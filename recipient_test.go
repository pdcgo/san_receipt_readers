package san_receipt_readers

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ledongthuc/pdf"
)

// rowsOfPDF opens a synthetic one-page PDF and rebuilds its rows, the way Extract does.
func rowsOfPDF(t *testing.T, page []text) []row {
	t.Helper()

	data := buildPDF(page)
	reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	p := reader.Page(1)

	return pageRows(pageTexts(p), fontMetricsOf(p))
}

// shopeeRecipientPage is the Shopee two-column block with the same-day label's geometry: an address
// line that runs on past the "Pengirim:" caption, the sender's phone and city in a column further
// right, a row of the sender's column only, a watermark glyph of another size laid over the address,
// then the centred region boxes and the CASHLESS banner. The address lines are 8 points apart, as on the
// real label, with the sender's city on a baseline between two of them. In this fixture's font a glyph
// is 4 points.
func shopeeRecipientPage() []text {
	return []text{
		{x: 10, y: 320, size: 8, s: "Penerima: Budi Santoso"},
		{x: 145, y: 320, size: 8, s: "Pengirim:"},
		{x: 200, y: 320, size: 8, s: "Toko Contoh"},
		// Runs to x≈170 — past the caption at 145, short of the sender column at 184.
		{x: 10, y: 305, size: 8, s: "Jl. Melati Raya Ujung No. 7, RT 003 / RW"},
		{x: 184, y: 305, size: 8, s: "6280000000000"},
		{x: 10, y: 297, size: 8, s: "008, KEMBANGAN,"},
		{x: 184, y: 293, size: 8, s: "KOTA TANGERANG"},
		{x: 10, y: 289, size: 8, s: "DKI JAKARTA"},
		// The grey COD watermark, much larger, over the last address line.
		{x: 60, y: 289, size: 40, s: "C"},
		{x: 30, y: 265, size: 9, s: "KOTA JAKARTA BARAT"},
		{x: 186, y: 265, size: 9, s: "KEMBANGAN"},
		{x: 50, y: 250, size: 8, s: "CASHLESS"},
	}
}

func TestShopeeRecipientIsTheLeftColumn(t *testing.T) {
	got := readShopeeRecipient(rowsOfPDF(t, shopeeRecipientPage()))

	if got.name != "Budi Santoso" {
		t.Errorf("name = %q", got.name)
	}
	// "/ RW" starts right of the "Pengirim:" caption and is still the recipient's — cut at the
	// caption, it was lost. The watermark, the sender's phone and city, and the region boxes are not
	// the address.
	want := "Jl. Melati Raya Ujung No. 7, RT 003 / RW 008, KEMBANGAN, DKI JAKARTA"
	if got.address != want {
		t.Errorf("address = %q\nwant      %q", got.address, want)
	}
	// The only phone on a Shopee label is the sender's.
	if got.phone != "" {
		t.Errorf("phone = %q, want none", got.phone)
	}
}

// The TikTok block: the masked phone at the right of the "Penerima" row, the masked name under it,
// then the address — and the rows after it are other sizes.
func TestTikTokRecipientIsTheAddressOnly(t *testing.T) {
	rows := rowsOfPDF(t, []text{
		{x: 12, y: 223, size: 7.5, s: "Penerima"},
		{x: 158, y: 223, size: 6.3, s: "(+62)81*******00"},
		{x: 12, y: 213, size: 6.3, s: "S** W**O"},
		{x: 12, y: 203, size: 6.3, s: "Banten, Kota Tangerang, Benda,"},
		{x: 12, y: 196, size: 6.3, s: "jl contoh no 2"},
		{x: 180, y: 168, size: 7.5, s: "COD:"},
		{x: 12, y: 164, size: 8, s: "Weight: 3 KG"},
	})

	got := readTikTokRecipient(rows)

	want := recipient{name: "", phone: "", address: "Banten, Kota Tangerang, Benda, jl contoh no 2"}
	if got != want {
		t.Fatalf("got  %+v\nwant %+v", got, want)
	}
}

// rowsOfText builds rows straight from text, one word per token, for the readers that use only the
// rows' words and text.
func rowsOfText(lines ...string) []row {
	rows := []row{}
	for _, line := range lines {
		r := row{}
		for _, token := range strings.Fields(line) {
			r.words = append(r.words, word{s: token})
		}
		rows = append(rows, r)
	}

	return rows
}

func TestJNTRecipientRunsToTheWeightRow(t *testing.T) {
	rows := rowsOfText(
		"Penerima :a**i (+62)81*******11",
		"KALIMANTAN TIMUR,BALIKPAPAN,BALIKPAPAN",
		"Contoh Dalam, Balikpapan ,jln contoh km",
		"Weight : 0.400 KG",
		"Jumlah : 1pcs",
	)

	got := readJNTRecipient(rows)

	want := recipient{name: "", phone: "", address: "KALIMANTAN TIMUR,BALIKPAPAN,BALIKPAPAN Contoh Dalam, Balikpapan ,jln contoh km"}
	if got != want {
		t.Fatalf("got  %+v\nwant %+v", got, want)
	}
}

// With no "Weight" row to end it, the address would run on into the rest of the label — so it is
// not read at all.
func TestJNTRecipientWithoutItsEndIsNotRead(t *testing.T) {
	got := readJNTRecipient(rowsOfText(
		"Penerima :Budi (+62)8123456789",
		"KALIMANTAN TIMUR",
		"Jumlah : 1pcs",
	))

	if got.address != "" {
		t.Errorf("address = %q, want none", got.address)
	}
	if got.name != "Budi" || got.phone != "(+62)8123456789" {
		t.Errorf("name %q phone %q — unmasked values are read as printed", got.name, got.phone)
	}
}

func TestAMaskedValueIsNotPrinted(t *testing.T) {
	cases := map[string]string{
		"a**i":             "",
		"S** W**O":         "",
		"(+62)81*******00": "",
		"  Budi Santoso ":  "Budi Santoso",
		"(+62)8123456789":  "(+62)8123456789",
	}

	for in, want := range cases {
		if got := printed(in); got != want {
			t.Errorf("printed(%q) = %q, want %q", in, got, want)
		}
	}
}

// The buyer's label for the address ("HOME") is printed in a box above it, in the address's own size
// and indented into the box. Taken as the first line, it became the whole address: the real first
// line, at the margin, ended the block.
func TestAnAddressTagIsNotTheAddress(t *testing.T) {
	rows := rowsOfPDF(t, []text{
		{x: 7, y: 320, size: 8, s: "Penerima:Budi"},
		{x: 145, y: 320, size: 8, s: "Pengirim:"},
		{x: 184, y: 320, size: 8, s: "Toko Contoh"},
		{x: 13, y: 309, size: 8, s: "HOME"},
		{x: 184, y: 307, size: 9, s: "6280000000000"},
		{x: 8.5, y: 296, size: 8, s: "Jl. Mawar No. 1,"},
		{x: 8.5, y: 288, size: 8, s: "KOTA BEKASI"},
		{x: 48, y: 257, size: 9, s: "KOTA BEKASI"},
	})

	got := readShopeeRecipient(rows)

	if got.address != "Jl. Mawar No. 1, KOTA BEKASI" {
		t.Fatalf("address = %q", got.address)
	}
}

// On the SPX reservation label the region box sits at the address's margin and in its size — only
// further down: 1.4 lines below the last address line, where the lines are one apart.
func TestARegionBoxFurtherDownIsNotTheAddress(t *testing.T) {
	rows := rowsOfPDF(t, []text{
		{x: 12, y: 320, size: 8, s: "Penerima: Hub Contoh"},
		{x: 170, y: 320, size: 8, s: "Pengirim:Toko"},
		{x: 196, y: 305, size: 8, s: "6280000000000"},
		{x: 14, y: 296, size: 8, s: "Jl. Mawar No. 1,"},
		{x: 14, y: 288, size: 8, s: "Kec. Contoh,"},
		{x: 14, y: 280, size: 8, s: "KOTA CONTOH"},
		{x: 15.5, y: 268.8, size: 8, s: "KOTA CONTOH"},
	})

	got := readShopeeRecipient(rows)

	if got.address != "Jl. Mawar No. 1, Kec. Contoh, KOTA CONTOH" {
		t.Fatalf("address = %q", got.address)
	}
}

// A long name runs into the "Pengirim:" caption with no gap, and its last letters become the start of
// the caption's word ("(TOKO CEPengirim:"). Cut where that word starts, they were lost.
func TestANameRunIntoTheSenderCaptionKeepsItsEnd(t *testing.T) {
	rows := rowsOfPDF(t, []text{
		{x: 7, y: 320, size: 8, s: "Penerima:Budi Santoso (TOKO CEPengirim:"},
		{x: 184, y: 320, size: 8, s: "Toko Contoh"},
		{x: 184, y: 307, size: 9, s: "6280000000000"},
		{x: 8.5, y: 296, size: 8, s: "Jl. Mawar No. 1"},
	})

	got := readShopeeRecipient(rows)

	if got.name != "Budi Santoso (TOKO CE" {
		t.Fatalf("name = %q", got.name)
	}
}
