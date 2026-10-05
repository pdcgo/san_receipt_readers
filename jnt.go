package san_receipt_readers

import (
	"regexp"
	"strings"
	"unicode"
)

// jntLayout is the J&T label as the first sample prints it: rendered from HTML
// (Creator wkhtmltopdf), with J&T's website in the terms line at its foot. A J&T label printed
// by another app is a different layout and gets its own file.
var jntLayout = layout{
	name: "jnt",
	match: func(v *pageView) bool {
		return containsLine(v.lines, "www.jet.co.id")
	},
	read: func(v *pageView) ReceiptData {
		return readJNT(v.lines).with(readJNTRecipient(v.rows))
	},
}

var (
	// The barcode caption: the tracking number ALONE on its line. The prefix is not pinned —
	// JP and JY are both real (the-form-warns-on-a-real-jnt-number, context_clarify.md) — this
	// finds the number, it does not judge it.
	jntTracking = regexp.MustCompile(`^[A-Z]{2}\d{10,14}$`)
	// "Order Id :585600000000000001" — the label and its value share a line once grouped.
	jntOrderID = regexp.MustCompile(`Order\s*Id\s*:\s*(\S+)`)
)

func readJNT(lines []string) ReceiptData {
	result := ReceiptData{}

	for _, line := range lines {
		if result.Receipt == "" && jntTracking.MatchString(line) {
			result.Receipt = line
		}

		m := jntOrderID.FindStringSubmatch(line)
		if result.OrderID == "" && m != nil && looksLikeOrderID(m[1]) {
			result.OrderID = m[1]
		}
	}

	return result
}

// looksLikeOrderID refuses the word that follows an EMPTY "Order Id :" on the same line —
// on this label that is "Estimated", from "Estimated Date:". A marketplace order number always
// carries digits.
func looksLikeOrderID(s string) bool {
	return len(s) >= 6 && strings.ContainsFunc(s, unicode.IsDigit)
}
