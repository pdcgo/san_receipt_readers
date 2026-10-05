package san_receipt_readers

import "regexp"

// spxPictureLayout is a Shopee SPX label printed as PICTURES: the page has no text. Measured on one
// sample (shopee_std_01.pdf, SPX STD, Microsoft Print To PDF): every word is drawn as outlines, and
// each graphic is its own JPEG — the Shopee and SPX logos, a QR code and two Code 128s.
//
// Its barcodes carry both numbers. The QR code and the large Code 128 hold the tracking number, and
// the small Code 128 the order number. ⚠ The recipient is outlines, and stays "" (Q13 in
// context_clarify.md).
var spxPictureLayout = layout{
	name: "spx_picture",
	match: func(v *pageView) bool {
		return agreedCode(v.codes, spxTracking) != ""
	},
	read: func(v *pageView) ReceiptData {
		return ReceiptData{
			Receipt: agreedCode(v.codes, spxTracking),
			OrderID: agreedCode(v.codes, shopeeOrderCode),
		}
	},
}

// shopeeOrderCode is a Shopee order number alone, as a barcode carries it: the shape after
// "No. Pesanan:" (shopeeOrderID). It is read only beside an SPX number, so it is never mistaken for
// a tracking number: a SiCepat one, twelve digits, has this shape too.
var shopeeOrderCode = regexp.MustCompile(`^\d{6}[0-9A-Z]{6,8}$`)
