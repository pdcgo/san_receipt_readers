package san_receipt_readers_test

import (
	"fmt"
	"log"
	"os"

	"github.com/pdcgo/san_receipt_readers"
)

// Compiled by `go test`, so it cannot drift from the API. No "Output:" comment on purpose — that
// keeps it compile-checked without a label on disk.

// Reading the tracking number and the marketplace's order id off a shipping label.
func ExampleExtract() {
	file, err := os.Open("label.pdf")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	receipt, err := san_receipt_readers.Extract(file)
	if err != nil {
		// Not a PDF, over 2 MB, unreadable, a label this package does not know yet, or more
		// than one label in the file. The person types the numbers instead.
		log.Fatal(err)
	}

	// An empty field means the label does not print it, or masks it.
	fmt.Println(receipt.Receipt, receipt.OrderID)
	fmt.Println(receipt.CustomerName, receipt.Phone, receipt.Address)
}
