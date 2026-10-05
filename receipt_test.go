package san_receipt_readers

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// These run on SYNTHETIC labels built below, with invented numbers, so CI needs no real label —
// a real one carries a buyer's street address, and this repository is public. The real samples
// are exercised by samples_test.go when they are on disk.

// text is one string drawn on a synthetic page.
type text struct {
	x, y, size float64
	s          string
	// rotated draws it at 90°, the way the J&T label prints the tracking number up its edges.
	rotated bool
}

// jntPage is the J&T label's structure with invented values.
func jntPage(tracking, orderID string) []text {
	return []text{
		{x: 130, y: 396, size: 20, s: "ECO"},
		{x: 284, y: 300, size: 6, s: tracking, rotated: true},
		{x: 20, y: 375, size: 7, s: "Pengirim : Toko Contoh"},
		{x: 20, y: 282, size: 6, s: "Weight : 0.400 KG"},
		{x: 150, y: 282, size: 6, s: "Ship : 01-01-2026"},
		{x: 140, y: 250, size: 30, s: "COD"},
		{x: 120, y: 160, size: 20, s: "000-XXX00-00"},
		{x: 120, y: 80, size: 20, s: tracking},
		{x: 60, y: 70, size: 5, s: "Syarat dan ketentuan pengiriman dapat dilihat pada website www.jet.co.id"},
		{x: 20, y: 55, size: 7, s: "Order Id :"},
		{x: 70, y: 55, size: 7, s: orderID},
		{x: 180, y: 55, size: 7, s: "Estimated Date:"},
	}
}

func TestExtractReadsAJNTLabel(t *testing.T) {
	got, err := Extract(bytes.NewReader(buildPDF(jntPage("JY1234567890", "581234567890123456"))))
	if err != nil {
		t.Fatal(err)
	}

	assertNumbers(t, got, "JY1234567890", "581234567890123456")
}

// The other prefix seen on J&T numbers — the reader finds the number, it does not judge it.
func TestExtractDoesNotPinTheJNTPrefix(t *testing.T) {
	got, err := Extract(bytes.NewReader(buildPDF(jntPage("JP1234567890", "581234567890123456"))))
	if err != nil {
		t.Fatal(err)
	}

	if got.Receipt != "JP1234567890" {
		t.Fatalf("receipt = %q", got.Receipt)
	}
}

// An empty "Order Id :" is followed on its line by "Estimated Date:" — that word is not an id.
func TestExtractLeavesAnUnprintedOrderIDEmpty(t *testing.T) {
	got, err := Extract(bytes.NewReader(buildPDF(jntPage("JY1234567890", ""))))
	if err != nil {
		t.Fatal(err)
	}

	if got.OrderID != "" {
		t.Fatalf("order id = %q, want empty", got.OrderID)
	}
	if got.Receipt != "JY1234567890" {
		t.Fatalf("receipt = %q", got.Receipt)
	}
}

func TestExtractRefusesAnUnknownLayout(t *testing.T) {
	page := []text{
		{x: 20, y: 300, size: 10, s: "Some other courier"},
		{x: 20, y: 280, size: 10, s: "AB1234567890"},
	}

	_, err := Extract(bytes.NewReader(buildPDF(page)))
	if !errors.Is(err, errUnknownLayout) {
		t.Fatalf("err = %v, want errUnknownLayout", err)
	}
}

// A bulk print must never be read as its first page — that would hand this order another
// order's numbers.
func TestExtractRefusesMoreThanOneLabel(t *testing.T) {
	pdf := buildPDF(
		jntPage("JY1234567890", "581234567890123456"),
		jntPage("JY0987654321", "589876543210987654"),
	)

	_, err := Extract(bytes.NewReader(pdf))
	if !errors.Is(err, errMultipleReceipts) {
		t.Fatalf("err = %v, want errMultipleReceipts", err)
	}
}

// A blank page beside the label is not a second label.
func TestExtractIgnoresAPageWithNoLabel(t *testing.T) {
	pdf := buildPDF(
		jntPage("JY1234567890", "581234567890123456"),
		[]text{{x: 20, y: 300, size: 10, s: "page 2"}},
	)

	got, err := Extract(bytes.NewReader(pdf))
	if err != nil {
		t.Fatal(err)
	}
	if got.Receipt != "JY1234567890" {
		t.Fatalf("receipt = %q", got.Receipt)
	}
}

func TestExtractRefusesWhatIsNotAPDF(t *testing.T) {
	for name, data := range map[string][]byte{
		"empty": {},
		"text":  []byte("hello, this is not a pdf"),
		"png":   {0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a},
	} {
		_, err := Extract(bytes.NewReader(data))
		if !errors.Is(err, errNotPDF) {
			t.Errorf("%s: err = %v, want errNotPDF", name, err)
		}
	}
}

func TestExtractRefusesAnOversizedFile(t *testing.T) {
	data := append([]byte("%PDF-1.4\n"), make([]byte, maxSize)...)

	_, err := Extract(bytes.NewReader(data))
	if !errors.Is(err, errTooLarge) {
		t.Fatalf("err = %v, want errTooLarge", err)
	}
}

func TestExtractRefusesATruncatedFile(t *testing.T) {
	data := buildPDF(jntPage("JY1234567890", "581234567890123456"))

	_, err := Extract(bytes.NewReader(data[:len(data)/2]))
	if !errors.Is(err, ErrUnreadable) {
		t.Fatalf("err = %v, want ErrUnreadable", err)
	}
}

// The PDF library panics on malformed content — an `end` with no `begin` is its
// `panic("mismatched begin/end")`. Extract must answer, not crash the server.
func TestExtractTurnsALibraryPanicIntoAnError(t *testing.T) {
	data := buildPDFContent("end\n")

	_, err := Extract(bytes.NewReader(data))
	if !errors.Is(err, ErrUnreadable) {
		t.Fatalf("err = %v, want ErrUnreadable", err)
	}
	// Proves it came through the recover, not through an ordinary parse error.
	if !strings.Contains(err.Error(), "mismatched begin/end") {
		t.Fatalf("err = %v, want the recovered panic", err)
	}
}

// An operator with the wrong operands — `Tc` with two — fails the file the way the library did,
// rather than placing every glyph after it wrongly.
func TestExtractRefusesAMalformedOperator(t *testing.T) {
	data := buildPDFContent("BT 1 2 Tc ET\n")

	_, err := Extract(bytes.NewReader(data))
	if !errors.Is(err, ErrUnreadable) {
		t.Fatalf("err = %v, want ErrUnreadable", err)
	}
	if !strings.Contains(err.Error(), "malformed Tc operator") {
		t.Fatalf("err = %v, want the malformed operator named", err)
	}
}

// assertNumbers checks the two numbers a reading is for. The recipient is checked by its own tests
// (recipient_test.go), on fixtures laid out the way the real labels are.
func assertNumbers(t *testing.T, got ReceiptData, receipt, orderID string) {
	t.Helper()

	if got.Receipt != receipt || got.OrderID != orderID {
		t.Fatalf("got receipt %q order id %q\nwant receipt %q order id %q", got.Receipt, got.OrderID, receipt, orderID)
	}
}

// A file that matches no layout and has no image on any page is not a courier's label: every real
// label carries a logo, a barcode or a QR. It is refused as ErrNotShippingLabel, which is still the
// unknown-layout refusal, so a caller can skip it.
func TestAFileWithNoImageIsNotAShippingLabel(t *testing.T) {
	data := buildPDF([]text{
		{x: 9, y: 365, size: 12, s: "Nama Penerima : Budi"},
		{x: 9, y: 331, size: 12, s: "Alamat Penerima : Jl. Mawar No. 1"},
	})

	_, err := Extract(bytes.NewReader(data))
	if !errors.Is(err, ErrNotShippingLabel) || !errors.Is(err, errUnknownLayout) {
		t.Fatalf("err = %v, want ErrNotShippingLabel, as an unknown layout", err)
	}
}

// A file with an image and a word shaped like a tracking number may be a courier's label the package
// hasn't learned: unknown, not spam.
func TestAnUnknownFileWithAnImageMayBeALabel(t *testing.T) {
	_, err := Extract(bytes.NewReader(shopeePDF(otherLogo, "Some Courier", "Resi: ZZ0000000001")))
	if !errors.Is(err, errUnknownLayout) || errors.Is(err, ErrNotShippingLabel) {
		t.Fatalf("err = %v, want errUnknownLayout and not ErrNotShippingLabel", err)
	}
}

// The two exported errors never overlap: ErrUnreadable is a file that could not be read at all,
// ErrNotShippingLabel one that was read and is no courier's label (the pdc_sample spam).
func TestUnreadableAndNotAShippingLabelAreTwoErrors(t *testing.T) {
	label := buildPDF(jntPage("JY1234567890", "581234567890123456"))
	spam := buildPDF([]text{{x: 9, y: 365, size: 12, s: "Nama Penerima : Budi"}})

	cases := []struct {
		name       string
		data       []byte
		unreadable bool
	}{
		{"not a pdf", []byte("hello, this is not a label"), true},
		{"over 2 MB", append([]byte("%PDF-1.4\n"), make([]byte, maxSize)...), true},
		{"truncated", label[:len(label)/2], true},
		{"a library panic", buildPDFContent("end\n"), true},
		{"read, and not a label", spam, false},
	}

	for _, c := range cases {
		_, err := Extract(bytes.NewReader(c.data))
		if errors.Is(err, ErrUnreadable) != c.unreadable || errors.Is(err, ErrNotShippingLabel) == c.unreadable {
			t.Errorf("%s: err = %v — unreadable %v, not a label %v", c.name, err,
				errors.Is(err, ErrUnreadable), errors.Is(err, ErrNotShippingLabel))
		}
	}
}

// A page with no text whose images carry no barcode or QR is a picture of something else — a
// screenshot of a document — not a label printed as a picture, which always carries one.
func TestAPictureWithNoCodeIsNotAShippingLabel(t *testing.T) {
	w := newPDF()
	image := w.rgbImage(2, 2, fakeLogo)
	w.page(fmt.Sprintf("/XObject << /Im1 %d 0 R >>", image), "q 200 0 0 200 50 50 cm /Im1 Do Q\n")

	_, err := Extract(bytes.NewReader(w.bytes()))
	if !errors.Is(err, ErrNotShippingLabel) {
		t.Fatalf("err = %v, want ErrNotShippingLabel", err)
	}
}

// A note designed over a background picture (pdc_samples/04.pdf): an image and text, but no barcode
// and no word a tracking number could be. Not a shipping label.
func TestANoteWithNoTrackingNumberIsNotAShippingLabel(t *testing.T) {
	w := newPDF()
	font := w.helvetica()
	image := w.rgbImage(2, 2, fakeLogo)
	content := "q 280 0 0 420 0 0 cm /Im1 Do Q\n" + helveticaContent([]text{
		{x: 36, y: 325, size: 20, s: "Team (09)"},
		{x: 38, y: 295, size: 12, s: "Order Cross Produk"},
		{x: 88, y: 175, size: 12, s: "(02102026"},
	})
	w.page(fmt.Sprintf("/Font << /F1 %d 0 R >> /XObject << /Im1 %d 0 R >>", font, image), content)

	_, err := Extract(bytes.NewReader(w.bytes()))
	if !errors.Is(err, ErrNotShippingLabel) {
		t.Fatalf("err = %v, want ErrNotShippingLabel", err)
	}
}

func TestATrackingShapeIsLongAndMostlyDigits(t *testing.T) {
	cases := map[string]bool{
		"ZZ0000000001":           true,
		"Resi:SPXID000000000001": true,  // split at the colon
		"081234567890":           true,  // a phone: only ever keeps a file "unknown"
		"02102026":               false, // a date
		"2026-09-28":             false, // a date with dashes
		"TTE-TTE024A-TA":         false, // a sort code
		"Order Cross Produk":     false,
	}

	for line, want := range cases {
		if got := hasTrackingShape([]string{line}); got != want {
			t.Errorf("%q → %v, want %v", line, got, want)
		}
	}
}
