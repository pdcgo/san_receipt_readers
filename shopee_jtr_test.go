package san_receipt_readers

import (
	"bytes"
	"errors"
	"testing"
)

func TestExtractReadsAShopeeJTRLabel(t *testing.T) {
	got, err := Extract(bytes.NewReader(shopeeResiPDF("JTR", "JT00000000001", "")))
	if err != nil {
		t.Fatal(err)
	}

	assertNumbers(t, got, "JT00000000001", "260101ABCDEF12")
}

// Eleven digits cut short by the box are ten: not read, so the label stays unknown.
func TestAJTRNumberTheBoxCutShortIsNotRead(t *testing.T) {
	_, err := Extract(bytes.NewReader(shopeeResiPDF("JTR", "JT0000000000", "1")))
	if !errors.Is(err, errUnknownLayout) {
		t.Fatalf("err = %v, want errUnknownLayout", err)
	}
}
