package cmd

import (
	"errors"
	"flag"
	"io/fs"
	"log"

	"github.com/etnz/portfolio/security"
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
func OpenSecurities() (db *security.Securities, err error) {
	// Load the security database from the specified file.
	db, err = security.Load(*securitiesPath)
	if errors.Is(err, fs.ErrNotExist) {
		log.Println("warning, database does not exist, creating an empty database instead")
		db, err = security.New(), nil

	}
	return
}

func CloseSecurities(db *security.Securities) error {
	// Close the security database if it is not nil.
	return db.Persist(*securitiesPath)
}
