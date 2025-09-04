package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/etnz/portfolio"
	"github.com/etnz/portfolio/date"
	"github.com/etnz/portfolio/renderer"
	"github.com/google/subcommands"
)

// dailyCmd holds the flags for the 'daily' subcommand.
type dailyCmd struct {
	date     string
	currency string
	update   bool
	watch    int
}

func (*dailyCmd) Name() string     { return "daily" }
func (*dailyCmd) Synopsis() string { return "display a daily portfolio performance report" }
func (*dailyCmd) Usage() string {
	return `pcs daily [-d <date>] [-c <currency>] [-u] [-w n] 

  Displays a summary of the portfolio for a single day.
`
}

func (c *dailyCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.date, "d", date.Today().String(), "Date for the report. See the user manual for supported date formats.")
	f.StringVar(&c.currency, "c", "EUR", "Reporting currency for the report")
	f.BoolVar(&c.update, "u", false, "update with latest intraday prices before calculating the report")
	f.IntVar(&c.watch, "w", 0, "run every n seconds")
}

func (c *dailyCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	on, err := date.Parse(c.date)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing date: %v\n", err)
		return subcommands.ExitUsageError
	}

	if on.IsToday() {
		c.update = true
	}

	for c.watch > 0 {
		market, err := DecodeMarketData()
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

		if c.update {
			err := market.UpdateIntraday()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error updating intraday prices: %v\n", err)
				return subcommands.ExitFailure
			}
		}

		report, err := as.NewDailyReport(on)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error calculating daily report: %v\n", err)
			return subcommands.ExitFailure
		}

		if c.watch > 0 {
			fmt.Println("\033[2J")
		}

		printMarkdown(renderer.DailyMarkdown(report))

		if c.watch > 0 {
			time.Sleep(time.Duration(c.watch) * time.Second)
		}
	}

	return subcommands.ExitSuccess
}
