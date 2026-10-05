package san_receipt_readers

import (
	"bytes"
	"image"
	"image/jpeg"
	"io"
	"regexp"

	"github.com/ledongthuc/pdf"
	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/oned"
	"github.com/makiuchi-d/gozxing/qrcode"
)

// maxScanPixels bounds an image decoded to look for a barcode: 2000×2000, four megabytes as grey.
// The J&T label printed as a picture is 1000×1000.
const maxScanPixels = 4_000_000

// maxScanImages bounds how many images one page is scanned for barcodes. The Shopee label printed
// through Microsoft Print To PDF carries five: two logos, a QR code and two Code 128s.
const maxScanImages = 16

// pageCodes is what the barcodes on the page's images say: each image's QR code, then its Code 128.
//
// It is read only for a page with NO TEXT — a label printed as pictures. Two shapes so far:
//
//   - one picture of the whole label (jnt_eco_01.pdf, re-wrapped by pdfcpu);
//   - the label's words drawn as outlines, with one JPEG per graphic (shopee_std_01.pdf, Microsoft
//     Print To PDF). Its codes are in images SMALLER than its logo, so every image is scanned, not
//     the largest.
//
// Its order id and recipient are pixels this package cannot read; a barcode it can. An image that
// can't be decoded gives no codes, never a failed label.
func pageCodes(p pdf.Page, raw []byte) []string {
	codes := []string{}
	for _, img := range pageImages(p, raw) {
		codes = append(codes, imageCodes(img)...)
	}

	return codes
}

// imageCodes is what one image's QR code and Code 128 say.
func imageCodes(img image.Image) []string {
	bitmap, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		return nil
	}
	hints := map[gozxing.DecodeHintType]interface{}{gozxing.DecodeHintType_TRY_HARDER: true}

	codes := []string{}
	readers := []gozxing.Reader{qrcode.NewQRCodeReader(), oned.NewCode128Reader()}
	for _, reader := range readers {
		result, err := reader.Decode(bitmap, hints)
		if err == nil {
			codes = append(codes, result.GetText())
		}
	}

	return codes
}

// pageImages is the page's images (its forms' included), decoded, up to maxScanImages. An image
// too large, in a format this package doesn't decode, or broken, is left out.
func pageImages(p pdf.Page, raw []byte) []image.Image {
	values := []pdf.Value{}
	eachResources(p.Resources(), func(res pdf.Value) {
		xobjects := res.Key("XObject")
		for _, name := range xobjects.Keys() {
			img := xobjects.Key(name)
			if img.Key("Subtype").Name() == "Image" && len(values) < maxScanImages {
				values = append(values, img)
			}
		}
	})

	images := []image.Image{}
	for _, v := range values {
		img := decodeImage(v, raw)
		if img != nil {
			images = append(images, img)
		}
	}

	return images
}

// decodeImage is one image XObject as an image, or nil.
//
// The PDF library panics on a filter it doesn't implement, and on some it does, so each image is
// guarded on its own: one broken picture must not cost the page the codes in the others.
func decodeImage(v pdf.Value, raw []byte) (img image.Image) {
	defer func() {
		if recover() != nil {
			img = nil
		}
	}()

	w, h := v.Key("Width").Int64(), v.Key("Height").Int64()
	if w <= 0 || h <= 0 || w*h > maxScanPixels {
		return nil
	}
	if v.Key("Filter").Name() == "DCTDecode" {
		return jpegImage(raw, v.Key("Length").Int64(), int(w), int(h))
	}

	return rawImage(v, int(w), int(h))
}

// rawImage is an 8-bit RGB or grey image whose stream the PDF library decodes, as grey.
func rawImage(v pdf.Value, w, h int) image.Image {
	if v.Key("BitsPerComponent").Int64() != 8 {
		return nil
	}
	channels := map[string]int{"DeviceRGB": 3, "DeviceGray": 1}[v.Key("ColorSpace").Name()]
	if channels == 0 {
		return nil
	}

	data, err := io.ReadAll(v.Reader())
	if err != nil || len(data) < w*h*channels {
		return nil
	}

	gray := image.NewGray(image.Rect(0, 0, w, h))
	for i := range gray.Pix {
		if channels == 1 {
			gray.Pix[i] = data[i]
			continue
		}
		r, g, b := int(data[3*i]), int(data[3*i+1]), int(data[3*i+2])
		gray.Pix[i] = uint8((r + 2*g + b) / 4)
	}

	return gray
}

// jpegImage decodes the JPEG an image XObject carries.
//
// ⚠ The PDF library has no JPEG filter and no way to hand over a stream UNdecoded, nor its offset,
// so the stream is found in the file's own bytes, by what only it can be: data that starts as a
// JPEG (FF D8 FF), is exactly the Length the image's dictionary declares, is followed by
// `endstream`, and whose own header gives the image's width and height. A stream that is not
// stored plainly (an encrypted file) matches nothing, and the picture gives no codes.
func jpegImage(raw []byte, length int64, w, h int) image.Image {
	if length <= 0 {
		return nil
	}

	for _, at := range streamStarts.FindAllIndex(raw, -1) {
		start := at[1]
		end := start + int(length)
		if end > len(raw) || !bytes.HasPrefix(raw[start:], jpegStart) {
			continue
		}
		if !bytes.HasPrefix(bytes.TrimLeft(raw[end:], "\r\n"), []byte("endstream")) {
			continue
		}

		data := raw[start:end]
		config, err := jpeg.DecodeConfig(bytes.NewReader(data))
		if err != nil || config.Width != w || config.Height != h {
			continue
		}

		img, err := jpeg.Decode(bytes.NewReader(data))
		if err != nil {
			return nil
		}

		return img
	}

	return nil
}

var (
	// streamStarts is the `stream` keyword and its line end; the data starts right after it. The
	// keyword ends `endstream` too, but no JPEG starts after one of those.
	streamStarts = regexp.MustCompile(`stream\r?\n`)
	jpegStart    = []byte{0xFF, 0xD8, 0xFF}
)

// agreedCode is the code of `pattern`'s shape that a page's barcodes carry — when every one that
// has that shape says the same. Two different numbers are a guess.
func agreedCode(codes []string, pattern *regexp.Regexp) string {
	found := ""
	for _, code := range codes {
		if !pattern.MatchString(code) || code == found {
			continue
		}
		if found != "" {
			return ""
		}
		found = code
	}

	return found
}
