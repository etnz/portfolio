package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/etnz/portfolio"
	"github.com/google/subcommands"
)

type searchSecurityCmd struct{
	addSecurityCmd *addSecurityCmd
	showErrors     bool
}

func (*searchSecurityCmd) Name() string     { return "search-security" }
func (*searchSecurityCmd) Synopsis() string { return "search for securities using EODHD API" }
func (*searchSecurityCmd) Usage() string {
	return `pcs search-security <search term>

  Searches for securities via EOD Historical Data API and prints
  ready-to-use 'add-security' commands for the results.
  Requires the EODHD_API_TOKEN environment variable to be set.
`
}

func (c *searchSecurityCmd) SetFlags(f *flag.FlagSet) {
	f.BoolVar(&c.showErrors, "show-errors", false, "Display entries with invalid ISINs and print error messages")
}

func (c *searchSecurityCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if f.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "Error: a search term is required.")
		return subcommands.ExitUsageError
	}
	searchTerm := strings.Join(f.Args(), " ")

	results, err := portfolio.Search(searchTerm)
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
		commandToCopy := c.addSecurityCmd.GenerateAddCommand(suggestedTicker, securityID.String(), item.Currency)

		fmt.Printf("    $ %s\n\n", commandToCopy)
	}

	return subcommands.ExitSuccess
}
