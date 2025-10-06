package portfolio

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// FindLedger return the unique ledger corresponding with the name.
// If there is only one ledger found, returns it.
// If the query is meant to match all ledgers and the list is empty returns an empty default ledger.
// In any other cases it returns an error.
func FindLedger(path, query string) (*Ledger, error) {

	ledgerPaths, err := findLedgerPaths(path, query)
	if err != nil {
		return nil, err
	}
	switch len(ledgerPaths) {
	case 0:
		// nothing found, return an error by default unless the query was ""
		if query == "" {
			l := NewLedger()
			// use a default name
			l.name = "transactions"
			return l, nil
		}
		return nil, fmt.Errorf("could not find ledger %q", query)
	case 1:
		return loadLedgerFile(path, ledgerPaths[0])
	default:
		return nil, fmt.Errorf("multiple ledgers found for %q", query)
	}
}

// FindLedgers discovers and loads ledger files from a given portfolio path.
// The query string can be used to filter which ledgers are loaded.
// If query is empty, all ledgers (.jsonl files) in the path are loaded.
// If query specifies a ledger name (e.g., "john/bnp"), only that ledger is loaded.
// A ledger name is its relative path from the portfolio path, without the .jsonl extension.
func FindLedgers(path, query string) ([]*Ledger, error) {
	ledgerPaths, err := findLedgerPaths(path, query)
	if err != nil {
		return nil, err
	}

	var loadedLedgers []*Ledger
	for _, fullPath := range ledgerPaths {
		ledger, err := loadLedgerFile(path, fullPath)
		if err != nil {
			// In a multi-file load, it's better to return a partial result with an error
			// or just the error, depending on the desired behavior. Here we fail fast.
			return nil, err
		}
		loadedLedgers = append(loadedLedgers, ledger)
	}

	return loadedLedgers, nil
}

// loadLedgerFile opens, decodes, and initializes a ledger from a given file path.
// It sets the ledger's name based on its relative path to the portfolio root.
func loadLedgerFile(portfolioPath, fullPath string) (*Ledger, error) {
	relPath, err := filepath.Rel(portfolioPath, fullPath)
	if err != nil {
		return nil, fmt.Errorf("could not determine relative path for %q: %w", fullPath, err)
	}
	ledgerName := strings.TrimSuffix(relPath, ".jsonl")

	f, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("could not open ledger file %q: %w", fullPath, err)
	}
	defer f.Close()

	ledger, err := DecodeLedger(f)
	if err != nil {
		return nil, fmt.Errorf("could not decode ledger file %q: %w", fullPath, err)
	}
	ledger.name = ledgerName
	return ledger, nil
}

// SaveLedger saves a single ledger to its corresponding file within the portfolio path.
// It uses the ledger's name to construct the file path (e.g., a ledger named "john/bnp"
// will be saved to "<path>/john/bnp.jsonl").
func SaveLedger(path string, ledger *Ledger) error {
	ledgerName := ledger.Name()
	if ledgerName == "" {
		return fmt.Errorf("cannot save ledger with an empty name")
	}

	filePath := filepath.Join(path, ledgerName+".jsonl")

	// Ensure the directory for the ledger file exists.
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("could not create directory for ledger %q: %w", filePath, err)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("error opening ledger file %q for writing: %w", filePath, err)
	}
	defer file.Close()

	return EncodeLedger(file, ledger)
}

// findLedgerPaths scans a directory and returns a map of ledger names to their full file paths.
func findLedgerPaths(path, query string) ([]string, error) {
	var ledgers []string

	err := filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(p, ".jsonl") {

			relPath, err := filepath.Rel(path, p)
			if err != nil {
				// This should not happen if p is in path
				return err
			}
			ledgerName := strings.TrimSuffix(relPath, ".jsonl")

			// test if ledgerName "matches" the query.
			// This is very very rudimentary to get started
			if query == "" || ledgerName == query {
				ledgers = append(ledgers, p)
			}
		}
		return nil
	})

	return ledgers, err
}
