package san_receipt_readers

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/ledongthuc/pdf"
)

// linesOfPDF opens a synthetic PDF and rebuilds its first page's lines, the way Extract does.
func linesOfPDF(t *testing.T, data []byte, metrics bool) []string {
	t.Helper()

	reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	page := reader.Page(1)

	m := map[string]fontMetrics{}
	if metrics {
		m = fontMetricsOf(page)
	}

	return pageLines(pageTexts(page), m)
}

// kernedPage draws text the way PDFium does in a Type0 font: a word split into two text-shows
// at a kerning pair, and a space drawn as a GAP between runs rather than as a glyph. Every glyph
// is 0.6 em wide at size 10, so a run of n glyphs is 6n points long.
func kernedPage() []byte {
	return kernedPageIn("TestSans")
}

// kernedPageIn is kernedPage in a font of the given BaseFont.
func kernedPageIn(name string) []byte {
	w := newPDF()
	font := w.type0(name, 600)

	show := func(x, y float64, s string) string {
		return fmt.Sprintf("BT /F1 10 Tf 1 0 0 1 %g %g Tm %s Tj ET\n", x, y, hex16(s))
	}
	content := show(20, 300, "INS") + // ends at 38
		show(38, 300, "TANT") + // touches it: one word
		show(200, 300, "9XYZ") + // far away: another column
		show(20, 280, "DKI") + // ends at 38
		show(40.8, 280, "JAKARTA") // 0.28 em later: a space drawn as a gap

	w.page(fmt.Sprintf("/Font << /F1 %d 0 R >>", font), content)

	return w.bytes()
}

func TestLinesJoinAWordSplitAtAKerningPair(t *testing.T) {
	got := linesOfPDF(t, kernedPage(), true)

	want := []string{"INSTANT 9XYZ", "DKI JAKARTA"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
}

// The regression the widths exist for: without them a run is stacked on its first glyph, its
// end is unknown, and the kerning split reads as a word break — "INS TANT". Inside an order id
// that is a truncated id, read without an error.
func TestLinesWithoutWidthsSplitTheKernedWord(t *testing.T) {
	got := linesOfPDF(t, kernedPage(), false)

	if !strings.Contains(strings.Join(got, "|"), "INS TANT") {
		t.Fatalf("got %q — the fixture no longer reproduces the stacked-run split", got)
	}
}

// A 90° turn is cos 90° in the text matrix, which a generator writes as a rounding remainder
// rather than as zero — the SPX label's right edge comes back at size 3.7e-16. Such text must be
// dropped as rotated, not merged into the line beside it.
func TestLinesDropTextTurnedByANearZeroMatrix(t *testing.T) {
	content := "BT /F1 8 Tf 4 Tc 0.00000000000000006 1 -1 0.00000000000000006 280 300 Tm (SPXID00000000000A) Tj ET\n" +
		"BT /F1 8 Tf 4 Tc 1 0 0 1 20 300 Tm (Pengirim:) Tj ET\n"

	got := linesOfPDF(t, buildPDFContent(content), true)

	if strings.Join(got, "|") != "Pengirim:" {
		t.Fatalf("got %q, want only the upright text", got)
	}
}

// The KiriminAja label fakes bold by drawing a run twice, 0.034 em apart, and every bold glyph came
// out doubled: "NNOONN--CCOODD". A letter that really repeats ("FF", "00") is a glyph's width on.
func TestLinesReadARunDrawnTwiceOnce(t *testing.T) {
	w := newPDF()
	font := w.type0("TestSans", 600)

	show := func(x, y float64, s string) string {
		return fmt.Sprintf("BT /F1 10 Tf 1 0 0 1 %g %g Tm %s Tj ET\n", x, y, hex16(s))
	}
	content := show(20, 300, "NON-COD") +
		show(120, 300, "DROP-OFF") +
		show(20.34, 300, "NON-COD") +
		show(120.34, 300, "DROP-OFF") +
		show(20, 280, "1200")
	w.page(fmt.Sprintf("/Font << /F1 %d 0 R >>", font), content)

	got := linesOfPDF(t, w.bytes(), true)

	want := []string{"NON-COD DROP-OFF", "1200"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
}
