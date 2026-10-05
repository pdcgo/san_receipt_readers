package san_receipt_readers

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/oned"
)

// shopeeBarcodePDF is a Shopee label no courier layout knows, with invented numbers: its "Resi" box
// says `box`, and a Code 128 barcode beside it encodes `coded`.
func shopeeBarcodePDF(t *testing.T, box, coded string) []byte {
	t.Helper()

	const width, height = 300, 60
	matrix, err := oned.NewCode128Writer().Encode(coded, gozxing.BarcodeFormat_CODE_128, width, height, nil)
	if err != nil {
		t.Fatal(err)
	}
	pixels := make([]byte, width*height*3)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if !matrix.Get(x, y) {
				copy(pixels[3*(y*width+x):], []byte{255, 255, 255})
			}
		}
	}

	w := newPDF()
	font := w.helvetica()
	image := w.rgbImage(width, height, pixels)
	page := []text{}
	for i, s := range []string{
		"STD",
		"ZZZ-01 No. Resi: " + box,
		"Penerima: Contoh Pengirim: Toko Contoh",
		"628000000000",
		"No. Pesanan: 260101ABCDEF12",
	} {
		page = append(page, text{x: 10, y: 400 - float64(i)*20, size: 8, s: s})
	}
	content := helveticaContent(page) + "q 150 0 0 30 20 200 cm /Im1 Do Q\n"
	w.page(fmt.Sprintf("/Font << /F1 %d 0 R >> /XObject << /Im1 %d 0 R >>", font, image), content)

	return w.bytes()
}

// A number no courier layout knows is read from the Resi box, because the barcode says the same.
func TestAShopeeResiConfirmedByItsBarcodeIsRead(t *testing.T) {
	got, err := Extract(bytes.NewReader(shopeeBarcodePDF(t, "ZZ0000000001", "ZZ0000000001")))
	if err != nil {
		t.Fatal(err)
	}

	assertNumbers(t, got, "ZZ0000000001", "260101ABCDEF12")
}

// The box and the barcode disagree — a box that cut its number short does — so nothing is read.
func TestAShopeeResiItsBarcodeContradictsIsUnknown(t *testing.T) {
	_, err := Extract(bytes.NewReader(shopeeBarcodePDF(t, "ZZ000000000", "ZZ0000000001")))
	if !errors.Is(err, errUnknownLayout) {
		t.Fatalf("err = %v, want errUnknownLayout", err)
	}
}
