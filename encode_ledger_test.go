package portfolio

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestDecodeLedger(t *testing.T) {
	// A multi-line string representing a JSONL stream with all command types

	jsonlStream := `
{"command":"declare","date":"2025-08-01","ticker":"AAPL","id":"US0378331005.XNAS","currency":"USD"}
{"command":"buy","date":"2025-08-01","security":"AAPL","quantity":10,"price":195.5}
{"command":"deposit","date":"2025-08-02","amount":5000,"currency":"USD"}
{"command":"declare","date":"2025-08-01","ticker":"GOOG","id":"US02079K1079.XNAS","currency":"USD"}
{"command":"sell","date":"2025-08-02","security":"GOOG","quantity":5,"price":140.2}
{"command":"dividend","date":"2025-08-03","security":"AAPL","amount":5.50}
{"command":"withdraw","date":"2025-08-04","amount":1000,"currency":"USD"}
{"command":"convert","date":"2025-08-05","fromCurrency":"USD","fromAmount":2000,"toCurrency":"EUR","toAmount":1850.50}
`
	reader := strings.NewReader(jsonlStream)

	ledger, err := DecodeLedger(reader)

	// 1. Check for unexpected errors
	if err != nil {
		t.Fatalf("DecodeLedger() returned an unexpected error: %v", err)
	}

	// 2. Check the number of transactions decoded
	expectedCount := 8
	if len(ledger.transactions) != expectedCount {
		t.Fatalf("DecodeLedger() decoded wrong number of transactions. Got: %d, want: %d", len(ledger.transactions), expectedCount)
	}

	// 3. Check the type of each decoded transaction
	expectedTypes := []reflect.Type{
		reflect.TypeOf(Declare{}),
		reflect.TypeOf(Buy{}),
		reflect.TypeOf(Deposit{}),
		reflect.TypeOf(Declare{}),
		reflect.TypeOf(Sell{}),
		reflect.TypeOf(Dividend{}),
		reflect.TypeOf(Withdraw{}),
		reflect.TypeOf(Convert{}),
	}

	for i, tx := range ledger.Transactions() {
		if reflect.TypeOf(tx) != expectedTypes[i] {
			t.Errorf("Transaction %d has wrong type. Got: %T, want: %v", i+1, tx, expectedTypes[i])
		}
	}
}

func TestEncodeLedger(t *testing.T) {
	// 1. Arrange: Create test data in a deliberately unsorted order.
	// Note that tx2 and tx3 have the same date. Their relative order must be preserved. (tx2 comes before tx3)
	tx1 := NewBuy(NewDate(2025, time.August, 3), "", "AAPL", Q(100), USD(15000.0))
	tx2 := NewDeposit(NewDate(2025, time.August, 1), "", USD(1000), "")
	tx3 := NewSell(NewDate(2025, time.August, 1), "", "GOOG", Q(50), USD(14000.0)) // Same date as tx2

	ledger := &Ledger{
		transactions: []Transaction{
			tx1, // Should be last
			tx2, // Should be first
			tx3, // Should be second (stable sort)
		},
	}

	// Manually sort the transactions to build the expected output string.
	expectedOrder := []Transaction{tx2, tx3, tx1}
	var expectedOutputBuffer bytes.Buffer
	for _, tx := range expectedOrder {
		// Use EncodeTransaction to generate canonical JSON for expected output
		if err := EncodeTransaction(&expectedOutputBuffer, tx); err != nil {
			t.Fatalf("Failed to encode expected transaction: %v", err)
		}
	}

	var buffer bytes.Buffer

	// 2. Act: Call the Save function.
	err := EncodeLedger(&buffer, ledger)

	// 3. Assert: Check the results.
	if err != nil {
		t.Fatalf("EncodeLedger() returned an unexpected error: %v", err)
	}

	if got := buffer.String(); got != expectedOutputBuffer.String() {
		t.Errorf("EncodeLedger() produced incorrect output.\nGot:\n%s\nWant:\n%s", got, expectedOutputBuffer.String())
	}
}
