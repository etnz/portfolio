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

type yearlyCmd struct {
	review reviewCmd
}

func (*yearlyCmd) Name() string     { return "yearly" }
func (*yearlyCmd) Synopsis() string { return "display a yearly portfolio performance report" }
func (*yearlyCmd) Usage() string {
	return `pcs yearly [-d <date>] [-method <method>]

  Displays a yearly performance review of the portfolio.
`
}

func (c *yearlyCmd) SetFlags(f *flag.FlagSet) {
	c.review.period = "year"
	f.StringVar(&c.review.date, "d", "", "End date for the report period (defaults to today)")
	f.StringVar(&c.review.method, "method", "fifo", "Cost basis method (average, fifo)")
}

func (c *yearlyCmd) Execute(ctx context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {
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

func (c *yearlyCmd) render(review *portfolio.Review, method portfolio.CostBasisMethod) {
	md := renderer.PeriodicMarkdown(review, method)
	printMarkdown(md)
}
