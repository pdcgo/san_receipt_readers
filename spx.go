package san_receipt_readers

import (
	"regexp"
)

// spxLayout is a Shopee label carrying an SPX tracking number, printed through PDFium. Measured on
// two samples of two services: STD (spx_02.pdf) and ECO (spx_03.pdf).
//
// The SPXID prefix is what makes the label SPX's, in its TEXT. A Shopee label carrying another
// courier's number does not match, and stays unknown until it has a sample of its own. The instant
// label prints no number at all and is shopeeInstantLayout.
var spxLayout = layout{
	name: "spx",
	match: func(v *pageView) bool {
		return isShopeeLabel(v) && readSPXTracking(v.lines) != ""
	},
	read: func(v *pageView) ReceiptData {
		return ReceiptData{
			Receipt: readSPXTracking(v.lines),
			OrderID: readShopeeOrderID(v.lines),
		}.with(readShopeeRecipient(v.rows))
	},
}

var spxTracking = regexp.MustCompile(`^SPXID\d{8,14}[A-Z]?$`)

// readSPXTracking is the SPX number the label prints most often — never the one in the "Resi"
// box, which wraps on the ECO label (see mostPrinted).
func readSPXTracking(lines []string) string {
	return mostPrinted(lines, spxTracking)
}
