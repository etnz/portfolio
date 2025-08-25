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

type buyCmd struct {
	date     string
	security string
	quantity float64
	price    float64
	memo     string
}

func (*buyCmd) Name() string     { return "buy" }
func (*buyCmd) Synopsis() string { return "record the purchase of a security" }
func (*buyCmd) Usage() string {
	return `buy -d <date> -s <security> -q <quantity> -p <price> [-m <memo>]

  Purchases shares of a security. The total cost is debited from the cash account in the security's currency.
`
}

func (c *buyCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.date, "d", date.Today().String(), "Transaction date (YYYY-MM-DD)")
	f.StringVar(&c.security, "s", "", "Security ticker")
	f.Float64Var(&c.quantity, "q", 0, "Number of shares")
	f.Float64Var(&c.price, "p", 0, "Price per share")
	f.StringVar(&c.memo, "m", "", "An optional rationale or note for the transaction")
}

func (c *buyCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	day, err := date.Parse(c.date) // Validate date format
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing date: %v\n", err)
		return subcommands.ExitUsageError
	}

	tx := portfolio.NewBuy(day, c.memo, c.security, c.quantity, c.price)
	return handleTransaction(tx, f)
}

// --- Sell Command ---

type sellCmd struct {
	date     string
	security string
	quantity float64
	price    float64
	memo     string
}

func (*sellCmd) Name() string     { return "sell" }
func (*sellCmd) Synopsis() string { return "record the sale of a security" }
func (*sellCmd) Usage() string {
	return `sell -d <date> -s <security> -q <quantity> -p <price> [-m <memo>]

  Sells shares of a security. The proceeds are credited to the cash account in the security's currency.
`
}
func (c *sellCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.date, "d", date.Today().String(), "Transaction date (YYYY-MM-DD)")
	f.StringVar(&c.security, "s", "", "Security ticker")
	f.Float64Var(&c.quantity, "q", 0, "Number of shares, if missing all shares are sold")
	f.Float64Var(&c.price, "p", 0, "Price per share")
	f.StringVar(&c.memo, "m", "", "An optional rationale or note for the transaction")
}
func (c *sellCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	day, err := date.Parse(c.date) // Validate date format
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing date: %v\n", err)
		return subcommands.ExitUsageError
	}
	tx := portfolio.NewSell(day, c.memo, c.security, c.quantity, c.price)
	return handleTransaction(tx, f)
}

// --- Dividend Command ---

type dividendCmd struct {
	date     string
	security string
	amount   float64
	memo     string
}

func (*dividendCmd) Name() string     { return "dividend" }
func (*dividendCmd) Synopsis() string { return "record a dividend payment for a security" }
func (*dividendCmd) Usage() string {
	return `dividend -d <date> -s <security> -a <amount> [-m <memo>]

  Records a dividend payment. The amount is credited to the cash account in the security's currency.
`
}
func (c *dividendCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.date, "d", date.Today().String(), "Transaction date (YYYY-MM-DD)")
	f.StringVar(&c.security, "s", "", "Security ticker receiving the dividend")
	f.Float64Var(&c.amount, "a", 0, "Total dividend amount received")
	f.StringVar(&c.memo, "m", "", "An optional rationale or note")
}
func (c *dividendCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	day, err := date.Parse(c.date) // Validate date format
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing date: %v\n", err)
		return subcommands.ExitUsageError
	}

	tx := portfolio.NewDividend(day, c.memo, c.security, c.amount)
	return handleTransaction(tx, f)
}

// --- Deposit Command ---

type depositCmd struct {
	date     string
	amount   float64
	currency string
	memo     string
}

func (*depositCmd) Name() string     { return "deposit" }
func (*depositCmd) Synopsis() string { return "record a cash deposit into the portfolio" }
func (*depositCmd) Usage() string {
	return `deposit -d <date> -a <amount> -c <currency> [-m <memo>]

  Records a cash deposit into the portfolio's cash account.
`
}
func (c *depositCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.date, "d", date.Today().String(), "Transaction date (YYYY-MM-DD)")
	f.Float64Var(&c.amount, "a", 0, "Amount of cash to deposit")
	f.StringVar(&c.currency, "c", "EUR", "Currency of the deposit (e.g., USD, EUR). Cash is kept in that currency")
	f.StringVar(&c.memo, "m", "", "An optional rationale or note")
}
func (c *depositCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	day, err := date.Parse(c.date) // Validate date format
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing date: %v\n", err)
		return subcommands.ExitUsageError
	}

	tx := portfolio.NewDeposit(day, c.memo, c.currency, c.amount)
	return handleTransaction(tx, f)
}

// --- Withdraw Command ---

type withdrawCmd struct {
	date     string
	amount   float64
	currency string
	memo     string
}

func (*withdrawCmd) Name() string     { return "withdraw" }
func (*withdrawCmd) Synopsis() string { return "record a cash withdrawal from the portfolio" }
func (*withdrawCmd) Usage() string {
	return `withdraw -d <date> -a <amount> -c <currency> [-m <memo>]

  Records a cash withdrawal from the portfolio's cash account.
`
}
func (c *withdrawCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.date, "d", date.Today().String(), "Transaction date (YYYY-MM-DD)")
	f.Float64Var(&c.amount, "a", 0, "Amount of cash to withdraw")
	f.StringVar(&c.currency, "c", "EUR", "Currency of the withdrawal (e.g., USD, EUR)")
	f.StringVar(&c.memo, "m", "", "An optional rationale or note")
}
func (c *withdrawCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	day, err := date.Parse(c.date) // Validate date format
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing date: %v\n", err)
		return subcommands.ExitUsageError
	}

	tx := portfolio.NewWithdraw(day, c.memo, c.currency, c.amount)
	return handleTransaction(tx, f)
}

// --- Convert Command ---

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
	return `convert -d <date> -from-c <currency> -from-a <amount> -to-c <currency> -to-a <amount> [-m <memo>]

  Records an internal cash conversion between two currency accounts.
  This does not represent a net portfolio deposit or withdrawal.
`
}

func (c *convertCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.date, "d", date.Today().String(), "Transaction date (YYYY-MM-DD)")
	f.StringVar(&c.fromCurrency, "from-c", "", "Source currency code (e.g., USD)")
	f.Float64Var(&c.fromAmount, "from-a", 0, "Amount of cash to convert from the source currency")
	f.StringVar(&c.toCurrency, "to-c", "", "Destination currency code (e.g., EUR)")
	f.Float64Var(&c.toAmount, "to-a", 0, "Amount of cash received in the destination currency")
	f.StringVar(&c.memo, "m", "", "An optional rationale or note for the transaction")
}

func (c *convertCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	day, err := date.Parse(c.date) // Validate date format
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing date: %v\n", err)
		return subcommands.ExitUsageError
	}

	tx := portfolio.NewConvert(day, c.memo, c.fromCurrency, c.fromAmount, c.toCurrency, c.toAmount)
	return handleTransaction(tx, f)
}

// handleTransaction calls the core EncodeTransaction function and manages the
// CLI feedback, printing errors or a success message and returning the
// appropriate exit status.
func handleTransaction(tx portfolio.Transaction, f *flag.FlagSet) subcommands.ExitStatus {
	if err := EncodeTransaction(tx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		f.Usage()
		return subcommands.ExitUsageError
	}

	fmt.Printf("Successfully appended transaction to %s\n", *ledgerFile)
	return subcommands.ExitSuccess
}

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
	return `pcs declare -ticker <ticker> -id <security-id> -currency <currency> [-d <date>] [-m <memo>]

  Declares a security, creating a mapping from a ledger-internal ticker to a
  globally unique security ID and its currency. This declaration is required
  before using the ticker in any transaction.
`
}

func (c *declareCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.ticker, "ticker", "", "Ledger-internal ticker to define (e.g., 'MY_AAPL')")
	f.StringVar(&c.id, "id", "", "Full, unique security ID (e.g., 'US0378331005.XNAS')")
	f.StringVar(&c.currency, "currency", "", "The currency of the security (e.g., 'USD')")
	f.StringVar(&c.date, "d", date.Today().String(), "Transaction date (YYYY-MM-DD)")
	f.StringVar(&c.memo, "m", "", "An optional rationale or note for the transaction")
}

func (c *declareCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if c.ticker == "" || c.id == "" || c.currency == "" {
		fmt.Fprintln(os.Stderr, "Error: -ticker, -id, and -currency flags are all required.")
		return subcommands.ExitUsageError
	}

	day, err := date.Parse(c.date)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing date: %v\n", err)
		return subcommands.ExitUsageError
	}
	tx := portfolio.NewDeclaration(day, c.memo, c.ticker, c.id, c.currency)

	return handleTransaction(tx, f)
}
