package san_receipt_readers

import (
	"regexp"
)

// shopeeInstantLayout is Shopee's instant / same-day template, printed through PDFium — the one
// with a pickup code ("Kode Pengambilan") for the rider. Measured on four samples, three couriers:
// spx_01.pdf (SPX), shopee_grab_express_01.pdf and shopee_grab_express_02.pdf (GrabExpress), and
// shopee_sameday_01.pdf (GoSend).
//
// ⚠ IT PRINTS NO TRACKING NUMBER, AND NAMES ITS COURIER ONLY IN A LOGO. On spx_01 the text, the
// barcode and the QR code all carry the same string, the Shopee order number. The order's receipt
// is the PICKUP CODE instead: that is what live orders store (the-pickup-code-is-the-receipt,
// context_decision.md).
//
// No courier is returned (courier-is-not-read), so the logo plays no part here: the label matches
// on its text — the pickup code and a header this template has been seen with.
var shopeeInstantLayout = layout{
	name: "shopee_instant",
	match: func(v *pageView) bool {
		if !isShopeeLabel(v) || !containsLine(v.lines, "Kode Pengambilan:") {
			return false
		}
		for _, service := range shopeeInstantServices {
			if hasLine(v.lines, service) {
				return true
			}
		}

		return false
	},
	read: func(v *pageView) ReceiptData {
		return ReceiptData{
			Receipt: readShopeePickupCode(v.lines),
			OrderID: readShopeeOrderID(v.lines),
		}.with(readShopeeRecipient(v.rows))
	},
}

// shopeeInstantServices are the headers this template has been seen with. A new one stays unknown
// until it has a sample — the same rule as every other layout.
var shopeeInstantServices = []string{
	"INSTANT / SAMEDAY", // spx_01.pdf, SPX
	"Same-day",          // shopee_grab_express_01.pdf and _02.pdf, GrabExpress
	"SAMEDAY",           // shopee_sameday_01.pdf, GoSend
	"Instant",           // shopee_instant_02.pdf, GoSend
}

// The pickup code is four capitals and digits beside the order number, on its row or one of its own:
// "Kode Pengambilan: 1ABC". Measured on four samples, and kept that narrow: a code fused to the next word,
// or of another length, reads as nothing rather than as a wrong receipt.
var shopeePickupCode = regexp.MustCompile(`Kode\s*Pengambilan:\s*([0-9A-Z]{4})(?:[^0-9A-Z]|$)`)

// readShopeePickupCode is the label's pickup code — when it prints exactly one. Two different codes
// are a guess, and read as nothing.
func readShopeePickupCode(lines []string) string {
	found := ""
	for _, line := range lines {
		m := shopeePickupCode.FindStringSubmatch(line)
		if m == nil || m[1] == found {
			continue
		}
		if found != "" {
			return ""
		}
		found = m[1]
	}

	return found
}
