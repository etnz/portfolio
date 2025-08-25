package portfolio

import (
	"errors"
	"fmt"

	"github.com/etnz/portfolio/date"
)

// CommandType is a typed string for identifying transaction commands.
type CommandType string

func (c CommandType) IsCashFlow() bool { return c == CmdDeposit || c == CmdWithdraw }

// Command types used for identifying transactions.
const (
	CmdBuy      CommandType = "buy"
	CmdSell     CommandType = "sell"
	CmdDividend CommandType = "dividend"
	CmdDeposit  CommandType = "deposit"
	CmdWithdraw CommandType = "withdraw"
	CmdConvert  CommandType = "convert"
	CmdDeclare  CommandType = "declare"
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
func (t baseCmd) What() CommandType {
	return t.Command
}

// When returns the date of the transaction.
func (t baseCmd) When() date.Date {
	return t.Date
}

// Rationale returns the memo associated with the transaction, which can provide additional context or rationale.
func (t baseCmd) Rationale() string {
	return t.Memo
}

// Validate checks the base command fields. It sets the date to today if it's zero.
// It's meant to be embedded in other transaction validation methods.
func (t *baseCmd) Validate(as *AccountingSystem) error {
	if t.Date == (date.Date{}) {
		t.Date = date.Today()
	}
	return nil
}

// secCmd is a component for security-based transactions (buy, sell, dividend).
type secCmd struct {
	baseCmd
	Security string `json:"security"`
}

// Validate checks the security command fields. It validates the base command,
// ensures a security ticker is present, and attempts to auto-populate the
// currency from the security's definition if it's missing.
func (t *secCmd) Validate(as *AccountingSystem) error {
	if err := t.baseCmd.Validate(as); err != nil {
		return err
	}

	if t.Security == "" {
		return errors.New("security ticker is missing")
	}

	// use ticker to resolve the ledger security
	ledgerSec := as.Ledger.Get(t.Security)
	if ledgerSec == nil {
		return fmt.Errorf("security %q not declared in ledger", t.Security)
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
func NewBuy(day date.Date, memo, security string, quantity, price float64) Buy {
	return Buy{
		secCmd:   secCmd{baseCmd: baseCmd{Command: CmdBuy, Date: day, Memo: memo}, Security: security},
		Quantity: quantity,
		Price:    price,
	}
}

// Validate checks the Buy transaction's fields. It ensures that the quantity
// and price are positive. It also verifies that there is enough cash in the
// corresponding currency account to cover the cost of the purchase on the
// transaction date.
func (t *Buy) Validate(as *AccountingSystem) error {
	if err := t.secCmd.Validate(as); err != nil {
		return err
	}

	if t.Quantity <= 0 {
		return fmt.Errorf("buy transaction quantity must be positive, got %f", t.Quantity)
	}
	if t.Price <= 0 {
		return fmt.Errorf("buy transaction price must be positive, got %f", t.Price)
	}

	ledgerSec := as.Ledger.Get(t.Security) // We know this is not nil from secCmd.Validate
	currency := ledgerSec.Currency()
	cash, cost := as.Ledger.CashBalance(currency, t.Date), t.Quantity*t.Price
	if cash < cost {
		return fmt.Errorf("cannot buy for %f %s cash balance is %f %s", cost, currency, cash, currency)
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
func NewSell(day date.Date, memo, security string, quantity, price float64) Sell {
	return Sell{
		secCmd:   secCmd{baseCmd: baseCmd{Command: CmdSell, Date: day, Memo: memo}, Security: security},
		Quantity: quantity,
		Price:    price,
	}
}

// Validate checks the Sell transaction's fields.
// It handles the "sell all" case by resolving a quantity of 0 to the total
// position size on the transaction date. It ensures the final quantity and
// price are positive and that the position is sufficient to cover the sale.
func (t *Sell) Validate(as *AccountingSystem) error {
	if err := t.secCmd.Validate(as); err != nil {
		return err
	}
	if t.Quantity == 0 {
		// quick fix, sell all.
		t.Quantity = as.Ledger.Position(t.Security, t.Date)
	}

	if t.Quantity <= 0 {
		// For Sell quantity == 0 is interpreted as sell all.
		return fmt.Errorf("sell transaction quantity must be positive, got %f", t.Quantity)
	}
	if t.Price <= 0 {
		return fmt.Errorf("sell transaction price must be positive, got %f", t.Price)
	}

	if as.Ledger.Position(t.Security, t.Date) < t.Quantity {
		return fmt.Errorf("cannot sell %f of %s, position is only %f", t.Quantity, t.Security, as.Ledger.Position(t.Security, t.Date))
	}

	return nil
}

// --- Declare Command ---

// Declare represents a transaction to declare a security for use in the ledger.
// This maps a ledger-internal ticker to a globally unique security ID and its currency.
type Declare struct {
	baseCmd
	Ticker   string `json:"ticker"`
	ID       ID     `json:"id"`
	Currency string `json:"currency"`
}

// NewDeclaration creates a new Declare transaction.
func NewDeclaration(day date.Date, memo, ticker, id, currency string) Declare {
	return Declare{
		baseCmd:  baseCmd{Command: CmdDeclare, Date: day, Memo: memo},
		Ticker:   ticker,
		ID:       ID(id),
		Currency: currency,
	}
}

// Validate checks the Declare transaction's fields.
// It ensures the ticker is not already declared and that the ID and currency are valid.
func (t *Declare) Validate(as *AccountingSystem) error {
	if err := t.baseCmd.Validate(as); err != nil {
		return err
	}
	if t.Ticker == "" {
		return errors.New("declaration ticker is missing")
	}
	if t.ID == "" {
		return errors.New("declaration security ID is missing")
	}
	if _, err := ParseID(t.ID.String()); err != nil {
		return fmt.Errorf("invalid security ID '%s' for declaration: %w", t.ID, err)
	}
	if err := ValidateCurrency(t.Currency); err != nil {
		return fmt.Errorf("invalid currency for declaration: %w", err)
	}

	return nil
}

// Dividend represents a dividend payment.
type Dividend struct {
	secCmd
	Amount float64 `json:"amount"`
}

// NewDividend creates a new Dividend transaction.
func NewDividend(day date.Date, memo, security string, amount float64) Dividend {
	return Dividend{
		secCmd: secCmd{baseCmd: baseCmd{Command: CmdDividend, Date: day, Memo: memo}, Security: security},
		Amount: amount,
	}
}

// Validate checks the Dividend transaction's fields. It ensures the dividend
// amount is positive.
func (t *Dividend) Validate(as *AccountingSystem) error {
	if err := t.secCmd.Validate(as); err != nil {
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

// Validate checks the Deposit transaction's fields. It ensures the deposit
// amount is positive and the currency code is valid.
func (t *Deposit) Validate(as *AccountingSystem) error {
	if err := t.baseCmd.Validate(as); err != nil {
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

// Validate checks the Withdraw transaction's fields.
// It handles a "withdraw all" case if the amount is 0, ensures the final
// amount is positive, and verifies there is sufficient cash to cover the withdrawal.
func (t *Withdraw) Validate(as *AccountingSystem) error {
	if err := t.baseCmd.Validate(as); err != nil {
		return err
	}

	if t.Currency != "" {
		if err := ValidateCurrency(t.Currency); err != nil {
			return fmt.Errorf("invalid currency for withdraw: %w", err)
		}
	}

	if t.Amount == 0 {
		// quick fix, cash all.
		t.Amount = as.Ledger.CashBalance(t.Currency, t.Date)
	}

	if t.Amount <= 0 {
		return fmt.Errorf("withdraw amount must be positive, got %f", t.Amount)
	}

	cash, cost := as.Ledger.CashBalance(t.Currency, t.Date), t.Amount
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

// Validate checks the Convert transaction's fields.
// It handles a "convert all" case if the from-amount is 0. It ensures both
// amounts are positive, currencies are valid, and there is sufficient cash in
// the source currency account to cover the conversion.
func (t *Convert) Validate(as *AccountingSystem) error {
	if err := t.baseCmd.Validate(as); err != nil {
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
		t.FromAmount = as.Ledger.CashBalance(t.FromCurrency, t.Date)
	}

	if t.FromAmount <= 0 {
		// fromAmount == 0 is interpreted as "convert all".
		return fmt.Errorf("convert 'from' amount must be positive, got %f", t.FromAmount)
	}

	cash, cost := as.Ledger.CashBalance(t.FromCurrency, t.Date), t.FromAmount
	if cash < cost {
		return fmt.Errorf("cannot withdraw for %f %s cash balance is %f %s", cost, t.FromCurrency, cash, t.FromCurrency)
	}

	return nil
}

// BySecurity returns a predicate that filters transactions by security ticker.
func BySecurity(ticker string) func(Transaction) bool {
	return func(tx Transaction) bool {
		switch v := tx.(type) {
		case Buy:
			return v.Security == ticker
		case Sell:
			return v.Security == ticker
		case Dividend:
			return v.Security == ticker
		case Declare:
			return v.Ticker == ticker
		default:
			return false
		}
	}
}

// ByCurrency returns a predicate that filters transactions by currency.
func ByCurrency(currency string) func(Transaction) bool {
	return func(tx Transaction) bool {
		switch v := tx.(type) {
		case Deposit:
			return v.Currency == currency
		case Withdraw:
			return v.Currency == currency
		case Convert:
			return v.FromCurrency == currency || v.ToCurrency == currency
		case Declare:
			return v.Currency == currency
		default:
			return false
		}
	}
}
