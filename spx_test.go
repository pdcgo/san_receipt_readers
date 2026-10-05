package san_receipt_readers

import (
	"bytes"
	"errors"
	"testing"
)

// An SPX label's structure with invented numbers: the tracking number in strips along the top and
// the bottom, in its caption box, and on the "Pesan:" line. The logo plays no part — the tracking
// number names the courier.
func spxPDF(service string, resiBox ...string) []byte {
	lines := []string{
		"SPXID01234567890A SPXID01234567890A SPXID01234567890A",
		service,
	}
	lines = append(lines, resiBox...)
	lines = append(lines,
		"Berat: 700 gr COD Cek Dulu: Ya",
		"No.Pesanan: 260101ABCDEF12",
		"Pesan: (260101ABCDEF12) (SPXID01234567890A)",
		"SPXID01234567890A SPXID01234567890A SPXID01234567890A",
	)

	return shopeePDF(otherLogo, lines...)
}

func TestExtractReadsAnSPXStandardLabel(t *testing.T) {
	got, err := Extract(bytes.NewReader(spxPDF("STD", "ANC-A-02 No. Resi: SPXID01234567890A")))
	if err != nil {
		t.Fatal(err)
	}

	assertNumbers(t, got, "SPXID01234567890A", "260101ABCDEF12")
}

// The ECO label's box is too narrow for the number, so it wraps — and its caption is "Resi:",
// not "No. Resi:". The number is read whole all the same, from everywhere else it is printed.
func TestExtractReadsAnSPXLabelWhoseResiBoxWraps(t *testing.T) {
	got, err := Extract(bytes.NewReader(spxPDF("ECO", "Resi: SPXID01234567890", "A")))
	if err != nil {
		t.Fatal(err)
	}

	assertNumbers(t, got, "SPXID01234567890A", "260101ABCDEF12")
}

// A Shopee label carrying ANOTHER courier's number is not SPX, and is not yet known — it fails
// loudly so it arrives as a sample, rather than being read as SPX.
func TestShopeeLabelWithAnotherCouriersNumberIsUnknown(t *testing.T) {
	data := shopeePDF(otherLogo,
		"JP1234567890 JP1234567890",
		"No. Resi: JP1234567890",
		"No.Pesanan: 260101ABCDEF12",
	)

	_, err := Extract(bytes.NewReader(data))
	if !errors.Is(err, errUnknownLayout) {
		t.Fatalf("err = %v, want errUnknownLayout", err)
	}
}

func TestSPXTrackingIsTheNumberPrintedMostOften(t *testing.T) {
	cases := []struct {
		name  string
		lines []string
		want  string
	}{
		{
			"the wrapped fragment is outvoted",
			[]string{"Resi: SPXID01234567890", "A", "SPXID01234567890A SPXID01234567890A", "(SPXID01234567890A)"},
			"SPXID01234567890A",
		},
		{
			"printed once is not trusted",
			[]string{"No. Resi: SPXID01234567890A"},
			"",
		},
		{
			"a tie is not a guess",
			[]string{"SPXID01234567890A SPXID01234567890A", "SPXID09876543210A SPXID09876543210A"},
			"",
		},
		{
			"not an SPX number",
			[]string{"JP1234567890 JP1234567890"},
			"",
		},
	}

	for _, c := range cases {
		got := readSPXTracking(c.lines)
		if got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}
