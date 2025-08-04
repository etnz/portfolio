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

type summaryCmd struct {
	date     string
	currency string
}

func (*summaryCmd) Name() string     { return "summary" }
func (*summaryCmd) Synopsis() string { return "display portfolio summary dashboard" }
func (*summaryCmd) Usage() string {
	return `summary [-d <date>] [-c <currency>]

  Displays a summary of the portfolio, including total market value.
`
}

func (c *summaryCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.date, "d", date.Today().String(), "Date for the summary (YYYY-MM-DD)")
	f.StringVar(&c.currency, "c", "EUR", "Reporting currency for the summary")
}

func (c *summaryCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	on, err := date.Parse(c.date)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing date: %v\n", err)
		return subcommands.ExitUsageError
	}

	market, err := DecodeSecurities()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading securities: %v\n", err)
		return subcommands.ExitFailure
	}

	ledger, err := DecodeLedger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading ledger: %v\n", err)
		return subcommands.ExitFailure
	}

	as, err := portfolio.NewAccountingSystem(ledger, market, c.currency)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating accounting system: %v\n", err)
		return subcommands.ExitFailure
	}

	totalValue, err := as.TotalMarketValue(on)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error calculating portfolio value: %v\n", err)
		return subcommands.ExitFailure
	}

	fmt.Printf("Portfolio Summary on %s\n", on)
	fmt.Println("---------------------------------")
	fmt.Printf("Total Market Value: %.2f %s\n", totalValue, c.currency)

	return subcommands.ExitSuccess
}
