package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/etnz/portfolio"
	"github.com/google/subcommands"
)

type addSecurityCmd struct {
	ticker   string
	id       string
	currency string
}

func (*addSecurityCmd) Name() string     { return "add-security" }
func (*addSecurityCmd) Synopsis() string { return "add a new security to the market data" }
func (*addSecurityCmd) Usage() string {
	return `add-security -ticker <ticker> -id <id> -currency <currency>

  Adds a new security to the definition file.
  - ticker: The ticker symbol for the security (e.g., "NVDA"). Must be unique.
  - id: The unique identifier for the security (e.g., "US67066G1040.XFRA").
  - currency: The 3-letter currency code for the security (e.g., "EUR").
`
}

func (c *addSecurityCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.ticker, "ticker", "", "Security ticker symbol (required)")
	f.StringVar(&c.id, "id", "", "Unique security identifier (required)")
	f.StringVar(&c.currency, "currency", "", "Security's currency, 3-letter code (required)")
}

func (c *addSecurityCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if c.ticker == "" || c.id == "" || c.currency == "" {
		fmt.Fprintln(os.Stderr, "Error: -ticker, -id, and -currency flags are all required.")
		return subcommands.ExitUsageError
	}
	if err := portfolio.ValidateCurrency(c.currency); err != nil {
		fmt.Fprintln(os.Stderr, "Error: ", err.Error())
		return subcommands.ExitUsageError
	}

	market, err := DecodeSecurities()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading securities database: %v\n", err)
		return subcommands.ExitFailure
	}

	if market.Get(c.ticker) != nil {
		fmt.Fprintf(os.Stderr, "Error: Ticker '%s' already exists in the market data.\n", c.ticker)
		return subcommands.ExitFailure
	}

	parsedID, err := portfolio.ParseID(c.id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing security ID '%s': %v\n", c.id, err)
		return subcommands.ExitUsageError
	}
	sec := portfolio.NewSecurity(parsedID, c.ticker, c.currency)
	market.Add(sec)

	if err := EncodeMarketData(market); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving securities database: %v\n", err)
		return subcommands.ExitFailure
	}

	fmt.Printf("✅ Successfully added security '%s' to the market data.\n", c.ticker)
	return subcommands.ExitSuccess
}
