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
	ticker    string
	id        string
	currency  string
	portfolio bool
}

func (*addSecurityCmd) Name() string     { return "add-security" }
func (*addSecurityCmd) Synopsis() string { return "add a new security to the market data" }
func (*addSecurityCmd) Usage() string {
	return `add-security -ticker <ticker> [-id <id> -currency <currency> | -portfolio]

  Adds a new security to the definition file:
  - ticker: The ticker symbol for the security (e.g., "NVDA"). Must be unique.
  - id: The unique identifier for the security (e.g., "US67066G1040.XFRA").
  - currency: The 3-letter currency code for the security (e.g., "EUR").

  Adds all securities declared in the portfolio.

  Both types of addition can be executed, but at least one must be attempted.
`
}

func (c *addSecurityCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.ticker, "ticker", "", "Security ticker symbol (required)")
	f.StringVar(&c.id, "id", "", "Unique security identifier (required)")
	f.StringVar(&c.currency, "currency", "", "Security's currency, 3-letter code (required)")
	f.BoolVar(&c.portfolio, "portfolio", false, "Declare all securities in the portfolio into the market data")
}

func (c *addSecurityCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {

	// If there is at least one of the three simple definition argument consider
	// it has an attempt to define a simple security.
	simple := c.ticker != "" || c.id != "" || c.currency != ""

	// It must be either a single or (inclusive) a porfolio addition.
	if !(simple || c.portfolio) {
		fmt.Fprintln(os.Stderr, "Error: either (-ticker, -id, and -currency) or -portfolio flags are required.")
		return subcommands.ExitUsageError
	}

	market, err := DecodeMarketData()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading securities database: %v\n", err)
		return subcommands.ExitFailure
	}

	if simple {

		if err := portfolio.ValidateCurrency(c.currency); err != nil {
			fmt.Fprintln(os.Stderr, "Error: ", err.Error())
			return subcommands.ExitUsageError
		}

		if market.Resolve(c.ticker) != "" {
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
	}

	if c.portfolio {
		// Declaring all securities from portfolio into the market data
		ledger, err := DecodeLedger()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading portfolio: %v\n", err)
			return subcommands.ExitFailure
		}
		_, err = portfolio.NewAccountingSystem(ledger, market, *defaultCurrency)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating accounting system: %v\n", err)
			return subcommands.ExitFailure
		}
		// as a nice side effect of NewAccountinSystem all securities have been defined.

	}

	if err := EncodeMarketData(market); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving securities database: %v\n", err)
		return subcommands.ExitFailure
	}

	fmt.Printf("✅ Successfully added security '%s' to the market data.\n", c.ticker)
	return subcommands.ExitSuccess
}
