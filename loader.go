package portfolio

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// FindLedgers discovers and loads ledger files from a given portfolio path.
// The query string can be used to filter which ledgers are loaded.
// If query is empty, all ledgers (.jsonl files) in the path are loaded.
// If query specifies a ledger name (e.g., "john/bnp"), only that ledger is loaded.
// A ledger name is its relative path from the portfolio path, without the .jsonl extension.
func FindLedgers(path, query string) ([]*Ledger, error) {
	ledgerPaths, err := findLedgerPaths(path)
	if err != nil {
		return nil, err
	}

	var filesToLoad []string
	if query == "" {
		// Load all ledgers
		for _, p := range ledgerPaths {
			filesToLoad = append(filesToLoad, p)
		}
	} else {
		// Load specific ledger
		p, ok := ledgerPaths[query]
		if !ok {
			return nil, fmt.Errorf("ledger %q not found in portfolio %q", query, path)
		}
		filesToLoad = append(filesToLoad, p)
	}

	var loadedLedgers []*Ledger
	for _, fullPath := range filesToLoad {
		relPath, err := filepath.Rel(path, fullPath)
		if err != nil {
			return nil, fmt.Errorf("could not determine relative path for %q: %w", fullPath, err)
		}
		ledgerName := strings.TrimSuffix(relPath, ".jsonl")

		f, err := os.Open(fullPath)
		if err != nil {
			return nil, fmt.Errorf("could not open ledger file %q: %w", fullPath, err)
		}

		ledger, err := DecodeLedger(f)
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("could not decode ledger file %q: %w", fullPath, err)
		}
		ledger.name = ledgerName
		loadedLedgers = append(loadedLedgers, ledger)
	}

	return loadedLedgers, nil
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
func findLedgerPaths(path string) (map[string]string, error) {
	ledgers := make(map[string]string)

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
			ledgers[ledgerName] = p
		}
		return nil
	})

	return ledgers, err
}
