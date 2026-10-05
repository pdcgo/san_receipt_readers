package san_receipt_readers

import (
	"math"
	"regexp"
	"strings"
)

// tiktokInstantLayout is TikTok Shop's instant / same-day label, printed by wkhtmltopdf like its other
// labels: a pickup code ("Kode Pengambilan") for the rider, and NO tracking number. Measured on one
// sample (tiktokshop_01.pdf). The courier is named nowhere — its one logo is TikTok Shop's own.
//
// Its receipt is the pickup code, as on Shopee's instant label (the-pickup-code-is-the-receipt,
// context_decision.md) — carried over to TikTok as the default, for the owner's iterate tool to confirm.
var tiktokInstantLayout = layout{
	name: "tiktok_instant",
	match: func(v *pageView) bool {
		return readTikTokShopOrderID(v.lines) != "" && readTikTokPickupCode(v.rows) != ""
	},
	read: func(v *pageView) ReceiptData {
		return ReceiptData{
			Receipt: readTikTokPickupCode(v.rows),
			OrderID: readTikTokShopOrderID(v.lines),
		}.with(readTikTokShopRecipient(v.rows))
	},
}

var (
	// "Nomor Order: 585600000000000001" — TikTok's eighteen-digit order id, under another caption than
	// the "TT Order ID：" of its courier labels.
	tiktokShopOrderID = regexp.MustCompile(`Nomor\s*Order\s*:\s*(\d{16,20})(?:\D|$)`)
	// The pickup code is six capitals and digits, alone on the row under its caption, printed large.
	// Measured on one sample, and kept that narrow.
	tiktokPickupCode = regexp.MustCompile(`^[0-9A-Z]{6}$`)
)

func readTikTokShopOrderID(lines []string) string {
	for _, line := range lines {
		m := tiktokShopOrderID.FindStringSubmatch(line)
		if m != nil {
			return m[1]
		}
	}

	return ""
}

// readTikTokPickupCode is the code on the row under the "Kode Pengambilan" caption — one word, with a
// digit in it, so a word of the label can't pass for one.
func readTikTokPickupCode(rows []row) string {
	for i, r := range rows {
		if strings.TrimSuffix(r.text(), ":") != "Kode Pengambilan" || i+1 >= len(rows) {
			continue
		}
		next := rows[i+1].words
		if len(next) == 1 && tiktokPickupCode.MatchString(next[0].s) && strings.ContainsAny(next[0].s, "0123456789") {
			return next[0].s
		}
	}

	return ""
}

// valueAfterColon is what a caption row says after its first colon: "Penerima : Budi" is "Budi".
func valueAfterColon(r row) string {
	_, value, _ := strings.Cut(r.text(), ":")

	return strings.TrimSpace(value)
}

// readTikTokShopRecipient reads three caption rows: "Penerima : <name>", "Nomor Telepon : <phone>" and
// "Alamat :", whose address follows on rows of its own. The name is printed in full; the phone is
// masked, and comes back "".
func readTikTokShopRecipient(rows []row) recipient {
	out := recipient{}
	for i, r := range rows {
		switch {
		case strings.HasPrefix(r.text(), "Penerima :"), strings.HasPrefix(r.text(), "Penerima:"):
			out.name = printed(valueAfterColon(r))
		case strings.HasPrefix(r.text(), "Nomor Telepon"):
			out.phone = printed(valueAfterColon(r))
		case strings.TrimSpace(strings.TrimSuffix(r.text(), ":")) == "Alamat":
			out.address = addressBlock(rows[i+1:], r.words[0].size, math.Inf(-1), math.Inf(1))
		}
	}

	return out
}
