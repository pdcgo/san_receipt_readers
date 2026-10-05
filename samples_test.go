package san_receipt_readers

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// notReceipts are samples that are NOT shipping labels, kept to prove they stay refused: an address
// label designed in Canva, with no courier, no tracking number, no order id and no image at all.
// To the reader it is spam (a-file-that-is-not-a-courier-label-is-refused, context_decision.md).
var notReceipts = map[string]bool{
	"pdc_sample_01.pdf": true,
	"pdc_sample_02.pdf": true,
	"pdc_sample_03.pdf": true,
}

// samplesDir is where a person drops real labels to try them:
//
//	go test -run TestSamples -v
//	RECEIPT_SAMPLES=/some/folder go test -run TestSamples -v
//
// The values are LOGGED, never asserted literally: a label is a real buyer's parcel, and this
// repository is public. Sample privacy is Q5 in context_clarify.md.
func samplesDir() string {
	dir := os.Getenv("RECEIPT_SAMPLES")
	if dir != "" {
		return dir
	}

	return "../../../examples/receipt_file_samples"
}

// Every PDF in the folder must read as exactly one label. A label this package does not know yet
// FAILS, naming the reason — that is the point of dropping it in. A field the label does not
// print is a warning, not a failure: the KiriminAja label prints no marketplace order id.
func TestSamples(t *testing.T) {
	dir := samplesDir()

	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		t.Skipf("no receipt samples at %s", dir)
	}
	if err != nil {
		t.Fatal(err)
	}

	read := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".pdf") {
			continue
		}
		read++

		t.Run(entry.Name(), func(t *testing.T) {
			file, err := os.Open(filepath.Join(dir, entry.Name()))
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()

			got, err := Extract(file)
			if notReceipts[entry.Name()] {
				if !errors.Is(err, ErrNotShippingLabel) {
					t.Fatalf("a file that is not a label must be refused as one: got %+v, %v", got, err)
				}
				t.Logf("refused, as it must be: not a courier's label")
				return
			}
			// Spam the list doesn't name yet is still a correct refusal, not a label left to learn.
			if errors.Is(err, ErrNotShippingLabel) {
				t.Logf("⚠ refused as not a courier's label (no image on any page): spam, or a label to look at")
				return
			}
			if err != nil {
				t.Fatalf("%v", err)
			}

			t.Logf("receipt=%s  order_id=%s", got.Receipt, got.OrderID)
			t.Logf("name=%q  phone=%q", got.CustomerName, got.Phone)
			t.Logf("address=%q", got.Address)

			if got.Receipt == "" {
				t.Logf("⚠ no tracking number printed, or not found")
			}
			if got.OrderID == "" {
				t.Logf("⚠ no order id printed, or not found")
			}
			if got.Receipt != "" && got.Receipt == got.OrderID {
				t.Errorf("the tracking number and the order id are the same string")
			}
		})
	}

	if read == 0 {
		t.Skipf("no .pdf in %s", dir)
	}
}

// TestNotShippingLabels walks the samples folder's pdc_samples/: files kept because they are NOT
// shipping labels — spam made in Canva, screenshots of documents — and every one must be refused as
// ErrNotShippingLabel, never read and never called unreadable.
func TestNotShippingLabels(t *testing.T) {
	dir := filepath.Join(samplesDir(), "pdc_samples")

	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		t.Skipf("no non-label samples at %s", dir)
	}
	if err != nil {
		t.Fatal(err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".pdf") {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			file, err := os.Open(filepath.Join(dir, entry.Name()))
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()

			got, err := Extract(file)
			if !errors.Is(err, ErrNotShippingLabel) || errors.Is(err, ErrUnreadable) {
				t.Fatalf("want ErrNotShippingLabel only: got %+v, %v", got, err)
			}
		})
	}
}
