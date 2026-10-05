package san_receipt_readers

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"strings"
	"testing"

	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/oned"
	"github.com/makiuchi-d/gozxing/qrcode"
)

// jpegImage stores a grey JPEG image, the way Microsoft Print To PDF stores each graphic of a label.
func (w *pdfWriter) jpegImage(t *testing.T, img *image.Gray) int {
	t.Helper()

	var data bytes.Buffer
	err := jpeg.Encode(&data, img, &jpeg.Options{Quality: 95})
	if err != nil {
		t.Fatal(err)
	}

	return w.stream(fmt.Sprintf(
		"/Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /DeviceGray /BitsPerComponent 8 /Filter /DCTDecode",
		img.Bounds().Dx(), img.Bounds().Dy(),
	), data.String())
}

// codeImage draws a barcode saying `contents`, black on white.
func codeImage(t *testing.T, writer gozxing.Writer, format gozxing.BarcodeFormat, contents string, w, h int) *image.Gray {
	t.Helper()

	matrix, err := writer.Encode(contents, format, w, h, nil)
	if err != nil {
		t.Fatal(err)
	}
	img := image.NewGray(image.Rect(0, 0, matrix.GetWidth(), matrix.GetHeight()))
	for y := 0; y < matrix.GetHeight(); y++ {
		for x := 0; x < matrix.GetWidth(); x++ {
			if !matrix.Get(x, y) {
				img.Pix[y*img.Stride+x] = 255
			}
		}
	}

	return img
}

// spxPicturesPDF is a page with no text and one JPEG per graphic: a blank "logo" LARGER than every
// code, a QR code and a Code 128 of the tracking number, and a Code 128 of the order number — the
// shape of shopee_std_01.pdf, with invented numbers.
func spxPicturesPDF(t *testing.T, qrTracking, barTracking, order string) []byte {
	t.Helper()

	logo := image.NewGray(image.Rect(0, 0, 900, 300))
	for i := range logo.Pix {
		logo.Pix[i] = 255
	}
	images := []*image.Gray{
		logo,
		codeImage(t, qrcode.NewQRCodeWriter(), gozxing.BarcodeFormat_QR_CODE, qrTracking, 200, 200),
		codeImage(t, oned.NewCode128Writer(), gozxing.BarcodeFormat_CODE_128, barTracking, 500, 120),
		codeImage(t, oned.NewCode128Writer(), gozxing.BarcodeFormat_CODE_128, order, 400, 80),
	}

	w := newPDF()
	names, content := []string{}, ""
	for i, img := range images {
		names = append(names, fmt.Sprintf("/Im%d %d 0 R", i, w.jpegImage(t, img)))
		content += fmt.Sprintf("q %d 0 0 %d 50 %d cm /Im%d Do Q\n", img.Bounds().Dx()/3, img.Bounds().Dy()/3, 50+150*i, i)
	}
	w.page(fmt.Sprintf("/XObject << %s >>", strings.Join(names, " ")), content)

	return w.bytes()
}

func TestExtractReadsAnSPXLabelPrintedAsJPEGs(t *testing.T) {
	got, err := Extract(bytes.NewReader(spxPicturesPDF(t, "SPXID000000000001", "SPXID000000000001", "261002AB12CD34")))
	if err != nil {
		t.Fatal(err)
	}

	// Both numbers come from barcodes; the recipient is outlines.
	want := ReceiptData{Receipt: "SPXID000000000001", OrderID: "261002AB12CD34"}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

// The QR code and the Code 128 disagreeing is two numbers, and a guess is never a receipt.
func TestAnSPXPictureWhoseCodesDisagreeIsUnknown(t *testing.T) {
	_, err := Extract(bytes.NewReader(spxPicturesPDF(t, "SPXID000000000001", "SPXID000000000002", "261002AB12CD34")))
	if !errors.Is(err, errUnknownLayout) {
		t.Fatalf("err = %v, want errUnknownLayout", err)
	}
}

// A JPEG is found in the file's bytes only at exactly its declared Length and size — never a
// neighbouring stream that merely starts like one.
func TestAJPEGIsFoundOnlyAtItsOwnLengthAndSize(t *testing.T) {
	raw := spxPicturesPDF(t, "SPXID000000000001", "SPXID000000000001", "261002AB12CD34")

	// The blank logo is the first image: 900×300.
	start := bytes.Index(raw, jpegStart)
	end := bytes.Index(raw[start:], []byte("\nendstream")) + start
	length := int64(end - start)

	if jpegImage(raw, length, 900, 300) == nil {
		t.Fatal("the logo at its own length and size: not found")
	}
	if jpegImage(raw, length-1, 900, 300) != nil {
		t.Error("one byte short: found, want nothing")
	}
	if jpegImage(raw, length, 300, 900) != nil {
		t.Error("the wrong size: found, want nothing")
	}
}
