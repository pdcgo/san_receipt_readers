package san_receipt_readers

import (
	"bytes"
	"errors"
	"testing"
)

// The TikTok Shop instant label's structure with invented values: a sort code, the pickup code large
// under its caption, the recipient as caption rows (the phone masked), the address under "Alamat :",
// the sender small, and the order id under "Nomor Order:".
func tiktokInstantPDF(code string) []byte {
	return buildPDF([]text{
		{x: 210, y: 396, size: 19, s: "GKS"},
		{x: 13, y: 379, size: 10, s: "Kode Pengambilan"},
		{x: 145, y: 366, size: 28, s: code},
		{x: 15, y: 350, size: 10, s: "Penerima : Budi"},
		{x: 15, y: 338, size: 10, s: "Nomor Telepon : (+62)81*******00"},
		{x: 15, y: 327, size: 10, s: "Alamat :"},
		{x: 15, y: 315, size: 10, s: "Jl. Mawar No. 1,"},
		{x: 15, y: 304, size: 10, s: "Kota Contoh, Banten"},
		{x: 15, y: 252, size: 6, s: "Pengirim: Toko (+62)85*******00"},
		{x: 83, y: 238, size: 10, s: "Nomor Order: 585600000000000001"},
	})
}

func TestExtractReadsATikTokInstantLabel(t *testing.T) {
	got, err := Extract(bytes.NewReader(tiktokInstantPDF("AB1234")))
	if err != nil {
		t.Fatal(err)
	}

	// No tracking number is printed: the receipt is the pickup code. The masked phone is not read.
	want := ReceiptData{
		Receipt:      "AB1234",
		OrderID:      "585600000000000001",
		CustomerName: "Budi",
		Address:      "Jl. Mawar No. 1, Kota Contoh, Banten",
	}
	if got != want {
		t.Fatalf("got  %+v\nwant %+v", got, want)
	}
}

// A word with no digit under the caption is not a pickup code, so nothing is read.
func TestATikTokPickupCodeHasADigit(t *testing.T) {
	_, err := Extract(bytes.NewReader(tiktokInstantPDF("ABCDEF")))
	if !errors.Is(err, errUnknownLayout) {
		t.Fatalf("err = %v, want errUnknownLayout", err)
	}
}
