package portfolio

import (
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"maps"
	"slices"

	"github.com/shopspring/decimal"
)

// CommandType is a typed string for identifying transaction commands.
type CommandType string

// Command types used for identifying transactions.
const (
	CmdInit        CommandType = "init"
	CmdAccrue      CommandType = "accrue"
	CmdBuy         CommandType = "buy"
	CmdSell        CommandType = "sell"
	CmdDividend    CommandType = "dividend"
	CmdDeposit     CommandType = "deposit"
	CmdWithdraw    CommandType = "withdraw"
	CmdConvert     CommandType = "convert"
	CmdDeclare     CommandType = "declare"
	CmdUpdatePrice CommandType = "update-price"
	CmdSplit       CommandType = "split"
)

// Transaction defines the common interface for all types of financial transactions
// that can be recorded in the ledger.
type Transaction interface {
	What() CommandType // What returns the command type of the transaction (e.g., "buy", "sell").
	When() Date        // When returns the date on which the transaction occurred.
	Equal(Transaction) bool
	Validate(ledger *Ledger) (Transaction, error)
}

type baseCmd struct {
	Command CommandType `json:"command"`        // Command specifies the type of transaction (e.g., "buy", "sell").
	Date    Date        `json:"date"`           // Date is the date when the transaction took place.
	Memo    string      `json:"memo,omitempty"` // Memo provides an optional rationale or note for the transaction.
}

// What returns the command name for the transaction, which is used to identify the type of transaction.
func (t baseCmd) What() CommandType {
	return t.Command
}

// When returns the date of the transaction.
func (t baseCmd) When() Date {
	return t.Date
}

// Rationale returns the memo associated with the transaction, which can provide additional context or rationale.
func (t baseCmd) Rationale() string {
	return t.Memo
}

// MarshalJSON implements the json.Marshaler interface for baseCmd.
func (t baseCmd) MarshalJSON() ([]byte, error) {
	var w jsonObjectWriter
	w.Append("command", t.Command)
	w.Append("date", t.Date)
	w.Optional("memo", t.Memo)
	return w.MarshalJSON()
}

// Validate checks the base command fields. It sets the date to today if it's zero.
// It's meant to be embedded in other transaction validation methods.
func (t *baseCmd) Validate() {
	if t.Date == (Date{}) {
		t.Date = Today()
	}
}

// secCmd is a component for security-based transactions (buy, sell, dividend).
type secCmd struct {
	baseCmd
	Security string `json:"security"` // Security is the ticker symbol of the security involved in the transaction.
}

// Validate checks the security command fields. It validates the base command,
// ensures a security ticker is present, and attempts to auto-populate the
// currency from the security's definition if it's missing.
func (t *secCmd) Validate(ledger *Ledger) error {
	t.baseCmd.Validate()

	if t.Security == "" {
		return errors.New("security ticker is missing")
	}

	// use ticker to resolve the ledger security
	ledgerSec := ledger.Security(t.Security)
	if ledgerSec == nil {
		return fmt.Errorf("security %q not declared in ledger", t.Security)
	}

	return nil
}

// MarshalJSON implements the json.Marshaler interface for secCmd.
func (t secCmd) MarshalJSON() ([]byte, error) {
	var w jsonObjectWriter
	w.EmbedFrom(t.baseCmd)
	w.Append("security", t.Security)
	return w.MarshalJSON()
}

// Buy represents a buy transaction.
// Buy represents a transaction where a quantity of a security is purchased
// for a specified amount.
type Buy struct {
	secCmd
	Quantity Quantity // Quantity is the number of shares or units bought.
	Amount   Money    // Amount is the total cost of the purchase.
}

// NewBuy creates a new Buy transaction.
func NewBuy(day Date, memo, security string, quantity Quantity, amount Money) Buy {
	return Buy{
		secCmd:   secCmd{baseCmd: baseCmd{Command: CmdBuy, Date: day, Memo: memo}, Security: security},
		Quantity: quantity,
		Amount:   amount,
	}
}

// MarshalJSON implements the json.Marshaler interface for Buy.
func (t Buy) MarshalJSON() ([]byte, error) {
	var w jsonObjectWriter
	w.EmbedFrom(t.secCmd)
	w.Append("quantity", t.Quantity)
	w.EmbedFrom(t.Amount)
	return w.MarshalJSON()
}

// UnmarshalJSON implements the json.Unmarshaler interface for Buy.
// It handles the custom structure where amount and currency are separate fields.
func (t *Buy) UnmarshalJSON(data []byte) error {
	// Use a temporary type that has all possible fields.
	var temp struct {
		secCmd
		amountCmd
		Quantity Quantity `json:"quantity"`
	}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	t.secCmd = temp.secCmd
	t.Quantity = temp.Quantity
	t.Amount = temp.Money()
	return nil
}

func (t Buy) Equal(other Transaction) bool {
	o, ok := other.(Buy)
	return ok && t.secCmd == o.secCmd && t.Quantity.Equal(o.Quantity) && t.Amount.Equal(o.Amount)
}

func (t *Buy) Currency() string { return t.Amount.Currency() }

// Validate checks the Buy transaction's fields. It ensures that the quantity
// and price are positive. It also verifies that there is enough cash in the
// corresponding currency account to cover the cost of the purchase on the
// transaction date. It now accepts a Ledger object.
func (t Buy) Validate(ledger *Ledger) (Transaction, error) {
	if err := t.secCmd.Validate(ledger); err != nil {
		return t, err
	}

	if t.Quantity.IsNegative() || t.Quantity.IsZero() {
		return t, fmt.Errorf("buy transaction quantity must be positive, got %s", t.Quantity.String())
	}
	if t.Amount.IsNegative() || t.Amount.IsZero() {
		return t, fmt.Errorf("buy transaction amount must be positive, got %s", t.Amount.String())
	}

	ledgerSec := ledger.Security(t.Security) // We know this is not nil from secCmd.Validate
	currency := ledgerSec.Currency()
	// first the quick fix
	if t.Currency() == "" {
		t.Amount = M(t.Amount.value, currency)
	} else if currency != t.Currency() {
		return t, fmt.Errorf("buy transaction currency %s does not match security currency %s", t.Currency(), currency)
	}

	cash, cost := ledger.CashBalance(t.Currency(), t.Date), t.Amount
	if cash.LessThan(cost) {
		return t, fmt.Errorf("on %s, cannot buy for %s cash balance is %s", t.When(), cost, cash)
	}
	return t, nil
}

// Sell represents a sell transaction.
// Sell represents a transaction where a quantity of a security is sold
// for a specified amount.
type Sell struct {
	secCmd
	Quantity Quantity // Quantity is the number of shares or units sold.
	Amount   Money    // Amount is the total proceeds from the sale.
}

// MarshalJSON implements the json.Marshaler interface for Sell.
func (t Sell) MarshalJSON() ([]byte, error) {
	var w jsonObjectWriter
	w.EmbedFrom(t.secCmd)
	w.Append("quantity", t.Quantity)
	w.EmbedFrom(t.Amount)
	return w.MarshalJSON()
}

// UnmarshalJSON implements the json.Unmarshaler interface for Sell.
// It handles the custom structure where amount and currency are separate fields.
func (t *Sell) UnmarshalJSON(data []byte) error {
	// Use a temporary type that has all possible fields.
	var temp struct {
		secCmd
		amountCmd
		Quantity Quantity `json:"quantity"`
	}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	t.secCmd = temp.secCmd
	t.Quantity = temp.Quantity
	t.Amount = temp.Money()
	return nil
}

func (t Sell) Equal(other Transaction) bool {
	o, ok := other.(Sell)
	return ok && t.secCmd == o.secCmd && t.Quantity.Equal(o.Quantity) && t.Amount.Equal(o.Amount)
}

// NewSell creates a new Sell transaction.
// If the quantity is set to 0, it signifies a "sell all" instruction.
// The actual number of shares will be determined during the validation phase
// based on the portfolio's position on the transaction date.
func NewSell(day Date, memo, security string, quantity Quantity, amount Money) Sell {
	return Sell{
		secCmd:   secCmd{baseCmd: baseCmd{Command: CmdSell, Date: day, Memo: memo}, Security: security},
		Quantity: quantity,
		Amount:   amount,
	}
}

func (t *Sell) Currency() string { return t.Amount.Currency() }

// Validate checks the Sell transaction's fields.
// It handles the "sell all" case by resolving a quantity of 0 to the total
// position size on the transaction date. It ensures the final quantity and
// price are positive and that the position is sufficient to cover the sale. It
// now accepts a Ledger object.
func (t Sell) Validate(ledger *Ledger) (Transaction, error) {
	if err := t.secCmd.Validate(ledger); err != nil {
		return t, err
	}

	// Quick fix currency and check
	ledgerSec := ledger.Security(t.Security) // We know this is not nil from secCmd.Validate
	currency := ledgerSec.Currency()
	// first the quick fix
	if t.Currency() == "" {
		t.Amount.cur = currency
	} else if currency != t.Currency() {
		return t, fmt.Errorf("sell transaction currency %s does not match security currency %s", t.Currency(), currency)
	}
	if !t.Amount.IsPositive() {
		return t, fmt.Errorf("sell transaction amount must be positive, got %v", t.Amount)
	}

	pos := ledger.Position(t.When(), t.Security)
	if t.Quantity.IsZero() {
		// quick fix, sell all.
		t.Quantity = pos
	}

	if !t.Quantity.IsPositive() {
		return t, fmt.Errorf("sell transaction quantity must be positive, got %s", t.Quantity.String())
	}

	if pos.LessThan(t.Quantity) {
		return t, fmt.Errorf("on %s, cannot sell %v of %s, position is only %v", t.When(), t.Quantity, t.Security, pos)
	}

	return t, nil
}

// --- Declare Command ---

// Declare represents a transaction to declare a security for use in the ledger.
// This maps a ledger-internal ticker to a globally unique security ID and its currency.
// Declare represents a transaction to declare a security for use in the ledger.
// This maps a ledger-internal ticker to a globally unique security ID and its currency.
type Declare struct {
	baseCmd
	Ticker   string `json:"ticker"`
	ID       ID     `json:"id"`
	Currency string `json:"currency"`
}

// NewDeclare creates a new Declare transaction.
func NewDeclare(day Date, memo, ticker string, id ID, currency string) Declare {
	return Declare{
		baseCmd:  baseCmd{Command: CmdDeclare, Date: day, Memo: memo},
		Ticker:   ticker,
		ID:       ID(id),
		Currency: currency,
	}
}

// MarshalJSON implements the json.Marshaler interface for Declare.
func (t Declare) MarshalJSON() ([]byte, error) {
	var w jsonObjectWriter
	w.EmbedFrom(t.baseCmd)
	w.Append("ticker", t.Ticker)
	w.Append("id", t.ID)
	w.Append("currency", t.Currency)
	return w.MarshalJSON()
}

func (t Declare) Equal(other Transaction) bool {
	o, ok := other.(Declare)
	return ok && t.baseCmd == o.baseCmd && t.Ticker == o.Ticker && t.ID == o.ID && t.Currency == o.Currency
}

// Validate checks the Declare transaction's fields.
// It ensures the ticker is not already declared and that the ID and currency are valid.
func (t Declare) Validate(ledger *Ledger) (Transaction, error) {
	t.baseCmd.Validate()
	if t.Ticker == "" {
		return t, errors.New("declaration ticker is missing")
	}
	if t.ID == "" {
		return t, errors.New("declaration security ID is missing")
	}
	if _, err := ParseID(t.ID.String()); err != nil {
		return t, fmt.Errorf("invalid security ID '%s' for declaration: %w", t.ID, err)
	}
	if err := ValidateCurrency(t.Currency); err != nil {
		return t, fmt.Errorf("invalid currency for declaration: %w", err)
	}

	ledgerSec := ledger.Security(t.Ticker)
	if ledgerSec != nil {
		return t, fmt.Errorf("security %q already declared in ledger", t.Ticker)
	}

	return t, nil
}

// --- Init Command ---

// Init represents the initialization of the ledger.
// It sets the base currency for the ledger. It has a date and must be the first transaction.
type Init struct {
	baseCmd
	Currency string `json:"currency"`
}

// NewInit creates a new Init transaction.
func NewInit(date Date, memo string, currency string) Init {
	return Init{
		baseCmd:  baseCmd{Command: CmdInit, Date: date, Memo: memo},
		Currency: currency,
	}
}

func (t Init) Equal(other Transaction) bool {
	o, ok := other.(Init)
	return ok && t.baseCmd == o.baseCmd && t.Currency == o.Currency
}

func (t Init) Validate(ledger *Ledger) (Transaction, error) {
	if err := ValidateCurrency(t.Currency); err != nil {
		return t, fmt.Errorf("invalid currency for init: %w", err)
	}

	if len(ledger.transactions) > 0 {
		// Case 1: Ledger is not empty.
		if existingInit, ok := ledger.transactions[0].(Init); ok {
			// Subcase 1.1: First tx is already Init -> update it idempotently.
			if !t.Date.IsZero() {
				existingInit.Date = t.Date
			}
			if t.Currency != "" {
				existingInit.Currency = t.Currency
			}
			if t.Memo != "" {
				existingInit.Memo = t.Memo
			}
			return existingInit, nil
		}

		// Subcase 1.2: First tx is not Init -> create and prepend Init.
		// Its date must be before the first existing transaction.
		firstTxDate := ledger.transactions[0].When()
		if t.Date.IsZero() {
			t.Date = firstTxDate // Quick fix: set date to the first day.
		} else if t.Date.After(firstTxDate) {
			return t, fmt.Errorf("init date %s must be before or equal to the first transaction date %s", t.Date, firstTxDate)
		}
	} else if t.Date.IsZero() {
		// Case 2: Ledger is empty, this is a new creation. Quick-fix date to today.
		if t.Date.IsZero() {
			t.Date = Today()
		}
	}
	return t, nil
}

// MarshalJSON implements the json.Marshaler interface for Init.
func (t Init) MarshalJSON() ([]byte, error) {
	var w jsonObjectWriter
	w.EmbedFrom(t.baseCmd)
	w.Append("currency", t.Currency)
	return w.MarshalJSON()
}

// --- Dividend Command ---

// Dividend represents a dividend payment.
// Dividend represents a transaction where a dividend payment is received
// for a held security.
type Dividend struct {
	secCmd
	Amount Money // Amount is the dividend paid per share.
}

// NewDividend creates a new Dividend transaction.
func NewDividend(day Date, memo, security string, amount Money) Dividend {
	return Dividend{
		secCmd: secCmd{baseCmd: baseCmd{Command: CmdDividend, Date: day, Memo: memo}, Security: security},
		Amount: amount,
	}
}

// MarshalJSON implements the json.Marshaler interface for Dividend.
func (t Dividend) MarshalJSON() ([]byte, error) {
	var w jsonObjectWriter
	w.EmbedFrom(t.secCmd)
	// by default money is persisted in its minor unit.
	// so we must call exact() to persist the dps.
	w.EmbedFrom(t.Amount.exact())
	return w.MarshalJSON()
}

// UnmarshalJSON implements the json.Unmarshaler interface for Dividend.
func (t *Dividend) UnmarshalJSON(data []byte) error {
	// Use a temporary type that has all possible fields.
	var temp struct {
		secCmd
		amountCmd
	}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}
	// Create the final transaction struct
	t.secCmd = temp.secCmd
	t.Amount = temp.Money()
	return nil
}

func (t Dividend) Equal(other Transaction) bool {
	o, ok := other.(Dividend)
	return ok && t.secCmd == o.secCmd && t.Amount.Equal(o.Amount)
}

// Validate checks the Dividend transaction's fields. It ensures the dividend
// amount is positive.
func (t Dividend) Validate(ledger *Ledger) (Transaction, error) {
	if err := t.secCmd.Validate(ledger); err != nil {
		return t, err
	}

	if !t.Amount.IsPositive() {
		return t, errors.New("dividend must have a positive amount per share")
	}

	// Quick fix currency if not provided
	if t.Amount.Currency() == "" {
		ledgerSec := ledger.Security(t.Security) // Not nil, checked in secCmd.Validate
		t.Amount = M(t.Amount.value, ledgerSec.Currency())
	} else if err := ValidateCurrency(t.Amount.Currency()); err != nil {
		// If currency is provided, validate it.
		return t, fmt.Errorf("invalid currency for dividend: %w", err)
	}

	return t, nil
}

// Deposit represents a cash deposit.
// Deposit represents a transaction where cash is added to a currency account
// within the portfolio.
type Deposit struct {
	baseCmd
	Amount  Money  // Amount is the quantity of cash deposited.
	Settles string // Settles is an optional counterparty account that this deposit settles.
}

func (t Deposit) Currency() string {
	return t.Amount.Currency()
}

// MarshalJSON implements the json.Marshaler interface for Deposit.
func (t Deposit) MarshalJSON() ([]byte, error) {
	var w jsonObjectWriter
	w.EmbedFrom(t.baseCmd)
	w.EmbedFrom(t.Amount)
	w.Optional("settles", t.Settles)
	return w.MarshalJSON()
}

// UnmarshalJSON implements the json.Unmarshaler interface for Deposit.
func (t *Deposit) UnmarshalJSON(data []byte) error {
	// Use a temporary type that has all possible fields.
	var temp struct {
		baseCmd
		amountCmd
		Settles string `json:"settles,omitempty"`
	}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}
	t.baseCmd = temp.baseCmd
	t.Amount = temp.Money()
	t.Settles = temp.Settles
	return nil
}

func (t Deposit) Equal(other Transaction) bool {
	o, ok := other.(Deposit)
	return ok && t.baseCmd == o.baseCmd && t.Amount.Equal(o.Amount) && t.Settles == o.Settles
}

// NewDeposit creates a new Deposit transaction.
func NewDeposit(day Date, memo string, amount Money, settles string) Deposit {
	return Deposit{
		baseCmd: baseCmd{Command: CmdDeposit, Date: day, Memo: memo},
		Amount:  amount,
		Settles: settles,
	}
}

// Validate checks the Deposit transaction's fields. It ensures the deposit
// amount is positive and the currency code is valid.
func (t Deposit) Validate(ledger *Ledger) (Transaction, error) {
	t.baseCmd.Validate()

	if !t.Amount.IsPositive() {
		return t, fmt.Errorf("deposit amount must be positive, got %v", t.Amount)
	}
	if err := ValidateCurrency(t.Amount.Currency()); err != nil {
		return t, fmt.Errorf("invalid currency for deposit: %w", err)
	}

	if t.Settles != "" {
		cur, exists := ledger.CounterPartyCurrency(t.Settles)
		if !exists {
			return t, fmt.Errorf("counterparty account %q not found", t.Settles)
		}
		if cur != t.Amount.Currency() {
			return t, fmt.Errorf("settlement currency %s does not match counterparty account currency %s", t.Amount.Currency(), cur)
		}
	}
	return t, nil
}

// Withdraw represents a cash withdrawal.
// Withdraw represents a transaction where cash is removed from a currency account
// within the portfolio.
type Withdraw struct {
	baseCmd
	Amount  Money  // Amount is the quantity of cash withdrawn.
	Settles string // Settles is an optional counterparty account that this withdrawal settles.
}

// MarshalJSON implements the json.Marshaler interface for Withdraw.
func (t Withdraw) MarshalJSON() ([]byte, error) {
	var w jsonObjectWriter
	w.EmbedFrom(t.baseCmd)
	w.EmbedFrom(t.Amount)
	w.Optional("settles", t.Settles)
	return w.MarshalJSON()
}

// UnmarshalJSON implements the json.Unmarshaler interface for Withdraw.
func (t *Withdraw) UnmarshalJSON(data []byte) error {
	// Use a temporary type that has all possible fields.
	var temp struct {
		baseCmd
		amountCmd
		Settles string `json:"settles,omitempty"`
	}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}
	t.baseCmd = temp.baseCmd
	t.Amount = temp.Money()
	t.Settles = temp.Settles
	return nil
}

func (t Withdraw) Equal(other Transaction) bool {
	o, ok := other.(Withdraw)
	return ok && t.baseCmd == o.baseCmd && t.Amount.Equal(o.Amount) && t.Settles == o.Settles
}

// NewWithdraw creates a new Withdraw transaction.
// If the amount is set to 0, it signifies a "withdraw all" instruction for the specified currency.
// The actual amount will be determined during the validation phase based on the cash balance on the transaction date.
func NewWithdraw(day Date, memo string, amount Money) Withdraw {
	return Withdraw{
		baseCmd: baseCmd{Command: CmdWithdraw, Date: day, Memo: memo},
		Amount:  amount,
	}
}

// Validate checks the Withdraw transaction's fields.
// It handles a "withdraw all" case if the amount is 0, ensures the final
// amount is positive, and verifies there is sufficient cash to cover the withdrawal
func (t Withdraw) Validate(ledger *Ledger) (Transaction, error) {
	t.baseCmd.Validate()

	if err := ValidateCurrency(t.Amount.Currency()); err != nil {
		return t, fmt.Errorf("invalid currency for withdraw: %w", err)
	}

	if t.Amount.IsZero() {
		t.Amount = ledger.CashBalance(t.Amount.Currency(), t.Date)
	}

	if !t.Amount.IsPositive() {
		return t, fmt.Errorf("withdraw amount must be positive, got %s", t.Amount.String())
	}

	cash := ledger.CashBalance(t.Amount.Currency(), t.Date)
	if cash.LessThan(t.Amount) {
		return t, fmt.Errorf("on %s, cannot withdraw for %s cash balance is %s", t.When(), t.Amount.String(), cash.String())
	}
	if t.Settles != "" {
		accounts := slices.Collect(ledger.AllCounterpartyAccounts())
		if !slices.Contains(accounts, t.Settles) {
			return t, fmt.Errorf("counterparty account %q not found", t.Settles)
		}

		balance := ledger.CounterpartyAccountBalance(t.Settles, t.Date)
		if balance.Currency() != t.Currency() {
			return t, fmt.Errorf("settlement currency %s does not match counterparty account currency %s", t.Currency(), balance.Currency())
		}
	}
	return t, nil
}

func (t *Withdraw) Currency() string { return t.Amount.Currency() }

// Accrue represents a non-cash transaction that affects a counterparty account.
// Accrue represents a non-cash transaction that affects a counterparty account,
// such as a loan or an accrued expense/income.
type Accrue struct {
	baseCmd
	Counterparty string // Counterparty is the name of the entity with whom the accrual is made.
	Amount       Money  // Amount is the value of the accrual. Positive for receivables, negative for payables.
	Create       bool   // Create is true if this accrual creates a new counterparty account.
}

// MarshalJSON implements the json.Marshaler interface for Accrue.
func (t Accrue) MarshalJSON() ([]byte, error) {
	var w jsonObjectWriter
	w.EmbedFrom(t.baseCmd)
	w.Append("counterparty", t.Counterparty)
	w.Optional("create", t.Create)
	w.EmbedFrom(t.Amount)
	return w.MarshalJSON()
}

// UnmarshalJSON implements the json.Unmarshaler interface for Accrue.
func (t *Accrue) UnmarshalJSON(data []byte) error {
	// Use a temporary type that has all possible fields.
	var temp struct {
		baseCmd
		amountCmd
		Counterparty string `json:"counterparty"`
		Create       bool   `json:"create,omitempty"`
	}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}
	t.baseCmd = temp.baseCmd
	t.Amount = temp.Money()
	t.Counterparty = temp.Counterparty
	t.Create = temp.Create
	return nil
}

func (t Accrue) Equal(other Transaction) bool {
	o, ok := other.(Accrue)
	return ok && t.baseCmd == o.baseCmd && t.Counterparty == o.Counterparty && t.Amount.Equal(o.Amount) && t.Create == o.Create
}

// NewAccrue creates a new Accrue transaction.
// A positive amount indicates a receivable (an asset), meaning the counterparty owes the user money.
// A negative amount indicates a payable (a liability), meaning the user owes the counterparty money.
func NewAccrue(day Date, memo, counterparty string, amount Money) Accrue {
	return Accrue{
		baseCmd:      baseCmd{Command: CmdAccrue, Date: day, Memo: memo},
		Counterparty: counterparty,
		Amount:       amount,
	}
}

// NewCreatedAccrue creates a new Accrue transaction.
// A positive amount indicates a receivable (an asset), meaning the counterparty owes the user money.
// A negative amount indicates a payable (a liability), meaning the user owes the counterparty money.
func NewCreatedAccrue(day Date, memo, counterparty string, amount Money) Accrue {
	return Accrue{
		baseCmd:      baseCmd{Command: CmdAccrue, Date: day, Memo: memo},
		Counterparty: counterparty,
		Amount:       amount,
		Create:       true,
	}
}

// Currency returns the currency of the transaction.
func (t *Accrue) Currency() string { return t.Amount.Currency() }

// Validate checks the Accrue transaction's fields.
func (t Accrue) Validate(ledger *Ledger) (Transaction, error) {
	t.baseCmd.Validate()
	if t.Counterparty == "" {
		return t, errors.New("accrue transaction counterparty is missing")
	}
	if t.Amount.IsZero() {
		return t, errors.New("accrue transaction amount cannot be zero")
	}
	if err := ValidateCurrency(t.Currency()); err != nil {
		return t, fmt.Errorf("invalid currency for accrue: %w", err)
	}

	// Check if the account already exists in the ledger at any point in time
	currency, exists := ledger.CounterPartyCurrency(t.Counterparty)
	// If the account does not exist in the ledger at all, then it's a new creation.
	if !exists {
		t.Create = true
	}

	if !t.Create && currency != t.Currency() {
		return t, fmt.Errorf("new accrue currency %s does not match counterparty account currency %s", t.Currency(), currency)
	}
	return t, nil
}

// Convert represents an internal currency conversion.
// Convert represents an internal currency conversion.
type Convert struct {
	baseCmd
	FromAmount Money
	ToAmount   Money
}

// MarshalJSON implements the json.Marshaler interface for Convert.
func (t Convert) MarshalJSON() ([]byte, error) {
	var w jsonObjectWriter
	w.EmbedFrom(t.baseCmd)
	w.PrefixFrom("from", t.FromAmount)
	w.PrefixFrom("to", t.ToAmount)
	return w.MarshalJSON()
}

// UnmarshalJSON implements the json.Unmarshaler interface for Convert.
func (t *Convert) UnmarshalJSON(data []byte) error {
	// Use a temporary type that has all possible fields.
	temp := convertCmd{}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	// Create the final transaction struct
	t.baseCmd = temp.baseCmd
	t.FromAmount = temp.FromMoney()
	t.ToAmount = temp.ToMoney()
	return nil
}

func (t Convert) Equal(other Transaction) bool {
	o, ok := other.(Convert)
	return ok && t.baseCmd == o.baseCmd && t.FromAmount.Equal(o.FromAmount) && t.ToAmount.Equal(o.ToAmount)
}

// NewConvert creates a new Convert transaction.
func NewConvert(day Date, memo string, fromAmount Money, toAmount Money) Convert {
	return Convert{
		baseCmd:    baseCmd{Command: CmdConvert, Date: day, Memo: memo},
		FromAmount: fromAmount,
		ToAmount:   toAmount,
	}
}

func (t *Convert) FromCurrency() string { return t.FromAmount.Currency() }
func (t *Convert) ToCurrency() string   { return t.ToAmount.Currency() }

// Validate checks the Convert transaction's fields.
// It handles a "convert all" case if the from-amount is 0. It ensures both
// amounts are positive, currencies are valid, and there is sufficient cash in
// the source currency account to cover the conversion.
func (t Convert) Validate(ledger *Ledger) (Transaction, error) {
	t.baseCmd.Validate()

	if err := ValidateCurrency(t.FromCurrency()); err != nil {
		return t, fmt.Errorf("invalid 'from' currency: %w", err)
	}
	if err := ValidateCurrency(t.ToCurrency()); err != nil {
		return t, fmt.Errorf("invalid 'to' currency: %w", err)
	}
	if t.FromCurrency() == t.ToCurrency() {
		return t, fmt.Errorf("cannot convert to the same currency: %s", t.FromCurrency())
	}

	if !t.ToAmount.IsPositive() {
		return t, fmt.Errorf("convert 'to' amount must be positive, got %v", t.ToAmount)
	}

	if t.FromAmount.IsZero() && t.FromAmount.Currency() != "" {
		t.FromAmount = ledger.CashBalance(t.FromCurrency(), t.Date)
	}
	if !t.FromAmount.IsPositive() {
		// fromAmount == 0 is interpreted as "convert all".
		return t, fmt.Errorf("convert 'from' amount must be positive, got %v", t.FromAmount)
	}

	cash, cost := ledger.CashBalance(t.FromCurrency(), t.Date), t.FromAmount
	if cash.LessThan(cost) {
		return t, fmt.Errorf("on %s, cannot convert for %v cash balance is %v", t.When(), cost, cash)
	}

	return t, nil
}

// --- UpdatePrice Command ---

// UpdatePrice represents a transaction to record the prices of multiple securities on a specific date.
type UpdatePrice struct {
	baseCmd
	Prices map[string]decimal.Decimal
}

// NewUpdatePrice creates a new UpdatePrice transaction for a single security.
// This is kept for backward compatibility and ease of transition.
func NewUpdatePrice(date Date, ticker string, price Money) UpdatePrice {
	return UpdatePrice{
		baseCmd: baseCmd{Command: CmdUpdatePrice, Date: date},
		Prices:  map[string]decimal.Decimal{ticker: price.value},
	}
}

// NewUpdatePrices creates a new UpdatePrice transaction for multiple securities.
func NewUpdatePrices(date Date, prices map[string]decimal.Decimal) UpdatePrice {
	if prices == nil {
		prices = make(map[string]decimal.Decimal)
	}
	return UpdatePrice{
		baseCmd: baseCmd{Command: CmdUpdatePrice, Date: date},
		Prices:  prices,
	}
}

// MarshalJSON implements the json.Marshaler interface for UpdatePrice.
func (t UpdatePrice) MarshalJSON() ([]byte, error) {
	var w jsonObjectWriter
	w.EmbedFrom(t.baseCmd)

	// Custom marshaling for the 'prices' map to ensure stable key order.
	var pricesObject jsonObjectWriter
	for ticker, price := range t.PricesIter() {
		pricesObject.Append(ticker, price)
	}
	pricesBytes, err := pricesObject.MarshalJSON()
	if err != nil {
		return nil, err
	}

	// Manually append the 'prices' object to the main writer.
	w.WriteString(`"prices":`)
	w.Write(pricesBytes)
	w.WriteString(",")

	return w.MarshalJSON()
}

// UnmarshalJSON implements the json.Unmarshaler interface for UpdatePrice.
func (t *UpdatePrice) UnmarshalJSON(data []byte) error {
	var temp struct {
		baseCmd
		Prices map[string]decimal.Decimal `json:"prices"`
	}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}
	t.baseCmd = temp.baseCmd
	t.Prices = temp.Prices
	return nil
}

// PricesIter returns an iterator that yields ticker and price pairs in a stable, sorted order.
func (t UpdatePrice) PricesIter() iter.Seq2[string, decimal.Decimal] {
	keys := slices.Collect(maps.Keys(t.Prices))
	slices.Sort(keys)
	// return the iterator in keys order
	return func(yield func(string, decimal.Decimal) bool) {
		for _, key := range keys {
			if !yield(key, t.Prices[key]) {
				return
			}
		}
	}
}

func (t UpdatePrice) Equal(other Transaction) bool {
	o, ok := other.(UpdatePrice)
	if !ok || t.baseCmd != o.baseCmd || len(t.Prices) != len(o.Prices) {
		return false
	}
	for k, v := range t.Prices {
		if ov, ok := o.Prices[k]; !ok || !v.Equal(ov) {
			return false
		}
	}
	return true
}

// Validate checks the UpdatePrice transaction's fields.
func (t UpdatePrice) Validate(ledger *Ledger) (Transaction, error) {
	t.baseCmd.Validate()
	for ticker, price := range t.Prices {
		if ledger.Security(ticker) == nil {
			return t, fmt.Errorf("security %q not declared in ledger", ticker)
		}
		if !price.IsPositive() {
			return t, fmt.Errorf("price for %s must be positive, got %v", ticker, price)
		}
	}
	return t, nil
}

// --- Split Command ---

// Split represents a stock split event for a security.
type Split struct {
	secCmd
	Numerator   int64 `json:"num"`
	Denominator int64 `json:"den"`
}

// NewSplit creates a new Split transaction.
func NewSplit(date Date, ticker string, num, den int64) Split {
	return Split{
		secCmd: secCmd{
			baseCmd:  baseCmd{Command: CmdSplit, Date: date},
			Security: ticker,
		},
		Numerator:   num,
		Denominator: den,
	}
}

func (t Split) Equal(other Transaction) bool {
	o, ok := other.(Split)
	return ok && t.secCmd == o.secCmd && t.Numerator == o.Numerator && t.Denominator == o.Denominator
}

// Validate checks the Split transaction's fields.
func (t Split) Validate(ledger *Ledger) (Transaction, error) {
	if err := t.secCmd.Validate(ledger); err != nil {
		return t, err
	}
	if t.Numerator <= 0 {
		return t, fmt.Errorf("split numerator must be positive, got %d", t.Numerator)
	}
	if t.Denominator <= 0 {
		return t, fmt.Errorf("split denominator must be positive, got %d", t.Denominator)
	}
	return t, nil
}

// MarshalJSON implements the json.Marshaler interface for Split.
func (t Split) MarshalJSON() ([]byte, error) {
	var w jsonObjectWriter
	w.EmbedFrom(t.secCmd)
	w.Append("num", t.Numerator)
	w.Append("den", t.Denominator)
	return w.MarshalJSON()
}

// UnmarshalJSON implements the json.Unmarshaler interface for Split.
func (t *Split) UnmarshalJSON(data []byte) error {
	var temp struct {
		secCmd
		Numerator   int64 `json:"num"`
		Denominator int64 `json:"den"`
	}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}
	// default Numerator and Denominator to 1
	if temp.Denominator == 0 {
		// Default to 1 if not present, which is a common case for JSON unmarshaling of optional int fields.
		temp.Denominator = 1
	}
	t.secCmd = temp.secCmd
	t.Numerator = temp.Numerator
	t.Denominator = temp.Denominator
	return nil
}
