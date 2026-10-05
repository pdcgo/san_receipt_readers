package san_receipt_readers

import (
	"regexp"
)

// What every TikTok / Tokopedia label shares, whichever courier carries it. Rendered from HTML
// (Creator "wkhtmltopdf", Producer "pdfcpu") — the same generator as jnttest.pdf, whose order id
// is TikTok's too.

// "TT Order ID：583400000000000001" — the colon is the FULL-WIDTH one (U+FF1A), as the label prints
// every caption. The Tokopedia-branded label drops the "TT" ("Order ID：…",
// tiktok_toped_anteraja_01.pdf). TikTok order ids are 18 digits; the capture must end at a non-digit,
// so an id fused to the next field reads as nothing.
var tiktokOrderID = regexp.MustCompile(`(?:TT\s*)?Order\s*ID\s*[:：]\s*(\d{16,20})(?:\D|$)`)

func readTikTokOrderID(lines []string) string {
	for _, line := range lines {
		m := tiktokOrderID.FindStringSubmatch(line)
		if m != nil {
			return m[1]
		}
	}

	return ""
}

func isTikTokLabel(v *pageView) bool {
	return containsLine(v.lines, "TT Order ID")
}
