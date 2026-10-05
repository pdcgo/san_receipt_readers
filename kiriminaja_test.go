package san_receipt_readers

import (
	"bytes"
	"errors"
	"testing"
)

// kiriminAjaPage is the KiriminAja label's structure with invented values: captions over their
// values, the recipient's column beside the sender's ("Dari"), and the sender's city on the same row
// as the recipient's first address line. awbX is where the tracking number is printed; its caption
// is at 190. In this fixture's font a glyph is 3 points.
func kiriminAjaPage(awbX float64) []byte {
	return buildPDF([]text{
		{x: 10, y: 400, size: 6, s: "ID Express Flat"},
		{x: 70, y: 400, size: 6, s: "No. Trx KiriminAja"},
		{x: 190, y: 400, size: 6, s: "AWB"},
		{x: 10, y: 392, size: 6, s: "Standard"},
		{x: 70, y: 392, size: 6, s: "OID-0000000001"},
		{x: awbX, y: 392, size: 6, s: "IDE7000000000001"},
		{x: 10, y: 370, size: 6, s: "Penerima"},
		{x: 150, y: 370, size: 6, s: "Dari"},
		{x: 10, y: 362, size: 6, s: "Budi Santoso"},
		{x: 150, y: 362, size: 6, s: "Toko Contoh"},
		{x: 10, y: 354, size: 6, s: "081200000000"},
		{x: 150, y: 354, size: 6, s: "081300000000"},
		// Ends at 118, short of the sender's column at 150.
		{x: 10, y: 346, size: 6, s: "Jl. Mawar No. 1, RT. 01 RW. 02, Kec."},
		{x: 150, y: 346, size: 6, s: "Tangerang, Banten"},
		{x: 10, y: 338, size: 6, s: "Contoh, Kab. Contoh, Jawa Tengah"},
		{x: 40, y: 326, size: 6, s: "10000 to 20000"},
		{x: 10, y: 316, size: 6, s: "Catatan :"},
	})
}

func TestExtractReadsAKiriminAjaLabel(t *testing.T) {
	got, err := Extract(bytes.NewReader(kiriminAjaPage(190)))
	if err != nil {
		t.Fatal(err)
	}

	// KiriminAja's own booking number is not the order id.
	assertNumbers(t, got, "IDE7000000000001", "")
}

// The first label that prints the recipient's phone. The sender's name, phone and city are the
// column right of "Dari", and the block ends at the first row off the margin.
func TestKiriminAjaRecipientIsTheLeftColumn(t *testing.T) {
	got, err := Extract(bytes.NewReader(kiriminAjaPage(190)))
	if err != nil {
		t.Fatal(err)
	}

	want := ReceiptData{
		Receipt:      "IDE7000000000001",
		Phone:        "081200000000",
		CustomerName: "Budi Santoso",
		Address:      "Jl. Mawar No. 1, RT. 01 RW. 02, Kec. Contoh, Kab. Contoh, Jawa Tengah",
	}
	if got != want {
		t.Fatalf("got  %+v\nwant %+v", got, want)
	}
}

// The caption is what names the tracking number. A code that is not under it is not read, and with
// no tracking number the label is unknown.
func TestKiriminAjaTrackingIsOnlyTheValueUnderItsCaption(t *testing.T) {
	_, err := Extract(bytes.NewReader(kiriminAjaPage(150)))
	if !errors.Is(err, errUnknownLayout) {
		t.Fatalf("err = %v, want errUnknownLayout", err)
	}
}
