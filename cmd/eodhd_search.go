package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/etnz/portfolio"
	"github.com/etnz/portfolio/eodhd"
	"github.com/google/subcommands"
)

// eodhdSearchCmd implements the "eodhd search" command.
type eodhdSearchCmd struct {
	eodhdApiFlag string
	Cmd          *declareCmd
	showErrors   bool
}

func (*eodhdSearchCmd) Name() string     { return "search" }
func (*eodhdSearchCmd) Synopsis() string { return "searches for securities on EODHD" }
func (*eodhdSearchCmd) Usage() string {
	return `pcs eodhd search <search term>

  Searches for securities via EOD Historical Data API and prints
  ready-to-use 'pcs' commands for the results.
  
  Requires the EODHD_API_TOKEN environment variable to be set or passed as a flag.
`
}

func (c *eodhdSearchCmd) SetFlags(f *flag.FlagSet) {
	flag.StringVar(&c.eodhdApiFlag, "eodhd-api-key", "", "EODHD API key to use for consuming EODHD.com API. This flag takes precedence over the "+eodhd_api_key+" environment variable. You can get one at https://eodhd.com/")
	f.BoolVar(&c.showErrors, "show-errors", false, "Display entries with invalid ISINs and print error messages")
}

// eodhdApiKey retrieves the EODHD API key from the command-line flag or the environment variable.
// It prioritizes the flag over the environment variable.
func (c *eodhdSearchCmd) eodhdApiKey() string {
	// If the flag is not set, we try to read it from the environment variable.
	if c.eodhdApiFlag == "" {
		c.eodhdApiFlag = os.Getenv(eodhd_api_key)
	}
	return c.eodhdApiFlag
}

func (c *eodhdSearchCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if f.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "Error: a search term is required.")
		return subcommands.ExitUsageError
	}
	searchTerm := strings.Join(f.Args(), " ")

	key := c.eodhdApiKey()
	if key == "" {
		fmt.Fprintf(os.Stderr, "Error: EODHD API key is not set. Use -eodhd-api-key flag or EODHD_API_KEY environment variable\n")
		return subcommands.ExitFailure
	}

	results, err := eodhd.Search(key, searchTerm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error searching securities: %v\n", err)
		return subcommands.ExitFailure
	}

	if len(results) == 0 {
		fmt.Printf("No results found for '%s'.\n", searchTerm)
		return subcommands.ExitSuccess
	}

	fmt.Printf("Found %d results for '%s':\n\n", len(results), searchTerm)

	for _, item := range results {
		fmt.Printf("➡️   Name       : %s (%s)\n", item.Name, item.Code)
		fmt.Printf("    Type        : %s, Country: %s, Currency: %s\n", item.Type, item.Country, item.Currency)
		fmt.Printf("    ISIN.MIC    : %s.%s\n", item.ISIN, item.MIC)
		fmt.Printf("    Prev. Close : %.2f on %s\n", item.PreviousClose, item.PreviousCloseDate)

		securityID, err := portfolio.NewMSSI(item.ISIN, item.MIC)
		if err != nil {
			if c.showErrors {
				fmt.Fprintf(os.Stderr, "    Error creating security ID for %s (%s): %v\n\n", item.Name, item.Code, err)
			}
			continue // skip invalid results
		}

		suggestedTicker := item.Code
		commandToCopy := c.Cmd.GenerateCommand(suggestedTicker, securityID.String(), item.Currency)

		fmt.Printf("    $ %s\n\n", commandToCopy)
	}

	return subcommands.ExitSuccess
}
