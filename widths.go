package san_receipt_readers

import (
	"strings"

	"github.com/ledongthuc/pdf"
)

// fontMetrics is one font's glyph widths, in thousandths of an em, by character.
type fontMetrics struct {
	widths map[rune]float64
	// missing is the width of a glyph the table does not list (the font's DW).
	missing float64
}

func (m fontMetrics) width(r rune) float64 {
	w, ok := m.widths[r]
	if ok {
		return w
	}

	return m.missing
}

// fontMetricsOf reads the widths of the page's composite (Type0) fonts, and of the fonts its forms
// use, keyed by the name pdf.Text.Font carries: BaseFont without its subset tag ("AAAEKH+ArialMT"
// is "ArialMT"). Keyed by the full BaseFont, a subset font's widths were never found.
//
// The PDF library reads widths only for simple fonts. Both labels measured so far use Type0
// fonts, so every glyph came back with no width — and on the SPX label, where a whole word is
// one text-show, every glyph of the word came back at the SAME x. Simple fonts are left to the
// library, which already fills pdf.Text.W for them.
func fontMetricsOf(p pdf.Page) map[string]fontMetrics {
	out := map[string]fontMetrics{}

	eachResources(p.Resources(), func(res pdf.Value) {
		fonts := res.Key("Font")
		for _, name := range fonts.Keys() {
			font := pdf.Font{V: fonts.Key(name)}
			if font.V.Key("Subtype").Name() != "Type0" {
				continue
			}

			base := font.BaseFont()
			if i := strings.Index(base, "+"); i >= 0 {
				base = base[i+1:]
			}

			// Two subsets of one typeface share a name, and each lists only its own glyphs — the
			// KiriminAja label draws from "AAAURT+Inter-Regular" and "Inter-Regular" alike. Their
			// widths are merged: the first subset's alone left the other's glyphs a whole em wide.
			descendant := font.V.Key("DescendantFonts").Index(0)
			metrics, seen := out[base]
			if !seen {
				metrics = fontMetrics{widths: map[rune]float64{}, missing: 1000}
				if dw := descendant.Key("DW"); !dw.IsNull() {
					metrics.missing = dw.Float64()
				}
			}

			readCIDWidths(descendant.Key("W"), font.Encoder(), metrics.widths)
			out[base] = metrics
		}
	})

	return out
}

// readCIDWidths walks a CIDFont's W array, whose entries come in two shapes:
//
//	c [w1 w2 …]   consecutive CIDs from c, one width each
//	c1 c2 w       every CID from c1 to c2, the same width
//
// Each CID is decoded through the font's own ToUnicode map, so the table ends up keyed by the
// character pdf.Text carries rather than by a CID the library never exposes.
func readCIDWidths(w pdf.Value, enc pdf.TextEncoding, out map[rune]float64) {
	set := func(cid int64, width float64) {
		r := []rune(enc.Decode(string([]byte{byte(cid >> 8), byte(cid)})))
		if len(r) != 1 {
			return
		}
		if _, seen := out[r[0]]; !seen {
			out[r[0]] = width
		}
	}

	for i := 0; i+1 < w.Len(); {
		first := w.Index(i).Int64()
		next := w.Index(i + 1)

		if next.Kind() == pdf.Array {
			for j := 0; j < next.Len(); j++ {
				set(first+int64(j), next.Index(j).Float64())
			}
			i += 2
			continue
		}

		if i+2 >= w.Len() {
			return
		}
		last, width := next.Int64(), w.Index(i+2).Float64()
		// A CID is two bytes, so a range past 0xFFFF is malformed — refuse it rather than loop.
		if last < first || last > 0xFFFF {
			return
		}
		for cid := first; cid <= last; cid++ {
			set(cid, width)
		}
		i += 3
	}
}

// positionGlyphs puts each glyph where the page actually draws it.
//
// When the library has no width it cannot advance between the glyphs of one text-show, so a
// run comes back stacked on its first glyph's x. A run is recognised by exactly that: the same
// x, y, font and size as the glyph before it, in content order. Each glyph is moved past the one
// before by that glyph's width, and every glyph gets the width the library left empty.
func positionGlyphs(texts []pdf.Text, metrics map[string]fontMetrics) []pdf.Text {
	out := make([]pdf.Text, len(texts))
	copy(out, texts)

	widthOf := func(t pdf.Text) (float64, bool) {
		m, ok := metrics[t.Font]
		r := []rune(t.S)
		if !ok || len(r) == 0 {
			return 0, false
		}

		return m.width(r[0]) * t.FontSize / 1000, true
	}

	for i := 1; i < len(out); i++ {
		// The ORIGINAL positions decide what is one run; the moved ones are what get written.
		prev, cur := texts[i-1], texts[i]
		stacked := cur.W == 0 &&
			cur.X == prev.X &&
			cur.Y == prev.Y &&
			cur.Font == prev.Font &&
			cur.FontSize == prev.FontSize
		if !stacked {
			continue
		}

		w, ok := widthOf(prev)
		if !ok {
			continue
		}
		out[i].X = out[i-1].X + w
	}

	for i := range out {
		if out[i].W != 0 {
			continue
		}
		w, ok := widthOf(out[i])
		if ok {
			out[i].W = w
		}
	}

	return out
}
