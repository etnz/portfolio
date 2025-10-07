package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/etnz/portfolio"
	"github.com/etnz/portfolio/renderer"
	"github.com/google/subcommands"
)

type monthlyCmd struct {
	review reviewCmd
}

func (*monthlyCmd) Name() string     { return "monthly" }
func (*monthlyCmd) Synopsis() string { return "display a monthly portfolio performance report" }
func (*monthlyCmd) Usage() string {
	return `pcs monthly [-d <date>] [-l <ledger>] [-method <method>]

  Displays a monthly performance review of the portfolio.
`
}

func (c *monthlyCmd) SetFlags(f *flag.FlagSet) {
	c.review.period = "month"
	f.StringVar(&c.review.date, "d", "", "End date for the report period (defaults to today)")
	f.StringVar(&c.review.method, "method", "fifo", "Cost basis method (average, fifo)")
	f.StringVar(&c.review.ledgerFile, "l", "", "Ledger to report on. Defaults to the only ledger if one exists.")
}

func (c *monthlyCmd) Execute(ctx context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {
	if err := c.review.init(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return subcommands.ExitUsageError
	}
	for _, ledger := range c.review.ledgers {
		review := c.review.generateReview(ledger)
		c.render(review, c.review.parsedMethod)
	}
	return subcommands.ExitSuccess
}

func (c *monthlyCmd) render(review *portfolio.Review, method portfolio.CostBasisMethod) {
	md := renderer.PeriodicMarkdown(review, method)
	printMarkdown(md)
}
