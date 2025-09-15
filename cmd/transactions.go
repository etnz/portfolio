package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/etnz/portfolio"
	"github.com/google/subcommands"
	"github.com/shopspring/decimal"
)

// --- Buy Command ---

// buyCmd holds the flags for the 'buy' subcommand.
type buyCmd struct {
	date     string
	security string
	quantity decimal.Decimal
	amount   decimal.Decimal
	memo     string
}

func (*buyCmd) Name() string     { return "buy" }
func (*buyCmd) Synopsis() string { return "record the purchase of a security" }
func (*buyCmd) Usage() string {
	return `pcs buy -d <date> -s <security> -q <quantity> -p <price> [-m <memo>]

	Purchases shares of a security. The total cost is debited from the cash account in the security's currency.
`
}

func (c *buyCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.date, "d", portfolio.Today().String(), "Transaction date. See the user manual for supported date formats.")
	f.StringVar(&c.security, "s", "", "Security ticker")
	f.Var(DecimalVar(&c.quantity, "0"), "q", "Number of shares")
	f.Var(DecimalVar(&c.amount, "0"), "a", "Total amount paid for the shares")
	f.StringVar(&c.memo, "m", "", "An optional rationale or note for the transaction")
}

func (c *buyCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if c.security == "" || c.quantity.IsZero() || c.amount.IsZero() {
		fmt.Fprintln(os.Stderr, "Error: -s, -q, and -a flags are all required.")
		return subcommands.ExitUsageError
	}
	day, err := portfolio.ParseDate(c.date) // Validate date format
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing date: %v\n", err)
		return subcommands.ExitUsageError
	}

	tx := portfolio.NewBuy(day, c.memo, c.security, portfolio.Q(c.quantity), portfolio.M(c.amount, ""))
	_, status := handleTransaction(tx, f)
	return status
}

// --- Sell Command ---

// sellCmd holds the flags for the 'sell' subcommand.
type sellCmd struct {
	date     string
	security string
	quantity portfolio.Quantity
	amount   decimal.Decimal
	memo     string
}

func (*sellCmd) Name() string     { return "sell" }
func (*sellCmd) Synopsis() string { return "record the sale of a security" }
func (*sellCmd) Usage() string {
	return `pcs sell -d <date> -s <security> -p <price> [-q <quantity>] [-m <memo>]

	Sells shares of a security. The proceeds are credited to the cash account in the security's currency.
	If -q is not specified, all shares of the security are sold.
`
}
func (c *sellCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.date, "d", portfolio.Today().String(), "Transaction date. See the user manual for supported date formats.")
	f.StringVar(&c.security, "s", "", "Security ticker")
	f.Var(QuantityVar(&c.quantity, "0"), "q", "Number of shares, if missing all shares are sold")
	f.Var(DecimalVar(&c.amount, "0"), "a", "Total amount received for the shares")
	f.StringVar(&c.memo, "m", "", "An optional rationale or note for the transaction")
}
func (c *sellCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if c.security == "" || c.amount.IsZero() {
		fmt.Fprintln(os.Stderr, "Error: -s and -a flags are required.")
		return subcommands.ExitUsageError
	}
	day, err := portfolio.ParseDate(c.date) // Validate date format
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing date: %v\n", err)
		return subcommands.ExitUsageError
	}
	tx := portfolio.NewSell(day, c.memo, c.security, c.quantity, portfolio.M(c.amount, ""))
	_, status := handleTransaction(tx, f)
	return status
}

// --- Dividend Command ---

// dividendCmd holds the flags for the 'dividend' subcommand.
type dividendCmd struct {
	date     string
	security string
	amount   decimal.Decimal
	memo     string
}

func (*dividendCmd) Name() string     { return "dividend" }
func (*dividendCmd) Synopsis() string { return "record a dividend payment for a security" }
func (*dividendCmd) Usage() string {
	return `pcs dividend -d <date> -s <security> -a <amount> [-m <memo>]

	Records a dividend payment. The amount is credited to the cash account in the security's currency.
`
}
func (c *dividendCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.date, "d", portfolio.Today().String(), "Transaction date. See the user manual for supported date formats.")
	f.StringVar(&c.security, "s", "", "Security ticker receiving the dividend")
	f.Var(DecimalVar(&c.amount, "0"), "a", "Total dividend amount received")
	f.StringVar(&c.memo, "m", "", "An optional rationale or note")
}
func (c *dividendCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if c.security == "" || c.amount.IsZero() {
		fmt.Fprintln(os.Stderr, "Error: -s and -a flags are required.")
		return subcommands.ExitUsageError
	}
	day, err := portfolio.ParseDate(c.date) // Validate date format
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing date: %v\n", err)
		return subcommands.ExitUsageError
	}

	tx := portfolio.NewDividend(day, c.memo, c.security, portfolio.M(c.amount, ""))
	_, status := handleTransaction(tx, f)
	return status
}

// --- Deposit Command ---

// depositCmd holds the flags for the 'deposit' subcommand.
type depositCmd struct {
	date     string
	amount   decimal.Decimal
	currency string
	memo     string
	settles  string
}

func (*depositCmd) Name() string     { return "deposit" }
func (*depositCmd) Synopsis() string { return "record a cash deposit into the portfolio" }
func (*depositCmd) Usage() string {
	return `pcs deposit -d <date> -a <amount> -c <currency> [-m <memo>] [-settles <account>]

	Records a cash deposit into the portfolio's cash account.
`
}
func (c *depositCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.date, "d", portfolio.Today().String(), "Transaction date. See the user manual for supported date formats.")
	f.Var(DecimalVar(&c.amount, "0"), "a", "Amount of cash to deposit")
	f.StringVar(&c.currency, "c", "EUR", "Currency of the deposit (e.g., USD, EUR). Cash is kept in that currency")
	f.StringVar(&c.memo, "m", "", "An optional rationale or note")
	f.StringVar(&c.settles, "settles", "", "Settle a counterparty account")
}
func (c *depositCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...any) subcommands.ExitStatus {
	if c.amount.IsZero() {
		fmt.Fprintln(os.Stderr, "Error: -a flag is required.")
		return subcommands.ExitUsageError
	}
	day, err := portfolio.ParseDate(c.date) // Validate date format
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing date: %v\n", err)
		return subcommands.ExitUsageError
	}

	tx := portfolio.NewDeposit(day, c.memo, portfolio.M(c.amount, c.currency), c.settles)
	_, status := handleTransaction(tx, f)
	return status
}

// --- Withdraw Command ---

// withdrawCmd holds the flags for the 'withdraw' subcommand.
type withdrawCmd struct {
	date     string
	amount   decimal.Decimal
	currency string
	memo     string
	settles  string
}

func (*withdrawCmd) Name() string     { return "withdraw" }
func (*withdrawCmd) Synopsis() string { return "record a cash withdrawal from the portfolio" }
func (*withdrawCmd) Usage() string {
	return `pcs withdraw -d <date> -a <amount> -c <currency> [-m <memo>] [-settles <account>]

	Records a cash withdrawal from the portfolio's cash account.
`
}
func (c *withdrawCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.date, "d", portfolio.Today().String(), "Transaction date. See the user manual for supported date formats.")
	f.Var(DecimalVar(&c.amount, "0"), "a", "Amount of cash to withdraw")
	f.StringVar(&c.currency, "c", "EUR", "Currency of the withdrawal (e.g., USD, EUR)")
	f.StringVar(&c.memo, "m", "", "An optional rationale or note")
	f.StringVar(&c.settles, "settles", "", "Settle a counterparty account")
}
func (c *withdrawCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if c.amount.IsZero() {
		fmt.Fprintln(os.Stderr, "Error: -a flag is required.")
		return subcommands.ExitUsageError
	}
	day, err := portfolio.ParseDate(c.date) // Validate date format
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing date: %v\n", err)
		return subcommands.ExitUsageError
	}

	tx := portfolio.NewWithdraw(day, c.memo, portfolio.M(c.amount, c.currency))
	tx.Settles = c.settles
	_, status := handleTransaction(tx, f)
	return status
}

// --- Convert Command ---

// convertCmd holds the flags for the 'convert' subcommand.
type convertCmd struct {
	date         string
	fromCurrency string
	fromAmount   decimal.Decimal
	toCurrency   string
	toAmount     decimal.Decimal
	memo         string
}

func (*convertCmd) Name() string { return "convert" }
func (*convertCmd) Synopsis() string {
	return "converts cash from one currency to another within the portfolio"
}
func (*convertCmd) Usage() string {
	return `pcs convert -d <date> -fc <currency> -fa <amount> -tc <currency> -ta <amount> [-m <memo>]

	Records an internal cash conversion between two currency accounts.
	This does not represent a net portfolio deposit or withdrawal.
`
}

func (c *convertCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.date, "d", portfolio.Today().String(), "Transaction date. See the user manual for supported date formats.")
	f.StringVar(&c.fromCurrency, "fc", "", "Source currency code (e.g., USD)")
	f.Var(DecimalVar(&c.fromAmount, "0"), "fa", "Amount of cash to convert from the source currency")
	f.StringVar(&c.toCurrency, "tc", "", "Destination currency code (e.g., EUR")
	f.Var(DecimalVar(&c.toAmount, "0"), "ta", "Amount of cash received in the destination currency")
	f.StringVar(&c.memo, "m", "", "An optional rationale or note for the transaction")
}

func (c *convertCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if c.fromCurrency == "" || c.fromAmount.IsZero() || c.toCurrency == "" || c.toAmount.IsZero() {
		fmt.Fprintln(os.Stderr, "Error: -fc, -fa, -tc, and -ta flags are all required.")
		return subcommands.ExitUsageError
	}
	day, err := portfolio.ParseDate(c.date) // Validate date format
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing date: %v\n", err)
		return subcommands.ExitUsageError
	}

	tx := portfolio.NewConvert(day, c.memo, portfolio.M(c.fromAmount, c.fromCurrency), portfolio.M(c.toAmount, c.toCurrency))
	_, status := handleTransaction(tx, f)
	return status
}

// --- Accrue Command ---

// accrueCmd holds the flags for the 'accrue' subcommand.
type accrueCmd struct {
	date       string
	payable    string
	receivable string
	amount     decimal.Decimal
	currency   string
	memo       string
}

func (*accrueCmd) Name() string     { return "accrue" }
func (*accrueCmd) Synopsis() string { return "record a non-cash transaction with a counterparty" }
func (*accrueCmd) Usage() string {
	return `pcs accrue -d <date> (-payable <account> | -receivable <account>) -a <amount> -c <currency> [-m <memo>]

	Records a non-cash transaction with a counterparty, such as a loan or rent.
`
}

func (c *accrueCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.date, "d", portfolio.Today().String(), "Transaction date. See the user manual for supported date formats.")
	f.StringVar(&c.payable, "payable", "", "The counterparty account to which the user owes money")
	f.StringVar(&c.receivable, "receivable", "", "The counterparty account that owes money to the user")
	f.Var(DecimalVar(&c.amount, "0"), "a", "Amount of cash to accrue")
	f.StringVar(&c.currency, "c", "EUR", "Currency of the accrual (e.g., USD, EUR)")
	f.StringVar(&c.memo, "m", "", "An optional rationale or note")
}

func (c *accrueCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if (c.payable == "" && c.receivable == "") || (c.payable != "" && c.receivable != "") {
		fmt.Fprintln(os.Stderr, "Error: either -payable or -receivable must be specified.")
		return subcommands.ExitUsageError
	}
	if !c.amount.IsPositive() {
		fmt.Fprintln(os.Stderr, "Error: -a flag must be a positive amount.")
		return subcommands.ExitUsageError
	}
	day, err := portfolio.ParseDate(c.date) // Validate date format
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing date: %v\n", err)
		return subcommands.ExitUsageError
	}

	var account string
	var amount decimal.Decimal
	switch {
	case c.payable != "":
		account = c.payable
		amount = c.amount.Neg()
	case c.receivable != "":
		account = c.receivable
		amount = c.amount
	}

	tx := portfolio.NewAccrue(day, c.memo, account, portfolio.M(amount, c.currency))

	// Call handleTransaction and receive the validated transaction
	validatedTx, status := handleTransaction(tx, f)
	if status != subcommands.ExitSuccess {
		return status
	}

	// Check if it's an Accrue transaction and if a new account was created
	if accrueTx, ok := validatedTx.(portfolio.Accrue); ok {
		if accrueTx.Create {
			fmt.Printf("A new counterparty account '%s' has been created.\n", accrueTx.Counterparty)
		}
	}
	return subcommands.ExitSuccess
}

// --- Price Command ---

type priceCmd struct {
	date   string
	ticker string
	price  decimal.Decimal
}

func (*priceCmd) Name() string     { return "price" }
func (*priceCmd) Synopsis() string { return "records a price for a security on a specific date" }
func (*priceCmd) Usage() string {
	return `pcs price -s <ticker> -d <date> -p <price>

Records the price of a security on a given date in the ledger.
This is an alternative to storing prices in the market.jsonl file.
`
}

func (c *priceCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.date, "d", portfolio.Today().String(), "date of the price")
	f.StringVar(&c.ticker, "s", "", "security ticker")
	f.Var(DecimalVar(&c.price, "0"), "p", "price per share")
}

func (c *priceCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if c.ticker == "" {
		fmt.Fprintln(os.Stderr, "Error: security ticker (-s) is required")
		return subcommands.ExitUsageError
	}
	if c.price.IsZero() {
		fmt.Fprintln(os.Stderr, "Error: price (-p) is required and cannot be zero")
		return subcommands.ExitUsageError
	}

	date, err := portfolio.ParseDate(c.date)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid date: %v\n", err)
		return subcommands.ExitUsageError
	}

	tx := portfolio.NewUpdatePrice(date, c.ticker, portfolio.M(c.price, ""))
	_, status := handleTransaction(tx, f)
	return status
}

// --- Split Command ---

type splitCmd struct {
	date   string
	ticker string
	num    int64
	den    int64
}

func (*splitCmd) Name() string     { return "split" }
func (*splitCmd) Synopsis() string { return "records a stock split for a security" }
func (*splitCmd) Usage() string {
	return `pcs split -s <ticker> -d <date> -num <numerator> -den <denominator>

Records a stock split for a security in the ledger.
For a 2-for-1 split, use -num 2 -den 1.
For a 1-for-5 reverse split, use -num 1 -den 5.
`
}

func (c *splitCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.date, "d", portfolio.Today().String(), "effective date of the split")
	f.StringVar(&c.ticker, "s", "", "security ticker")
	f.Int64Var(&c.num, "num", 0, "numerator of the split ratio (e.g., 2 in a 2-for-1 split)")
	f.Int64Var(&c.den, "den", 1, "denominator of the split ratio (e.g., 1 in a 2-for-1 split)")
}

func (c *splitCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if c.ticker == "" {
		fmt.Fprintln(os.Stderr, "Error: security ticker (-s) is required")
		return subcommands.ExitUsageError
	}
	if c.num <= 0 {
		fmt.Fprintln(os.Stderr, "Error: numerator (-num) must be a positive integer")
		return subcommands.ExitUsageError
	}

	date, err := portfolio.ParseDate(c.date)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid date: %v\n", err)
		return subcommands.ExitUsageError
	}

	tx := portfolio.NewSplit(date, c.ticker, c.num, c.den)
	_, status := handleTransaction(tx, f)
	return status
}

// handleTransaction processes a transaction by validating it against the current
// accounting system and then encoding it to the ledger file. It also manages
// the CLI feedback, printing errors or a success message and returning the
// appropriate exit status.
//
// This function also applies "quick fixes" during validation, such as resolving
// "sell all" quantities. The returned `portfolio.Transaction` is the validated
// and potentially modified transaction.
//
func handleTransaction(tx portfolio.Transaction, f *flag.FlagSet) (portfolio.Transaction, subcommands.ExitStatus) {
	validatedTx, err := EncodeTransaction(tx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		f.Usage()
		return nil, subcommands.ExitUsageError
	}

	fmt.Printf("Successfully appended transaction to %s\n", *ledgerFile)
	return validatedTx, subcommands.ExitSuccess
}

// declareCmd holds the flags for the 'declare' subcommand.
type declareCmd struct {
	ticker   string
	id       string
	currency string
	date     string
	memo     string
}

func (*declareCmd) Name() string     { return "declare" }
func (*declareCmd) Synopsis() string { return "declare a new security" }
func (*declareCmd) Usage() string {
	return `pcs declare -s <ticker> -id <security-id> -c <currency> [-d <date>] [-m <memo>]

	Declares a security, creating a mapping from a ledger-internal ticker to a
	globally unique security ID and its currency. This declaration is required
	before using the ticker in any transaction.
`
}

func (c *declareCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.ticker, "s", "", "Ledger-internal ticker to define (e.g., 'MY_AAPL')")
	f.StringVar(&c.id, "id", "", "Full, unique security ID (e.g., 'US0378331005.XNAS')")
	f.StringVar(&c.currency, "c", "", "The currency of the security (e.g., 'USD')")
	f.StringVar(&c.date, "d", portfolio.Today().String(), "Transaction date. See the user manual for supported date formats.")
	f.StringVar(&c.memo, "m", "", "An optional rationale or note for the transaction")
}

// GenerateCommand generates the 'pcs add-security' command string with the given parameters.
// This function is primarily used by the 'search-security' command to construct the
// command needed to add a selected security. It's crucial to keep the flags and
// their names in sync with the 'add-security' command's SetFlags method.
func (c *declareCmd) GenerateCommand(ticker, id, currency string) string {
	// TODO: use const for flag names.
	return fmt.Sprintf("pcs declare -%s='%s' -%s='%s' -%s='%s'", "s", ticker, "id", id, "c", currency)
}

func (c *declareCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if c.ticker == "" || c.id == "" || c.currency == "" {
		fmt.Fprintln(os.Stderr, "Error: -s, -id, and -c flags are all required.")
		return subcommands.ExitUsageError
	}

	day, err := portfolio.ParseDate(c.date)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing date: %v\n", err)
		return subcommands.ExitUsageError
	}
	id, err := portfolio.ParseID(c.id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid security ID: %v\n", err)
		return subcommands.ExitUsageError
	}
	tx := portfolio.NewDeclare(day, c.memo, c.ticker, id, c.currency)
	_, status := handleTransaction(tx, f)
	return status
}
