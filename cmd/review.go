package cmd

import (
	"context"
	"flag"
	"log"

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
	opts       renderer.ReviewRenderOptions
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
	f.BoolVar(&c.opts.SimplifiedView, "s", false, "provide a simplified asset review")
	f.BoolVar(&c.opts.SkipTransactions, "t", false, "skip transactions in the report")
	f.StringVar(&c.start, "start", "", "Start date of the reporting period. Overrides -p.")
	f.StringVar(&c.method, "method", "fifo", "Cost basis method (average, fifo)")
	f.StringVar(&c.ledgerFile, "l", "", "Ledger to report on. Defaults to the only ledger if one exists.")
}

func (c *reviewCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if c.date == "" {
		c.date = portfolio.Today().String()
	}
	endDate, err := portfolio.ParseDate(c.date)
	if err != nil {
		log.Printf("Error parsing end date: %v", err)
		return subcommands.ExitUsageError
	}

	var rng portfolio.Range
	if c.start != "" {
		// Custom range using start and end dates
		startDate, err := portfolio.ParseDate(c.start)
		if err != nil {
			log.Printf("Error parsing start date: %v", err)
			return subcommands.ExitUsageError
		}
		rng = portfolio.NewRange(startDate, endDate)
	} else {
		// Predefined period
		p, err := portfolio.ParsePeriod(c.period)
		if err != nil {
			log.Printf("Error parsing period: %v", err)
			return subcommands.ExitUsageError
		}
		rng = p.Range(endDate)
	}

	if !rng.To.Before(portfolio.Today()) {
		c.update = true
	}

	parsedMethod, err := portfolio.ParseCostBasisMethod(c.method)
	if err != nil {
		log.Printf("Error parsing cost basis method: %v", err)
		return subcommands.ExitUsageError
	}

	ledgers, err := DecodeLedgers(c.ledgerFile)
	if err != nil {
		log.Printf("Error decoding ledgers: %v", err)
		return subcommands.ExitFailure
	}

	var reviews []*portfolio.Review
	for _, ledger := range ledgers {
		if c.update {
			if err := ledger.UpdateIntraday(); err != nil {
				log.Printf("Warning: could not update some intraday prices for ledger %q: %v\n", ledger.Name(), err)
			}
		}
		reviews = append(reviews, ledger.NewReview(rng))
	}

	var md string
	if len(reviews) == 1 {
		r := renderer.NewReview(reviews[0], parsedMethod)
		md = renderer.RenderReview(r, c.opts)
	} else {
		cr := renderer.NewConsolidatedReview(reviews, parsedMethod)
		md = renderer.RenderConsolidatedReview(cr, c.opts)
	}
	printMarkdown(md)

	return subcommands.ExitSuccess
}
