package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/etnz/portfolio"
	"github.com/google/subcommands"
)

// addSecurityCmd holds the flags for the 'add-security' subcommand.
type addSecurityCmd struct {
	ticker     string
	id         string
	currency   string
	fromLedger bool

	tickerFlagName   string
	idFlagName       string
	currencyFlagName string
}

func (*addSecurityCmd) Name() string     { return "add-security" }
func (*addSecurityCmd) Synopsis() string { return "add a new security to the market data" }
func (*addSecurityCmd) Usage() string {
	return `pcs add-security -s <ticker> -id <id> -c <currency> | pcs add-security -from-ledger

  Adds a new security to the definition file, or adds all securities from the ledger file.

  When adding a single security, all of -s, -id, and -c are required.
  - ticker: The ticker symbol for the security (e.g., "NVDA"). Must be unique.
  - id: The unique identifier for the security (e.g., "US67066G1040.XFRA").
  - currency: The 3-letter currency code for the security (e.g., "EUR").

  The -from-ledger flag adds all securities declared in the ledger.

  Both forms can be used in a single invocation.
`
}

func NewAddSecurityCmd() *addSecurityCmd {
	return &addSecurityCmd{
		tickerFlagName:   "s",
		idFlagName:       "id",
		currencyFlagName: "c",
	}
}

func (c *addSecurityCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.ticker, c.tickerFlagName, "", "Security ticker symbol (required)")
	f.StringVar(&c.id, c.idFlagName, "", "Unique security identifier (required)")
	f.StringVar(&c.currency, c.currencyFlagName, "", "Security's currency, 3-letter code (required)")
	f.BoolVar(&c.fromLedger, "from-ledger", false, "Declare all securities in the ledger into the market data")
}

// GenerateAddCommand generates the 'pcs add-security' command string with the given parameters.
// This function is primarily used by the 'search-security' command to construct the
// command needed to add a selected security. It's crucial to keep the flags and
// their names in sync with the 'add-security' command's SetFlags method.
func (c *addSecurityCmd) GenerateAddCommand(ticker, id, currency string) string {
	return fmt.Sprintf("pcs add-security -%s='%s' -%s='%s' -%s='%s'", c.tickerFlagName, ticker, c.idFlagName, id, c.currencyFlagName, currency)
}

// Execute runs the add-security command. It either adds a single security
// based on provided flags or adds all securities found in the ledger to the
// market data.
func (c *addSecurityCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {

	// If there is at least one of the three simple definition argument consider
	// it has an attempt to define a simple security.
	simple := c.ticker != "" || c.id != "" || c.currency != ""

	// It must be either a single or (inclusive) a fromLedger addition.
	if !(simple || c.fromLedger) {
		fmt.Fprintln(os.Stderr, "Error: either (-s, -id, and -c) or -from-ledger flags are required.")
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

	if c.fromLedger {
		// Declaring all securities from ledger into the market data
		ledger, err := DecodeLedger()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading ledger: %v\n", err)
			return subcommands.ExitFailure
		}
		as, err := portfolio.NewAccountingSystem(ledger, market, *defaultCurrency)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating accounting system: %v\n", err)
			return subcommands.ExitFailure
		}
		err = as.DeclareSecurities()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error declaring securities: %v\n", err)
			return subcommands.ExitFailure
		}
	}

	if err := EncodeMarketData(market); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving securities database: %v\n", err)
		return subcommands.ExitFailure
	}

	fmt.Printf("✅ Successfully added security '%s' to the market data.\n", c.ticker)
	return subcommands.ExitSuccess
}
