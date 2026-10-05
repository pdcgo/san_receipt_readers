package san_receipt_readers

import (
	"regexp"
)

// jntCargoLayout is a TikTok / Tokopedia label carried by J&T Cargo — J&T's freight brand, not
// J&T Express. Measured on one sample (tiktok_tokped_jnt_cargo.pdf, service FastTrack).
//
// ⚠ THE TEXT NEVER SAYS J&T. The courier is named only by its logo, matched in courierLogos — a
// TikTok label with another logo is unknown, not read as J&T Cargo.
var jntCargoLayout = layout{
	name: "jnt_cargo",
	match: func(v *pageView) bool {
		return isTikTokLabel(v) &&
			logoCourier(v) == courierJNTCargo &&
			readJNTCargoTracking(v.lines) != ""
	},
	read: func(v *pageView) ReceiptData {
		return ReceiptData{
			Receipt: readJNTCargoTracking(v.lines),
			OrderID: readTikTokOrderID(v.lines),
		}.with(readTikTokRecipient(v.rows))
	},
}

// A J&T Cargo tracking number is twelve digits: "570000000001". Measured on one sample.
var jntCargoTracking = regexp.MustCompile(`^\d{12}$`)

// readJNTCargoTracking is the barcode caption: the twelve digits ALONE on their line, printed once.
// The phones beside it are captioned and bracketed ("(+62)85123746739"), so they never stand alone.
func readJNTCargoTracking(lines []string) string {
	return onlyLine(lines, jntCargoTracking)
}
