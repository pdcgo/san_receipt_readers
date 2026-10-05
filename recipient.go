package san_receipt_readers

import (
	"math"
	"regexp"
	"strings"
)

// recipient is who the parcel goes to, as the label prints them.
type recipient struct {
	name    string
	phone   string
	address string
}

// with is the reading with the recipient filled in.
func (rd ReceiptData) with(r recipient) ReceiptData {
	rd.CustomerName = r.name
	rd.Phone = r.phone
	rd.Address = r.address

	return rd
}

// printed is the value as printed — or "" when the label MASKED it. TikTok prints its recipient as
// "a**i" and "(+62)81*******00": a name or a phone with stars in it is not the name or the phone,
// and must not be autofilled as one.
func printed(s string) string {
	s = strings.TrimSpace(s)
	if strings.Contains(s, "*") {
		return ""
	}

	return s
}

// rowStartingWith is the index of the first row whose first word begins with `prefix`, or -1.
func rowStartingWith(rows []row, prefix string) int {
	for i, r := range rows {
		if len(r.words) > 0 && strings.HasPrefix(r.words[0].s, prefix) {
			return i
		}
	}

	return -1
}

const (
	// A word belongs to the left column when it starts this far left of the right column's caption.
	columnSlack = 1.0
	// The sender's column on a Shopee label is aligned on its phone, but the city under it can start
	// up to 5 points further left ("KOTA" at 184 under a phone at 189). The recipient's address
	// column ends this far left of the phone.
	senderColumnSlack = 8.0
	// An address line starts at the block's margin within this many points. The region boxes under
	// a Shopee address are centred, and start 12 points or more to the right.
	marginTolerance = 3.0
	// Address lines are about one size apart. A line further than this many sizes below the one
	// before it is not the same address.
	maxLineGap = 2.5
	// Once two lines have set the block's spacing, a line further below than this many times it is
	// not the same address: the SPX reservation label's region box sits at the address's margin, in its
	// size, but 1.4 lines down. The spacing differs between labels (1.0 em on Shopee, 1.4 on
	// KiriminAja), so it is measured, not assumed.
	maxSpacingGrowth = 1.25
)

// barePhone is a phone printed as bare digits: the sender's on a Shopee label ("6281234567890"), the
// recipient's on a KiriminAja one ("081234567890").
var barePhone = regexp.MustCompile(`^(?:62|0)\d{8,13}$`)

// senderColumnX is where the sender's column starts under a Shopee "Penerima/Pengirim" row: the x
// of the sender's phone, the first bare phone number below. NaN when there is none.
func senderColumnX(rows []row) float64 {
	for _, r := range rows {
		for _, w := range r.words {
			if barePhone.MatchString(w.s) {
				return w.x0
			}
		}
	}

	return math.NaN()
}

// addressBlock reads an address printed as left-aligned lines of one font size, in the column of
// words starting from `from` up to `limit`.
//
// A row with no word of that size in the column is passed over — a row of the other column
// ("KOTA TANGERANG", the sender's phone) or a tag in another size (SiCepat's "HOME"). The block ends
// at the first row whose text does not start at the margin (the Shopee label's centred region boxes,
// or its "CASHLESS" banner), or that sits further below the line before it than the block's own
// spacing allows (maxSpacingGrowth, maxLineGap): on the Lazada label and the SPX reservation label,
// nothing else ends it.
//
// The lines are joined by a space. The box wraps at any character, so a word can be split across
// two lines ("Gro" / "gol Sel"), and nothing on the page tells that from a wrap between words — a
// split word stays split rather than two words being glued.
func addressBlock(rows []row, size, from, limit float64) string {
	margin, last, spacing := math.NaN(), math.NaN(), math.NaN()
	lines := []string{}
	for _, r := range rows {
		words := column(r.sized(size).leftOf(limit), from)
		if len(words) == 0 {
			continue
		}
		if len(lines) == 0 && len(words) == 1 && addressTags[strings.ToUpper(words[0].s)] {
			continue
		}
		if math.IsNaN(margin) {
			margin = words[0].x0
		}
		if math.Abs(words[0].x0-margin) > marginTolerance {
			break
		}
		if !math.IsNaN(last) {
			gap := last - r.y
			if gap > size*maxLineGap || gap > spacing*maxSpacingGrowth {
				break
			}
			if math.IsNaN(spacing) {
				spacing = gap
			}
		}
		lines = append(lines, joinWords(words))
		last = r.y
	}

	return strings.Join(lines, " ")
}

// addressTags are the labels a buyer gives an address in the marketplace app, which the label prints
// in a box above the address. Skipped only before the address starts: in the address's own size and
// indented into its box, "HOME" read as the whole address once, and ended it (shopee_eco_01.pdf).
var addressTags = map[string]bool{"HOME": true, "OFFICE": true, "RUMAH": true, "KANTOR": true}

// column is the words that start at `from` or right of it.
func column(words []word, from float64) []word {
	out := []word{}
	for _, w := range words {
		if w.x0 >= from {
			out = append(out, w)
		}
	}

	return out
}

// readShopeeRecipient reads the Shopee label's two-column block: "Penerima:" and the recipient's
// name on the left, "Pengirim:" and the sender on the right, and under each its address.
//
// The two columns are cut at DIFFERENT x. The name row is cut where the "Pengirim:" caption starts.
// The address rows are cut where the sender's column starts — and that is further right: the
// caption sits left of the sender's own text, and the recipient's address runs on past it
// ("RT 003 / RW" ends at 166 on the same-day label, under a caption at 143 and beside a sender
// column at 184). Cut at the caption, "/ RW" and "dusun" were lost.
//
// ⚠ A Shopee label prints no RECIPIENT phone — the one phone on it is the sender's, under
// "Pengirim:". Phone stays "".
func readShopeeRecipient(rows []row) recipient {
	i := rowStartingWith(rows, "Penerima:")
	if i < 0 {
		return recipient{}
	}

	head := rows[i]
	size := head.words[0].size
	senderX, fused := math.Inf(1), ""
	for _, w := range head.words {
		if at := strings.Index(w.s, "Pengirim:"); at >= 0 {
			senderX, fused = w.x0, w.s[:at]
			break
		}
	}
	nameLimit := senderX - columnSlack

	addressLimit := senderColumnX(rows[i+1:]) - senderColumnSlack
	if math.IsNaN(addressLimit) {
		addressLimit = nameLimit
	}

	// The caption and the name can come out as one word ("Penerima:Budi") — the gap between them is
	// narrower than a space.
	name := joinWords(head.sized(size).leftOf(nameLimit))
	name = strings.TrimPrefix(name, "Penerima:")
	// A long name runs into the caption with no gap ("(TOKO CEPengirim:"), and then its last letters
	// are the start of the caption's word (shopee_jtr_01.pdf).
	if fused != "" {
		name += " " + fused
	}

	return recipient{
		name:    printed(name),
		address: addressBlock(rows[i+1:], size, math.Inf(-1), addressLimit),
	}
}

// readTikTokRecipient reads the TikTok label's block: a "Penerima" row with the phone at its right,
// the name on the row under it, and the address under that. TikTok masks the name and the phone
// ("S** W**O", "(+62)81*******00"), so both come back "" — only the address is printed in full.
func readTikTokRecipient(rows []row) recipient {
	i := rowStartingWith(rows, "Penerima")
	if i < 0 || i+1 >= len(rows) {
		return recipient{}
	}

	phone := joinWords(rows[i].words[1:])
	nameRow := rows[i+1]

	return recipient{
		name:    printed(nameRow.text()),
		phone:   printed(phone),
		address: addressBlock(rows[i+2:], nameRow.words[0].size, math.Inf(-1), math.Inf(1)),
	}
}

// readJNTRecipient reads the older TikTok J&T label: "Penerima :a**i (+62)81*******11" on one row,
// then the region and the street, in two different sizes, down to the "Weight" row. With no
// "Weight" row to end it, the address is not read — rather than run on into the rest of the label.
func readJNTRecipient(rows []row) recipient {
	i := rowStartingWith(rows, "Penerima")
	if i < 0 {
		return recipient{}
	}

	name, phone := []string{}, ""
	for _, w := range rows[i].words[1:] {
		if strings.HasPrefix(w.s, "(+") {
			phone = w.s
			continue
		}
		name = append(name, w.s)
	}

	address := ""
	lines := []string{}
	for _, r := range rows[i+1:] {
		if strings.HasPrefix(r.text(), "Weight") {
			address = strings.Join(lines, " ")
			break
		}
		lines = append(lines, r.text())
	}

	return recipient{
		name:    printed(strings.TrimPrefix(strings.Join(name, " "), ":")),
		phone:   printed(phone),
		address: address,
	}
}
