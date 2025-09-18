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

// summaryCmd holds the flags for the 'summary' subcommand.
type summaryCmd struct {
	date   string
	update bool
}

func (*summaryCmd) Name() string     { return "summary" }
func (*summaryCmd) Synopsis() string { return "display a portfolio performance summary" }
func (*summaryCmd) Usage() string {
	return `pcs summary [-d <date>]

  Displays a summary of the portfolio, including total market value.
`
}

func (c *summaryCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.date, "d", portfolio.Today().String(), "Date for the summary. See the user manual for supported date formats.")
}

func (c *summaryCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	on, err := portfolio.ParseDate(c.date)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing date: %v\n", err)
		return subcommands.ExitUsageError
	}

	if on.IsToday() {
		c.update = true
	}

	ledger, err := DecodeLedger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding ledger: %v\n", err)
		return subcommands.ExitFailure
	}

	if c.update {
		// err := as.MarketData.UpdateIntraday()
		// if err != nil {
		// 	fmt.Fprintf(os.Stderr, "Error updating intraday prices: %v\n", err)
		// 	return subcommands.ExitFailure
		// }
	}

	// Helper to create a review for a period and calculate TWR
	calculateTWR := func(period portfolio.Range) (portfolio.Percent, error) {
		review, err := ledger.NewReview(period)
		if err != nil {
			return 0, err
		}
		// TWR for the whole portfolio is calculated on a virtual asset with an empty ticker.
		return review.TimeWeightedReturn(), nil
	}

	inceptionDate := ledger.GlobalInceptionDate()

	// Calculate TWR for all periods
	daily, errD := calculateTWR(portfolio.Daily.Range(on))
	wtd, errW := calculateTWR(portfolio.Weekly.Range(on))
	mtd, errM := calculateTWR(portfolio.Monthly.Range(on))
	qtd, errQ := calculateTWR(portfolio.Quarterly.Range(on))
	ytd, errY := calculateTWR(portfolio.Yearly.Range(on))
	inception, errI := calculateTWR(portfolio.NewRange(inceptionDate.Add(1), on))

	if errD != nil || errW != nil || errM != nil || errQ != nil || errY != nil || errI != nil {
		// Handle or log errors as needed
		fmt.Fprintln(os.Stderr, "Error calculating performance metrics.")
		return subcommands.ExitFailure
	}

	endSnapshot := ledger.NewSnapshot(on)

	summaryData := &renderer.SummaryData{
		Date:             on,
		TotalMarketValue: endSnapshot.TotalPortfolio(),
		Daily:            daily,
		WTD:              wtd,
		MTD:              mtd,
		QTD:              qtd,
		YTD:              ytd,
		Inception:        inception,
	}

	md := renderer.SummaryMarkdown(summaryData)
	printMarkdown(md)

	return subcommands.ExitSuccess
}
