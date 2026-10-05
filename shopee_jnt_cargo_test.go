package san_receipt_readers

import (
	"bytes"
	"errors"
	"testing"
)

// The Shopee J&T Cargo label's structure with invented numbers: twelve digits in strips and in the
// Resi box, and an order number its box cut short, printed whole in the "Pesan:" line.
func shopeeJNTCargoPDF(logo []byte) []byte {
	return shopeePDF(logo,
		"200000000001 200000000001",
		"CARGO",
		"TTE-TTE000A-TA No. Resi: 200000000001",
		"Penerima: Contoh Pengirim: Toko Contoh",
		"628000000000",
		"Berat:3000 gr COD:Rp0",
		"No. Pesanan: 260101ABCDEF\x83", // the ellipsis as one byte: this fixture's Helvetica has no UTF-8
		"Pesan: (260101ABCDEF12)",
		"200000000001 200000000001",
	)
}

func TestExtractReadsAShopeeJNTCargoLabel(t *testing.T) {
	useFakeLogo(t, courierJNTCargo)

	got, err := Extract(bytes.NewReader(shopeeJNTCargoPDF(fakeLogo)))
	if err != nil {
		t.Fatal(err)
	}

	// The order number whole, not the "260101ABCDEF" its box cut it to.
	assertNumbers(t, got, "200000000001", "260101ABCDEF12")
}

// Twelve digits are SiCepat's shape too: without J&T Cargo's logo, the label stays unknown.
func TestAShopeeCargoLabelWithAnUnknownLogoIsUnknown(t *testing.T) {
	useFakeLogo(t, courierJNTCargo)

	_, err := Extract(bytes.NewReader(shopeeJNTCargoPDF(otherLogo)))
	if !errors.Is(err, errUnknownLayout) {
		t.Fatalf("err = %v, want errUnknownLayout", err)
	}
}

// A box too narrow for the order number cuts it short with an ellipsis. The stump is never the order
// number: the whole one comes from the "Pesan:" line, and only when it starts with the stump.
func TestACutShopeeOrderNumberIsReadWhole(t *testing.T) {
	cases := []struct {
		lines []string
		want  string
	}{
		{[]string{"No. Pesanan: 260101ABCDEF…", "Pesan: (260101ABCDEF12)"}, "260101ABCDEF12"},
		// No whole copy: nothing, rather than the stump.
		{[]string{"No. Pesanan: 260101ABCDEF…"}, ""},
		// A whole copy of another number: nothing, rather than a guess.
		{[]string{"No. Pesanan: 260101ABCDEF…", "Pesan: (260101ZZZZZZ12)"}, ""},
	}

	for _, c := range cases {
		if got := readShopeeOrderID(c.lines); got != c.want {
			t.Errorf("%q → %q, want %q", c.lines, got, c.want)
		}
	}
}
