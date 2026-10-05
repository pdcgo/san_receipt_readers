package san_receipt_readers

import (
	"bytes"
	"errors"
	"testing"
)

// The SiCepat label's structure with invented numbers: the tracking number in strips along the top
// and the bottom and in its "No. Resi" box, and the sender's phone — digits too, printed once.
func sicepatPDF(header, resi string, logo []byte) []byte {
	return shopeePDF(logo,
		resi+" "+resi,
		header,
		"PLM-ITR-B No. Resi: "+resi,
		"Penerima: Contoh Pengirim: Toko Contoh",
		"628000000000",
		"Berat: 500 gr",
		"No. Pesanan: 260101ABCDEF12",
		"Pesan: (260101ABCDEF12)",
		resi+" "+resi,
	)
}

// HALU is named by its header alone — no known logo needed.
func TestExtractReadsASicepatHaluLabel(t *testing.T) {
	got, err := Extract(bytes.NewReader(sicepatPDF("HALU", "001234567890", otherLogo)))
	if err != nil {
		t.Fatal(err)
	}

	assertNumbers(t, got, "001234567890", "260101ABCDEF12")
}

// REG is named by the SiCepat logo: the header is shared with JNE and J&T.
func TestExtractReadsASicepatRegLabelByItsLogo(t *testing.T) {
	useFakeLogo(t, courierSicepat)

	got, err := Extract(bytes.NewReader(sicepatPDF("REG", "001234567890", fakeLogo)))
	if err != nil {
		t.Fatal(err)
	}

	assertNumbers(t, got, "001234567890", "260101ABCDEF12")
}

// A REG label with twelve digits and a logo the package does not know could be JNE's — unknown,
// never read as SiCepat.
func TestRegWithAnUnknownLogoIsUnknown(t *testing.T) {
	useFakeLogo(t, courierSicepat)

	_, err := Extract(bytes.NewReader(sicepatPDF("REG", "001234567890", otherLogo)))
	if !errors.Is(err, errUnknownLayout) {
		t.Fatalf("err = %v, want errUnknownLayout", err)
	}
}

// The header is matched as a whole line: "HALUS" is an ordinary word, and can sit in a product
// title on any Shopee label.
func TestHaluIsTheHeaderNotAWord(t *testing.T) {
	_, err := Extract(bytes.NewReader(sicepatPDF("KAIN HALUS", "001234567890", otherLogo)))
	if !errors.Is(err, errUnknownLayout) {
		t.Fatalf("err = %v, want errUnknownLayout", err)
	}
}

// A HALU label whose number is not twelve digits is not a shape this package has seen — unknown,
// so it arrives as a sample, rather than misread.
func TestHaluWithAnotherNumberShapeIsUnknown(t *testing.T) {
	_, err := Extract(bytes.NewReader(sicepatPDF("HALU", "0012345678901", otherLogo)))
	if !errors.Is(err, errUnknownLayout) {
		t.Fatalf("err = %v, want errUnknownLayout", err)
	}
}

// An older print has no strips repeating the number: it is printed once, in its "Resi" box.
func sicepatOncePDF(resiBox ...string) []byte {
	lines := append([]string{"HALU"}, resiBox...)
	lines = append(lines,
		"Penerima: Contoh Pengirim: Toko Contoh",
		"628000000000",
		"No. Pesanan: 260101ABCDEF12",
	)

	return shopeePDF(otherLogo, lines...)
}

func TestSicepatPrintedOnceIsReadFromItsResiBox(t *testing.T) {
	got, err := Extract(bytes.NewReader(sicepatOncePDF("SUB21003 No. Resi: 001234567890")))
	if err != nil {
		t.Fatal(err)
	}

	assertNumbers(t, got, "001234567890", "260101ABCDEF12")
}

// Printed once AND wrapped: the box holds eleven digits and the twelfth sits on the next line. Eleven
// digits are not a SiCepat number, so nothing is read — never the number cut short.
func TestSicepatWrappedOnceIsNotReadShort(t *testing.T) {
	_, err := Extract(bytes.NewReader(sicepatOncePDF("No. Resi: 00123456789", "0")))
	if !errors.Is(err, errUnknownLayout) {
		t.Fatalf("err = %v, want errUnknownLayout", err)
	}
}
