package san_receipt_readers

import (
	"bytes"
	"errors"
	"testing"
)

func TestExtractReadsAShopeePosLabel(t *testing.T) {
	got, err := Extract(bytes.NewReader(shopeeResiPDF("REG", "SHPE000000000000000001", "")))
	if err != nil {
		t.Fatal(err)
	}

	assertNumbers(t, got, "SHPE000000000000000001", "260101ABCDEF12")
}

// Eighteen characters after "SHPE" cut to seventeen by the box are not the number: unknown.
func TestAPosNumberTheBoxCutShortIsNotRead(t *testing.T) {
	_, err := Extract(bytes.NewReader(shopeeResiPDF("REG", "SHPE00000000000000000", "1")))
	if !errors.Is(err, errUnknownLayout) {
		t.Fatalf("err = %v, want errUnknownLayout", err)
	}
}
