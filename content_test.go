package san_receipt_readers

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/ledongthuc/pdf"
)

// textsOfPDF opens a synthetic PDF and returns its first page's glyphs, the way Extract reads them.
func textsOfPDF(t *testing.T, data []byte) (library, read []pdf.Text) {
	t.Helper()

	reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	page := reader.Page(1)

	return page.Content().Text, pageTexts(page)
}

// The Lazada label draws its whole face as one form, and the library read nothing from it.
func TestTextInsideAFormIsRead(t *testing.T) {
	data := buildFormPDF([]text{{x: 10, y: 300, size: 8, s: "Penerima: Budi"}})

	library, _ := textsOfPDF(t, data)
	if len(library) != 0 {
		t.Fatalf("the library read %d glyphs — the fixture no longer reproduces the empty page", len(library))
	}

	got := linesOfPDF(t, data, true)
	if strings.Join(got, "|") != "Penerima: Budi" {
		t.Fatalf("got %q", got)
	}
}

// A form inside a form: each one's matrix places it inside its caller, on top of the page's own cm. The
// outer one scales, so composing them in the wrong order lands the glyph elsewhere.
func TestNestedFormsArePlacedByTheirMatrices(t *testing.T) {
	w := newPDF()
	font := w.helvetica()
	inner := w.form(
		fmt.Sprintf("/Font << /F1 %d 0 R >>", font),
		"BT /F1 8 Tf 1 0 0 1 10 300 Tm (A) Tj ET\n",
		"1 0 0 1 5 -100",
	)
	outer := w.form(fmt.Sprintf("/XObject << /Fm2 %d 0 R >>", inner), "q /Fm2 Do Q\n", "2 0 0 2 50 0")
	w.page(fmt.Sprintf("/XObject << /Fm1 %d 0 R >>", outer), "q 1 0 0 1 0 10 cm /Fm1 Do Q\n")

	_, got := textsOfPDF(t, w.bytes())

	// (10,300) → inner (15,200) → outer, doubled (80,400) → the page's cm (80,410).
	if len(got) != 1 || got[0].X != 80 || got[0].Y != 410 || got[0].FontSize != 16 {
		t.Fatalf("got %+v, want one glyph at 80,410 size 16", got)
	}
}

// A form that draws itself must end, not recurse until the stack runs out. Drawn three times over, the
// depth bound alone would still allow 3,280 drawings; the page-wide budget caps them.
func TestAFormThatDrawsItselfEnds(t *testing.T) {
	cases := map[string]struct {
		draws string
		max   int
	}{
		"once":        {"/Fm1 Do\n", maxFormDepth},
		"three times": {"/Fm1 Do /Fm1 Do /Fm1 Do\n", maxForms},
	}

	for name, c := range cases {
		w := newPDF()
		font := w.helvetica()
		self := len(w.objects) + 1 // the number the form is about to get
		form := w.form(
			fmt.Sprintf("/Font << /F1 %d 0 R >> /XObject << /Fm1 %d 0 R >>", font, self),
			"BT /F1 8 Tf 1 0 0 1 10 300 Tm (A) Tj ET\n"+c.draws,
			"",
		)
		if form != self {
			t.Fatalf("the form is object %d, not %d", form, self)
		}
		w.page(fmt.Sprintf("/XObject << /Fm1 %d 0 R >>", form), "/Fm1 Do\n")

		_, got := textsOfPDF(t, w.bytes())

		if len(got) == 0 || len(got) > c.max {
			t.Errorf("%s: %d glyphs, want between 1 and %d", name, len(got), c.max)
		}
	}
}

// Without forms, pageTexts must read exactly what the library reads: it replaced Page.Content in
// Extract. One stream with every text operator the interpreter handles, and a Type0 page.
func TestPageTextsMatchesTheLibraryWithoutForms(t *testing.T) {
	operators := "q 1 0 0 1 5 5 cm BT /F1 8 Tf 2 Tc 110 Tz 1 Ts 12 TL 1 0 0 1 10 300 Tm (Ab) Tj T* (Cd) Tj " +
		"0 -10 Td (Ef) Tj 0 -10 TD (Gh) Tj [(I) -500 (j)] TJ (Kl) ' 1 2 (Mn) \" ET Q " +
		"BT /F1 6 Tf 20 100 Td (Op) Tj ET\n"

	for name, data := range map[string][]byte{
		"every operator": buildPDFContent(operators),
		"type0":          kernedPage(),
	} {
		library, read := textsOfPDF(t, data)
		if len(library) == 0 {
			t.Fatalf("%s: the library read nothing — the fixture tests nothing", name)
		}
		if !reflect.DeepEqual(library, read) {
			t.Errorf("%s:\nlibrary %+v\nread    %+v", name, library, read)
		}
	}
}
