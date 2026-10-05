// Package san_receipt_readers reads the shipping label (the "resi") a seller attaches to an
// order, and answers with the numbers and the recipient printed on it.
//
// A label is a PDF printed from some app, so its layout belongs to the app that GENERATED it,
// not to the courier. Each layout is one file here, recognised by what is printed on the label
// itself — the caller hands over a file and nothing else.
package san_receipt_readers

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/ledongthuc/pdf"
)

// CourierType is which courier's label it is.
//
// ⚠ The contract no longer returns it — the owner dropped Courier from ReceiptData as too hard to
// extract (courier-is-not-read, context_decision.md). It still names a logo's courier internally,
// where a layout needs the logo to MATCH (the SiCepat REG and J&T Cargo labels name their courier
// nowhere else).
type CourierType string

// The couriers a logo in courierLogos can name. Unexported: nothing returns them.
const (
	courierSPX         CourierType = "spx"
	courierSicepat     CourierType = "sicepat"
	courierGrabExpress CourierType = "grabexpress"
	// J&T's freight brand, a different service from J&T Express.
	courierJNTCargo CourierType = "jntcargo"
)

// ReceiptData is what a label says. An empty string means the label does not print it — that
// is not an error, the same meaning the order form gives an empty scan.
//
// A MASKED value is not printed either: TikTok prints the recipient as "a**i" and
// "(+62)81*******00", and those come back "" (see masked).
type ReceiptData struct {
	// Receipt is the tracking number — what lands in orders.receipt.
	Receipt string
	// OrderID is the MARKETPLACE's order number — orders.order_external_ref_id, not orders.id.
	OrderID string
	// Phone is the RECIPIENT's phone. ⚠ No sample prints it: a Shopee label prints only the
	// sender's, and a TikTok label masks it.
	Phone string
	// CustomerName is the recipient's name.
	CustomerName string
	// Address is the recipient's street address as one string, its printed lines joined by a
	// space. ⚠ A label's address box wraps inside words too ("Gro" / "gol Sel"), and a wrap
	// inside a word can't be told from one between words, so such a word stays split.
	Address string
}

// maxSize caps what Extract will buffer. A label is tens of KB — the one sample is 60 KB and
// already allocates ~12 MB to read — while the order form's 10 MB upload limit exists for photos.
const maxSize = 2 << 20

// Two errors are exported, and they never overlap (unreadable-and-not-a-label-are-two-errors,
// context_decision.md): ErrUnreadable — the file could not be read at all — and ErrNotShippingLabel —
// it was read, and it is no courier's label. The rest stay unexported: the contract declares no other
// errors yet (critique 4 in context_clarify.md).
var (
	// ErrUnreadable is a file that could not be read at all: not a PDF, over 2 MB, a PDF the library
	// cannot parse, or a reader that failed. Nothing on it was looked at.
	ErrUnreadable = errors.New("receipt_readers: the file could not be read")

	errNotPDF           = fmt.Errorf("%w: not a pdf", ErrUnreadable)
	errTooLarge         = fmt.Errorf("%w: over 2 MB", ErrUnreadable)
	errUnknownLayout    = errors.New("receipt_readers: no known label layout")
	errMultipleReceipts = errors.New("receipt_readers: more than one label in the file")
)

// ErrNotShippingLabel is a file that was READ, and is no courier's label at all — spam, to the reader
// (a-file-that-is-not-a-courier-label-is-refused, context_decision.md), like the address label
// designed in Canva. A caller can skip such a file rather than treat it as a label the package has yet
// to learn. It also matches errUnknownLayout: it is that refusal, with the reason known.
//
// How a file is told to be no label at all — no image; no text and no code; or no code and no word
// shaped like a tracking number — is notAShippingLabel. Anything else stays errUnknownLayout: it may be a
// courier's label the package hasn't seen yet.
var ErrNotShippingLabel = errors.New("receipt_readers: not a shipping label")

// Extract reads a label PDF and returns what it prints.
//
// It reads every page. Exactly one page must carry a known label: none is errUnknownLayout, and
// more than one is errMultipleReceipts — a bulk print is never silently read as its first page,
// which would hand one order another order's numbers.
func Extract(data io.Reader) (result ReceiptData, err error) {
	raw, err := io.ReadAll(io.LimitReader(data, maxSize+1))
	if err != nil {
		return ReceiptData{}, fmt.Errorf("%w: %w", ErrUnreadable, err)
	}
	if len(raw) > maxSize {
		return ReceiptData{}, errTooLarge
	}
	// The spec lets the header sit anywhere in the first 1024 bytes.
	if !bytes.Contains(raw[:min(len(raw), 1024)], []byte("%PDF-")) {
		return ReceiptData{}, errNotPDF
	}

	// The PDF library panics on some malformed content (`panic("bad g.Tc")` and others). A label
	// is uploaded by a person, so a broken one is an answer, never a crashed server.
	defer func() {
		p := recover()
		if p != nil {
			result, err = ReceiptData{}, fmt.Errorf("%w: %v", ErrUnreadable, p)
		}
	}()

	reader, err := pdf.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return ReceiptData{}, fmt.Errorf("%w: %v", ErrUnreadable, err)
	}

	found, views := []ReceiptData{}, []*pageView{}
	pictured, written, shaped := false, false, false
	for i := 1; i <= reader.NumPage(); i++ {
		page := reader.Page(i)
		if page.V.IsNull() {
			continue
		}
		pictured = pictured || hasImage(page)

		rows := pageRows(pageTexts(page), fontMetricsOf(page))
		view := &pageView{
			rows:   rows,
			lines:  linesOf(rows),
			images: imageFingerprints(page),
			page:   page,
			raw:    raw,
		}
		// A page with no text is a label printed as pictures. Its barcodes are all it says.
		if len(rows) == 0 {
			view.barcodes()
		}
		written = written || len(rows) > 0
		shaped = shaped || hasTrackingShape(view.lines)
		views = append(views, view)
		for _, l := range layouts {
			if !l.match(view) {
				continue
			}
			found = append(found, l.read(view))
			break
		}
	}

	switch len(found) {
	case 0:
		if notAShippingLabel(views, pictured, written, shaped) {
			return ReceiptData{}, fmt.Errorf("%w (%w)", ErrNotShippingLabel, errUnknownLayout)
		}
		return ReceiptData{}, errUnknownLayout
	case 1:
		return found[0], nil
	default:
		return ReceiptData{}, fmt.Errorf("%w: %d labels", errMultipleReceipts, len(found))
	}
}

// pageView is what a layout gets to look at: the page's text as lines, and its logos.
type pageView struct {
	// rows are the page's printed lines as positioned words; lines are the same rows as plain text.
	rows  []row
	lines []string
	// images holds the fingerprints of the page's small images — see imageFingerprints.
	images map[string]bool
	// codes are what the barcodes say — decoded up front only on a page with no text (pageCodes), and
	// on any other page only when a layout asks (barcodes).
	codes   []string
	decoded bool
	page    pdf.Page
	raw     []byte
}

// barcodes is what the page's barcodes say, decoded the first time a layout asks. Decoding costs a
// few milliseconds an image, so a page a layout reads from its text never pays for it.
func (v *pageView) barcodes() []string {
	if !v.decoded {
		v.codes = pageCodes(v.page, v.raw)
		v.decoded = true
	}

	return v.codes
}

// layout is one generator's label for one courier.
type layout struct {
	name string
	// match reports whether this page is this layout — by a marker printed on it.
	match func(v *pageView) bool
	read  func(v *pageView) ReceiptData
}

// layouts is tried in order, and the first match wins.
var layouts = []layout{
	jntLayout,
	// Before the instant layout: a Shopee label that prints an SPX tracking number is read for it,
	// whatever else it says.
	spxLayout,
	sicepatLayout,
	shopeeJNTCargoLayout,
	shopeeCMLayout,
	shopeeAnterAjaLayout,
	shopeeJTRLayout,
	shopeeIDExpressLayout,
	shopeePosLayout,
	shopeeInstantLayout,
	jntCargoLayout,
	tiktokInstantLayout,
	tiktokAnterAjaLayout,
	kiriminAjaLayout,
	lazadaLayout,
	jntPictureLayout,
	spxPictureLayout,
	// Last: a Shopee label no courier layout knows, read by its Resi box when a barcode confirms it.
	shopeeBarcodeLayout,
}
