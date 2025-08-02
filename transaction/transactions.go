package transaction

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/etnz/portfolio/date"
)

// CommandType is a typed string for identifying transaction commands.
type CommandType string

// Command types used for identifying transactions.
const (
	CmdBuy      CommandType = "buy"
	CmdSell     CommandType = "sell"
	CmdDividend CommandType = "dividend"
	CmdDeposit  CommandType = "deposit"
	CmdWithdraw CommandType = "withdraw"
	CmdConvert  CommandType = "convert"
)

type Transaction interface {
	What() CommandType // Returns the command type of the transaction
	When() date.Date   // Returns the date of the transaction
	Rationale() string // Returns the memo or rationale for the transaction
}

// Base contains fields common to all transaction types.
type Base struct {
	Command CommandType `json:"command"`
	Date    date.Date   `json:"date"`
	Memo    string      `json:"memo,omitempty"`
}

// What returns the command name for the transaction, which is used to identify the type of transaction.
func (b Base) What() CommandType {
	return b.Command
}

// When returns the date of the transaction.
func (b Base) When() date.Date {
	return b.Date
}

// Rationale returns the memo associated with the transaction, which can provide additional context or rationale.
func (b Base) Rationale() string {
	return b.Memo
}

// Buy represents a buy transaction.
type Buy struct {
	Base
	Security string  `json:"security"`
	Quantity float64 `json:"quantity"`
	Price    float64 `json:"price"`
}

// Sell represents a sell transaction.
type Sell struct {
	Base
	Security string  `json:"security"`
	Quantity float64 `json:"quantity"`
	Price    float64 `json:"price"`
}

// Dividend represents a dividend payment.
type Dividend struct {
	Base
	Security string  `json:"security"`
	Amount   float64 `json:"amount"`
}

// Deposit represents a cash deposit.
type Deposit struct {
	Base
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

// Withdraw represents a cash withdrawal.
type Withdraw struct {
	Base
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

// Convert represents an internal currency conversion.
type Convert struct {
	Base
	FromCurrency string  `json:"fromCurrency"`
	FromAmount   float64 `json:"fromAmount"`
	ToCurrency   string  `json:"toCurrency"`
	ToAmount     float64 `json:"toAmount"`
}

// Load reads a stream of JSONL data from an io.Reader, decodes each line into the
// appropriate transaction struct, and returns a slice of transactions.
func Load(r io.Reader) ([]Transaction, error) {
	var transactions []Transaction
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		lineBytes := scanner.Bytes()
		if len(lineBytes) == 0 {
			continue // Skip empty lines
		}

		var identifier struct {
			Command CommandType `json:"command"`
		}
		if err := json.Unmarshal(lineBytes, &identifier); err != nil {
			return nil, fmt.Errorf("could not identify command in line %q: %w", string(lineBytes), err)
		}

		var decodedTx Transaction
		var err error

		switch identifier.Command {
		case CmdBuy:
			var tx Buy
			err = json.Unmarshal(lineBytes, &tx)
			decodedTx = tx
		case CmdSell:
			var tx Sell
			err = json.Unmarshal(lineBytes, &tx)
			decodedTx = tx
		case CmdDividend:
			var tx Dividend
			err = json.Unmarshal(lineBytes, &tx)
			decodedTx = tx
		case CmdDeposit:
			var tx Deposit
			err = json.Unmarshal(lineBytes, &tx)
			decodedTx = tx
		case CmdWithdraw:
			var tx Withdraw
			err = json.Unmarshal(lineBytes, &tx)
			decodedTx = tx
		case CmdConvert:
			var tx Convert
			err = json.Unmarshal(lineBytes, &tx)
			decodedTx = tx
		default:
			err = fmt.Errorf("unknown transaction command: %q", identifier.Command)
		}

		if err != nil {
			return nil, err
		}
		transactions = append(transactions, decodedTx)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading from input: %w", err)
	}

	// 1. Perform a stable sort on the slice based on the transaction date.
	sort.SliceStable(transactions, func(i, j int) bool {
		return transactions[i].When().Before(transactions[j].When())
	})

	return transactions, nil
}

// Save reorders transactions by date and persists them to an io.Writer in JSONL format.
// The sort is stable, meaning transactions on the same day maintain their original relative order.
func Save(w io.Writer, transactions []Transaction) error {
	// 1. Perform a stable sort on the slice based on the transaction date.
	sort.SliceStable(transactions, func(i, j int) bool {
		return transactions[i].When().Before(transactions[j].When())
	})

	// 2. Iterate through the sorted transactions and write each one as a JSON line.
	for _, tx := range transactions {
		jsonData, err := json.Marshal(tx)
		if err != nil {
			return err // Stop and return marshalling error
		}

		// Write the JSON data followed by a newline to create the JSONL format.
		if _, err := w.Write(append(jsonData, '\n')); err != nil {
			return err // Stop and return write error
		}
	}

	return nil
}
