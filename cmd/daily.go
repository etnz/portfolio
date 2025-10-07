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
	review reviewCmd
	watch  int
}

func (*dailyCmd) Name() string     { return "daily" }
func (*dailyCmd) Synopsis() string { return "display a daily portfolio performance report" }
func (*dailyCmd) Usage() string {
	return `pcs daily [-d <date>] [-l <ledger>] [-method <method>] [-w n] 

  Displays a summary of the portfolio for a single day.
`
}

func (c *dailyCmd) SetFlags(f *flag.FlagSet) {
	c.review.period = "day"
	f.StringVar(&c.review.date, "d", "", "Date for the report (defaults to today)")
	f.StringVar(&c.review.method, "method", "fifo", "Cost basis method (average, fifo)")
	f.StringVar(&c.review.ledgerFile, "l", "", "Ledger to report on. Defaults to the only ledger if one exists.")
	f.IntVar(&c.watch, "w", 0, "run every n seconds")
}

func (c *dailyCmd) Execute(ctx context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {
	if err := c.review.init(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return subcommands.ExitUsageError
	}
	for {
		for _, ledger := range c.review.ledgers {
			review := c.review.generateReview(ledger)
			c.render(review, c.review.parsedMethod)
		}

		if c.watch > 0 {
			time.Sleep(time.Duration(c.watch) * time.Second)
		} else {
			break
		}
	}
	return subcommands.ExitSuccess
}

func (c *dailyCmd) render(review *portfolio.Review, method portfolio.CostBasisMethod) {
	md := renderer.PeriodicMarkdown(review, method)
	if c.watch > 0 {
		fmt.Println("\033[2J")
	}
	printMarkdown(md)

}
