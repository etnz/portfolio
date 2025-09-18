package cmd

import (
	"context"
	"flag"
	"fmt"
	"github.com/etnz/portfolio"
	"github.com/etnz/portfolio/renderer"
	"os"
	"time"

	"github.com/google/subcommands"
)

type weeklyCmd struct {
	review reviewCmd
	watch  int
}

func (*weeklyCmd) Name() string     { return "weekly" }
func (*weeklyCmd) Synopsis() string { return "display a weekly portfolio performance report" }
func (*weeklyCmd) Usage() string {
	return `pcs weekly [-d <date>] [-method <method>] [-w n]

  Displays a weekly performance review of the portfolio.
`
}

func (c *weeklyCmd) SetFlags(f *flag.FlagSet) {
	c.review.period = "week"
	f.StringVar(&c.review.date, "d", "", "End date for the report period (defaults to today)")
	f.StringVar(&c.review.method, "method", "fifo", "Cost basis method (average, fifo)")
	f.IntVar(&c.watch, "w", 0, "run every n seconds")
}

func (c *weeklyCmd) Execute(ctx context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {
	if err := c.review.init(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return subcommands.ExitUsageError
	}
	for {
		review, err := c.review.generateReview()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			if c.watch == 0 {
				return subcommands.ExitFailure
			}
		} else {
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

func (c *weeklyCmd) render(review *portfolio.Review, method portfolio.CostBasisMethod) {
	md := renderer.WeeklyMarkdown(review, method)
	if c.watch > 0 {
		fmt.Println("\033[2J")
	}
	printMarkdown(md)
}
