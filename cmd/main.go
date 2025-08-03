package cmd

import (
	"errors"
	"flag"
	"io/fs"
	"log"

	"github.com/etnz/portfolio"
	"github.com/google/subcommands"
)

// Register the subcommands.
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

var securitiesPath = flag.String("securities-path", ".security", "Path to the securities database folder")
var portfolioFile = flag.String("portfolio-file", "transactions.jsonl", "Path to the portfolio transactions file (JSONL format)")

// OpenSecurities is the central function to open the securities database.
func OpenSecurities() (db *portfolio.Securities, err error) {
	// Load the portfolio database from the specified file.
	db, err = portfolio.DecodeSecurities(*securitiesPath)
	if errors.Is(err, fs.ErrNotExist) {
		log.Println("warning, database does not exist, creating an empty database instead")
		db, err = portfolio.NewSecurities(), nil

	}
	return
}

func CloseSecurities(s *portfolio.Securities) error {
	// Close the portfolio database if it is not nil.
	return portfolio.EncodeSecurities(*securitiesPath, s)
}
