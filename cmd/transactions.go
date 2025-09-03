package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/etnz/portfolio"
	"github.com/etnz/portfolio/date"
	"github.com/google/subcommands"
)

// --- Buy Command ---

// buyCmd holds the flags for the 'buy' subcommand.
type buyCmd struct {
	date     string
	security string
	quantity float64
	amount   float64
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
	f.StringVar(&c.date, "d", date.Today().String(), "Transaction date. See the user manual for supported date formats.")
	f.StringVar(&c.security, "s", "", "Security ticker")
	f.Float64Var(&c.quantity, "q", 0, "Number of shares")
	f.Float64Var(&c.amount, "a", 0, "Total amount paid for the shares")
	f.StringVar(&c.memo, "m", "", "An optional rationale or note for the transaction")
}

func (c *buyCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if c.security == "" || c.quantity == 0 || c.amount == 0 {
		fmt.Fprintln(os.Stderr, "Error: -s, -q, and -a flags are all required.")
		return subcommands.ExitUsageError
	}
	day, err := date.Parse(c.date) // Validate date format
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing date: %v\n", err)
		return subcommands.ExitUsageError
	}

	tx := portfolio.NewBuy(day, c.memo, c.security, c.quantity, c.amount)
	_, status := handleTransaction(tx, f)
	return status
}

// --- Sell Command ---

// sellCmd holds the flags for the 'sell' subcommand.
type sellCmd struct {
	date     string
	security string
	quantity float64
	amount   float64
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
	f.StringVar(&c.date, "d", date.Today().String(), "Transaction date. See the user manual for supported date formats.")
	f.StringVar(&c.security, "s", "", "Security ticker")
	f.Float64Var(&c.quantity, "q", 0, "Number of shares, if missing all shares are sold")
	f.Float64Var(&c.amount, "a", 0, "Total amount received for the shares")
	f.StringVar(&c.memo, "m", "", "An optional rationale or note for the transaction")
}
func (c *sellCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if c.security == "" || c.amount == 0 {
		fmt.Fprintln(os.Stderr, "Error: -s and -a flags are required.")
		return subcommands.ExitUsageError
	}
	day, err := date.Parse(c.date) // Validate date format
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing date: %v\n", err)
		return subcommands.ExitUsageError
	}
	tx := portfolio.NewSell(day, c.memo, c.security, c.quantity, c.amount)
	_, status := handleTransaction(tx, f)
	return status
}

// --- Dividend Command ---

// dividendCmd holds the flags for the 'dividend' subcommand.
type dividendCmd struct {
	date     string
	security string
	amount   float64
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
	f.StringVar(&c.date, "d", date.Today().String(), "Transaction date. See the user manual for supported date formats.")
	f.StringVar(&c.security, "s", "", "Security ticker receiving the dividend")
	f.Float64Var(&c.amount, "a", 0, "Total dividend amount received")
	f.StringVar(&c.memo, "m", "", "An optional rationale or note")
}
func (c *dividendCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if c.security == "" || c.amount == 0 {
		fmt.Fprintln(os.Stderr, "Error: -s and -a flags are required.")
		return subcommands.ExitUsageError
	}
	day, err := date.Parse(c.date) // Validate date format
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing date: %v\n", err)
		return subcommands.ExitUsageError
	}

	tx := portfolio.NewDividend(day, c.memo, c.security, c.amount)
	_, status := handleTransaction(tx, f)
	return status
}

// --- Deposit Command ---

// depositCmd holds the flags for the 'deposit' subcommand.
type depositCmd struct {
	date     string
	amount   float64
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
	f.StringVar(&c.date, "d", date.Today().String(), "Transaction date. See the user manual for supported date formats.")
	f.Float64Var(&c.amount, "a", 0, "Amount of cash to deposit")
	f.StringVar(&c.currency, "c", "EUR", "Currency of the deposit (e.g., USD, EUR). Cash is kept in that currency")
	f.StringVar(&c.memo, "m", "", "An optional rationale or note")
	f.StringVar(&c.settles, "settles", "", "Settle a counterparty account")
}
func (c *depositCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if c.amount == 0 {
		fmt.Fprintln(os.Stderr, "Error: -a flag is required.")
		return subcommands.ExitUsageError
	}
	day, err := date.Parse(c.date) // Validate date format
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing date: %v\n", err)
		return subcommands.ExitUsageError
	}

	tx := portfolio.NewDeposit(day, c.memo, c.currency, c.amount, c.settles)
	_, status := handleTransaction(tx, f)
	return status
}

// --- Withdraw Command ---

// withdrawCmd holds the flags for the 'withdraw' subcommand.
type withdrawCmd struct {
	date     string
	amount   float64
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
	f.StringVar(&c.date, "d", date.Today().String(), "Transaction date. See the user manual for supported date formats.")
	f.Float64Var(&c.amount, "a", 0, "Amount of cash to withdraw")
	f.StringVar(&c.currency, "c", "EUR", "Currency of the withdrawal (e.g., USD, EUR)")
	f.StringVar(&c.memo, "m", "", "An optional rationale or note")
	f.StringVar(&c.settles, "settles", "", "Settle a counterparty account")
}
func (c *withdrawCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if c.amount == 0 {
		fmt.Fprintln(os.Stderr, "Error: -a flag is required.")
		return subcommands.ExitUsageError
	}
	day, err := date.Parse(c.date) // Validate date format
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing date: %v\n", err)
		return subcommands.ExitUsageError
	}

	tx := portfolio.NewWithdraw(day, c.memo, c.currency, c.amount)
	tx.Settles = c.settles
	_, status := handleTransaction(tx, f)
	return status
}

// --- Convert Command ---

// convertCmd holds the flags for the 'convert' subcommand.
type convertCmd struct {
	date         string
	fromCurrency string
	fromAmount   float64
	toCurrency   string
	toAmount     float64
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
	f.StringVar(&c.date, "d", date.Today().String(), "Transaction date. See the user manual for supported date formats.")
	f.StringVar(&c.fromCurrency, "fc", "", "Source currency code (e.g., USD)")
	f.Float64Var(&c.fromAmount, "fa", 0, "Amount of cash to convert from the source currency")
	f.StringVar(&c.toCurrency, "tc", "", "Destination currency code (e.g., EUR")
	f.Float64Var(&c.toAmount, "ta", 0, "Amount of cash received in the destination currency")
	f.StringVar(&c.memo, "m", "", "An optional rationale or note for the transaction")
}

func (c *convertCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if c.fromCurrency == "" || c.fromAmount == 0 || c.toCurrency == "" || c.toAmount == 0 {
		fmt.Fprintln(os.Stderr, "Error: -fc, -fa, -tc, and -ta flags are all required.")
		return subcommands.ExitUsageError
	}
	day, err := date.Parse(c.date) // Validate date format
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing date: %v\n", err)
		return subcommands.ExitUsageError
	}

	tx := portfolio.NewConvert(day, c.memo, c.fromCurrency, c.fromAmount, c.toCurrency, c.toAmount)
	_, status := handleTransaction(tx, f)
	return status
}

// --- Accrue Command ---

// accrueCmd holds the flags for the 'accrue' subcommand.
type accrueCmd struct {
	date       string
	payable    string
	receivable string
	amount     float64
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
	f.StringVar(&c.date, "d", date.Today().String(), "Transaction date. See the user manual for supported date formats.")
	f.StringVar(&c.payable, "payable", "", "The counterparty account to which the user owes money")
	f.StringVar(&c.receivable, "receivable", "", "The counterparty account that owes money to the user")
	f.Float64Var(&c.amount, "a", 0, "Amount of cash to accrue")
	f.StringVar(&c.currency, "c", "EUR", "Currency of the accrual (e.g., USD, EUR)")
	f.StringVar(&c.memo, "m", "", "An optional rationale or note")
}

func (c *accrueCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if (c.payable == "" && c.receivable == "") || (c.payable != "" && c.receivable != "") {
		fmt.Fprintln(os.Stderr, "Error: either -payable or -receivable must be specified.")
		return subcommands.ExitUsageError
	}
	if c.amount <= 0 {
		fmt.Fprintln(os.Stderr, "Error: -a flag must be a positive amount.")
		return subcommands.ExitUsageError
	}
	day, err := date.Parse(c.date) // Validate date format
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing date: %v\n", err)
		return subcommands.ExitUsageError
	}

	var account string
	var amount float64
	switch {
	case c.payable != "":
		account = c.payable
		amount = -c.amount
	case c.receivable != "":
		account = c.receivable
		amount = c.amount
	}

	tx := portfolio.NewAccrue(day, c.memo, account, amount, c.currency)

	// Call handleTransaction and receive the validated transaction
	validatedTx, status := handleTransaction(tx, f)
	if status != subcommands.ExitSuccess {
		return status
	}

	// Check if it's an Accrue transaction and if a new account was created
	if accrueTx, ok := validatedTx.(portfolio.Accrue); ok {
		if accrueTx.Create {
			fmt.Printf("A new counterparty account '%s' will be created.\n", accrueTx.Counterparty)
		}
	}

	fmt.Printf("Successfully appended transaction to %s\n", *ledgerFile)
	return subcommands.ExitSuccess
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
// TODO(etnz): Make this function more generic to handle different types of
// encoding and feedback, possibly by passing in an interface for output.
// Create a GitHub issue for this refactoring.
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
	f.StringVar(&c.date, "d", date.Today().String(), "Transaction date. See the user manual for supported date formats.")
	f.StringVar(&c.memo, "m", "", "An optional rationale or note for the transaction")
}

func (c *declareCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if c.ticker == "" || c.id == "" || c.currency == "" {
		fmt.Fprintln(os.Stderr, "Error: -s, -id, and -c flags are all required.")
		return subcommands.ExitUsageError
	}

	day, err := date.Parse(c.date)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing date: %v\n", err)
		return subcommands.ExitUsageError
	}
	tx := portfolio.NewDeclaration(day, c.memo, c.ticker, c.id, c.currency)
	_, status := handleTransaction(tx, f)
	return status
}
