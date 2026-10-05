package san_receipt_readers

import (
	"bytes"
	"fmt"
	"strings"
)

// A minimal PDF writer, for the tests only — so CI exercises Extract end to end on synthetic
// labels instead of on a real one, which would put a buyer's address in a public repository.

type pdfWriter struct {
	// objects[0] is the catalog and objects[1] the page tree; both are written last, once the
	// pages are known.
	objects []string
	pages   []int
}

func newPDF() *pdfWriter {
	return &pdfWriter{objects: []string{"", ""}}
}

// add stores an object and returns its number.
func (w *pdfWriter) add(object string) int {
	w.objects = append(w.objects, object)
	return len(w.objects)
}

func (w *pdfWriter) stream(dict, data string) int {
	return w.add(fmt.Sprintf("<< %s /Length %d >>\nstream\n%s\nendstream", dict, len(data), data))
}

func (w *pdfWriter) page(resources, content string) {
	contents := w.stream("", content)
	w.pages = append(w.pages, w.add(fmt.Sprintf(
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 300 420] /Resources << %s >> /Contents %d 0 R >>",
		resources, contents,
	)))
}

// helvetica is a standard font: the PDF carries no widths for it, so the library reports none.
func (w *pdfWriter) helvetica() int {
	return w.add("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>")
}

// type0 is a composite font like the ones both real labels use: two-byte codes equal to the
// character (Identity-H), a ToUnicode map, and every printable character `width` thousandths of
// an em wide. The library reads none of those widths — which is what positionGlyphs is for.
func (w *pdfWriter) type0(name string, width int) int {
	return w.type0Range(name, ' ', '~', width)
}

// type0Range is type0 with widths listed only for the characters first to last — the way a SUBSET
// font lists only the glyphs it carries. Any other character gets the default width, DW 0.
func (w *pdfWriter) type0Range(name string, first, last rune, width int) int {
	cmap := strings.Join([]string{
		"/CIDInit /ProcSet findresource begin",
		"12 dict begin",
		"begincmap",
		"/CMapName /Test def",
		"1 begincodespacerange",
		"<0000> <FFFF>",
		"endcodespacerange",
		"1 beginbfrange",
		"<0020> <007E> <0020>",
		"endbfrange",
		"endcmap",
		"CMapName currentdict /CMap defineresource pop",
		"end",
		"end",
	}, "\n")
	toUnicode := w.stream("", cmap)

	cid := w.add(fmt.Sprintf(
		"<< /Type /Font /Subtype /CIDFontType2 /BaseFont /%s "+
			"/CIDSystemInfo << /Registry (Adobe) /Ordering (Identity) /Supplement 0 >> "+
			"/DW 0 /W [%d %d %d] /CIDToGIDMap /Identity >>",
		name, first, last, width,
	))

	return w.add(fmt.Sprintf(
		"<< /Type /Font /Subtype /Type0 /BaseFont /%s /Encoding /Identity-H "+
			"/DescendantFonts [%d 0 R] /ToUnicode %d 0 R >>",
		name, cid, toUnicode,
	))
}

// form is a Form XObject: a drawing of its own, with its own resources, that a page calls with "Do".
// `matrix` is its six-number placement, or "" for none.
func (w *pdfWriter) form(resources, content, matrix string) int {
	placement := ""
	if matrix != "" {
		placement = fmt.Sprintf(" /Matrix [%s]", matrix)
	}

	return w.stream(fmt.Sprintf(
		"/Type /XObject /Subtype /Form /BBox [0 0 300 420]%s /Resources << %s >>",
		placement, resources,
	), content)
}

// buildFormPDF writes one page whose whole face is ONE form drawing `page` in Helvetica — the way
// the Lazada label is drawn.
func buildFormPDF(page []text) []byte {
	w := newPDF()
	font := w.helvetica()
	form := w.form(fmt.Sprintf("/Font << /F1 %d 0 R >>", font), helveticaContent(page), "")
	w.page(fmt.Sprintf("/XObject << /Fm1 %d 0 R >>", form), "q /Fm1 Do Q\n")

	return w.bytes()
}

// rgbImage is an uncompressed w×h image whose pixels are `pixels`, three bytes each.
func (w *pdfWriter) rgbImage(width, height int, pixels []byte) int {
	return w.stream(fmt.Sprintf(
		"/Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /DeviceRGB /BitsPerComponent 8",
		width, height,
	), string(pixels))
}

// rgbImageWithMask is rgbImage with an alpha mask (SMask) — the way the J&T Cargo logo carries its
// whole shape, over solid-black colour data.
func (w *pdfWriter) rgbImageWithMask(width, height int, pixels, alpha []byte) int {
	mask := w.stream(fmt.Sprintf(
		"/Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /DeviceGray /BitsPerComponent 8",
		width, height,
	), string(alpha))

	return w.stream(fmt.Sprintf(
		"/Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /DeviceRGB /BitsPerComponent 8 /SMask %d 0 R",
		width, height, mask,
	), string(pixels))
}

func (w *pdfWriter) bytes() []byte {
	kids := []string{}
	for _, p := range w.pages {
		kids = append(kids, fmt.Sprintf("%d 0 R", p))
	}
	w.objects[0] = "<< /Type /Catalog /Pages 2 0 R >>"
	w.objects[1] = fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(kids, " "), len(w.pages))

	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n")

	offsets := make([]int, len(w.objects))
	for i, object := range w.objects {
		offsets[i] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, object)
	}

	// Every xref entry is exactly 20 bytes, its own line ending included.
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(w.objects)+1)
	for _, offset := range offsets {
		fmt.Fprintf(&b, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(w.objects)+1, xref)

	return b.Bytes()
}

// helveticaContent draws each text in Helvetica.
//
// Helvetica carries no widths, so the library would draw every glyph of a string at the same x.
// Character spacing (Tc) advances each one by half its size, the way the J&T label's generator
// positions every glyph itself.
func helveticaContent(page []text) string {
	var cs strings.Builder
	for _, t := range page {
		// Rotation is the text matrix: [0 1 -1 0] turns it 90°.
		matrix := "1 0 0 1"
		if t.rotated {
			matrix = "0 1 -1 0"
		}
		fmt.Fprintf(&cs, "BT /F1 %g Tf %g Tc %s %g %g Tm (%s) Tj ET\n",
			t.size, t.size*0.5, matrix, t.x, t.y, escape(t.s))
	}

	return cs.String()
}

// buildPDF writes one Helvetica page per argument.
func buildPDF(pages ...[]text) []byte {
	w := newPDF()
	font := w.helvetica()
	for _, page := range pages {
		w.page(fmt.Sprintf("/Font << /F1 %d 0 R >>", font), helveticaContent(page))
	}

	return w.bytes()
}

// buildPDFContent writes one page with a raw content stream.
func buildPDFContent(content string) []byte {
	w := newPDF()
	font := w.helvetica()
	w.page(fmt.Sprintf("/Font << /F1 %d 0 R >>", font), content)

	return w.bytes()
}

// hex16 is s as a two-byte-per-character hex string, for a Type0 font with Identity-H.
func hex16(s string) string {
	var b strings.Builder
	b.WriteByte('<')
	for _, r := range s {
		fmt.Fprintf(&b, "%04X", r)
	}
	b.WriteByte('>')

	return b.String()
}

func escape(s string) string {
	return strings.NewReplacer(`\`, `\\`, `(`, `\(`, `)`, `\)`).Replace(s)
}
