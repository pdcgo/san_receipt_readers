package san_receipt_readers

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/ledongthuc/pdf"
)

// fingerprintsOf builds a one-page PDF carrying one 2×2 image with an alpha mask, and returns the
// page's fingerprints.
func fingerprintsOf(t *testing.T, alpha []byte) map[string]bool {
	t.Helper()

	black := make([]byte, 2*2*3)

	w := newPDF()
	image := w.rgbImageWithMask(2, 2, black, alpha)
	w.page(fmt.Sprintf("/XObject << /Im1 %d 0 R >>", image), "q 2 0 0 2 10 10 cm /Im1 Do Q\n")
	data := w.bytes()

	reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}

	return imageFingerprints(reader.Page(1))
}

// The J&T Cargo logo's colour data is solid black — its shape is entirely in its alpha mask. Two
// black images with different masks are different logos, so they must not share a fingerprint.
func TestFingerprintIncludesTheAlphaMask(t *testing.T) {
	a := fingerprintsOf(t, []byte{255, 0, 0, 255})
	b := fingerprintsOf(t, []byte{0, 255, 255, 0})

	if len(a) != 1 || len(b) != 1 {
		t.Fatalf("want one fingerprint each, got %v and %v", a, b)
	}
	for fp := range a {
		if b[fp] {
			t.Fatalf("two different masks share the fingerprint %s", fp)
		}
	}
}

// The Lazada label's images are inside its one form; a logo there is still a logo.
func TestAnImageInsideAFormIsFingerprinted(t *testing.T) {
	w := newPDF()
	image := w.rgbImage(2, 2, fakeLogo)
	form := w.form(fmt.Sprintf("/XObject << /Im1 %d 0 R >>", image), "q 2 0 0 2 10 10 cm /Im1 Do Q\n", "")
	w.page(fmt.Sprintf("/XObject << /Fm1 %d 0 R >>", form), "/Fm1 Do\n")
	data := w.bytes()

	reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}

	if got := imageFingerprints(reader.Page(1)); len(got) != 1 {
		t.Fatalf("got %v, want the one image inside the form", got)
	}
}
