package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/etnz/portfolio"
	"github.com/etnz/portfolio/date"
	"github.com/etnz/portfolio/renderer"
	"github.com/google/subcommands"
)

// holdingCmd holds the flags for the 'holding' subcommand.
type holdingCmd struct {
	date     string
	currency string
	update   bool
}

func (*holdingCmd) Name() string     { return "holding" }
func (*holdingCmd) Synopsis() string { return "display detailed holdings for a specific date" }
func (*holdingCmd) Usage() string {
	return `pcs holding [-d <date>] [-c <currency>] [-u]

  Displays the portfolio holdings (securities and cash) on a given date.
`
}

func (c *holdingCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.date, "d", date.Today().String(), "Date for the holdings report. See the user manual for supported date formats.")
	f.StringVar(&c.currency, "c", "EUR", "Reporting currency for market values")
	f.BoolVar(&c.update, "u", false, "update with latest intraday prices before calculating the report")

}

func (c *holdingCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	on, err := date.Parse(c.date)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing date: %v\n", err)
		return subcommands.ExitUsageError
	}

	if on.IsToday() {
		c.update = true
	}

	market, err := DecodeMarketData()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading securities: %v\n", err)
		return subcommands.ExitFailure
	}

	if c.update {
		err := market.UpdateIntraday()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error updating intraday prices: %v\n", err)
			return subcommands.ExitFailure
		}
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

	report, err := as.NewHoldingReport(on)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating holding report: %v\n", err)
		return subcommands.ExitFailure
	}

	printMarkdown(renderer.HoldingMarkdown(report))

	return subcommands.ExitSuccess
}
