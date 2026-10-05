package san_receipt_readers

import (
	"regexp"
)

// shopeeCMLayout is a Shopee label whose tracking number is "CM" and eleven digits, printed through
// PDFium. Measured on one sample (shopee_reguler_01.pdf, service Reguler).
//
// ⚠ THE LABEL NAMES NO COURIER — not in its text, and it carries no courier logo (its one small image
// is a "REGULAR SERVICE" badge). Probably JNE, which numbers its marketplace parcels "CM…" — but only
// the prefix says so: the sort code beside the number ("BTH10000" in shape) is Shopee's, printed on a
// SiCepat label too. No courier is returned (courier-is-not-read), so the layout is named for what it
// reads: the number's shape.
//
// The number is printed ONCE, in the "No. Resi:" box, with no strips repeating it — and confirmed by
// the barcode beside it, which says the same. The exact shape is what makes the box safe to read: a
// number the box cut short has fewer than eleven digits and does not match.
var shopeeCMLayout = layout{
	name: "shopee_cm",
	match: func(v *pageView) bool {
		return isShopeeLabel(v) && readCMTracking(v.lines) != ""
	},
	read: func(v *pageView) ReceiptData {
		return ReceiptData{
			Receipt: readCMTracking(v.lines),
			OrderID: readShopeeOrderID(v.lines),
		}.with(readShopeeRecipient(v.rows))
	},
}

// cmTracking is "CM" and eleven digits. Measured on one sample, and kept that narrow: another length
// or prefix stays unknown until it has a sample.
var cmTracking = regexp.MustCompile(`^CM\d{11}$`)

// readCMTracking is the CM number the label prints most often, or — printed once, as on the one
// sample — the one in its "Resi" box.
func readCMTracking(lines []string) string {
	return readShopeeTracking(lines, cmTracking)
}
