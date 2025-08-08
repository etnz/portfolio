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
}

func (*holdingCmd) Name() string     { return "holding" }
func (*holdingCmd) Synopsis() string { return "display portfolio holdings at a specific date" }
func (*holdingCmd) Usage() string {
	return `holding [-d <date>] [-c <currency>]

  Displays the portfolio holdings (securities and cash) on a given date.
`
}

func (c *holdingCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.date, "d", date.Today().String(), "Date for the holdings report (YYYY-MM-DD)")
	f.StringVar(&c.currency, "c", "EUR", "Reporting currency for market values")
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

	fmt.Printf("Holdings on %s in reporting currency %s\n\n", on, c.currency)

	// Securities
	fmt.Println("Securities:")
	fmt.Println("-----------------------------------------------------------------")
	fmt.Printf("%-10s %15s %15s %15s\n", "Ticker", "Quantity", "Price", "Market Value")
	fmt.Println("-----------------------------------------------------------------")

	for sec := range ledger.AllSecurities() {
		ticker := sec.Ticker()
		id := sec.ID()
		currency := sec.Currency()
		position := ledger.Position(ticker, on)
		if position <= 1e-9 {
			continue
		}
		price, ok := market.PriceAsOf(id, on)
		if !ok {
			fmt.Fprintf(os.Stderr, "Warning: could not find price for %s (%s) on %s\n", ticker, id, on)
			continue
		}
		value := position * price
		convertedValue, err := as.ConvertCurrency(value, currency, c.currency, on)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not convert currency for %s: %v\n", ticker, err)
			continue
		}
		fmt.Printf("%-10s %15.4f %15.4f %15.2f\n", ticker, position, price, convertedValue)
	}
	fmt.Println("-----------------------------------------------------------------")

	// Cash
	fmt.Println("\nCash Balances:")
	fmt.Println("-------------------------------------------------")
	fmt.Printf("%-10s %15s %15s\n", "Currency", "Balance", "Value")
	fmt.Println("-------------------------------------------------")

	for currency := range ledger.AllCurrencies() {
		balance := ledger.CashBalance(currency, on)
		if balance <= 1e-9 {
			continue
		}
		convertedBalance, err := as.ConvertCurrency(balance, currency, c.currency, on)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not convert currency for cash %s: %v\n", currency, err)
			continue
		}
		fmt.Printf("%-10s %15.2f %15.2f\n", currency, balance, convertedBalance)
	}
	fmt.Println("-------------------------------------------------")

	tmv, err := as.TotalMarketValue(on)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error calculating total market value: %v\n", err)
		return subcommands.ExitFailure
	}
	fmt.Printf("\nTotal Portfolio Value: %.2f %s\n", tmv, c.currency)

	return subcommands.ExitSuccess
}
