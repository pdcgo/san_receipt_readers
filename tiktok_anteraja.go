package san_receipt_readers

import (
	"regexp"
)

// tiktokAnterAjaLayout is a TikTok / Tokopedia label (the "tokopedia | Shop" logo) carried by
// AnterAja, printed by wkhtmltopdf like TikTok's others. Measured on one sample
// (tiktok_toped_anteraja_01.pdf, service INS); the courier is the owner's file name — the label's text
// never says AnterAja.
//
// Its tracking number stands alone on its line, "TSA-" and eleven digits; its order id follows
// "Order ID：" with no "TT". The recipient is TikTok's usual block, masked.
var tiktokAnterAjaLayout = layout{
	name: "tiktok_anteraja",
	match: func(v *pageView) bool {
		return readTikTokOrderID(v.lines) != "" && onlyLine(v.lines, tsaTracking) != ""
	},
	read: func(v *pageView) ReceiptData {
		return ReceiptData{
			Receipt: onlyLine(v.lines, tsaTracking),
			OrderID: readTikTokOrderID(v.lines),
		}.with(readTikTokRecipient(v.rows))
	},
}

// tsaTracking is three capitals, a dash and eleven digits: "TSA-…". Measured on one sample; its exact
// length keeps the order id and the sort codes out.
var tsaTracking = regexp.MustCompile(`^[A-Z]{3}-\d{11}$`)
