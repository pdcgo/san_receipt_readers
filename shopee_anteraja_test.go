package san_receipt_readers

import (
	"bytes"
	"errors"
	"testing"
)

func TestExtractReadsAShopeeAnterAjaLabel(t *testing.T) {
	got, err := Extract(bytes.NewReader(shopeeResiPDF("ECO", "10000000000001", "")))
	if err != nil {
		t.Fatal(err)
	}

	assertNumbers(t, got, "10000000000001", "260101ABCDEF12")
}

// Fourteen digits cut short by the box are thirteen: not read, so the label stays unknown.
func TestAnAnterAjaNumberTheBoxCutShortIsNotRead(t *testing.T) {
	_, err := Extract(bytes.NewReader(shopeeResiPDF("ECO", "1000000000000", "1")))
	if !errors.Is(err, errUnknownLayout) {
		t.Fatalf("err = %v, want errUnknownLayout", err)
	}
}
