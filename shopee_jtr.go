package san_receipt_readers

import (
	"regexp"
)

// shopeeJTRLayout is a Shopee label carried by JNE Trucking (JTR, JNE's service for heavy parcels),
// printed through PDFium. Measured on one sample (shopee_jtr_01.pdf: a 4 kg parcel, the "JTR — JNE
// TRUCKING" logo).
//
// ⚠ "JT…" IS JNE'S, NOT J&T'S, here: the logo says so. No courier is returned, so all that matters is
// the shape — "JT" and eleven digits, printed ONCE in the "No. Resi:" box and confirmed by the barcode
// beside it. The exact length is what makes the box safe to read: a number it cut short does not match.
var shopeeJTRLayout = layout{
	name: "shopee_jtr",
	match: func(v *pageView) bool {
		return isShopeeLabel(v) && readJTRTracking(v.lines) != ""
	},
	read: func(v *pageView) ReceiptData {
		return ReceiptData{
			Receipt: readJTRTracking(v.lines),
			OrderID: readShopeeOrderID(v.lines),
		}.with(readShopeeRecipient(v.rows))
	},
}

// jtrTracking is "JT" and eleven digits. Measured on one sample, and kept that narrow: another length
// stays unknown until it has a sample.
var jtrTracking = regexp.MustCompile(`^JT\d{11}$`)

func readJTRTracking(lines []string) string {
	return readShopeeTracking(lines, jtrTracking)
}
