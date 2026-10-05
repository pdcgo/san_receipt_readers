package san_receipt_readers

import (
	"fmt"
	"strings"

	"github.com/ledongthuc/pdf"
)

// The PDF library's Page.Content reads the page's own content stream and nothing it calls: a Form
// XObject — a drawing the page invokes with "Do" — is skipped whole. The Lazada label draws its
// ENTIRE face as one form, and came back with no text at all. So the text is read here, by the same
// rules as Page.Content (pageTexts returns what it returns, glyph for glyph, on a page without
// forms), with "Do" followed into each form.

// maxFormDepth bounds forms nested in forms, and maxForms how many form drawings a page may make in
// all. A form that draws itself, or two that draw each other, would otherwise never end.
const (
	maxFormDepth = 8
	maxForms     = 256
)

type matrix [3][3]float64

var identity = matrix{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}}

func (x matrix) mul(y matrix) matrix {
	var z matrix
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			for k := 0; k < 3; k++ {
				z[i][j] += x[i][k] * y[k][j]
			}
		}
	}

	return z
}

// matrixOf reads six operands — a b c d e f — as a PDF transformation matrix.
func matrixOf(args []pdf.Value) matrix {
	var m matrix
	for i := 0; i < 6; i++ {
		m[i/2][i%2] = args[i].Float64()
	}
	m[2][2] = 1

	return m
}

// textState is the part of the graphics state that places glyphs.
type textState struct {
	Tc, Th, Tl, Tfs, Trise float64
	Tf                     pdf.Font
	Tm, Tlm, CTM           matrix
}

// rawEncoding is the library's fallback when a font has no encoding: the bytes as they are.
type rawEncoding struct{}

func (rawEncoding) Decode(raw string) string {
	return raw
}

// malformed is the panic for an operator with the wrong operands — the library's own response, kept
// so a broken file fails the same way whichever code reads it. Extract recovers it as ErrUnreadable.
func malformed(op string) string {
	return fmt.Sprintf("receipt_readers: malformed %s operator", op)
}

// pageTexts is every glyph the page draws, in content order — Page.Content's Text, plus the text
// inside the forms the page draws.
func pageTexts(p pdf.Page) []pdf.Text {
	out := []pdf.Text{}
	contents := p.V.Key("Contents")
	if p.V.IsNull() || contents.IsNull() {
		return out
	}

	forms := 0
	readText(contents, p.Resources(), identity, 0, &forms, &out)

	return out
}

// readText interprets one content stream: the page's, or a form's with the form's own resources and
// its matrix applied to the caller's.
func readText(strm, resources pdf.Value, ctm matrix, depth int, forms *int, out *[]pdf.Text) {
	var enc pdf.TextEncoding = rawEncoding{}
	g := textState{Th: 1, CTM: ctm}
	stack := []textState{}

	show := func(s string) {
		n := 0
		for _, ch := range enc.Decode(s) {
			var w0 float64
			if n < len(s) {
				w0 = g.Tf.Width(int(s[n]))
			}
			n++

			font := g.Tf.BaseFont()
			if i := strings.Index(font, "+"); i >= 0 {
				font = font[i+1:]
			}

			trm := matrix{{g.Tfs * g.Th, 0, 0}, {0, g.Tfs, 0}, {0, g.Trise, 1}}.mul(g.Tm).mul(g.CTM)
			*out = append(*out, pdf.Text{
				Font:     font,
				FontSize: trm[0][0],
				X:        trm[2][0],
				Y:        trm[2][1],
				W:        w0 / 1000 * trm[0][0],
				S:        string(ch),
			})

			tx := (w0/1000*g.Tfs + g.Tc) * g.Th
			g.Tm = matrix{{1, 0, 0}, {0, 1, 0}, {tx, 0, 1}}.mul(g.Tm)
		}
	}

	nextLine := func() {
		g.Tlm = matrix{{1, 0, 0}, {0, 1, 0}, {0, -g.Tl, 1}}.mul(g.Tlm)
		g.Tm = g.Tlm
	}

	pdf.Interpret(strm, func(stk *pdf.Stack, op string) {
		args := make([]pdf.Value, stk.Len())
		for i := len(args) - 1; i >= 0; i-- {
			args[i] = stk.Pop()
		}
		need := func(n int) {
			if len(args) != n {
				panic(malformed(op))
			}
		}

		switch op {
		case "cm":
			need(6)
			g.CTM = matrixOf(args).mul(g.CTM)

		case "q":
			stack = append(stack, g)

		case "Q":
			if len(stack) == 0 {
				panic(malformed(op))
			}
			g = stack[len(stack)-1]
			stack = stack[:len(stack)-1]

		case "Do":
			need(1)
			form := resources.Key("XObject").Key(args[0].Name())
			if form.Key("Subtype").Name() != "Form" || depth >= maxFormDepth || *forms >= maxForms {
				return
			}
			*forms++

			m := identity
			if mx := form.Key("Matrix"); mx.Len() == 6 {
				m = matrixOf([]pdf.Value{mx.Index(0), mx.Index(1), mx.Index(2), mx.Index(3), mx.Index(4), mx.Index(5)})
			}
			// A form without resources of its own uses its caller's.
			res := form.Key("Resources")
			if res.IsNull() {
				res = resources
			}
			readText(form, res, m.mul(g.CTM), depth+1, forms, out)

		case "BT":
			g.Tm = identity
			g.Tlm = g.Tm

		case "T*":
			nextLine()

		case "Tc":
			need(1)
			g.Tc = args[0].Float64()

		case "TD":
			need(2)
			g.Tl = -args[1].Float64()
			fallthrough
		case "Td":
			need(2)
			g.Tlm = matrix{{1, 0, 0}, {0, 1, 0}, {args[0].Float64(), args[1].Float64(), 1}}.mul(g.Tlm)
			g.Tm = g.Tlm

		case "Tf":
			need(2)
			g.Tf = pdf.Font{V: resources.Key("Font").Key(args[0].Name())}
			enc = g.Tf.Encoder()
			if enc == nil {
				enc = rawEncoding{}
			}
			g.Tfs = args[1].Float64()

		case `"`:
			need(3)
			g.Tc = args[1].Float64()
			args = args[2:]
			fallthrough
		case "'":
			if len(args) != 1 {
				panic(malformed(op))
			}
			nextLine()
			fallthrough
		case "Tj":
			if len(args) != 1 {
				panic(malformed(op))
			}
			show(args[0].RawString())

		case "TJ":
			need(1)
			v := args[0]
			for i := 0; i < v.Len(); i++ {
				x := v.Index(i)
				if x.Kind() == pdf.String {
					show(x.RawString())
					continue
				}
				tx := -x.Float64() / 1000 * g.Tfs * g.Th
				g.Tm = matrix{{1, 0, 0}, {0, 1, 0}, {tx, 0, 1}}.mul(g.Tm)
			}
			// The library ends every TJ with a newline glyph; kept, so a page reads as it did.
			show("\n")

		case "TL":
			need(1)
			g.Tl = args[0].Float64()

		case "Tm":
			need(6)
			g.Tm = matrixOf(args)
			g.Tlm = g.Tm

		case "Ts":
			need(1)
			g.Trise = args[0].Float64()

		case "Tz":
			need(1)
			g.Th = args[0].Float64() / 100
		}
	})
}

// eachResources calls fn with `res` and with the resources of every form it holds, nested forms
// included, within the same bounds as readText.
func eachResources(res pdf.Value, fn func(pdf.Value)) {
	forms := 0

	var walk func(res pdf.Value, depth int)
	walk = func(res pdf.Value, depth int) {
		if res.IsNull() {
			return
		}
		fn(res)

		xobjects := res.Key("XObject")
		for _, name := range xobjects.Keys() {
			form := xobjects.Key(name)
			if form.Key("Subtype").Name() != "Form" || depth >= maxFormDepth || forms >= maxForms {
				continue
			}
			forms++
			walk(form.Key("Resources"), depth+1)
		}
	}
	walk(res, 0)
}
