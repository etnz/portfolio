package portfolio

import (
	"errors"
	"fmt"

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

func (s *baseCmd) Validate(m *MarketData, l *Ledger) error {
	if s.Date == (date.Date{}) {
		s.Date = date.Today()
	}
	return nil
}

// secCmd is a component for security based transactions (sell buy dividends)
type secCmd struct {
	baseCmd
	Security string `json:"security"`
	Currency string `json:"currency,omitempty"`
}

func (t *secCmd) Validate(m *MarketData, l *Ledger) error {
	if err := t.baseCmd.Validate(m, l); err != nil {
		return err
	}

	if t.Security == "" {
		return errors.New("security ticker is missing")
	}

	// Quickfix to copy security currency
	sec := m.Get(t.Security)
	if t.Currency == "" && sec != nil {
		t.Currency = sec.currency
	}

	if err := ValidateCurrency(t.Currency); err != nil {
		return fmt.Errorf("invalid currency: %w", err)
	}
	return nil
}

// Buy represents a buy transaction.
type Buy struct {
	secCmd
	Quantity float64 `json:"quantity"`
	Price    float64 `json:"price"`
}

// NewBuy creates a new Buy transaction.
func NewBuy(day date.Date, memo, security string, quantity, price float64, currency string) Buy {
	return Buy{
		secCmd:   secCmd{baseCmd: baseCmd{Command: CmdBuy, Date: day, Memo: memo}, Security: security, Currency: currency},
		Quantity: quantity,
		Price:    price,
	}
}

// Validate performs basic validation of the Buy transaction's fields.
func (t *Buy) Validate(m *MarketData, l *Ledger) error {
	if err := t.secCmd.Validate(m, l); err != nil {
		return err
	}

	if t.Quantity <= 0 {
		return fmt.Errorf("buy transaction quantity must be positive, got %f", t.Quantity)
	}
	if t.Price <= 0 {
		return fmt.Errorf("buy transaction price must be positive, got %f", t.Price)
	}

	cash, cost := l.CashBalance(t.Currency, t.Date), t.Quantity*t.Price
	if cash < cost {
		return fmt.Errorf("cannot buy for %f %s cash balance is %f %s", cost, t.Currency, cash, t.Currency)
	}
	return nil
}

// Sell represents a sell transaction.
type Sell struct {
	secCmd
	Quantity float64 `json:"quantity"`
	Price    float64 `json:"price"`
}

// NewSell creates a new Sell transaction.
//
// Quantity to exactly 0 is interpreted as a sell all on the position.
func NewSell(day date.Date, memo, security string, quantity, price float64, currency string) Sell {
	return Sell{
		secCmd:   secCmd{baseCmd: baseCmd{Command: CmdBuy, Date: day, Memo: memo}, Security: security, Currency: currency},
		Quantity: quantity,
		Price:    price,
	}
}

// Validate performs basic validation of the Sell transaction's fields.
func (t Sell) Validate(m *MarketData, l *Ledger) error {
	if err := t.secCmd.Validate(m, l); err != nil {
		return err
	}
	if t.Quantity == 0 {
		// quick fix, sell all.
		t.Quantity = l.Position(t.Security, t.Date)
	}

	if t.Quantity <= 0 {
		// For Sell quantity == 0 is interpreted as sell all.
		return fmt.Errorf("sell transaction quantity must be non-negative, got %f", t.Quantity)
	}
	if t.Price <= 0 {
		return fmt.Errorf("sell transaction price must be non-negative, got %f", t.Price)
	}

	return nil
}

// Dividend represents a dividend payment.
type Dividend struct {
	secCmd
	Amount float64 `json:"amount"`
}

// NewDividend creates a new Dividend transaction.
func NewDividend(day date.Date, memo, security string, amount float64, currency string) Dividend {
	return Dividend{
		secCmd: secCmd{baseCmd: baseCmd{Command: CmdBuy, Date: day, Memo: memo}, Security: security, Currency: currency},
		Amount: amount,
	}
}

// Validate performs basic validation of the Dividend transaction's fields.
func (t Dividend) Validate(m *MarketData, l *Ledger) error {
	if err := t.secCmd.Validate(m, l); err != nil {
		return err
	}

	if t.Amount <= 0 {
		return fmt.Errorf("dividend amount must be positive, got %f", t.Amount)
	}
	return nil
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

// Validate performs basic validation of the Deposit transaction's fields.
func (t Deposit) Validate(m *MarketData, l *Ledger) error {
	if err := t.baseCmd.Validate(m, l); err != nil {
		return err
	}

	if t.Amount <= 0 {
		return fmt.Errorf("deposit amount must be positive, got %f", t.Amount)
	}
	if t.Currency != "" {
		if err := ValidateCurrency(t.Currency); err != nil {
			return fmt.Errorf("invalid currency for deposit: %w", err)
		}
	}
	return nil
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

// Validate performs basic validation of the Withdraw transaction's fields.
func (t Withdraw) Validate(m *MarketData, l *Ledger) error {
	if err := t.baseCmd.Validate(m, l); err != nil {
		return err
	}

	if t.Currency != "" {
		if err := ValidateCurrency(t.Currency); err != nil {
			return fmt.Errorf("invalid currency for withdraw: %w", err)
		}
	}

	if t.Amount == 0 {
		// quick fix, cash all.
		t.Amount = l.CashBalance(t.Currency, t.Date)
	}

	if t.Amount <= 0 {
		return fmt.Errorf("withdraw amount must be positive, got %f", t.Amount)
	}

	cash, cost := l.CashBalance(t.Currency, t.Date), t.Amount
	if cash < cost {
		return fmt.Errorf("cannot withdraw for %f %s cash balance is %f %s", cost, t.Currency, cash, t.Currency)
	}
	return nil
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

// Validate performs basic validation of the Convert transaction's fields.
func (t Convert) Validate(m *MarketData, l *Ledger) error {
	if err := t.baseCmd.Validate(m, l); err != nil {
		return err
	}

	if err := ValidateCurrency(t.FromCurrency); err != nil {
		return fmt.Errorf("invalid 'from' currency: %w", err)
	}
	if err := ValidateCurrency(t.ToCurrency); err != nil {
		return fmt.Errorf("invalid 'to' currency: %w", err)
	}
	if t.ToAmount <= 0 {
		return fmt.Errorf("convert 'to' amount must be positive, got %f", t.ToAmount)
	}

	if t.FromAmount == 0 {
		// quick fix, cash all.
		t.FromAmount = l.CashBalance(t.FromCurrency, t.Date)
	}

	if t.FromAmount <= 0 {
		// from amount ==0 is as a convert all from source currency.
		return fmt.Errorf("convert 'from' amount must be non-negative, got %f", t.FromAmount)
	}

	cash, cost := l.CashBalance(t.FromCurrency, t.Date), t.FromAmount
	if cash < cost {
		return fmt.Errorf("cannot withdraw for %f %s cash balance is %f %s", cost, t.FromCurrency, cash, t.FromCurrency)
	}

	return nil
}
