package san_receipt_readers

import (
	"regexp"
)

// sicepatLayout is a Shopee label carried by SiCepat, printed through PDFium. Measured on four
// samples of three services: HALU (shopee_halu_01.pdf), REG (shopee_sicepat_01.pdf, and an older
// print without the repeating strips, shopee_sicepat_02.pdf) and BEST, Shopee's NEXT DAY
// (shopee_nextday_01.pdf, its number printed once, named by SiCepat's BEST logo).
//
// ⚠ THE TEXT NEVER SAYS "SICEPAT". The tracking number is bare digits, so the layout needs other
// evidence that the label is SiCepat's — no courier is returned, but the twelve-digit rule below is
// SiCepat's, and must not read a JNE label. Either of two is enough:
//
//   - the header is HALU — SiCepat's budget service. An inference from the service name; with no
//     courier returned, all it decides is which number rule reads the label;
//   - a known SiCepat logo (courierLogos). The REG label needs this: "REG" is also a JNE and a J&T
//     service, so the header proves nothing there.
//
// Without either, a Shopee label with a twelve-digit number stays unknown rather than being read
// as SiCepat — a JNE REG label must not come out as SiCepat.
var sicepatLayout = layout{
	name: "sicepat",
	match: func(v *pageView) bool {
		if !isShopeeLabel(v) || readSicepatTracking(v.lines) == "" {
			return false
		}

		return hasLine(v.lines, "HALU") || logoCourier(v) == courierSicepat
	},
	read: func(v *pageView) ReceiptData {
		return ReceiptData{
			Receipt: readSicepatTracking(v.lines),
			OrderID: readShopeeOrderID(v.lines),
		}.with(readShopeeRecipient(v.rows))
	},
}

// A SiCepat tracking number is twelve digits: "004600000001", "004600000002", "004500000003".
// Measured on three samples, and kept that narrow on purpose — a label with any other number is
// unknown, not misread. The narrowness is also what makes the caption fallback safe.
var sicepatTracking = regexp.MustCompile(`^\d{12}$`)

// readSicepatTracking is the twelve-digit number the label prints most often — counting keeps the
// sender's phone out, since it is digits too but printed once. An older print has no repeating
// strips at all (shopee_sicepat_02.pdf), and then the number is read from its "Resi" box.
func readSicepatTracking(lines []string) string {
	return readShopeeTracking(lines, sicepatTracking)
}
