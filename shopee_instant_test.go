package san_receipt_readers

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"testing"
)

// The real courier logos are not copied into the tests. A synthetic 2×2 image stands in for one,
// and the test makes it the only known logo, as SPX's, for its duration.
var (
	fakeLogo  = []byte{255, 0, 0, 0, 255, 0, 0, 0, 255, 255, 255, 255}
	otherLogo = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
)

func useFakeSPXLogo(t *testing.T) {
	t.Helper()
	useFakeLogo(t, courierSPX)
}

// useFakeLogo makes fakeLogo the only known logo, as `courier`'s, for the test's duration.
func useFakeLogo(t *testing.T, courier CourierType) {
	t.Helper()

	sum := sha256.Sum256(fakeLogo)
	previous := courierLogos
	courierLogos = map[string]CourierType{
		fmt.Sprintf("2x2:%s", hex.EncodeToString(sum[:8])): courier,
	}
	t.Cleanup(func() {
		courierLogos = previous
	})
}

// shopeePDF draws `lines` (top to bottom, one per row) in Helvetica, with `logo` as the page's one
// small image.
func shopeePDF(logo []byte, lines ...string) []byte {
	w := newPDF()
	font := w.helvetica()
	image := w.rgbImage(2, 2, logo)

	page := []text{}
	for i, s := range lines {
		page = append(page, text{x: 10, y: 400 - float64(i)*20, size: 8, s: s})
	}
	content := helveticaContent(page) + "q 68 0 0 27 200 395 cm /Im1 Do Q\n"

	w.page(fmt.Sprintf("/Font << /F1 %d 0 R >> /XObject << /Im1 %d 0 R >>", font, image), content)

	return w.bytes()
}

// The instant label's structure with an invented order number.
func instantPDF(logo []byte) []byte {
	return instantPDFWithHeader("INSTANT / SAMEDAY", logo)
}

func instantPDFWithHeader(header string, logo []byte) []byte {
	return shopeePDF(logo,
		header,
		"No. Pesanan: 260101ABCDEF12",
		"Kode Pengambilan: 1ABC",
		"Berat: 500 gr",
	)
}

func TestExtractReadsAShopeeInstantLabel(t *testing.T) {
	useFakeSPXLogo(t)

	got, err := Extract(bytes.NewReader(instantPDF(fakeLogo)))
	if err != nil {
		t.Fatal(err)
	}

	// No tracking number is printed: the receipt is the pickup code (the-pickup-code-is-the-receipt).
	assertNumbers(t, got, "1ABC", "260101ABCDEF12")
}

func TestShopeeOrderIDIsReadByItsShape(t *testing.T) {
	cases := []struct {
		line string
		want string
	}{
		{"No. Pesanan: 261002AB12CD34 Kode Pengambilan: 9XYZ", "261002AB12CD34"},
		{"No. Pesanan:261002AB12CD34", "261002AB12CD34"},
		// The standard label prints it with no space after the dot.
		{"No.Pesanan: 261002EF56GH78", "261002EF56GH78"},
		// Fused to the next field: nothing, rather than a wrong, longer number.
		{"No. Pesanan: 261002AB12CD34Kode Pengambilan: 9XYZ", ""},
		// Split by a stray space: nothing, rather than the first half.
		{"No. Pesanan: 2610 02AB12CD34", ""},
	}

	for _, c := range cases {
		got := readShopeeOrderID([]string{c.line})
		if got != c.want {
			t.Errorf("%q → %q, want %q", c.line, got, c.want)
		}
	}
}

// GrabExpress prints the same template with "Same-day" as its header, and GoSend with "SAMEDAY" or
// "Instant". Each is read like the SPX one, whatever logo it carries.
func TestShopeeSameDayIsTheInstantTemplate(t *testing.T) {
	useFakeSPXLogo(t)

	for _, header := range []string{"Same-day", "SAMEDAY", "Instant"} {
		got, err := Extract(bytes.NewReader(instantPDFWithHeader(header, otherLogo)))
		if err != nil {
			t.Fatalf("%s: %v", header, err)
		}

		assertNumbers(t, got, "1ABC", "260101ABCDEF12")
	}
}

// A header this template has not been seen with stays unknown, so it arrives as a sample.
func TestShopeeInstantWithAnUnseenHeaderIsUnknown(t *testing.T) {
	_, err := Extract(bytes.NewReader(instantPDFWithHeader("Express", otherLogo)))
	if !errors.Is(err, errUnknownLayout) {
		t.Fatalf("err = %v, want errUnknownLayout", err)
	}
}

func TestShopeePickupCodeIsReadByItsShape(t *testing.T) {
	cases := []struct {
		lines []string
		want  string
	}{
		{[]string{"No. Pesanan: 260101ABCDEF12 Kode Pengambilan: 9XYZ"}, "9XYZ"},
		{[]string{"Kode Pengambilan:9XYZ"}, "9XYZ"},
		// Printed twice, the same: one code.
		{[]string{"Kode Pengambilan: 9XYZ", "Kode Pengambilan: 9XYZ"}, "9XYZ"},
		// Fused to the next word, or another length: nothing, rather than a wrong receipt.
		{[]string{"Kode Pengambilan: 9XYZBerat"}, ""},
		{[]string{"Kode Pengambilan: 9XY"}, ""},
		// Two different codes are a guess.
		{[]string{"Kode Pengambilan: 9XYZ", "Kode Pengambilan: 8ABC"}, ""},
	}

	for _, c := range cases {
		got := readShopeePickupCode(c.lines)
		if got != c.want {
			t.Errorf("%q → %q, want %q", c.lines, got, c.want)
		}
	}
}

// A label for a reservation (an SPX parcel addressed to an SPX hub) prints "No.Reservasi:" where an
// order's prints "No. Pesanan:", in a box too narrow for it. The "Pesan:" line has it whole.
func TestExtractReadsAShopeeReservationLabel(t *testing.T) {
	data := shopeePDF(otherLogo,
		"SPXID000000000001 SPXID000000000001",
		"ECO",
		"Resi:SPXID000000000001",
		"Penerima: Hub Contoh Pengirim:Toko Contoh",
		"No.Reservasi:260101ABCDEFGHIJK",
		"LM",
		"Pesan: (260101ABCDEFGHIJKLM) (SPXID000000000001)",
		"SPXID000000000001 SPXID000000000001",
	)

	got, err := Extract(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}

	assertNumbers(t, got, "SPXID000000000001", "260101ABCDEFGHIJKLM")
}

// The two copies must be one number: the box's wrapped start, and the whole one in brackets.
func TestShopeeReservationIsBothCopiesAgreeing(t *testing.T) {
	cases := []struct {
		lines []string
		want  string
	}{
		{[]string{"No.Reservasi:260101ABCDEFGHIJK", "Pesan: (260101ABCDEFGHIJKLM)"}, "260101ABCDEFGHIJKLM"},
		// The brackets hold another number: nothing, rather than a guess.
		{[]string{"No.Reservasi:260101ABCDEFGHIJK", "Pesan: (260101ZZZZZZZZZZZLM)"}, ""},
		// No reservation caption: an order label, read by its order number instead.
		{[]string{"Pesan: (260101ABCDEFGHIJKLM)"}, ""},
	}

	for _, c := range cases {
		if got := readShopeeWholeNumber(c.lines, shopeeReservation); got != c.want {
			t.Errorf("%q → %q, want %q", c.lines, got, c.want)
		}
	}
}
