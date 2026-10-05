package san_receipt_readers

import (
	"regexp"
)

// shopeePosLayout is a Shopee label carried by Pos Indonesia (its "PosAja! | POS IND" logo), service
// REG, printed through PDFium. Measured on one sample (shopee_reg_02.pdf): "SHPE" and eighteen capitals
// and digits, printed most often — strips along the top and the "Resi" box — and the same in its
// barcode. No courier is returned (courier-is-not-read); the shape is all the layout reads.
//
// A hidden tag sits over the address's last line in another size ("SIPP_MIXED_VA"); the address
// reader rebuilds each line from the address's own size, so it is left out.
var shopeePosLayout = layout{
	name: "shopee_pos",
	match: func(v *pageView) bool {
		return isShopeeLabel(v) && readShopeeTracking(v.lines, posTracking) != ""
	},
	read: func(v *pageView) ReceiptData {
		return ReceiptData{
			Receipt: readShopeeTracking(v.lines, posTracking),
			OrderID: readShopeeOrderID(v.lines),
		}.with(readShopeeRecipient(v.rows))
	},
}

// posTracking is "SHPE" and eighteen capitals and digits. The exact length is what makes the Resi box
// safe to read when the number is printed once.
var posTracking = regexp.MustCompile(`^SHPE[0-9A-Z]{18}$`)
