package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/etnz/portfolio/date"
	"github.com/etnz/portfolio"
	"github.com/google/subcommands"
)

// appendTransaction appends a transaction to the specified portfolio file.
func appendTransaction(filename string, tx portfolio.Transaction) subcommands.ExitStatus {
	// Open the file in append mode, creating it if it doesn't exist.
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening portfolio file %q: %v\n", filename, err)
		return subcommands.ExitFailure
	}
	defer f.Close()

	if err := portfolio.EncodeTransaction(f, tx); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing to portfolio file %q: %v\n", filename, err)
		return subcommands.ExitFailure
	}

	fmt.Printf("Successfully appended transaction to %s\n", filename)
	return subcommands.ExitSuccess
}

// --- Buy Command ---

type buyCmd struct {
	date     string
	security string
	quantity float64
	price    float64
	memo     string
}

func (*buyCmd) Name() string     { return "buy" }
func (*buyCmd) Synopsis() string { return "purchase shares to open or add to a position" }
func (*buyCmd) Usage() string {
	return `buy -d <date> -s <security> -q <quantity> -p <price> [-m <memo>]

  Purchases shares of a security. The total cost is debited from the cash account.
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
	if c.security == "" || c.quantity <= 0 || c.price <= 0 {
		f.Usage()
		return subcommands.ExitUsageError
	}
	day, err := date.Parse(c.date) // Validate date format
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing date: %v\n", err)
		return subcommands.ExitUsageError
	}

	tx := portfolio.Buy{
		Base:     portfolio.Base{Command: "buy", Date: day, Memo: c.memo},
		Security: c.security,
		Quantity: c.quantity,
		Price:    c.price,
	}
	return appendTransaction(*portfolioFile, tx)
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
func (*sellCmd) Synopsis() string { return "sell shares to trim or close a position" }
func (*sellCmd) Usage() string {
	return `sell -d <date> -s <security> -q <quantity> -p <price> [-m <memo>]

  Sells shares of a security. The proceeds are credited to the cash account.
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
	if c.security == "" || c.quantity <= 0 || c.price < 0 {
		f.Usage()
		return subcommands.ExitUsageError
	}

	day, err := date.Parse(c.date) // Validate date format
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing date: %v\n", err)
		return subcommands.ExitUsageError
	}
	tx := portfolio.Sell{
		Base:     portfolio.Base{Command: "sell", Date: day, Memo: c.memo},
		Security: c.security,
		Quantity: c.quantity,
		Price:    c.price,
	}
	return appendTransaction(*portfolioFile, tx)
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

  Records a dividend payment. The amount is credited to the cash account.
`
}
func (c *dividendCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.date, "d", date.Today().String(), "Transaction date (YYYY-MM-DD)")
	f.StringVar(&c.security, "s", "", "Security ticker receiving the dividend")
	f.Float64Var(&c.amount, "a", 0, "Total dividend amount received")
	f.StringVar(&c.memo, "m", "", "An optional rationale or note")
}
func (c *dividendCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if c.security == "" || c.amount <= 0 {
		f.Usage()
		return subcommands.ExitUsageError
	}

	day, err := date.Parse(c.date) // Validate date format
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing date: %v\n", err)
		return subcommands.ExitUsageError
	}

	tx := portfolio.Dividend{
		Base:     portfolio.Base{Command: "dividend", Date: day, Memo: c.memo},
		Security: c.security,
		Amount:   c.amount,
	}
	return appendTransaction(*portfolioFile, tx)
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
	if c.amount <= 0 {
		f.Usage()
		return subcommands.ExitUsageError
	}

	day, err := date.Parse(c.date) // Validate date format
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing date: %v\n", err)
		return subcommands.ExitUsageError
	}

	tx := portfolio.Deposit{
		Base:     portfolio.Base{Command: "deposit", Date: day, Memo: c.memo},
		Amount:   c.amount,
		Currency: c.currency,
	}
	return appendTransaction(*portfolioFile, tx)
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
	f.StringVar(&c.date, "d", time.Now().Format("2006-01-02"), "Transaction date (YYYY-MM-DD)")
	f.Float64Var(&c.amount, "a", 0, "Amount of cash to withdraw")
	f.StringVar(&c.currency, "c", "EUR", "Currency of the withdrawal (e.g., USD, EUR)")
	f.StringVar(&c.memo, "m", "", "An optional rationale or note")
}
func (c *withdrawCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if c.amount <= 0 {
		f.Usage()
		return subcommands.ExitUsageError
	}
	day, err := date.Parse(c.date) // Validate date format
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing date: %v\n", err)
		return subcommands.ExitUsageError
	}

	tx := portfolio.Withdraw{
		Base:     portfolio.Base{Command: "withdraw", Date: day, Memo: c.memo},
		Amount:   c.amount,
		Currency: c.currency,
	}
	return appendTransaction(*portfolioFile, tx)
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
	if c.fromCurrency == "" || c.toCurrency == "" || c.fromAmount <= 0 || c.toAmount <= 0 {
		f.Usage()
		return subcommands.ExitUsageError
	}
	if c.fromCurrency == c.toCurrency {
		fmt.Fprintln(os.Stderr, "Error: from and to currencies cannot be the same.")
		return subcommands.ExitUsageError
	}

	day, err := date.Parse(c.date) // Validate date format
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing date: %v\n", err)
		return subcommands.ExitUsageError
	}

	tx := portfolio.Convert{
		Base:         portfolio.Base{Command: "convert", Date: day, Memo: c.memo},
		FromCurrency: c.fromCurrency,
		FromAmount:   c.fromAmount,
		ToCurrency:   c.toCurrency,
		ToAmount:     c.toAmount,
	}
	return appendTransaction(*portfolioFile, tx)
}
