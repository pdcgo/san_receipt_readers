package san_receipt_readers

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/ledongthuc/pdf"
)

// A subset font's BaseFont carries a tag ("AAAEKH+ArialMT") that the glyphs' font name does not.
// Keyed by the full name, its widths were never found, and kerning split the word again.
func TestASubsetFontsWidthsAreFound(t *testing.T) {
	got := linesOfPDF(t, kernedPageIn("ABCDEF+TestSans"), true)

	want := []string{"INSTANT 9XYZ", "DKI JAKARTA"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
}

// Two subsets of one typeface share a name, and each lists only its own glyphs. The KiriminAja label
// draws from both; taking the first one's widths left the other's glyphs without theirs.
func TestSubsetsOfOneFontShareTheirWidths(t *testing.T) {
	w := newPDF()
	first := w.type0Range("AAAAAA+TestSans", 'A', 'M', 600)
	second := w.type0Range("TestSans", 'N', 'Z', 500)
	w.page(fmt.Sprintf("/Font << /F1 %d 0 R /F2 %d 0 R >>", first, second), "")
	data := w.bytes()

	reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}

	m := fontMetricsOf(reader.Page(1))["TestSans"]
	if m.widths['A'] != 600 || m.widths['Z'] != 500 {
		t.Fatalf("A = %v, Z = %v — want both subsets' widths under one name", m.widths['A'], m.widths['Z'])
	}
}
