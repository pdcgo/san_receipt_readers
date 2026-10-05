package san_receipt_readers

import (
	"bytes"
	"errors"
	"testing"
)

// The TikTok J&T Cargo label's structure with invented numbers. Helvetica can't draw the
// full-width colon the real label uses, so the fixture's caption has an ASCII one; the regex test
// below covers the full-width form.
func jntCargoPDF(logo []byte) []byte {
	return shopeePDF(logo,
		"FastTrack",
		"570000000001",
		"TGR-TGR078A",
		"Pengirim (+62)85100000000",
		"Weight: 3 KG",
		"TT Order ID: 583400000000000001",
	)
}

func TestExtractReadsATikTokJNTCargoLabel(t *testing.T) {
	useFakeLogo(t, courierJNTCargo)

	got, err := Extract(bytes.NewReader(jntCargoPDF(fakeLogo)))
	if err != nil {
		t.Fatal(err)
	}

	assertNumbers(t, got, "570000000001", "583400000000000001")
}

// The text never names J&T: a TikTok label with an unknown logo is unknown, not J&T Cargo.
func TestTikTokLabelWithAnUnknownLogoIsUnknown(t *testing.T) {
	useFakeLogo(t, courierJNTCargo)

	_, err := Extract(bytes.NewReader(jntCargoPDF(otherLogo)))
	if !errors.Is(err, errUnknownLayout) {
		t.Fatalf("err = %v, want errUnknownLayout", err)
	}
}

func TestTikTokOrderIDIsReadWithEitherColon(t *testing.T) {
	cases := []struct {
		line string
		want string
	}{
		{"TT Order ID：583400000000000001", "583400000000000001"},
		{"TT Order ID: 583400000000000001", "583400000000000001"},
		{"TT Order ID：583400000000000001 Estimated", "583400000000000001"},
		// The Tokopedia-branded label drops the "TT".
		{"Order ID：583400000000000001", "583400000000000001"},
		// Too short to be a TikTok id.
		{"TT Order ID：5834000000", ""},
		{"Print Time：2026-04-07 11:49", ""},
	}

	for _, c := range cases {
		got := readTikTokOrderID([]string{c.line})
		if got != c.want {
			t.Errorf("%q → %q, want %q", c.line, got, c.want)
		}
	}
}

// Two different twelve-digit lines are a guess between two numbers — nothing is read.
func TestJNTCargoTrackingIsTheOnlyTwelveDigitLine(t *testing.T) {
	if got := readJNTCargoTracking([]string{"570000000001", "TGS TGR"}); got != "570000000001" {
		t.Errorf("one line: got %q", got)
	}
	if got := readJNTCargoTracking([]string{"570000000001", "570000000002"}); got != "" {
		t.Errorf("two different lines: got %q, want nothing", got)
	}
}
