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

type declareCmd struct {
	ticker   string
	id       string
	currency string
	date     string
	memo     string
}

func (*declareCmd) Name() string     { return "declare-security" }
func (*declareCmd) Synopsis() string { return "declare a security for use within the ledger" }
func (*declareCmd) Usage() string {
	return `declare-security -ticker <ticker> -id <security-id> -currency <currency> [-d <date>] [-m <memo>]

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
