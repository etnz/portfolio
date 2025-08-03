// Package cmd implements the CLI application to manage a portfolio.
package cmd

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"

	"github.com/etnz/portfolio"
	"github.com/google/subcommands"
)

// Register the subcommands.
// A main package will call Register() to allow subcommands, and Execute() on the user-selected one.
func Register(c *subcommands.Commander) {
	c.Register(&importInvestingCmd{}, "securities")
	c.Register(&updateCmd{}, "securities")

	c.Register(&buyCmd{}, "transactions")
	c.Register(&sellCmd{}, "transactions")
	c.Register(&dividendCmd{}, "transactions")
	c.Register(&depositCmd{}, "transactions")
	c.Register(&withdrawCmd{}, "transactions")
	c.Register(&convertCmd{}, "transactions")

}

// as a CLI application, it has a very short lived lifecycle, so it is ok to use global vaariables.

var securitiesPath = flag.String("securities-path", ".security", "Path to the securities database folder")
var portfolioFile = flag.String("portfolio-file", "transactions.jsonl", "Path to the portfolio transactions file (JSONL format)")

// DecodeSecurities decode securities from the app securities path folder.
func DecodeSecurities() (m *portfolio.Market, err error) {
	// Load the portfolio database from the specified file.
	m, err = portfolio.DecodeMarketData(*securitiesPath)
	if errors.Is(err, fs.ErrNotExist) {
		log.Println("warning, database does not exist, creating an empty database instead")
		m, err = portfolio.NewMarket(), nil

	}
	return
}

// EncodeMarketData encode securities into the app securities path folder.
func EncodeMarketData(s *portfolio.Market) error {
	// Close the portfolio database if it is not nil.
	return portfolio.EncodeMarketData(*securitiesPath, s)
}

// EncodeTransaction appends a single transaction into the app default portfolio file.
func EncodeTransaction(tx portfolio.Transaction) subcommands.ExitStatus {
	filename := *portfolioFile
	// Open the file in append mode, creating it if it doesn't exist.
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening portfolio file %q: %v\n", filename, err)
		return subcommands.ExitFailure
	}
	defer f.Close()

	if err := portfolio.EncodeTransaction(f, tx); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing to portfolio file %q: %v\n", filename, err)
		return subcommands.ExitFailure
	}

	fmt.Printf("Successfully appended transaction to %s\n", filename)
	return subcommands.ExitSuccess
}
