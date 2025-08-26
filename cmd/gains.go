package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/etnz/portfolio"
	"github.com/etnz/portfolio/date"
	"github.com/google/subcommands"
)

type gainsCmd struct {
	period   string
	start    string
	end      string
	currency string
	method   string
	update   bool
}

func (*gainsCmd) Name() string     { return "gains" }
func (*gainsCmd) Synopsis() string { return "realized and unrealized gain analysis" }
func (*gainsCmd) Usage() string {
	return `pcs gains [-period <period>] [-start <date>] [-end <date>] [-c <currency>] [-method <method>] [-u]

  Calculates and displays realized and unrealized gains for each security.
`
}

func (c *gainsCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.period, "period", "", "Predefined period (day, week, month, quarter, year)")
	f.StringVar(&c.start, "start", "", "Start date of the reporting period (YYYY-MM-DD)")
	f.StringVar(&c.end, "end", date.Today().String(), "End date of the reporting period (YYYY-MM-DD)")
	f.StringVar(&c.currency, "c", "EUR", "Reporting currency")
	f.StringVar(&c.method, "method", "average", "Cost basis method (average, fifo)")
	f.BoolVar(&c.update, "u", false, "update with latest intraday prices before calculating gains")
}

func (c *gainsCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	// Determine the reporting period
	var period date.Range
	endDate, err := date.Parse(c.end)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing end date: %v\n", err)
		return subcommands.ExitUsageError
	}

	if c.start != "" && c.period != "" {
		fmt.Fprintln(os.Stderr, "-start and -period flags cannot be used together")
		return subcommands.ExitUsageError
	}

	if c.start != "" {
		startDate, err := date.Parse(c.start)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing start date: %v\n", err)
			return subcommands.ExitUsageError
		}
		period = date.Range{From: startDate, To: endDate}
	} else {
		period = date.NewRangeFrom(endDate, c.period)
	}

	// Decode market data and ledger
	market, err := DecodeMarketData()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading securities: %v\n", err)
		return subcommands.ExitFailure
	}

	if c.update {
		err := market.UpdateIntraday()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error updating intraday prices: %v\n", err)
			return subcommands.ExitFailure
		}
	}

	ledger, err := DecodeLedger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading ledger: %v\n", err)
		return subcommands.ExitFailure
	}

	// Create accounting system
	as, err := portfolio.NewAccountingSystem(ledger, market, c.currency)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating accounting system: %v\n", err)
		return subcommands.ExitFailure
	}

	// Parse cost basis method
	method, err := portfolio.ParseCostBasisMethod(c.method)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing cost basis method: %v\n", err)
		return subcommands.ExitUsageError
	}

	// Calculate gains
	report, err := as.CalculateGains(period, method)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error calculating gains: %v\n", err)
		return subcommands.ExitFailure
	}

	// Print report
	fmt.Printf("Capital Gains Report (Method: %s) for %s to %s (in %s)\n",
		report.Method, report.Range.From, report.Range.To, report.ReportingCurrency)
	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("%-" + "20s %20s %20s %20s\n", "Security", "Realized Gain/Loss", "Unrealized Gain/Loss", "Total Gain/Loss")
	fmt.Println(strings.Repeat("-", 80))

	var totalRealized, totalUnrealized, totalGain float64

	for _, s := range report.Securities {
		fmt.Printf("%-" + "20s %20.2f %20.2f %20.2f\n",
			s.Security, s.Realized, s.Unrealized, s.Total)
		totalRealized += s.Realized
		totalUnrealized += s.Unrealized
		totalGain += s.Total
	}

	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("%-" + "20s %20.2f %20.2f %20.2f\n",
		"Total", totalRealized, totalUnrealized, totalGain)

	return subcommands.ExitSuccess
}
