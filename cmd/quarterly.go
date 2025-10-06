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

type quarterlyCmd struct {
	review reviewCmd
}

func (*quarterlyCmd) Name() string     { return "quarterly" }
func (*quarterlyCmd) Synopsis() string { return "display a quarterly portfolio performance report" }
func (*quarterlyCmd) Usage() string {
	return `pcs quarterly [-d <date>] [-l <ledger>] [-method <method>]

  Displays a quarterly performance review of the portfolio.
`
}

func (c *quarterlyCmd) SetFlags(f *flag.FlagSet) {
	c.review.period = "quarter"
	f.StringVar(&c.review.date, "d", "", "End date for the report period (defaults to today)")
	f.StringVar(&c.review.method, "method", "fifo", "Cost basis method (average, fifo)")
	f.StringVar(&c.review.ledgerFile, "l", "", "Ledger to report on. Defaults to the only ledger if one exists.")
}

func (c *quarterlyCmd) Execute(ctx context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {
	if err := c.review.init(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return subcommands.ExitUsageError
	}
	review, err := c.review.generateReview()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return subcommands.ExitFailure
	}
	c.render(review, c.review.parsedMethod)
	return subcommands.ExitSuccess
}

func (c *quarterlyCmd) render(review *portfolio.Review, method portfolio.CostBasisMethod) {
	md := renderer.PeriodicMarkdown(review, method)
	printMarkdown(md)
}
