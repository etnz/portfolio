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

// baseCmd contains fields common to all transaction types.
type baseCmd struct {
	Command CommandType `json:"command"`
	Date    date.Date   `json:"date"`
	Memo    string      `json:"memo,omitempty"`
}

// What returns the command name for the transaction, which is used to identify the type of transaction.
func (b baseCmd) What() CommandType {
	return b.Command
}

// When returns the date of the transaction.
func (b baseCmd) When() date.Date {
	return b.Date
}

// Rationale returns the memo associated with the transaction, which can provide additional context or rationale.
func (b baseCmd) Rationale() string {
	return b.Memo
}

// Buy represents a buy transaction.
type Buy struct {
	baseCmd
	Security string  `json:"security"`
	Quantity float64 `json:"quantity"`
	Price    float64 `json:"price"`
}

// NewBuy creates a new Buy transaction.
func NewBuy(day date.Date, memo, security string, quantity, price float64) Buy {
	return Buy{
		baseCmd:  baseCmd{Command: CmdBuy, Date: day, Memo: memo},
		Security: security,
		Quantity: quantity,
		Price:    price,
	}
}

// Sell represents a sell transaction.
type Sell struct {
	baseCmd
	Security string  `json:"security"`
	Quantity float64 `json:"quantity"`
	Price    float64 `json:"price"`
}

// NewSell creates a new Sell transaction.
func NewSell(day date.Date, memo, security string, quantity, price float64) Sell {
	return Sell{
		baseCmd:  baseCmd{Command: CmdSell, Date: day, Memo: memo},
		Security: security,
		Quantity: quantity,
		Price:    price,
	}
}

// Dividend represents a dividend payment.
type Dividend struct {
	baseCmd
	Security string  `json:"security"`
	Amount   float64 `json:"amount"`
}

// NewDividend creates a new Dividend transaction.
func NewDividend(day date.Date, memo, security string, amount float64) Dividend {
	return Dividend{
		baseCmd:  baseCmd{Command: CmdDividend, Date: day, Memo: memo},
		Security: security,
		Amount:   amount,
	}
}

// Deposit represents a cash deposit.
type Deposit struct {
	baseCmd
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency,omitempty"`
}

// NewDeposit creates a new Deposit transaction.
func NewDeposit(day date.Date, memo, currency string, amount float64) Deposit {
	return Deposit{
		baseCmd:  baseCmd{Command: CmdDeposit, Date: day, Memo: memo},
		Amount:   amount,
		Currency: currency,
	}
}

// Withdraw represents a cash withdrawal.
type Withdraw struct {
	baseCmd
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency,omitempty"`
}

// NewWithdraw creates a new Withdraw transaction.
func NewWithdraw(day date.Date, memo, currency string, amount float64) Withdraw {
	return Withdraw{
		baseCmd:  baseCmd{Command: CmdWithdraw, Date: day, Memo: memo},
		Amount:   amount,
		Currency: currency,
	}
}

// Convert represents an internal currency conversion.
type Convert struct {
	baseCmd
	FromCurrency string  `json:"fromCurrency"`
	FromAmount   float64 `json:"fromAmount"`
	ToCurrency   string  `json:"toCurrency"`
	ToAmount     float64 `json:"toAmount"`
}

// NewConvert creates a new Convert transaction.
func NewConvert(day date.Date, memo, fromCurrency string, fromAmount float64, toCurrency string, toAmount float64) Convert {
	return Convert{
		baseCmd:      baseCmd{Command: CmdConvert, Date: day, Memo: memo},
		FromCurrency: fromCurrency,
		FromAmount:   fromAmount,
		ToCurrency:   toCurrency,
		ToAmount:     toAmount,
	}
}
