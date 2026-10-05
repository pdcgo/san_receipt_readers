package san_receipt_readers

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
)

// qrPicturePDF is a page whose only content is one image: a QR code saying `contents`, the way the
// J&T label printed as a picture carries its tracking number. With `withText`, the page also prints a
// line of text that names no layout.
func qrPicturePDF(t *testing.T, contents string, withText bool) []byte {
	t.Helper()

	const size = 200
	matrix, err := qrcode.NewQRCodeWriter().Encode(contents, gozxing.BarcodeFormat_QR_CODE, size, size, nil)
	if err != nil {
		t.Fatal(err)
	}
	pixels := make([]byte, size*size*3)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			if !matrix.Get(x, y) {
				copy(pixels[3*(y*size+x):], []byte{255, 255, 255})
			}
		}
	}

	w := newPDF()
	image := w.rgbImage(size, size, pixels)
	resources := fmt.Sprintf("/XObject << /Im1 %d 0 R >>", image)
	content := "q 200 0 0 200 50 50 cm /Im1 Do Q\n"
	if withText {
		resources += fmt.Sprintf(" /Font << /F1 %d 0 R >>", w.helvetica())
		content += helveticaContent([]text{{x: 10, y: 400, size: 8, s: "Weight: 1 KG"}})
	}
	w.page(resources, content)

	return w.bytes()
}

func TestExtractReadsAJNTLabelPrintedAsAPicture(t *testing.T) {
	got, err := Extract(bytes.NewReader(qrPicturePDF(t, "JX0000000001", false)))
	if err != nil {
		t.Fatal(err)
	}

	// Only the tracking number: the order id and the recipient are pixels.
	if got != (ReceiptData{Receipt: "JX0000000001"}) {
		t.Fatalf("got %+v", got)
	}
}

// A picture's code is read only when it has a J&T tracking number's shape. The Shopee instant label's
// QR carries the ORDER number; read from a picture, it must not become a receipt.
func TestAPictureWhoseCodeIsNotATrackingNumberIsUnknown(t *testing.T) {
	_, err := Extract(bytes.NewReader(qrPicturePDF(t, "261002AB12CD34", false)))
	// It carries a code, so it may be a label to learn: unknown, not a non-label.
	if !errors.Is(err, errUnknownLayout) || errors.Is(err, ErrNotShippingLabel) {
		t.Fatalf("err = %v, want errUnknownLayout and not ErrNotShippingLabel", err)
	}
}

// A page with text is read from its text. Its images are scanned for codes only on a Shopee label, and
// only to confirm its Resi box (shopeeBarcodeLayout), so this label stays unknown rather than half-read
// from a barcode.
func TestAPageWithTextIsNotScannedForCodes(t *testing.T) {
	_, err := Extract(bytes.NewReader(qrPicturePDF(t, "JX0000000001", true)))
	if !errors.Is(err, errUnknownLayout) {
		t.Fatalf("err = %v, want errUnknownLayout", err)
	}
}

func TestPictureTrackingIsOneNumber(t *testing.T) {
	if got := readPictureTracking([]string{"JX0000000001", "JX0000000001"}); got != "JX0000000001" {
		t.Errorf("the QR and the Code 128 agreeing: got %q", got)
	}
	if got := readPictureTracking([]string{"JX0000000001", "JX0000000002"}); got != "" {
		t.Errorf("two different numbers: got %q, want nothing", got)
	}
}
