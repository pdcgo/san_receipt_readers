package san_receipt_readers

import (
	"regexp"
	"strings"
)

// shopeeBarcodeLayout reads a Shopee label that no courier layout knows — a new logo size, a new number
// shape — by its "Resi" box, confirmed by its barcode: the box's value is the receipt only when a barcode
// on the page decodes to exactly that value (a-shopee-resi-is-confirmed-by-its-barcode,
// context_decision.md).
//
// It is tried after every courier layout, so a label one of them reads never has its barcodes decoded.
// A box that cut its number short no longer matches the barcode, and a label whose box and barcodes
// disagree, or whose barcode can't be read, stays unknown as before.
var shopeeBarcodeLayout = layout{
	name: "shopee_barcode",
	match: func(v *pageView) bool {
		return isShopeeLabel(v) && readConfirmedResi(v) != ""
	},
	read: func(v *pageView) ReceiptData {
		return ReceiptData{
			Receipt: readConfirmedResi(v),
			OrderID: readShopeeOrderID(v.lines),
		}.with(readShopeeRecipient(v.rows))
	},
}

// anyResi is any value a "Resi" box can hold: capitals and digits, eight or more. Its shape is not what
// is trusted — the barcode is.
var anyResi = regexp.MustCompile(`^[0-9A-Z]{8,30}$`)

// readConfirmedResi is the "Resi" box's value when a barcode on the page says the same, or "".
func readConfirmedResi(v *pageView) string {
	box := captionValue(v.lines, anyResi)
	if box == "" || !strings.ContainsAny(box, "0123456789") {
		return ""
	}
	for _, code := range v.barcodes() {
		if code == box {
			return box
		}
	}

	return ""
}
