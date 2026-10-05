package san_receipt_readers

import (
	"math"
	"regexp"
	"strings"
)

// lazadaLayout is Lazada's label, printed through PDFium. Measured on three samples: LEX's own (lazada_lex_01.pdf,
// lazada_lex_02.pdf, service STANDARD, "LXAD-…") and a partner's (lazada_lex_03.pdf, service TAP, "JNAP-…",
// two pages: the item list runs on to page 2, which matches no layout).
//
// ⚠ THE PAGE DRAWS ITS WHOLE FACE AS ONE FORM — the library read no text from it at all (content.go
// follows the form). Its images can't be decoded either (a Flate predictor the library doesn't
// implement), so no logo is read: the layout matches on its header and the tracking number's shape.
var lazadaLayout = layout{
	name: "lazada_lex",
	match: func(v *pageView) bool {
		return containsLine(v.lines, "Diserahkan ke") &&
			containsLine(v.lines, "Diantar oleh") &&
			readLEXTracking(v.lines) != ""
	},
	// The receipt is the courier's tracking number, and the order id the sixteen-digit order number
	// (a-lazada-receipt-is-its-tracking-number, context_decision.md). An earlier decision had it the
	// other way round, on a value the owner later withdrew.
	read: func(v *pageView) ReceiptData {
		return ReceiptData{
			Receipt: readLEXTracking(v.lines),
			OrderID: readLazadaOrderID(v.lines),
		}.with(readLazadaRecipient(v.rows))
	},
}

var (
	// A Lazada tracking number is alone on its line under the barcode, in its courier's own shape:
	//
	//   - four capitals, a dash and ten digits: "LXAD-…" (LEX), "JNAP-…" (service TAP) — two prefixes of
	//     one shape, so the shape is checked, not a list of prefixes;
	//   - J&T's two capitals and ten to fourteen digits: "JZ…" (service TAP, lazada_lex_04.pdf).
	//
	// Exact lengths are what keep the sort codes ("3-TAP-…") and the 16-digit order number out.
	lexTracking = regexp.MustCompile(`^(?:[A-Z]{4}-\d{10}|[A-Z]{2}\d{10,14})$`)
	// The order number is sixteen digits alone on a line, under the second barcode, with NO caption on
	// page 1. A two-page label captions the same number "Nomor Order :" on page 2 (lazada_lex_03.pdf),
	// which confirms it.
	lazadaOrderID = regexp.MustCompile(`^\d{16}$`)
)

func readLEXTracking(lines []string) string {
	return onlyLine(lines, lexTracking)
}

func readLazadaOrderID(lines []string) string {
	return onlyLine(lines, lazadaOrderID)
}

// readLazadaRecipient reads the Lazada two-column block: "Pengirim:" and the sender on the LEFT,
// "Penerima:" and the recipient on the RIGHT, and under each its address. Every row is cut where
// the "Penerima:" caption starts; the sender's address lines sit between the recipient's, on
// baselines of their own. The address-type tag ("Rumah") is printed larger, and left out.
//
// Lazada prints no recipient phone. Phone stays "".
func readLazadaRecipient(rows []row) recipient {
	for i, r := range rows {
		for _, w := range r.words {
			if !strings.HasPrefix(w.s, "Penerima:") {
				continue
			}

			from := w.x0 - columnSlack
			name := joinWords(column(r.sized(w.size).words, from))
			name = strings.TrimPrefix(name, "Penerima:")

			return recipient{
				name:    printed(name),
				address: addressBlock(rows[i+1:], w.size, from, math.Inf(1)),
			}
		}
	}

	return recipient{}
}
