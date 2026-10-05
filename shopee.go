package san_receipt_readers

import (
	"regexp"
	"strings"
	"unicode"
)

// What every Shopee-printed label shares, whichever courier carries it.

// A Shopee order number is six date digits and then capitals and digits: "261002AB12CD34". The
// label prints it as "No. Pesanan:" or, on the standard label, "No.Pesanan:".
//
// The capture must END at something that is not part of one, so a value fused to the next field
// ("…TT9Kode") reads as nothing rather than as a wrong, longer number. An ellipsis ends nothing: it
// is the box cutting the number short ("260101ABCDEF…", shopee_cargo_01.pdf), and the part before it
// is not the order number.
var shopeeOrderID = regexp.MustCompile(`No\.\s*Pesanan:\s*(\d{6}[0-9A-Z]{6,8})(?:[^0-9A-Z…]|$)`)

// readShopeeOrderID is the order number — whole, even when its box cut it short — or, on a label for
// a RESERVATION, the reservation number (Q14 in context_clarify.md).
func readShopeeOrderID(lines []string) string {
	for _, line := range lines {
		m := shopeeOrderID.FindStringSubmatch(line)
		if m != nil {
			return m[1]
		}
	}

	whole := readShopeeWholeNumber(lines, shopeeCutOrderID)
	if whole != "" {
		return whole
	}

	return readShopeeWholeNumber(lines, shopeeReservation)
}

var (
	// An order number its box cut short ends in an ellipsis: "No. Pesanan: 260101ABCDEF…".
	shopeeCutOrderID = regexp.MustCompile(`No\.\s*Pesanan:\s*(\d{6}[0-9A-Z]+)…`)
	// A reservation label prints "No.Reservasi:" where an order's prints "No. Pesanan:" — an SPX
	// parcel addressed to an SPX hub rather than to a buyer (shopee_spx_02.pdf). Its box wraps it.
	shopeeReservation = regexp.MustCompile(`No\.\s*Reservasi:\s*(\d{6}[0-9A-Z]+)`)
	// The "Pesan:" line repeats the label's number in brackets, whole.
	shopeeMessageNumber = regexp.MustCompile(`Pesan:\s*\((\d{6}[0-9A-Z]{8,16})\)`)
)

// readShopeeWholeNumber is the number whose box (`caption`) shows only its start — cut short with an
// ellipsis, or wrapped onto the next line ("260628…" / "KI") — read whole from the "Pesan:" line, and
// only when the part in the box is how it starts, so both copies are the same number.
func readShopeeWholeNumber(lines []string, caption *regexp.Regexp) string {
	start, whole := "", ""
	for _, line := range lines {
		if m := caption.FindStringSubmatch(line); m != nil && start == "" {
			start = m[1]
		}
		if m := shopeeMessageNumber.FindStringSubmatch(line); m != nil && whole == "" {
			whole = m[1]
		}
	}
	if start == "" || !strings.HasPrefix(whole, start) {
		return ""
	}

	return whole
}

// isShopeeLabel reports a label Shopee printed: for an order ("Pesanan:") or a reservation
// ("Reservasi:").
func isShopeeLabel(v *pageView) bool {
	return containsLine(v.lines, "Pesanan:") || containsLine(v.lines, "Reservasi:")
}

// hasLine reports a line that is exactly `s` — the service a Shopee label prints as its header
// ("HALU", "Same-day", "INSTANT / SAMEDAY"). Exact, because a product title on the same label can
// contain the word: "HALUS" is ordinary Indonesian. Spaces are ignored on both sides, so how a gap
// was rebuilt ("INSTANT /SAMEDAY") cannot decide a match.
func hasLine(lines []string, s string) bool {
	want := strings.ReplaceAll(s, " ", "")
	for _, line := range lines {
		if strings.ReplaceAll(line, " ", "") == want {
			return true
		}
	}

	return false
}

// readShopeeTracking is the number of the courier's exact shape that the label prints most often
// (mostPrinted), or — printed once, with no strips repeating it — the one in its "Resi" box
// (captionValue).
func readShopeeTracking(lines []string, pattern *regexp.Regexp) string {
	most := mostPrinted(lines, pattern)
	if most != "" {
		return most
	}

	return captionValue(lines, pattern)
}

// resiCaption finds the number printed after the "Resi:" caption ("No. Resi:" on most labels).
var resiCaption = regexp.MustCompile(`Resi:\s*([0-9A-Z]+)`)

// captionValue is the number in the label's "Resi" box, when it has exactly the courier's shape.
//
// It is the fallback for a label that prints its number ONCE — an older Shopee print has no strips
// repeating it (shopee_sicepat_02.pdf). The box can wrap (the SPX ECO label's does), so this is only
// safe for a shape a cut-off number cannot pass for: twelve digits cut short is eleven, and fails.
func captionValue(lines []string, pattern *regexp.Regexp) string {
	for _, line := range lines {
		m := resiCaption.FindStringSubmatch(line)
		if m != nil && pattern.MatchString(m[1]) {
			return m[1]
		}
	}

	return ""
}

// minRepeats is how often a number must be printed to be trusted as THE tracking number.
const minRepeats = 2

// mostPrinted returns the token matching `pattern` that the label prints most often.
//
// A Shopee label prints its tracking number many times — strips along its top and bottom, the
// "Resi" box, and on SPX the "Pesan:" line — and the "Resi" box is the one copy that can't be
// trusted: on the SPX ECO label it is too narrow, the number WRAPS ("SPXID09876543210" / "A"),
// and its caption is "Resi:" rather than "No. Resi:". Counting makes the wrapped fragment one vote
// against seven. A number printed fewer than minRepeats times, or a tie, reads as nothing — never
// a guess.
func mostPrinted(lines []string, pattern *regexp.Regexp) string {
	count := map[string]int{}
	for _, line := range lines {
		// Tokens are runs of capitals and digits, so "Resi: SPXID…" and "(SPXID…)" both yield the
		// number alone.
		tokens := strings.FieldsFunc(line, func(r rune) bool {
			return !unicode.IsDigit(r) && !unicode.IsUpper(r)
		})
		for _, token := range tokens {
			if pattern.MatchString(token) {
				count[token]++
			}
		}
	}

	best, bestCount, tied := "", 0, false
	for number, n := range count {
		switch {
		case n > bestCount:
			best, bestCount, tied = number, n, false
		case n == bestCount:
			tied = true
		}
	}

	if bestCount < minRepeats || tied {
		return ""
	}

	return best
}
