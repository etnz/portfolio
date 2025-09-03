package portfolio

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/etnz/portfolio/date"
)

func TestDecodeLedger(t *testing.T) {
	// A multi-line string representing a JSONL stream with all command types
	jsonlStream := `
{"command":"buy","date":"2025-08-01","security":"AAPL","quantity":10,"price":195.5}
{"command":"deposit","date":"2025-08-02","amount":5000,"currency":"USD"}
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
	expectedCount := 6
	if len(ledger.transactions) != expectedCount {
		t.Fatalf("DecodeLedger() decoded wrong number of transactions. Got: %d, want: %d", len(ledger.transactions), expectedCount)
	}

	// 3. Check the type of each decoded transaction
	expectedTypes := []reflect.Type{
		reflect.TypeOf(Buy{}),
		reflect.TypeOf(Deposit{}),
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
	// Note that tx2 and tx3 have the same date. Their relative order must be preserved.
	tx1 := NewBuy(date.New(2025, time.August, 3), "", "AAPL", 0, 0*0)
	tx2 := NewDeposit(date.New(2025, time.August, 1), "", "", 1000, "")
	tx3 := NewSell(date.New(2025, time.August, 1), "", "GOOG", 0, 0*0) // Same date as tx2

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

// TestEncodeDecodeLedger verifies that loading an unsorted JSONL file and immediately
// saving it results in a correctly and stably sorted file.
func TestDecodeLedger_BackwardCompatibility(t *testing.T) {
	// 1. Arrange: Define a JSONL stream with both old (price) and new (amount) formats.
	jsonlStream := `
{"command":"buy","date":"2025-08-01","security":"AAPL","quantity":10,"price":195.5}
{"command":"sell","date":"2025-08-02","security":"GOOG","quantity":5,"amount":701}
{"command":"buy","date":"2025-08-03","security":"MSFT","quantity":15,"price":410,"amount":6150}
{"command":"sell","date":"2025-08-04","security":"TSLA","quantity":2,"price":142,"amount":284}
{"command":"buy","date":"2025-08-05","security":"AMZN","quantity":10,"amount":1500}
{"command":"sell","date":"2025-08-06","security":"NFLX","quantity":5,"price":400}
`
	// 2. Act: Decode the stream.
	ledger, err := DecodeLedger(strings.NewReader(jsonlStream))
	if err != nil {
		t.Fatalf("DecodeLedger() returned an unexpected error: %v", err)
	}

	// 3. Assert: Check the decoded transactions.
	expectedTransactions := []Transaction{
		Buy{
			secCmd:   secCmd{baseCmd: baseCmd{Command: CmdBuy, Date: date.New(2025, time.August, 1)}, Security: "AAPL"},
			Quantity: 10,
			Amount:   1955,
		},
		Sell{
			secCmd:   secCmd{baseCmd: baseCmd{Command: CmdSell, Date: date.New(2025, time.August, 2)}, Security: "GOOG"},
			Quantity: 5,
			Amount:   701,
		},
		Buy{
			secCmd:   secCmd{baseCmd: baseCmd{Command: CmdBuy, Date: date.New(2025, time.August, 3)}, Security: "MSFT"},
			Quantity: 15,
			Amount:   6150,
		},
		Sell{
			secCmd:   secCmd{baseCmd: baseCmd{Command: CmdSell, Date: date.New(2025, time.August, 4)}, Security: "TSLA"},
			Quantity: 2,
			Amount:   284,
		},
		Buy{
			secCmd:   secCmd{baseCmd: baseCmd{Command: CmdBuy, Date: date.New(2025, time.August, 5)}, Security: "AMZN"},
			Quantity: 10,
			Amount:   1500,
		},
		Sell{
			secCmd:   secCmd{baseCmd: baseCmd{Command: CmdSell, Date: date.New(2025, time.August, 6)}, Security: "NFLX"},
			Quantity: 5,
			Amount:   2000,
		},
	}

	if len(ledger.transactions) != len(expectedTransactions) {
		t.Fatalf("DecodeLedger() decoded wrong number of transactions. Got: %d, want: %d", len(ledger.transactions), len(expectedTransactions))
	}

	for i, tx := range ledger.transactions {
		// We need to compare the structs without the unexported price field in Sell
		if buy, ok := tx.(Buy); ok {
			expectedBuy := expectedTransactions[i].(Buy)
			if buy.What() != expectedBuy.What() || buy.When() != expectedBuy.When() || buy.Security != expectedBuy.Security || buy.Quantity != expectedBuy.Quantity || buy.Amount != expectedBuy.Amount {
				t.Errorf("Transaction %d is incorrect.\nGot:  %+v\nWant: %+v", i, buy, expectedBuy)
			}
		} else if sell, ok := tx.(Sell); ok {
			expectedSell := expectedTransactions[i].(Sell)
			if sell.What() != expectedSell.What() || sell.When() != expectedSell.When() || sell.Security != expectedSell.Security || sell.Quantity != expectedSell.Quantity || sell.Amount != expectedSell.Amount {
				t.Errorf("Transaction %d is incorrect.\nGot:  %+v\nWant: %+v", i, sell, expectedSell)
			}
		}
	}
}
