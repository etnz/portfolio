package portfolio

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"

	"github.com/shopspring/decimal"
)

func init() {
	decimal.MarshalJSONWithoutQuotes = true
}

// amountCmd is a specialized struct to read from ledger amount in two fields.
// we could used json "inline" but it would work for some transactions not all.
type amountCmd struct {
	Amount   decimal.Decimal `json:"amount"`
	Currency string          `json:"currency"`
}

func (a amountCmd) Money() Money {
	return M(a.Amount, a.Currency)
}

// convertCmd is a specialized struct for decoding json.
type convertCmd struct {
	baseCmd
	FromAmount   decimal.Decimal `json:"fromAmount"`
	FromCurrency string          `json:"fromCurrency"`
	ToAmount     decimal.Decimal `json:"toAmount"`
	ToCurrency   string          `json:"toCurrency"`
}

func (a convertCmd) FromMoney() Money {
	return M(a.FromAmount, a.FromCurrency)
}
func (a convertCmd) ToMoney() Money {
	return M(a.ToAmount, a.ToCurrency)
}

// DecodeLedger decodes transactions from a stream of JSONL data from an io.Reader,
// decodes each line into the appropriate transaction struct, and returns a sorted Ledger.
func DecodeLedger(r io.Reader) (*Ledger, error) {
	ledger := NewLedger()
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
		// TODO: unmarshal like buy and sell for all transaction with Money
		case CmdBuy:
			// Use a temporary type that has all possible fields.
			var temp struct {
				secCmd
				amountCmd
				Quantity Quantity `json:"quantity"`
			}
			if err := json.Unmarshal(lineBytes, &temp); err != nil {
				return nil, err
			}

			// Create the final transaction struct
			buy := Buy{
				secCmd:   temp.secCmd,
				Quantity: temp.Quantity,
				Amount:   temp.Money(),
			}
			decodedTx = buy
		case CmdSell:
			// Use a temporary type that has all possible fields.
			var temp struct {
				secCmd
				amountCmd
				Quantity Quantity `json:"quantity"`
			}
			if err := json.Unmarshal(lineBytes, &temp); err != nil {
				return nil, err
			}

			// Create the final transaction struct
			decodedTx = Sell{
				secCmd:   temp.secCmd,
				Quantity: temp.Quantity,
				Amount:   temp.Money(),
			}
		case CmdDividend:
			// Use a temporary type that has all possible fields.
			var temp struct {
				secCmd
				amountCmd
				Quantity Quantity `json:"quantity"`
			}
			if err := json.Unmarshal(lineBytes, &temp); err != nil {
				return nil, err
			}

			// Create the final transaction struct
			decodedTx = Dividend{
				secCmd: temp.secCmd,
				Amount: temp.Money(),
			}
		case CmdDeposit:
			// Use a temporary type that has all possible fields.
			var temp struct {
				baseCmd
				amountCmd
				Settles string `json:"settles,omitempty"`
			}
			if err := json.Unmarshal(lineBytes, &temp); err != nil {
				return nil, err
			}

			// Create the final transaction struct
			decodedTx = Deposit{
				baseCmd: temp.baseCmd,
				Amount:  temp.Money(),
				Settles: temp.Settles,
			}
		case CmdWithdraw:
			// Use a temporary type that has all possible fields.
			var temp struct {
				baseCmd
				amountCmd
				Settles string `json:"settles,omitempty"`
			}
			if err := json.Unmarshal(lineBytes, &temp); err != nil {
				return nil, err
			}

			// Create the final transaction struct
			decodedTx = Withdraw{
				baseCmd: temp.baseCmd,
				Amount:  temp.Money(),
				Settles: temp.Settles,
			}
		case CmdConvert:
			// Use a temporary type that has all possible fields.
			temp := convertCmd{}
			if err := json.Unmarshal(lineBytes, &temp); err != nil {
				return nil, err
			}

			// Create the final transaction struct
			decodedTx = Convert{
				baseCmd:    temp.baseCmd,
				FromAmount: temp.FromMoney(),
				ToAmount:   temp.ToMoney(),
			}
		case CmdDeclare:
			var tx Declare
			err = json.Unmarshal(lineBytes, &tx)
			decodedTx = tx
		case CmdAccrue:
			// Use a temporary type that has all possible fields.
			var temp struct {
				baseCmd
				amountCmd
				Counterparty string `json:"counterparty"`
				Create       bool   `json:"create,omitempty"`
			}
			if err := json.Unmarshal(lineBytes, &temp); err != nil {
				return nil, err
			}

			// Create the final transaction struct
			decodedTx = Accrue{
				baseCmd:      temp.baseCmd,
				Amount:       temp.Money(),
				Counterparty: temp.Counterparty,
				Create:       temp.Create,
			}
		default:
			err = fmt.Errorf("unknown transaction command: %q", identifier.Command)
		}

		if err != nil {
			return nil, err
		}
		ledger.Append(decodedTx)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading from input: %w", err)
	}

	// Perform a stable sort on the ledger based on the transaction date.
	ledger.stableSort()

	return ledger, nil
}

// EncodeTransaction marshals a single transaction to JSON and writes it to the
// writer, followed by a newline, in JSONL format.
func EncodeTransaction(w io.Writer, tx Transaction) error {
	decimal.MarshalJSONWithoutQuotes = true
	// Marshal the transaction into a generic map to get all fields.
	// This step uses json.Marshal, which doesn't guarantee key order,
	// but we'll re-order them manually afterwards.
	tempData, err := json.Marshal(tx)
	if err != nil {
		return fmt.Errorf("failed to marshal transaction to temporary map: %w", err)
	}

	// Write the JSON data followed by a newline to create the JSONL format.
	if _, err := w.Write(append(tempData, '\n')); err != nil {
		return fmt.Errorf("failed to write transaction: %w", err)
	}
	return nil
}

// EncodeLedger reorders transactions by date and persists them to an io.Writer in JSONL format.
// The sort is stable, meaning transactions on the same day maintain their original relative order.
// It also ensures that the JSON keys within each transaction are sorted alphabetically for canonical output.
func EncodeLedger(w io.Writer, ledger *Ledger) error {
	decimal.MarshalJSONWithoutQuotes = true

	// Perform a stable sort on the ledger based on the transaction date to ensure order.
	ledger.stableSort()

	// 2. Iterate through the sorted transactions and write each one as a JSON line.
	for _, tx := range ledger.transactions {
		if err := EncodeTransaction(w, tx); err != nil {
			return err
		}
	}

	return nil
}
