package portfolio

import (
	"errors"
	"fmt"
	"slices"
)

// CommandType is a typed string for identifying transaction commands.
type CommandType string

func (c CommandType) IsCashFlow() bool { return c == CmdDeposit || c == CmdWithdraw }

// Command types used for identifying transactions.
const (
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

func (t Buy) Equal(other Transaction) bool {
	o, ok := other.(Buy)
	return ok && t.secCmd == o.secCmd && t.Quantity.Equal(o.Quantity) && t.Amount.Equal(o.Amount)
}

func (t *Buy) Currency() string { return t.Amount.Currency() }

// Validate checks the Buy transaction's fields. It ensures that the quantity
// and price are positive. It also verifies that there is enough cash in the
// corresponding currency account to cover the cost of the purchase on the
// transaction date. It now accepts a Ledger object.
func (t *Buy) Validate(ledger *Ledger) error {
	if err := t.secCmd.Validate(ledger); err != nil {
		return err
	}

	if t.Quantity.IsNegative() || t.Quantity.IsZero() {
		return fmt.Errorf("buy transaction quantity must be positive, got %s", t.Quantity.String())
	}
	if t.Amount.IsNegative() || t.Amount.IsZero() {
		return fmt.Errorf("buy transaction amount must be positive, got %s", t.Amount.String())
	}

	ledgerSec := ledger.Security(t.Security) // We know this is not nil from secCmd.Validate
	currency := ledgerSec.Currency()
	// first the quick fix
	if t.Currency() == "" {
		t.Amount = M(t.Amount.value, currency)
	} else if currency != t.Currency() {
		return fmt.Errorf("buy transaction currency %s does not match security currency %s", t.Currency(), currency)
	}

	cash, cost := ledger.CashBalance(t.Currency(), t.Date), t.Amount
	if cash.LessThan(cost) {
		return fmt.Errorf("cannot buy for %s cash balance is %s", cost, cash)
	}
	return nil
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
// now accepts a Ledger and a Balance object.
func (t *Sell) Validate(ledger *Ledger, b *Balance) error {
	if err := t.secCmd.Validate(ledger); err != nil {
		return err
	}

	// Quick fix currency and check
	ledgerSec := ledger.Security(t.Security) // We know this is not nil from secCmd.Validate
	currency := ledgerSec.Currency()
	// first the quick fix
	if t.Currency() == "" {
		t.Amount = M(t.Amount.value, currency)
	} else if currency != t.Currency() {
		return fmt.Errorf("sell transaction currency %s does not match security currency %s", t.Currency(), currency)
	}
	if !t.Amount.IsPositive() {
		return fmt.Errorf("sell transaction amount must be positive, got %v", t.Amount)
	}

	if t.Quantity.IsZero() {
		// quick fix, sell all.
		t.Quantity = b.Position(t.Security)
	}

	if !t.Quantity.IsPositive() {
		return fmt.Errorf("sell transaction quantity must be positive, got %s", t.Quantity.String())
	}

	if b.Position(t.Security).LessThan(t.Quantity) {
		return fmt.Errorf("cannot sell %v of %s, position is only %v", t.Quantity, t.Security, b.Position(t.Security))
	}

	return nil
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

// NewDeclare creates a new Declare transaction.
func NewDeclare(day Date, memo, ticker string, id ID, currency string) Declare {
	return Declare{
		baseCmd:  baseCmd{Command: CmdDeclare, Date: day, Memo: memo},
		Ticker:   ticker,
		ID:       ID(id),
		Currency: currency,
	}
}

// Validate checks the Declare transaction's fields.
// It ensures the ticker is not already declared and that the ID and currency are valid.
func (t *Declare) Validate(ledger *Ledger) error {
	t.baseCmd.Validate()
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

	// TODO add a check that the dclared does not already exists
	ledgerSec := ledger.Security(t.Ticker)
	if ledgerSec != nil {
		return fmt.Errorf("security %q already declared in ledger", t.Ticker)
	}

	return nil
}

// Dividend represents a dividend payment.
// Dividend represents a transaction where a dividend payment is received
// for a held security.
type Dividend struct {
	secCmd
	Amount           Money // Amount is the total dividend amount received.
	DividendPerShare Money // DividendPerShare is the amount paid per share.
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
	w.EmbedFrom(t.Amount)
	w.Optional("dividendPerShare", t.DividendPerShare.value)
	return w.MarshalJSON()
}

func (t Dividend) Equal(other Transaction) bool {
	o, ok := other.(Dividend)
	return ok && t.secCmd == o.secCmd && t.Amount.Equal(o.Amount)
}

// Validate checks the Dividend transaction's fields. It ensures the dividend
// amount is positive.
func (t *Dividend) Validate(ledger *Ledger, b *Balance) error {
	if err := t.secCmd.Validate(ledger); err != nil {
		return err
	}

	// Quick fix: if amount is missing but dividend per share is provided, calculate it.
	if t.Amount.IsZero() && t.DividendPerShare.IsPositive() {
		position := b.Position(t.Security)
		if position.IsZero() {
			return fmt.Errorf("cannot calculate total dividend for %s, position is zero on %s", t.Security, t.When())
		}
		// TODO: Broker-paid dividends are often net of taxes. The actual amount might be
		// slightly less than (position * dividend_per_share). This could be modeled
		// in the future by creating an associated tax withdrawal transaction.
		t.Amount = t.DividendPerShare.Mul(position)
	}

	if !t.Amount.IsPositive() && !t.DividendPerShare.IsPositive() {
		return errors.New("dividend must have a positive amount or dividendPerShare")
	}

	// Final check on the amount, which should now be populated.
	if !t.Amount.IsZero() && !t.Amount.IsPositive() {
		return fmt.Errorf("dividend amount must be positive, got %v", t.Amount)
	}
	return nil
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
func (t *Deposit) Validate(ledger *Ledger) error {
	t.baseCmd.Validate()

	if !t.Amount.IsPositive() {
		return fmt.Errorf("deposit amount must be positive, got %v", t.Amount)
	}
	// TODO: this validation is not correct, it should be done in the journal
	if err := ValidateCurrency(t.Amount.Currency()); err != nil {
		return fmt.Errorf("invalid currency for deposit: %w", err)
	}

	if t.Settles != "" {
		cur, exists := ledger.CounterPartyCurrency(t.Settles)
		if !exists {
			return fmt.Errorf("counterparty account %q not found", t.Settles)
		}
		if cur != t.Amount.Currency() {
			return fmt.Errorf("settlement currency %s does not match counterparty account currency %s", t.Amount.Currency(), cur)
		}
	}
	return nil
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
func (t *Withdraw) Validate(ledger *Ledger) error {
	t.baseCmd.Validate()

	if err := ValidateCurrency(t.Amount.Currency()); err != nil {
		return fmt.Errorf("invalid currency for withdraw: %w", err)
	}

	if t.Amount.IsZero() {
		t.Amount = ledger.CashBalance(t.Amount.Currency(), t.Date)
	}

	if !t.Amount.IsPositive() {
		return fmt.Errorf("withdraw amount must be positive, got %s", t.Amount.String())
	}

	cash := ledger.CashBalance(t.Amount.Currency(), t.Date)
	if cash.LessThan(t.Amount) {
		return fmt.Errorf("cannot withdraw for %s cash balance is %s", t.Amount.String(), cash.String())
	}
	if t.Settles != "" {
		accounts := slices.Collect(ledger.AllCounterpartyAccounts())
		if !slices.Contains(accounts, t.Settles) {
			return fmt.Errorf("counterparty account %q not found", t.Settles)
		}

		balance := ledger.CounterpartyAccountBalance(t.Settles, t.Date)
		if balance.Currency() != t.Currency() {
			return fmt.Errorf("settlement currency %s does not match counterparty account currency %s", t.Currency(), balance.Currency())
		}
	}
	return nil
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
func (t *Accrue) Validate(ledger *Ledger) error {
	t.baseCmd.Validate()
	if t.Counterparty == "" {
		return errors.New("accrue transaction counterparty is missing")
	}
	if t.Amount.IsZero() {
		return errors.New("accrue transaction amount cannot be zero")
	}
	if err := ValidateCurrency(t.Currency()); err != nil {
		return fmt.Errorf("invalid currency for accrue: %w", err)
	}

	// Check if the account already exists in the ledger at any point in time
	currency, exists := ledger.CounterPartyCurrency(t.Counterparty)
	// If the account does not exist in the ledger at all, then it's a new creation.
	if !exists {
		t.Create = true
	}

	if !t.Create && currency != t.Currency() {
		return fmt.Errorf("new accrue currency %s does not match counterparty account currency %s", t.Currency(), currency)
	}
	return nil
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
func (t *Convert) Validate(ledger *Ledger) error {
	t.baseCmd.Validate()

	if err := ValidateCurrency(t.FromCurrency()); err != nil {
		return fmt.Errorf("invalid 'from' currency: %w", err)
	}
	if err := ValidateCurrency(t.ToCurrency()); err != nil {
		return fmt.Errorf("invalid 'to' currency: %w", err)
	}
	if t.FromCurrency() == t.ToCurrency() {
		return fmt.Errorf("cannot convert to the same currency: %s", t.FromCurrency())
	}

	if !t.ToAmount.IsPositive() {
		return fmt.Errorf("convert 'to' amount must be positive, got %v", t.ToAmount)
	}

	if t.FromAmount.IsZero() && t.FromAmount.Currency() != "" {
		t.FromAmount = ledger.CashBalance(t.FromCurrency(), t.Date)
	}
	if !t.FromAmount.IsPositive() {
		// fromAmount == 0 is interpreted as "convert all".
		return fmt.Errorf("convert 'from' amount must be positive, got %v", t.FromAmount)
	}

	cash, cost := ledger.CashBalance(t.FromCurrency(), t.Date), t.FromAmount
	if cash.LessThan(cost) {
		return fmt.Errorf("cannot withdraw for %v cash balance is %v", cost, cash)
	}

	return nil
}

// --- UpdatePrice Command ---

// UpdatePrice represents a transaction to record the price of a security on a specific date.
type UpdatePrice struct {
	secCmd
	Price Money `json:"price"`
}

// NewUpdatePrice creates a new UpdatePrice transaction.
func NewUpdatePrice(date Date, ticker string, price Money) UpdatePrice {
	return UpdatePrice{
		secCmd: secCmd{
			baseCmd:  baseCmd{Command: CmdUpdatePrice, Date: date},
			Security: ticker,
		},
		Price: price,
	}
}

// MarshalJSON implements the json.Marshaler interface for UpdatePrice.
func (t UpdatePrice) MarshalJSON() ([]byte, error) {
	var w jsonObjectWriter
	w.EmbedFrom(t.secCmd)
	w.EmbedFrom(t.Price)
	return w.MarshalJSON()
}

func (t UpdatePrice) Equal(other Transaction) bool {
	o, ok := other.(UpdatePrice)
	return ok && t.secCmd == o.secCmd && t.Price.Equal(o.Price)
}

// Validate checks the UpdatePrice transaction's fields.
func (t *UpdatePrice) Validate(ledger *Ledger) error {
	if err := t.secCmd.Validate(ledger); err != nil {
		return err
	}
	if !t.Price.IsPositive() {
		return fmt.Errorf("price must be positive, got %v", t.Price)
	}
	return nil
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
func (t *Split) Validate(ledger *Ledger) error {
	if err := t.secCmd.Validate(ledger); err != nil {
		return err
	}
	if t.Numerator <= 0 {
		return fmt.Errorf("split numerator must be positive, got %d", t.Numerator)
	}
	if t.Denominator <= 0 {
		return fmt.Errorf("split denominator must be positive, got %d", t.Denominator)
	}
	return nil
}

// MarshalJSON implements the json.Marshaler interface for Split.
func (t Split) MarshalJSON() ([]byte, error) {
	var w jsonObjectWriter
	w.EmbedFrom(t.secCmd)
	w.Append("num", t.Numerator)
	w.Append("den", t.Denominator)
	return w.MarshalJSON()
}
