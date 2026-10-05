package san_receipt_readers

// jntPictureLayout is a J&T Express label printed as ONE PICTURE: the page has no text at all, only
// an image of the label. Measured on one sample (jnt_eco_01.pdf: the TikTok J&T template, service
// ECO, a 1000×1000 image re-wrapped by pdfcpu).
//
// ⚠ ONLY THE TRACKING NUMBER IS READ. The QR code and the Code 128 carry it; the order id and the
// recipient are pixels, and stay "" (Q13 in context_clarify.md).
var jntPictureLayout = layout{
	name: "jnt_picture",
	match: func(v *pageView) bool {
		return readPictureTracking(v.codes) != ""
	},
	read: func(v *pageView) ReceiptData {
		return ReceiptData{Receipt: readPictureTracking(v.codes)}
	},
}

// readPictureTracking is the J&T tracking number the page's barcodes carry — when every one that
// has that shape says the same. Two different numbers are a guess.
func readPictureTracking(codes []string) string {
	return agreedCode(codes, jntTracking)
}
