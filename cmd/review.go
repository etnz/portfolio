package cmd

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/etnz/portfolio"
	"github.com/etnz/portfolio/renderer"
	"github.com/google/subcommands"
)

// reviewCmd holds the flags for the 'review' subcommand.
type reviewCmd struct {
	period     string
	date       string
	start      string
	method     string
	update     bool
	ledgerFile string
	options    renderer.ReviewRenderOptions
	// processed
	parsedMethod portfolio.CostBasisMethod
	rng          portfolio.Range
	ledgers      []*portfolio.Ledger
}

func (*reviewCmd) Name() string { return "review" }

func (*reviewCmd) Synopsis() string { return "review a portfolio performance" }
func (*reviewCmd) Usage() string {
	return `pcs review [-p <period>| -start <date>] [-d <date>] [-l <ledger>] [-s]
	
  Review the portfolio for a given period.
`
}

func (c *reviewCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.date, "d", "", "Date for the report. See the user manual for supported date formats.")
	f.StringVar(&c.period, "p", portfolio.Daily.String(), "period for the review (day, week, month, quarter, year)")
	f.BoolVar(&c.options.SimplifiedView, "s", false, "provide a simplified asset review")
	f.BoolVar(&c.options.SkipTransactions, "t", false, "skip transactions in the report")
	f.StringVar(&c.start, "start", "", "Start date of the reporting period. Overrides -p.")
	f.StringVar(&c.method, "method", "fifo", "Cost basis method (average, fifo)")
	f.StringVar(&c.ledgerFile, "l", "", "Ledger to report on. Defaults to the only ledger if one exists.")
}

func (c *reviewCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if err := c.init(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return subcommands.ExitUsageError
	}

	for _, ledger := range c.ledgers {
		review := c.generateReview(ledger)
		c.render(review, c.parsedMethod)
	}

	return subcommands.ExitSuccess
}

func (c *reviewCmd) init() error {
	if c.date == "" {
		c.date = portfolio.Today().String()
	}
	endDate, err := portfolio.ParseDate(c.date)
	if err != nil {
		return fmt.Errorf("parsing end date: %w", err)
	}
	if c.start != "" {
		// Custom range using start and end dates
		startDate, err := portfolio.ParseDate(c.start)
		if err != nil {
			return fmt.Errorf("parsing start date: %w", err)
		}
		c.rng = portfolio.NewRange(startDate, endDate)
	} else {
		// Predefined period
		p, err := portfolio.ParsePeriod(c.period)
		if err != nil {
			return fmt.Errorf("parsing period: %w", err)
		}
		c.rng = p.Range(endDate)
	}

	if !c.rng.To.Before(portfolio.Today()) {
		c.update = true
	}

	method, err := portfolio.ParseCostBasisMethod(c.method)
	if err != nil {
		return fmt.Errorf("parsing cost basis method: %w", err)
	}
	c.parsedMethod = method

	c.ledgers, err = DecodeLedgers(c.ledgerFile)
	if err != nil {
		return fmt.Errorf("decoding ledgers: %w", err)
	}
	return nil
}

func (c *reviewCmd) generateReview(ledger *portfolio.Ledger) *portfolio.Review {

	if c.update {
		if err := ledger.UpdateIntraday(); err != nil {
			log.Printf("Warning: could not update some intraday prices: %v\n", err)
		}
	}
	return ledger.NewReview(c.rng)
}

func (c *reviewCmd) render(review *portfolio.Review, method portfolio.CostBasisMethod) {
	r := renderer.NewReview(review, method)
	md := renderer.RenderReview(r, c.options)
	printMarkdown(md)
}
