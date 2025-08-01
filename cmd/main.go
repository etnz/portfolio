package cmd

import (
	"errors"
	"flag"
	"io/fs"
	"log"

	"github.com/etnz/porfolio/security"
	"github.com/google/subcommands"
)

// Global variable to hold all subcommands
// This map is used to register and access subcommands by their names.
// It allows for easy addition and retrieval of commands in the application.
var Commands []subcommands.Command

func init() {
	// Register the subcommands.
	Commands = append(Commands, &importInvesting{})
}

var securitiesPath = flag.String("securities-path", ".security", "Path to the securities database folder")

// OpenSecurities is the central function to open the securities database.
func OpenSecurities() (db *security.DB, err error) {
	// Load the security database from the specified file.
	db, err = security.Load(*securitiesPath)
	if errors.Is(err, fs.ErrNotExist) {
		log.Println("warning, database does not exist, creating an empty database instead")
		db, err = security.NewDB(), nil

	}
	return
}

func CloseSecurities(db *security.DB) error {
	// Close the security database if it is not nil.
	return db.Persist(*securitiesPath)
}
