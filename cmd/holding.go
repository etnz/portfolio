package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/etnz/portfolio"
	"github.com/etnz/portfolio/date"
	"github.com/google/subcommands"
)

type holdingCmd struct {
	date     string
	currency string
	update   bool
}

func (*holdingCmd) Name() string     { return "holding" }
func (*holdingCmd) Synopsis() string { return "display detailed holdings for a specific date" }
func (*holdingCmd) Usage() string {
	return `pcs holding [-d <date>] [-c <currency>] [-u]

  Displays the portfolio holdings (securities and cash) on a given date.
`
}

func (c *holdingCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.date, "d", date.Today().String(), "Date for the holdings report (YYYY-MM-DD)")
	f.StringVar(&c.currency, "c", "EUR", "Reporting currency for market values")
	f.BoolVar(&c.update, "u", false, "update with latest intraday prices before calculating the report")

}

func (c *holdingCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
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

	as, err := portfolio.NewAccountingSystem(ledger, market, c.currency)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating accounting system: %v\n", err)
		return subcommands.ExitFailure
	}

	report, err := as.NewHoldingReport(on)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating holding report: %v\n", err)
		return subcommands.ExitFailure
	}

	fmt.Printf("Holdings on %s in reporting currency %s\n\n", report.Date, report.ReportingCurrency)

	// Securities
	fmt.Println("Securities:")
	fmt.Println("-----------------------------------------------------------------")
	fmt.Printf("%-10s %15s %15s %15s\n", "Ticker", "Quantity", "Price", "Market Value")
	fmt.Println("-----------------------------------------------------------------")

	for _, h := range report.Securities {
		fmt.Printf("%-10s %15.4f %15.4f %15.2f\n", h.Ticker, h.Quantity, h.Price, h.MarketValue)
	}
	fmt.Println("-----------------------------------------------------------------")

	// Cash
	fmt.Println("\nCash Balances:")
	fmt.Println("-------------------------------------------------")
	fmt.Printf("%-10s %15s %15s\n", "Currency", "Balance", "Value")
	fmt.Println("-------------------------------------------------")

	for _, h := range report.Cash {
		fmt.Printf("%-10s %15.2f %15.2f\n", h.Currency, h.Balance, h.Value)
	}
	fmt.Println("-------------------------------------------------")

	fmt.Printf("\nTotal Portfolio Value: %.2f %s\n", report.TotalValue, report.ReportingCurrency)

	return subcommands.ExitSuccess
}
