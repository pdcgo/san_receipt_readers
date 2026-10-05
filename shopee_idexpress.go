package san_receipt_readers

import (
	"regexp"
)

// shopeeIDExpressLayout is a Shopee label carried by ID Express, printed through PDFium. Measured on
// one sample (shopee_id_01.pdf): "IDS" and thirteen digits, printed most often — strips along the top
// and bottom and the "Resi" box — and the same in its barcode. No courier is returned
// (courier-is-not-read), but the shape is ID Express's own: the KiriminAja ID Express label prints
// "IDE" and thirteen digits.
var shopeeIDExpressLayout = layout{
	name: "shopee_idexpress",
	match: func(v *pageView) bool {
		return isShopeeLabel(v) && readShopeeTracking(v.lines, idExpressTracking) != ""
	},
	read: func(v *pageView) ReceiptData {
		return ReceiptData{
			Receipt: readShopeeTracking(v.lines, idExpressTracking),
			OrderID: readShopeeOrderID(v.lines),
		}.with(readShopeeRecipient(v.rows))
	},
}

// idExpressTracking is "ID", one capital and thirteen digits: "IDS…" (Shopee), "IDE…" (KiriminAja).
// The exact length is what makes the Resi box safe to read when the number is printed once.
var idExpressTracking = regexp.MustCompile(`^ID[A-Z]\d{13}$`)
