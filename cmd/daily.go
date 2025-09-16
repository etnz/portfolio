package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/etnz/portfolio"
	"github.com/etnz/portfolio/renderer"
	"github.com/google/subcommands"
)

// dailyCmd holds the flags for the 'daily' subcommand.
type dailyCmd struct {
	date   string
	update bool
	watch  int
}

func (*dailyCmd) Name() string     { return "daily" }
func (*dailyCmd) Synopsis() string { return "display a daily portfolio performance report" }
func (*dailyCmd) Usage() string {
	return `pcs daily [-d <date>] [-c <currency>] [-u] [-w n] 

  Displays a summary of the portfolio for a single day.
`
}

func (c *dailyCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.date, "d", portfolio.Today().String(), "Date for the report. See the user manual for supported date formats.")
	f.IntVar(&c.watch, "w", 0, "run every n seconds")
}

func (c *dailyCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	on, err := portfolio.ParseDate(c.date)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing date: %v\n", err)
		return subcommands.ExitUsageError
	}

	if on.IsToday() {
		c.update = true
	}

	for {
		ledger, err := DecodeLedger()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading ledger: %v\n", err)
			return subcommands.ExitFailure
		}

		if c.update {
			err := ledger.UpdateIntraday()
			if err != nil {
				// This is not a fatal error, we can continue with stale prices.
				//fmt.Fprintf(os.Stderr, "Warning: could not update some intraday prices: %v\n", err)
			}
		}

		// TODO: handle report currency
		review, err := ledger.NewReview(portfolio.NewRange(on, on))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error calculating daily report: %v\n", err)
			return subcommands.ExitFailure
		}

		md := renderer.DailyMarkdown(review, portfolio.AverageCost)

		if c.watch > 0 {
			fmt.Println("\033[2J")
		}
		printMarkdown(md)
		if c.watch > 0 {
			time.Sleep(time.Duration(c.watch) * time.Second)
		} else {
			return subcommands.ExitSuccess
		}
	}
}
