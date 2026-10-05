package san_receipt_readers

import (
	"bytes"
	"errors"
	"testing"
)

func TestExtractReadsAShopeeIDExpressLabel(t *testing.T) {
	got, err := Extract(bytes.NewReader(shopeeResiPDF("STD", "IDS0000000000001", "")))
	if err != nil {
		t.Fatal(err)
	}

	assertNumbers(t, got, "IDS0000000000001", "260101ABCDEF12")
}

// Thirteen digits cut to twelve by the box are not the number: the label stays unknown.
func TestAnIDExpressNumberTheBoxCutShortIsNotRead(t *testing.T) {
	_, err := Extract(bytes.NewReader(shopeeResiPDF("STD", "IDS000000000000", "1")))
	if !errors.Is(err, errUnknownLayout) {
		t.Fatalf("err = %v, want errUnknownLayout", err)
	}
}
