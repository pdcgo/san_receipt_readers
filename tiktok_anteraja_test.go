package san_receipt_readers

import (
	"bytes"
	"errors"
	"testing"
)

// The Tokopedia-branded TikTok label's structure with invented values: the "TSA-" number alone, the
// usual masked recipient block, and the order id after "Order ID" with no "TT". Helvetica can't draw
// the full-width colon the real label uses; TestTikTokOrderIDIsReadWithEitherColon covers it.
func tiktokAnterAjaPDF(tracking string) []byte {
	return buildPDF([]text{
		{x: 233, y: 399, size: 13, s: "INS"},
		{x: 100, y: 334, size: 12, s: tracking},
		{x: 11, y: 317, size: 7.5, s: "Pengirim"},
		{x: 11, y: 306, size: 6.3, s: "Toko Contoh"},
		{x: 11, y: 247, size: 7.5, s: "Penerima"},
		{x: 168, y: 247, size: 6.3, s: "(+62)81*******00"},
		{x: 11, y: 236, size: 6.3, s: "b**i"},
		{x: 11, y: 224, size: 6.3, s: "Jawa Barat, Kota Contoh,"},
		{x: 11, y: 217, size: 6.3, s: "Jl. Mawar no. 1"},
		{x: 11, y: 182, size: 7.5, s: "Print Time: 2026-01-01 08:00"},
		{x: 11, y: 138, size: 6.5, s: "Order ID: 586000000000000001"},
	})
}

func TestExtractReadsATikTokAnterAjaLabel(t *testing.T) {
	got, err := Extract(bytes.NewReader(tiktokAnterAjaPDF("TSA-00000000001")))
	if err != nil {
		t.Fatal(err)
	}

	want := ReceiptData{
		Receipt: "TSA-00000000001",
		OrderID: "586000000000000001",
		// The name and the phone are masked: not read.
		Address: "Jawa Barat, Kota Contoh, Jl. Mawar no. 1",
	}
	if got != want {
		t.Fatalf("got  %+v\nwant %+v", got, want)
	}
}

// Eleven digits cut to ten are not the number: the label stays unknown.
func TestATSANumberCutShortIsNotRead(t *testing.T) {
	_, err := Extract(bytes.NewReader(tiktokAnterAjaPDF("TSA-0000000001")))
	if !errors.Is(err, errUnknownLayout) {
		t.Fatalf("err = %v, want errUnknownLayout", err)
	}
}
