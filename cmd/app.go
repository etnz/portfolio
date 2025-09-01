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

// Register registers all the application's subcommands with the provided Commander.
// A main package will call Register() to set up the CLI.
func Register(c *subcommands.Commander) {
	c.Register(&importInvestingCmd{}, "securities")
	c.Register(&updateSecurityCmd{}, "securities")
	c.Register(NewAddSecurityCmd(), "securities")
	c.Register(&searchSecurityCmd{addSecurityCmd: NewAddSecurityCmd()}, "securities")

	c.Register(&updateAmundiCmd{}, "amundi")
	c.Register(&importAmundiCmd{}, "amundi")

	c.Register(&buyCmd{}, "transactions")
	c.Register(&sellCmd{}, "transactions")
	c.Register(&dividendCmd{}, "transactions")
	c.Register(&depositCmd{}, "transactions")
	c.Register(&declareCmd{}, "transactions")
	c.Register(&withdrawCmd{}, "transactions")
	c.Register(&convertCmd{}, "transactions")
	c.Register(&accrueCmd{}, "transactions")

	c.Register(&formatLedgerCmd{}, "tools")

	c.Register(&summaryCmd{}, "analysis")
	c.Register(&holdingCmd{}, "analysis")
	c.Register(&historyCmd{}, "analysis")
	c.Register(&gainsCmd{}, "analysis")
	c.Register(&dailyCmd{}, "analysis")

}

// As a CLI application, it has a very short-lived lifecycle, so it is ok to use global variables for flags.

var (
	marketFile      = flag.String("market-file", "market.jsonl", "Path to the market data file containing securities (JSONL format)")
	ledgerFile      = flag.String("ledger-file", "transactions.jsonl", "Path to the ledger file containing transactions (JSONL format)")
	defaultCurrency = flag.String("default-currency", "EUR", "default currency")
	Verbose         = flag.Bool("v", false, "enable verbose logging")
)

// DecodeAccountingSystem decodes the market data and the ledger to create a new
// AccountingSystem. This system provides a comprehensive view of the portfolio
// by combining transactional history with market information.
func DecodeAccountingSystem() (*portfolio.AccountingSystem, error) {
	market, err := DecodeMarketData()
	if err != nil {
		return nil, fmt.Errorf("could not load securities database: %w", err)
	}
	ledger, err := DecodeLedger()
	if err != nil {
		return nil, fmt.Errorf("could not load ledger: %w", err)
	}
	return portfolio.NewAccountingSystem(ledger, market, *defaultCurrency)
}

// DecodeMarketData decodes securities from the application's securities path folder.
func DecodeMarketData() (*portfolio.MarketData, error) {
	// Load the portfolio database from the specified file.
	m, err := portfolio.DecodeMarketData(*marketFile)
	if errors.Is(err, fs.ErrNotExist) {
		log.Println("warning, database does not exist, creating an empty database instead")
		return portfolio.NewMarketData(), nil
	}
	return m, err
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

// EncodeMarketData encodes securities into the application's securities path folder.
func EncodeMarketData(s *portfolio.MarketData) error {
	// Close the portfolio database if it is not nil.
	return portfolio.EncodeMarketData(*marketFile, s)
}

// EncodeTransaction validates a transaction against the market data and existing
// ledger, then appends it to the ledger file.
func EncodeTransaction(tx portfolio.Transaction) (portfolio.Transaction, error) {
	market, err := DecodeMarketData()
	if err != nil {
		return nil, fmt.Errorf("could not load securities database: %w", err)
	}
	ledger, err := DecodeLedger()
	if err != nil {
		return nil, fmt.Errorf("could not load ledger: %w", err)
	}

	// For validation, a reporting currency is not needed. We pass an empty string.
	as, err := portfolio.NewAccountingSystem(ledger, market, "")
	if err != nil {
		// This error is unexpected here since we pass an empty currency.
		return nil, fmt.Errorf("could not create accounting system: %w", err)
	}
	tx, err = as.Validate(tx)
	if err != nil {
		return nil, err
	}

	// Open the file in append mode, creating it if it doesn't exist.
	f, err := os.OpenFile(*ledgerFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("error opening portfolio file %q: %w", *ledgerFile, err)
	}
	defer f.Close()

	if err := portfolio.EncodeTransaction(f, tx); err != nil {
		return nil, fmt.Errorf("error writing to portfolio file %q: %w", *ledgerFile, err)
	}
	return tx, nil
}
