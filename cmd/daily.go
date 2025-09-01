package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/etnz/portfolio"
	"github.com/etnz/portfolio/date"
	"github.com/google/subcommands"
)

// dailyCmd holds the flags for the 'daily' subcommand.
type dailyCmd struct {
	date     string
	currency string
	update   bool
}

func (*dailyCmd) Name() string     { return "daily" }
func (*dailyCmd) Synopsis() string { return "display a daily portfolio performance report" }
func (*dailyCmd) Usage() string {
	return `pcs daily [-d <date>] [-c <currency>] [-u]

  Displays a summary of the portfolio for a single day.
`
}

func (c *dailyCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.date, "d", date.Today().String(), "Date for the report. See the user manual for supported date formats.")
	f.StringVar(&c.currency, "c", "EUR", "Reporting currency for the report")
	f.BoolVar(&c.update, "u", false, "update with latest intraday prices before calculating the report")
}

func (c *dailyCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	on, err := date.Parse(c.date)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing date: %v\n", err)
		return subcommands.ExitUsageError
	}

	market, err := DecodeMarketData()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading securities: %v\n", err)
		return subcommands.ExitFailure
	}

	ledger, err := DecodeLedger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading ledger: %v\n", err)
		return subcommands.ExitFailure
	}

	as, err := portfolio.NewAccountingSystem(ledger, market, c.currency)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating accounting system: %v\n", err)
		return subcommands.ExitFailure
	}

	if c.update {
		err := market.UpdateIntraday()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error updating intraday prices: %v\n", err)
			return subcommands.ExitFailure
		}
	}

	report, err := as.NewDailyReport(on, time.Now())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error calculating daily report: %v\n", err)
		return subcommands.ExitFailure
	}

	// Overall Performance
	fmt.Printf("Daily Report for %s (in %s)\n", report.Date, report.ReportingCurrency)
	if on == date.Today() {
		fmt.Printf("Report generated at %s\n", report.Time.Format("15:04:05"))
	}
	fmt.Println()
	fmt.Println("Overall Performance")
	fmt.Println("--------------------------------------------------")
	fmt.Printf("Value at Prev. Close:        %.2f\n", report.ValueAtPrevClose)
	fmt.Printf("Value at Day's Close:        %.2f\n", report.ValueAtClose)
	fmt.Println("--------------------------------------------------")
	fmt.Printf("Total Day's Gain / Loss:       %+.2f (%+.2f%%)\n", report.TotalGain, (report.TotalGain/report.ValueAtPrevClose)*100)

	// Breakdown of Change
	nonZeroCount := 0
	if report.MarketGains != 0 {
		nonZeroCount++
	}
	if report.RealizedGains != 0 {
		nonZeroCount++
	}
	if report.NetCashFlow != 0 {
		nonZeroCount++
	}

	if nonZeroCount > 0 {
		fmt.Println()
		fmt.Println("Breakdown of Change")
		fmt.Println("--------------------------------------------------")
		if report.MarketGains != 0 {
			fmt.Printf("Market Gains (Unrealized):      %+.2f\n", report.MarketGains)
		}
		if report.RealizedGains != 0 {
			fmt.Printf("Realized Gains (from sales):     %+.2f\n", report.RealizedGains)
		}
		if report.NetCashFlow != 0 {
			fmt.Printf("Net Cash Flow (today):          %+.2f\n", report.NetCashFlow)
		}
		if nonZeroCount > 1 {
			fmt.Println("--------------------------------------------------")
			fmt.Printf("Total Change:                  %+.2f\n", report.TotalGain)
		}
	}

	// Active Assets
	if len(report.ActiveAssets) > 0 {
		fmt.Println()
		fmt.Println("Active Assets")
		fmt.Println("--------------------------------------------------")
		fmt.Printf("% -10s % -20s %s\n", "Ticker", "Day's Gain / Loss", "Change")
		fmt.Println("--------------------------------------------------")
		for _, asset := range report.ActiveAssets {
			if asset.Gain != 0 {
				fmt.Printf("% -10s % 15.2f       %+.2f%%\n", asset.Security, asset.Gain, asset.Return*100)
			}
		}
		fmt.Println("--------------------------------------------------")
	}

	// Today's Transactions
	if len(report.Transactions) > 0 {
		fmt.Println()
		fmt.Println("Today's Transactions")
		fmt.Println("--------------------------------------------------")
		for _, tx := range report.Transactions {
			fmt.Printf("- %s\n", tx.What())
		}
	}

	return subcommands.ExitSuccess
}
