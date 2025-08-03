package portfolio

import (
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
