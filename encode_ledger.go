package portfolio

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

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
		case CmdBuy:
			// Use a temporary type that has all possible fields.
			var temp struct {
				secCmd
				Quantity float64 `json:"quantity"`
				Price    float64 `json:"price"`  // Old field
				Amount   float64 `json:"amount"` // New field
			}
			if err := json.Unmarshal(lineBytes, &temp); err != nil {
				return nil, err
			}

			// Create the final transaction struct
			buy := Buy{
				secCmd:   temp.secCmd,
				Quantity: temp.Quantity,
			}

			// Logic to handle both formats
			if temp.Amount != 0 {
				// New format: amount is present
				buy.Amount = temp.Amount
			} else {
				// Old format: price is present, calculate amount
				buy.Amount = temp.Price * temp.Quantity
			}
			decodedTx = buy
		case CmdSell:
			// Use a temporary type that has all possible fields.
			var temp struct {
				secCmd
				Quantity float64 `json:"quantity"`
				Price    float64 `json:"price"`  // Old field
				Amount   float64 `json:"amount"` // New field
			}
			if err := json.Unmarshal(lineBytes, &temp); err != nil {
				return nil, err
			}

			// Create the final transaction struct
			sell := Sell{
				secCmd:   temp.secCmd,
				Quantity: temp.Quantity,
			}

			// Logic to handle both formats
			if temp.Amount != 0 {
				// New format: amount is present
				sell.Amount = temp.Amount
			} else {
				// Old format: price is present, calculate amount
				sell.Amount = temp.Price * temp.Quantity
			}
			decodedTx = sell
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
		case CmdDeclare:
			var tx Declare
			err = json.Unmarshal(lineBytes, &tx)
			decodedTx = tx
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
// It ensures that the JSON keys are sorted alphabetically for canonical output.
func EncodeTransaction(w io.Writer, tx Transaction) error {
	// Marshal the transaction into a generic map to get all fields.
	// This step uses json.Marshal, which doesn't guarantee key order,
	// but we'll re-order them manually afterwards.
	tempData, err := json.Marshal(tx)
	if err != nil {
		return fmt.Errorf("failed to marshal transaction to temporary map: %w", err)
	}

	var rawMap map[string]interface{}
	if err := json.Unmarshal(tempData, &rawMap); err != nil {
		return fmt.Errorf("failed to unmarshal temporary data to map: %w", err)
	}

	// Extract keys and sort them alphabetically.
	keys := make([]string, 0, len(rawMap))
	for k := range rawMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Manually build the JSON string with sorted keys.
	var b strings.Builder
	b.WriteString("{")
	for i, key := range keys {
		if i > 0 {
			b.WriteString(",")
		}
		// Marshal each value individually to handle different types correctly.
		valData, err := json.Marshal(rawMap[key])
		if err != nil {
			return fmt.Errorf("failed to marshal value for key %q: %w", key, err)
		}
		fmt.Fprintf(&b, "%q:%s", key, valData)
	}
	b.WriteString("}")

	// Write the JSON data followed by a newline to create the JSONL format.
	if _, err := w.Write([]byte(b.String() + "\n")); err != nil {
		return fmt.Errorf("failed to write transaction: %w", err)
	}
	return nil
}

// EncodeLedger reorders transactions by date and persists them to an io.Writer in JSONL format.
// The sort is stable, meaning transactions on the same day maintain their original relative order.
// It also ensures that the JSON keys within each transaction are sorted alphabetically for canonical output.
func EncodeLedger(w io.Writer, ledger *Ledger) error {
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
