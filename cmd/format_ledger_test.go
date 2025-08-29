package cmd

import (
	"context"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/subcommands"
)

// Helper function to create a temporary ledger file
func createTempLedger(t *testing.T, content string) string {
	t.Helper()
	tmp := t.TempDir()
	tmpfile, err := os.Create(filepath.Join(tmp, "test_ledger.jsonl"))
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer tmpfile.Close()

	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	return tmpfile.Name()
}

// TestFormatLedgerDefaultOutput tests the default behavior (writes to default ledger file)
func TestFormatLedgerDefaultOutput(t *testing.T) {
	// Arrange
	originalLedgerContent := `{"command":"deposit","date":"2025-08-01","amount":1000, "memo":"this is a comment"}
{"command":"buy","date":"2025-08-03","security":"AAPL","quantity":10,"amount":1500}
`
	expectedFormattedContent := `{"command":"deposit","date":"2025-08-01","memo":"this is a comment","amount":1000}
{"command":"buy","date":"2025-08-03","security":"AAPL","quantity":10,"amount":1500}
`

	// Create a temporary default ledger file
	tempLedgerFile := createTempLedger(t, originalLedgerContent)

	cmd := &formatLedgerCmd{}
	f := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(f)

	// Override global ledgerFile for the test
	oldLedgerFile := ledgerFile
	ledgerFile = &tempLedgerFile
	defer func() { ledgerFile = oldLedgerFile }()

	// Act
	status := cmd.Execute(context.Background(), f)

	// Assert
	if status != subcommands.ExitSuccess {
		t.Errorf("Expected ExitSuccess, got %v", status)
	}

	// Read the content of the (now formatted) temporary ledger file
	gotContent, err := os.ReadFile(tempLedgerFile)
	if err != nil {
		t.Fatalf("Failed to read formatted ledger file: %v", err)
	}

	if strings.TrimSpace(string(gotContent)) != strings.TrimSpace(expectedFormattedContent) {
		t.Errorf("Default output mismatch.\nGot:\n%s\nWant:\n%s", string(gotContent), expectedFormattedContent)
	}
}

// TestFormatLedgerToFileOutput tests writing to a specified output file
func TestFormatLedgerToFileOutput(t *testing.T) {
	// Arrange
	originalLedgerContent := `{"command":"deposit","date":"2025-08-01","amount":1000}
{"command":"buy","date":"2025-08-03","security":"AAPL","quantity":10,"amount":1500}
`
	expectedFormattedContent := `{"command":"deposit","date":"2025-08-01","amount":1000}
{"command":"buy","date":"2025-08-03","security":"AAPL","quantity":10,"amount":1500}
`

	// Create a temporary input ledger file
	tempInputLedger := createTempLedger(t, originalLedgerContent)

	// Create a temporary output file path
	tempOutputFile := filepath.Join(t.TempDir(), "test_output.jsonl")

	cmd := &formatLedgerCmd{}
	f := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(f)
	f.Set("o", tempOutputFile) // Set the output file flag

	// Override global ledgerFile for the test (input)
	oldLedgerFile := ledgerFile
	ledgerFile = &tempInputLedger
	defer func() { ledgerFile = oldLedgerFile }()

	// Act
	status := cmd.Execute(context.Background(), f)

	// Assert
	if status != subcommands.ExitSuccess {
		t.Errorf("Expected ExitSuccess, got %v", status)
	}

	// Read the content of the output file
	gotContent, err := os.ReadFile(tempOutputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	if strings.TrimSpace(string(gotContent)) != strings.TrimSpace(expectedFormattedContent) {
		t.Errorf("File output mismatch.\nGot:\n%s\nWant:\n%s", string(gotContent), expectedFormattedContent)
	}
}

// TestFormatLedgerToStdoutOutput tests writing to stdout
func TestFormatLedgerToStdoutOutput(t *testing.T) {
	// Arrange
	originalLedgerContent := `{"command":"deposit","date":"2025-08-01","amount":1000}
{"command":"buy","date":"2025-08-03","security":"AAPL","quantity":10,"amount":1500}
`
	expectedFormattedContent := `{"command":"deposit","date":"2025-08-01","amount":1000}
{"command":"buy","date":"2025-08-03","security":"AAPL","quantity":10,"amount":1500}
`

	// Create a temporary input ledger file
	tempInputLedger := createTempLedger(t, originalLedgerContent)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
	}()

	cmd := &formatLedgerCmd{}
	f := flag.NewFlagSet("test", flag.ContinueOnError)
	cmd.SetFlags(f)
	f.Set("o", "-") // Set the output to stdout

	// Override global ledgerFile for the test (input)
	oldLedgerFile := ledgerFile
	ledgerFile = &tempInputLedger
	defer func() { ledgerFile = oldLedgerFile }()

	// Act
	status := cmd.Execute(context.Background(), f)

	// Assert
	w.Close() // Close the write end of the pipe
	gotOutput, _ := io.ReadAll(r)

	if status != subcommands.ExitSuccess {
		t.Errorf("Expected ExitSuccess, got %v", status)
	}

	gotFormattedContent := string(gotOutput)

	if strings.TrimSpace(gotFormattedContent) != strings.TrimSpace(expectedFormattedContent) {
		t.Errorf("Stdout output mismatch.\nGot:\n%s\nWant:\n%s", gotFormattedContent, expectedFormattedContent)
	}
}
