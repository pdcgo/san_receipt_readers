package san_receipt_readers

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/ledongthuc/pdf"
)

// maxLogoPixels bounds which images get fingerprinted. Logos run from 65×59 (HALU) to 683×157
// (J&T Cargo, on the TikTok label); the TikTok label's 2000×2000 QR code is twelve megabytes
// decoded and is not worth it to find a logo.
const maxLogoPixels = 250_000

// courierLogos are the courier logos measured on the samples (imageFingerprints), whichever
// template carries them.
//
// A logo is compared byte for byte, so it only ever CONFIRMS a courier — the same logo comes in
// several sizes (SPX 68×27 and 77×24, SiCepat 122×34, 136×45, 140×40 and 145×40, J&T Cargo 683×157 and 680×156), and a logo not in this table names
// nothing. Each new size is added when its sample arrives; the owner kept logos as courier evidence
// rather than dropping them (a-logo-still-names-the-courier, context_decision.md). Each layout
// decides what an unknown logo means.
//
// A var only so a test can stand in a synthetic logo.
//
// A logo IS byte-identical wherever it is drawn at the same size: the Shopee logo hashes the same
// on all seven Shopee samples, and SPX's 68×27 on both spx_01 and spx_03. The sizes vary between
// template versions, not between prints.
var courierLogos = map[string]CourierType{
	"68x27:dd3cde93d01d404f":   courierSPX,         // spx_01.pdf (instant), spx_03.pdf (ECO)
	"77x24:1ac075c83801bde9":   courierSPX,         // spx_02.pdf (STD)
	"127x29:3428676592f95e11":  courierGrabExpress, // shopee_grab_express_01.pdf, Shopee same-day
	"122x34:1cd0bf0789f59c6e":  courierSicepat,     // shopee_sicepat_01.pdf, Shopee REG
	"145x40:2667e991e6601fab":  courierSicepat,     // shopee_sicepat_02.pdf, Shopee REG, an older print
	"140x40:72d36f5ab3d03352":  courierSicepat,     // shopee_reg_01.pdf, Shopee REG, a third size
	"136x45:ac90669482b56efe":  courierSicepat,     // shopee_reg_03.pdf, Shopee REG, a fourth size (a larger page)
	"65x59:e5323fc7872323ad":   courierSicepat,     // shopee_halu_01.pdf, HALU's own logo
	"40x40:8a9d7ab99b93bab0":   courierSicepat,     // shopee_nextday_01.pdf, SiCepat BEST (next day)
	"683x157:2e69f4f857a5e1d9": courierJNTCargo,    // tiktok_tokped_jnt_cargo.pdf, TikTok / Tokopedia
	"680x156:6f86e93b2760b241": courierJNTCargo,    // tiktok_tokped_jnt_01.pdf, the same logo a size smaller
	"84x43:cf1ef151f9a459a7":   courierJNTCargo,    // shopee_cargo_01.pdf, Shopee CARGO
}

// logoCourier is the courier a known logo on the page names, or "".
func logoCourier(v *pageView) CourierType {
	for fp, courier := range courierLogos {
		if v.images[fp] {
			return courier
		}
	}

	return ""
}

// hasImage reports an image anywhere on the page, its forms included, whatever its size and
// whether or not it can be decoded. Every real label has one; the spam has none (ErrNotShippingLabel).
func hasImage(p pdf.Page) bool {
	found := false
	eachResources(p.Resources(), func(res pdf.Value) {
		xobjects := res.Key("XObject")
		for _, name := range xobjects.Keys() {
			if xobjects.Key(name).Key("Subtype").Name() == "Image" {
				found = true
			}
		}
	})

	return found
}

// imageFingerprints names the small images drawn on a page by their size and a hash of their
// decoded pixels.
//
// A courier's name can be printed ONLY as its logo: the SPX label says "SPX" nowhere in its
// text, and the same Shopee template is printed for the other instant couriers. A logo is what
// tells them apart. Images inside the page's forms count too: the Lazada label draws its whole face
// as one form.
func imageFingerprints(p pdf.Page) map[string]bool {
	out := map[string]bool{}

	eachResources(p.Resources(), func(res pdf.Value) {
		xobjects := res.Key("XObject")
		for _, name := range xobjects.Keys() {
			img := xobjects.Key(name)
			if img.Key("Subtype").Name() != "Image" {
				continue
			}

			w, h := img.Key("Width").Int64(), img.Key("Height").Int64()
			if w <= 0 || h <= 0 || w*h > maxLogoPixels {
				continue
			}

			fp, ok := fingerprint(img, w, h)
			if ok {
				out[fp] = true
			}
		}
	})

	return out
}

// fingerprint hashes one image's decoded pixels AND its alpha mask.
//
// The mask is not decoration: the J&T Cargo logo's colour data is solid black, and its whole shape
// is in the mask. Hashing the colour alone would name any black rectangle of that size J&T Cargo.
//
// The PDF library panics on a filter it cannot decode — a JPEG is DCTDecode — so an image it
// cannot read is skipped, not allowed to fail the whole label.
func fingerprint(img pdf.Value, w, h int64) (fp string, ok bool) {
	defer func() {
		p := recover()
		if p != nil {
			fp, ok = "", false
		}
	}()

	hash := sha256.New()

	data, err := io.ReadAll(img.Reader())
	if err != nil {
		return "", false
	}
	hash.Write(data)

	mask := img.Key("SMask")
	if !mask.IsNull() {
		alpha, err := io.ReadAll(mask.Reader())
		if err != nil {
			return "", false
		}
		hash.Write(alpha)
	}

	return fmt.Sprintf("%dx%d:%s", w, h, hex.EncodeToString(hash.Sum(nil)[:8])), true
}
