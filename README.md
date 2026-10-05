# `san_receipt_readers`

Reads the shipping label (the "resi") a seller attaches to an order, and returns the tracking number, the
marketplace's order id, and the recipient printed on it.

- **Spec:** [docs/technical/packages/receipt_readers/context.md](https://github.com/pdcgo/warehouse_revamp/blob/dev/docs/technical/packages/receipt_readers/context.md) (owner-written)
- **Decisions:** [context_decision.md](https://github.com/pdcgo/warehouse_revamp/blob/dev/docs/technical/packages/receipt_readers/context_decision.md)
- **Open questions / findings:** [context_clarify.md](https://github.com/pdcgo/warehouse_revamp/blob/dev/docs/technical/packages/receipt_readers/context_clarify.md)
- **Runnable usage:** [example_test.go](example_test.go), compiled by `go test` so it cannot drift

```go
import "github.com/pdcgo/san_receipt_readers"

receipt, err := san_receipt_readers.Extract(file)
// ReceiptData{Receipt, OrderID, Phone, CustomerName, Address} — "" = not printed, or masked
```

No courier is returned: the owner dropped it as too hard to extract.

This repo is a submodule of [warehouse_revamp](https://github.com/pdcgo/warehouse_revamp), checked out at
`backend/packages/san_receipt_readers`. It's developed there, and its design docs live there.

## Try a real label

Drop the PDF into warehouse_revamp's `examples/receipt_file_samples/`, three folders up (untracked: a label
carries a buyer's address, and both repositories are public; `*.pdf` is gitignored here), then, from this
folder:

```sh
go test -run TestSamples -v
RECEIPT_SAMPLES=/some/folder go test -run TestSamples -v
```

Each file is one subtest that logs all five fields. A label this package does not know yet **fails**,
saying why.

## What it reads today

| layout | recognised by | file |
| --- | --- | --- |
| TikTok J&T Express (`wkhtmltopdf`) | `www.jet.co.id` on the label | [jnt.go](jnt.go) |
| TikTok / Tokopedia J&T Cargo (`wkhtmltopdf`) | `TT Order ID` + 12 digits alone on a line + the J&T Cargo logo (two sizes so far) | [jnt_cargo.go](jnt_cargo.go) |
| TikTok Shop instant / same-day (`wkhtmltopdf`) | `Nomor Order:` + an 18-digit id + a six-character pickup code under `Kode Pengambilan` | [tiktok_instant.go](tiktok_instant.go) |
| TikTok / Tokopedia AnterAja (`wkhtmltopdf`, the "tokopedia | Shop" logo) | `Order ID：` (no `TT`) + three capitals, a dash and eleven digits alone on a line | [tiktok_anteraja.go](tiktok_anteraja.go) |
| Shopee SPX `STD` / `ECO` (`PDFium`) | `Pesanan:` + an `SPXID…` number printed at least twice | [spx.go](spx.go) |
| Shopee SiCepat `HALU` / `REG` / `BEST` (next day) (`PDFium`) | `Pesanan:` + a 12-digit number + (a `HALU` header **or** a known SiCepat logo; `REG` is also JNE's) | [sicepat.go](sicepat.go) |
| Shopee J&T Cargo, `CARGO` (`PDFium`) | `Pesanan:` + twelve digits + the J&T Cargo logo (the digits are SiCepat's shape too) | [shopee_jnt_cargo.go](shopee_jnt_cargo.go) |
| Shopee, a `CM…` number, probably JNE by its prefix (`PDFium`) | `Pesanan:` + `CM` and eleven digits, printed most often or in its `Resi:` box | [shopee_cm.go](shopee_cm.go) |
| Shopee AnterAja (`PDFium`) | `Pesanan:` + fourteen digits, printed most often or in its `Resi:` box | [shopee_anteraja.go](shopee_anteraja.go) |
| Shopee JNE Trucking, `JTR` (`PDFium`) | `Pesanan:` + `JT` and eleven digits, printed most often or in its `Resi:` box | [shopee_jtr.go](shopee_jtr.go) |
| Shopee ID Express (`PDFium`) | `Pesanan:` + `ID`, one capital and thirteen digits (`IDS…`), printed most often or in its `Resi:` box | [shopee_idexpress.go](shopee_idexpress.go) |
| Shopee Pos Indonesia (`PDFium`) | `Pesanan:` + `SHPE` and eighteen capitals and digits, printed most often or in its `Resi:` box | [shopee_pos.go](shopee_pos.go) |
| Shopee instant / same-day (`PDFium`) | `Pesanan:` + `Kode Pengambilan:` + a known header (`INSTANT / SAMEDAY`, `Same-day`, `SAMEDAY`, `Instant`) | [shopee_instant.go](shopee_instant.go) |
| KiriminAja, any courier it books (`wkhtmltopdf`) | `KiriminAja` on a caption row + a code under its `AWB` caption | [kiriminaja.go](kiriminaja.go) |
| Lazada (`PDFium`, the whole face one form) | `Diserahkan ke` + `Diantar oleh` + a tracking number alone on a line, in its courier's shape: four capitals, a dash and ten digits (`LXAD-…`, `JNAP-…`) or J&T's (`JZ…`) | [lazada.go](lazada.go) |
| J&T Express printed as **one picture** (`pdfcpu`, no text at all) | a page with no text whose QR code or Code 128 holds a J&T tracking number | [jnt_picture.go](jnt_picture.go), [barcodes.go](barcodes.go) |
| Shopee SPX printed as **pictures** (Microsoft Print To PDF: words as outlines, one JPEG per graphic) | a page with no text whose QR code or Code 128 holds an `SPXID…` number, every copy agreeing | [spx_picture.go](spx_picture.go), [barcodes.go](barcodes.go) |
| **any other Shopee label** (tried last) | `Pesanan:` + a `Resi:` box value that a barcode on the page decodes to exactly (a-shopee-resi-is-confirmed-by-its-barcode) | [shopee_barcode.go](shopee_barcode.go) |

Layouts are tried in that order, and the first match wins. A layout belongs to the app that **generated** the
PDF, not to the courier.

| field | Shopee | TikTok | KiriminAja | Lazada |
| --- | --- | --- | --- | --- |
| `Receipt` | the number printed most often, or its `Resi:` box when printed once (SiCepat only). ⚠ The instant / same-day label prints none, so its receipt is the pickup code after `Kode Pengambilan:` (the-pickup-code-is-the-receipt) | the barcode caption; on the instant label, its pickup code | the code under `AWB` | the courier's tracking number (`LXAD-…`, `JNAP-…`, `JZ…`) |
| `OrderID` | after `No. Pesanan:` (cut short with `…`: read whole from the `Pesan:` line), or on a **reservation** label (an SPX parcel to an SPX hub) the `No.Reservasi:` number, read whole from the `Pesan:` line (Q14) | after `TT Order ID：` (full-width colon) or `Order Id :` | ⚠ `""`: only KiriminAja's own `OID-…` is printed (Q11) | the 16-digit order number, uncaptioned on page 1 (`Nomor Order :` on a page 2) |
| `CustomerName` | the left column of the `Penerima:` row | ⚠ masked (`S** W**O`) → `""` | under `Penerima` | after `Penerima:`, the right column |
| `Phone` | ⚠ **never printed**: the one phone is the sender's | ⚠ masked → `""` | ✅ under the name, in full | ⚠ not printed |
| `Address` | the rows under `Penerima:`, left of the sender's column | the rows under the name | the rows under the phone, left of `Dari` | the rows under the name, right of `Penerima:`'s x |

⚠ **A label printed as pictures gives only what its barcodes say**, decoded with
[gozxing](https://github.com/makiuchi-d/gozxing) (MIT, pure Go): the `Receipt`, and on SPX the `OrderID` too. Its
recipient is pixels or outlines and comes back `""` (Q13). A page with text is never scanned for codes. A page with
none has **every** image scanned (up to 16), because Print To PDF puts the codes in images smaller than the logo.
⚠ The PDF library has no JPEG filter and no way to hand over a stream undecoded, so a JPEG is found in the file's own
bytes by its declared `Length` and its own header's size ([barcodes.go](barcodes.go)), then decoded by `image/jpeg`.

⚠ **Bold can be drawn twice.** The KiriminAja label prints each bold run a second time 0.034 em to the right, so
every bold letter came out doubled (`NNOONN--CCOODD`). A glyph drawn again within 0.1 em of itself is dropped
([lines.go](lines.go)); a letter that really repeats is a glyph's width on.

⚠ **A logo only gates a layout, and on Shopee no longer blocks a label**: one whose logo is new is read by `shopee_barcode.go` when its barcode confirms its Resi box. Logos are compared byte for byte, pixels **and** alpha mask
([images.go](images.go)): the J&T Cargo logo's colour data is solid black, its shape all in its mask. The same
logo comes in several sizes, so a match is certain while a miss means nothing. A miss leaves the SiCepat REG and
J&T Cargo labels **unknown**.

## How it works

⚠ **The library reads only the page's own content, never a form it draws.** The Lazada label draws its whole face
as one Form XObject, and the library read no text from it. [content.go](content.go) runs the same text rules
and follows `Do` into each form, with the form's matrix and resources. Without forms it reads exactly what the
library reads (`TestPageTextsMatchesTheLibraryWithoutForms`). Nesting is bounded, so a form that draws
itself ends.

The PDF library returns one glyph at a time with its position, and can't group them itself: its own
row grouping returns the whole label as one row. So [lines.go](lines.go) rebuilds rows of positioned
words. Rotated glyphs are dropped, glyphs are clustered into rows by baseline, and a new word starts at a
space or at a gap over 0.2 em. Each layout matches regexes against the rows' text, and reads the recipient
from the words' positions ([recipient.go](recipient.go)).

⚠ **The library reads no widths from Type0 fonts, and every real label uses them.** When a whole word is one
text-show (PDFium does this), every glyph comes back at the word's first x, and a word split at a kerning
pair reads as two words. That would truncate an alphanumeric order id with no error. [widths.go](widths.go)
reads each font's `W` array and puts the glyphs back where they're drawn. Widths are keyed by the font's name without its subset tag
(`AAAEKH+ArialMT` is `ArialMT`, the name a glyph carries), and two subsets of one typeface are merged. `TestLinesWithoutWidthsSplitTheKernedWord`
pins the failure it prevents.

⚠ **The address is cut out of a two-column layout at two different x:** the name row where `Pengirim:`
starts, and the address rows where the sender's phone starts, because the caption sits left of the sender's
column and the address runs past it. Glyphs of another size (the `COD` watermark, a hidden overlapping element)
are filtered out, and so is the buyer's address tag (`HOME`, `RUMAH`, …) printed above the address. A word the box wrapped in two stays split (`Gro gol Sel`); see Q10 in the clarify.

| input | result |
| --- | --- |
| one known label | `ReceiptData`; a field the label does not print, or masks, is `""` |
| not a PDF / over 2 MB / a PDF that can't be parsed | **`ErrUnreadable`**: nothing on it was looked at |
| no known layout on any page | an error, so the person types the numbers |
| no known layout, and **no image on any page** (spam made in Canva), **no text and no readable barcode or QR** (a screenshot), or **no readable barcode and no word shaped like a tracking number** (a note) | **`ErrNotShippingLabel`**, which is also the unknown-layout error, so a caller can skip it (a-file-that-is-not-a-courier-label-is-refused, Q16) |
| more than one label (a bulk print) | an error, never the first page |
| a malformed file that makes the library panic | **`ErrUnreadable`**; `Extract` recovers |

Two errors are exported, and a file is never both (unreadable-and-not-a-label-are-two-errors): `ErrUnreadable`, the file
could not be read at all, and `ErrNotShippingLabel`, it was read and is spam (every real label carries a logo, a barcode
or a QR, and spam carries none). The rest stay unexported for now (critique 4 in the clarify).
