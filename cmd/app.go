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
var ledgerFile = flag.String("ledger-file", "transactions.jsonl", "Path to the ledger file containing transactions (JSONL format)")

// DecodeSecurities decode securities from the app securities path folder.
func DecodeSecurities() (m *portfolio.MarketData, err error) {
	// Load the portfolio database from the specified file.
	m, err = portfolio.DecodeMarketData(*securitiesPath)
	if errors.Is(err, fs.ErrNotExist) {
		log.Println("warning, database does not exist, creating an empty database instead")
		m, err = portfolio.NewMarketData(), nil

	}
	return
}

// DecodeLedger decodes the ledger from the application's default ledger file.
// If the file does not exist, it returns a new empty ledger.
func DecodeLedger() (*portfolio.Ledger, error) {
	f, err := os.Open(*ledgerFile)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// If the file doesn't exist, it's an empty ledger.
			return portfolio.NewLedger(), nil
		}
		return nil, fmt.Errorf("could not open ledger file %q: %w", *ledgerFile, err)
	}
	defer f.Close()

	ledger, err := portfolio.DecodeLedger(f)
	if err != nil {
		return nil, fmt.Errorf("could not decode ledger file %q: %w", *ledgerFile, err)
	}
	return ledger, nil
}

// EncodeMarketData encode securities into the app securities path folder.
func EncodeMarketData(s *portfolio.MarketData) error {
	// Close the portfolio database if it is not nil.
	return portfolio.EncodeMarketData(*securitiesPath, s)
}

// EncodeTransaction validates a transaction against the market data and existing
// ledger, then appends it to the ledger file.
func EncodeTransaction(tx portfolio.Transaction) error {
	market, err := DecodeSecurities()
	if err != nil {
		return fmt.Errorf("could not load securities database: %w", err)
	}
	ledger, err := DecodeLedger()
	if err != nil {
		return fmt.Errorf("could not load ledger: %w", err)
	}
	tx, err = portfolio.Validate(market, ledger, tx)
	if err != nil {
		return err
	}

	// Open the file in append mode, creating it if it doesn't exist.
	f, err := os.OpenFile(*ledgerFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("error opening portfolio file %q: %w", *ledgerFile, err)
	}
	defer f.Close()

	if err := portfolio.EncodeTransaction(f, tx); err != nil {
		return fmt.Errorf("error writing to portfolio file %q: %w", *ledgerFile, err)
	}
	return nil
}
