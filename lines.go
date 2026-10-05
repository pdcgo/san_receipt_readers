package san_receipt_readers

import (
	"math"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/ledongthuc/pdf"
)

// The PDF library hands back one glyph at a time with its position, and cannot group them
// itself: its own GetTextByRow returned the J&T label's whole page as ONE row. So the grouping
// is done here, from positions — corrected first by positionGlyphs, since the library also
// reads no widths from the fonts these labels use.
const (
	// A glyph joins a line when its baseline is within this fraction of its size. Lines on a
	// label are a full size apart, while a label and its value can sit half a point off.
	lineTolerance = 0.35
	// With widths known, a gap wider than this fraction of the size after the previous glyph's
	// END starts a new word. It sits between kerning (a few hundredths of an em) and a space
	// (~0.28 em) because PDFium — the SPX label's generator — often draws a space as a GAP between
	// two runs rather than as a space glyph: at 0.5 "DKI JAKARTA" came out "DKIJAKARTA".
	columnGap = 0.2
	// Without a width, only the START of the previous glyph is known, so the threshold has to
	// cover a whole glyph's advance as well.
	wordGap = 1.05
	// Text turned 90° comes back with a size of cos 90° times its real size: zero, or a rounding
	// remainder of it — the SPX label's right edge reports 3.7e-16, which "≤ 0" let through. No
	// printed text on a label is below a point.
	minFontSize = 1.0
	// Two glyphs are the same size when they differ by less than this — sizes come out of a matrix
	// product, so equal ones can differ in the last digits.
	sizeTolerance = 0.05
	// A glyph drawn again within this fraction of its size is one glyph printed twice. The KiriminAja
	// label fakes bold by drawing a whole run twice 0.034 em apart, and "NON-COD" came out "NNOONN--
	// CCOODD". A letter that really repeats is a glyph's width further on: 0.24 em even for "ll".
	overprintShift = 0.1
)

// word is a run of glyphs with no word break inside it, and where it sits on the page.
type word struct {
	x0, x1 float64
	size   float64
	s      string
}

// row is one printed line, its words left to right.
//
// trailingSpace records that the row ENDS in a space glyph. A wrapped address is where it matters:
// a box that wraps at any character breaks a word in two ("Gro" / "gol Sel") as readily as it
// breaks between words, and a hanging space is what tells the two apart (joinWrapped).
type row struct {
	y             float64
	words         []word
	trailingSpace bool
	// glyphs are the row's glyphs left to right, kept so a row can be rebuilt from a subset (sized).
	glyphs []pdf.Text
}

// sized is the row rebuilt from the glyphs of ONE font size.
//
// A label lays other text over its address: the SPX label's grey "COD" watermark runs through it
// ("JAWA TIMUR" came out "JAWCA TIMUROD"), and a hidden element overlaps a SiCepat address glyph by
// glyph ("Sukamaju" came out "1S0uk0am2aJju"). Both are a different size from the address.
func (r row) sized(size float64) row {
	keep := []pdf.Text{}
	for _, g := range r.glyphs {
		if math.Abs(g.FontSize-size) < sizeTolerance {
			keep = append(keep, g)
		}
	}
	if len(keep) == 0 {
		return row{y: r.y}
	}

	return rowOf(keep)
}

// leftOf is the row's words that start left of x — one column of a two-column label.
func (r row) leftOf(x float64) []word {
	out := []word{}
	for _, w := range r.words {
		if w.x0 < x {
			out = append(out, w)
		}
	}

	return out
}

// joinWords is words as one string, separated by one space.
func joinWords(words []word) string {
	parts := make([]string, len(words))
	for i, w := range words {
		parts[i] = w.s
	}

	return strings.Join(parts, " ")
}

// text is the row as one string, words separated by one space.
func (r row) text() string {
	return joinWords(r.words)
}

// pageLines rebuilds a page's text as lines, top to bottom, words separated by one space.
func pageLines(texts []pdf.Text, metrics map[string]fontMetrics) []string {
	return linesOf(pageRows(texts, metrics))
}

// linesOf is rows as plain text.
func linesOf(rows []row) []string {
	lines := make([]string, 0, len(rows))
	for _, r := range rows {
		lines = append(lines, r.text())
	}

	return lines
}

// pageRows rebuilds a page's text as rows of positioned words, top to bottom.
func pageRows(texts []pdf.Text, metrics map[string]fontMetrics) []row {
	glyphs := []pdf.Text{}
	for _, t := range positionGlyphs(texts, metrics) {
		// Rotated text is dropped. On both the J&T and the SPX label it is only the copies of the
		// tracking number printed up the edges, which the label also prints upright.
		if math.Abs(t.FontSize) < minFontSize {
			continue
		}
		glyphs = append(glyphs, t)
	}

	// PDF y grows UP the page, so the top line has the largest y.
	sort.SliceStable(glyphs, func(a, b int) bool {
		return glyphs[a].Y > glyphs[b].Y
	})

	grouped := [][]pdf.Text{}
	for _, g := range glyphs {
		last := len(grouped) - 1
		if last >= 0 && math.Abs(grouped[last][0].Y-g.Y) <= g.FontSize*lineTolerance {
			grouped[last] = append(grouped[last], g)
			continue
		}
		grouped = append(grouped, []pdf.Text{g})
	}

	rows := make([]row, 0, len(grouped))
	for _, glyphs := range grouped {
		r := rowOf(glyphs)
		if len(r.words) > 0 {
			rows = append(rows, r)
		}
	}

	return rows
}

// rowOf splits one line's glyphs into words: at a space glyph, or at a gap (startsNewWord).
func rowOf(glyphs []pdf.Text) row {
	sort.SliceStable(glyphs, func(a, b int) bool {
		return glyphs[a].X < glyphs[b].X
	})
	glyphs = withoutOverprint(glyphs)

	r := row{y: glyphs[0].Y, glyphs: glyphs}
	var cur *word
	var prev *pdf.Text
	for i := range glyphs {
		g := &glyphs[i]
		if strings.TrimFunc(g.S, unicode.IsSpace) == "" {
			cur = nil
			r.trailingSpace = true
			continue
		}
		r.trailingSpace = false

		if cur == nil || startsNewWord(*prev, *g) {
			r.words = append(r.words, word{x0: g.X, size: g.FontSize})
			cur = &r.words[len(r.words)-1]
		}
		cur.s += g.S
		cur.x1 = g.X + g.W
		prev = g
	}

	return r
}

// withoutOverprint drops each glyph that is the glyph before it drawn again (overprintShift), from
// glyphs sorted by x.
//
// Only a glyph with a width is compared. Without one, a whole text-show can sit at one x
// (positionGlyphs), and "ll" there is two letters, not one printed twice.
func withoutOverprint(glyphs []pdf.Text) []pdf.Text {
	out := make([]pdf.Text, 0, len(glyphs))
	for _, g := range glyphs {
		last := len(out) - 1
		overprint := last >= 0 &&
			g.W > 0 &&
			g.S == out[last].S &&
			math.Abs(g.FontSize-out[last].FontSize) < sizeTolerance &&
			g.X-out[last].X < g.FontSize*overprintShift
		if overprint {
			continue
		}
		out = append(out, g)
	}

	return out
}

func startsNewWord(prev, g pdf.Text) bool {
	if prev.W > 0 {
		return g.X-(prev.X+prev.W) >= g.FontSize*columnGap
	}

	return g.X-prev.X >= g.FontSize*wordGap
}

// onlyLine is the line that is, in full, a match for `pattern` — when exactly one distinct line is.
// None, or two different ones, reads as nothing: a guess between two numbers is not a reading.
func onlyLine(lines []string, pattern *regexp.Regexp) string {
	found := ""
	for _, line := range lines {
		if !pattern.MatchString(line) || line == found {
			continue
		}
		if found != "" {
			return ""
		}
		found = line
	}

	return found
}

func containsLine(lines []string, needle string) bool {
	for _, line := range lines {
		if strings.Contains(line, needle) {
			return true
		}
	}

	return false
}
