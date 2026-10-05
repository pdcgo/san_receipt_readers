package san_receipt_readers

import (
	"regexp"
)

// shopeeAnterAjaLayout is a Shopee label carried by AnterAja, printed through PDFium. Measured on one
// sample (shopee_eco_01.pdf, service ECO, AnterAja's "PakEkoAja" logo).
//
// The tracking number is fourteen digits, printed ONCE in the "No. Resi:" box with no strips
// repeating it — and confirmed by the barcode beside it, which says the same. The exact length is
// what makes the box safe to read: a number the box cut short has thirteen digits and does not match.
// The sender's phone is digits too, but never sits in the box.
var shopeeAnterAjaLayout = layout{
	name: "shopee_anteraja",
	match: func(v *pageView) bool {
		return isShopeeLabel(v) && readAnterAjaTracking(v.lines) != ""
	},
	read: func(v *pageView) ReceiptData {
		return ReceiptData{
			Receipt: readAnterAjaTracking(v.lines),
			OrderID: readShopeeOrderID(v.lines),
		}.with(readShopeeRecipient(v.rows))
	},
}

// anterAjaTracking is fourteen digits. Measured on one sample, and kept that narrow: another length
// stays unknown until it has a sample.
var anterAjaTracking = regexp.MustCompile(`^\d{14}$`)

func readAnterAjaTracking(lines []string) string {
	return readShopeeTracking(lines, anterAjaTracking)
}
