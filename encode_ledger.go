package portfolio

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"

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
	ledger, err := decodeLedger(r)
	if err != nil {
		return nil, err
	}
	// Perform a stable sort and index securities and accounts.
	ledger.stableSort()
	// fill up the maps
	ledger.processTx(ledger.transactions...)
	err = ledger.newJournal()

	return ledger, err
}

// DecodeValidateLedger decode the ledger and validate every transactions.
// if market is nil, skip all validations.
func DecodeValidateLedger(r io.Reader) (*Ledger, error) {
	// Same as DecodeLedger except that it will perform a stricter validation.
	ledger, err := decodeLedger(r)
	if err != nil {
		return nil, err
	}

	// perform strict validation, and quick fixes.
	ledger.stableSort()
	// move all transactions out and insert them one by one with strict validation from
	// the accounting system.
	// For validation, a reporting currency is not needed. We pass an empty string.
	txs := ledger.transactions
	ledger.transactions = make([]Transaction, 0, len(txs))
	for _, tx := range txs {
		log.Println("validating", tx)
		t, err := ledger.Validate(tx)
		if err != nil {
			return nil, err
		}
		ledger.Append(t)
	}
	return ledger, nil
}

// decodeLedger read transactions from the reader, and simply append them to the ledger.
// this method is private, since the ledger could be unsorted, and with invalid indexes
// of securities etc. It is meant to be called to either simply sort transactions and compute indexes
// or perform a strict validation.
func decodeLedger(r io.Reader) (*Ledger, error) {
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

		decodedTx, err := decodeTransaction(identifier.Command, lineBytes)
		if err != nil {
			return nil, fmt.Errorf("error decoding transaction: %w", err)
		}

		// Raw append in this function.
		ledger.transactions = append(ledger.transactions, decodedTx)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading from input: %w", err)
	}
	return ledger, nil
}

// decodeTx is a generic function to decode a transaction of type T from JSON bytes.
func decodeTx[T any](lineBytes []byte, a *T) (t T, err error) {
	if err = json.Unmarshal(lineBytes, a); err != nil {
		return t, err
	}
	return *a, nil
}

func decodeTransaction(command CommandType, lineBytes []byte) (Transaction, error) {
	switch command {
	case CmdInit:
		return decodeTx(lineBytes, &Init{})
	case CmdBuy:
		return decodeTx(lineBytes, &Buy{})
	case CmdSell:
		return decodeTx(lineBytes, &Sell{})
	case CmdDividend:
		return decodeTx(lineBytes, &Dividend{})
	case CmdDeposit:
		return decodeTx(lineBytes, &Deposit{})
	case CmdWithdraw:
		return decodeTx(lineBytes, &Withdraw{})
	case CmdConvert:
		return decodeTx(lineBytes, &Convert{})
	case CmdDeclare:
		return decodeTx(lineBytes, &Declare{})
	case CmdAccrue:
		return decodeTx(lineBytes, &Accrue{})
	case CmdUpdatePrice:
		return decodeTx(lineBytes, &UpdatePrice{})
	case CmdSplit:
		return decodeTx(lineBytes, &Split{})
	default:
		return nil, fmt.Errorf("unknown transaction command: %q", command)
	}
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
