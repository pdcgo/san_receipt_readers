package san_receipt_readers

import (
	"bytes"
	"errors"
	"testing"
)

// shopeeResiPDF is a standard Shopee label with invented numbers that prints its tracking number ONCE,
// in its "No. Resi" box beside the sort code, with no strips repeating it (the Reguler and ECO
// labels). `resi` is what the box shows, and `wrapped` what it pushed onto the next line.
func shopeeResiPDF(header, resi, wrapped string) []byte {
	return shopeePDF(otherLogo,
		header,
		"BTH10000 No. Resi: "+resi,
		wrapped,
		"Penerima: Contoh Pengirim: Toko Contoh",
		"628000000000",
		"Berat: 400 gr",
		"No. Pesanan: 260101ABCDEF12",
		"Pesan: (260101ABCDEF12)",
	)
}

func TestExtractReadsAShopeeCMLabel(t *testing.T) {
	got, err := Extract(bytes.NewReader(shopeeResiPDF("Reguler", "CM00000000001", "")))
	if err != nil {
		t.Fatal(err)
	}

	assertNumbers(t, got, "CM00000000001", "260101ABCDEF12")
}

// A box too narrow for the number wraps it (the SPX ECO label does). Printed once, the part left in
// the box is all there is — and it is one digit short, so it is not read: the label stays unknown.
func TestACMNumberTheBoxCutShortIsNotRead(t *testing.T) {
	_, err := Extract(bytes.NewReader(shopeeResiPDF("Reguler", "CM0000000000", "1")))
	if !errors.Is(err, errUnknownLayout) {
		t.Fatalf("err = %v, want errUnknownLayout", err)
	}
}
