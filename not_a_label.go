package san_receipt_readers

import (
	"strings"
	"unicode"
)

// notAShippingLabel decides, for a file no layout read, whether it is no courier's label at all
// (ErrNotShippingLabel) rather than a label the package has yet to learn (errUnknownLayout). The signs
// are measured, not learned, and any one of them is enough:
//
//   - no image on any page. Every real label carries one — a logo, a barcode or a QR — and the spam (an
//     address label designed in Canva) carries none (a-non-label-gets-its-own-error);
//   - no text, and no barcode or QR read from the images: a label printed as a picture always carries
//     one, and a screenshot of a document carries none (a-picture-with-no-code-is-not-a-label);
//   - no barcode or QR, and no word shaped like a tracking number (hasTrackingShape): every real label
//     prints its tracking number or encodes it, and a note designed in Canva over a background picture
//     does neither (pdc_samples/04.pdf, a-note-with-no-tracking-number-is-not-a-label).
//
// The barcodes of a page with text are decoded here only, on the way to an error — a label a layout
// reads never pays for it.
func notAShippingLabel(views []*pageView, pictured, written, shaped bool) bool {
	if !pictured {
		return true
	}

	coded := false
	for _, v := range views {
		if !written || !shaped {
			coded = coded || len(v.barcodes()) > 0
		}
	}

	return (!written && !coded) || (!shaped && !coded)
}

// hasTrackingShape reports a word that could be a tracking number: a run of capitals, digits and dashes,
// ten to thirty long, with ten digits or more, or six digits or more and a letter. A date ("02102026",
// "2026-09-28") or a sort code ("TTE-TTE024A-TA") is not one. A phone number written with dashes is —
// which only ever makes a file "unknown" rather than "not a label", the side that keeps it as a sample.
func hasTrackingShape(lines []string) bool {
	for _, line := range lines {
		words := strings.FieldsFunc(line, func(r rune) bool {
			return !unicode.IsDigit(r) && !(r >= 'A' && r <= 'Z') && r != '-'
		})
		for _, w := range words {
			if len(w) < 10 || len(w) > 30 {
				continue
			}
			digits, letters := 0, 0
			for _, r := range w {
				switch {
				case unicode.IsDigit(r):
					digits++
				case r != '-':
					letters++
				}
			}
			if digits >= 10 || (letters > 0 && digits >= 6) {
				return true
			}
		}
	}

	return false
}
