package san_receipt_readers

import (
	"bytes"
	"testing"
)

// lazadaPDF is the Lazada label's structure with invented values, drawn inside one form as the real
// one is: the sender's column left, the recipient's right, a sender's address line between two of
// the recipient's, a larger address tag ("Rumah") and sort code, and an item line in the recipient's
// column far below the address. In this fixture's font a glyph is half its size wide.
func lazadaPDF() []byte {
	return buildFormPDF([]text{
		{x: 14, y: 408, size: 8, s: "Diserahkan ke"},
		{x: 90, y: 408, size: 8, s: "Diantar oleh"},
		{x: 157, y: 408, size: 10, s: "STANDARD"},
		{x: 106, y: 293, size: 10, s: "LXAD-0000000001"},
		{x: 6, y: 182, size: 7, s: "Pengirim: Toko Contoh"},
		{x: 108, y: 182, size: 7, s: "Penerima: Budi Santoso"},
		{x: 261, y: 182, size: 8, s: "Rumah"},
		{x: 108, y: 168, size: 7, s: "Jl. Mawar No. 1 -"},
		{x: 108, y: 160, size: 7, s: "Sukamaju, Kab. Contoh"},
		{x: 6, y: 154, size: 7, s: "Banten, Kota Tangerang"},
		{x: 249, y: 156, size: 12, s: "XYZ01"},
		{x: 108, y: 152, size: 7, s: "Barat, Jawa Barat"},
		{x: 200, y: 119, size: 6, s: "lzd.co/pengembalian-gratis"},
		{x: 51, y: 114, size: 7, s: "1234000000000001"},
		{x: 108, y: 90, size: 7, s: "Item Contoh 1"},
	})
}

func TestExtractReadsALazadaLabel(t *testing.T) {
	got, err := Extract(bytes.NewReader(lazadaPDF()))
	if err != nil {
		t.Fatal(err)
	}

	// The receipt is the courier's tracking number, and the order id the sixteen-digit order number
	// (a-lazada-receipt-is-its-tracking-number).
	want := ReceiptData{
		Receipt:      "LXAD-0000000001",
		OrderID:      "1234000000000001",
		CustomerName: "Budi Santoso",
		// Not the sender's line between, not the tag, not the item line 60 points below.
		Address: "Jl. Mawar No. 1 - Sukamaju, Kab. Contoh Barat, Jawa Barat",
	}
	if got != want {
		t.Fatalf("got  %+v\nwant %+v", got, want)
	}
}

func TestLEXTrackingIsReadByItsShape(t *testing.T) {
	cases := []struct {
		lines []string
		want  string
	}{
		{[]string{"LXAD-0000000001"}, "LXAD-0000000001"},
		{[]string{"LXAD-000000001"}, ""},
		// Another prefix of the same shape is read: Lazada prints several ("LXAD-", "JNAP-").
		{[]string{"JNAP-0000000001"}, "JNAP-0000000001"},
		// A partner courier's own shape: J&T's two capitals and digits.
		{[]string{"JZ0000000001"}, "JZ0000000001"},
		// Sixteen digits are the order number, not a tracking number.
		{[]string{"2700000000000001"}, ""},
		// Not the shape: three capitals, a missing dash.
		{[]string{"LXA-0000000001"}, ""},
		{[]string{"LXAD0000000001"}, ""},
		{[]string{"LXAD-0000000001", "LXAD-0000000002"}, ""},
	}

	for _, c := range cases {
		if got := readLEXTracking(c.lines); got != c.want {
			t.Errorf("%q → %q, want %q", c.lines, got, c.want)
		}
	}
}
