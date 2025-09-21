package cmd

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestExtensionMechanism(t *testing.T) {
	// 1. Create a temporary directory
	tempDir := t.TempDir()

	// 2. Create pcs-hello executable
	helloCmdSource := fmt.Sprintf(`
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Printf("%s=%%s\n", os.Getenv("%s"))
	fmt.Printf("%s=%%s\n", os.Getenv("%s"))
	fmt.Printf("%s=%%s\n", os.Getenv("%s"))
}
`, EnvLedgerFile, EnvLedgerFile, EnvDefaultCurrency, EnvDefaultCurrency, EnvVerbose, EnvVerbose)

	helloCmdPath := filepath.Join(tempDir, "pcs-hello")

	// Write source to a temporary file
	srcFile := helloCmdPath + ".go"
	if err := os.WriteFile(srcFile, []byte(helloCmdSource), 0644); err != nil {
		t.Fatalf("Failed to write pcs-hello source: %v", err)
	}
	log.Printf("Written pcs-hello source to %s", srcFile)

	// Compile pcs-hello
	cmd := exec.Command("go", "build", "-o", helloCmdPath, srcFile)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to compile pcs-hello: %v", err)
	}
	log.Printf("Compiled pcs-hello to %s", helloCmdPath)

	// 3. Compile the main pcs binary
	pcsBinaryPath := filepath.Join(tempDir, "pcs")
	cmd = exec.Command("go", "build", "-o", pcsBinaryPath, "../pcs")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to compile pcs binary: %v", err)
	}
	log.Printf("Compiled pcs binary to %s", pcsBinaryPath)

	// Define random values for global flags
	expectedLedgerFile := filepath.Join(tempDir, "random_ledger.jsonl")
	expectedDefaultCurrency := "XYZ"
	expectedVerbose := true

	// 5. Call pcs binary with extension and global flags
	args := []string{
		"--ledger-file", expectedLedgerFile,
		"--default-currency", expectedDefaultCurrency,
		"-v",
		"hello", // The extension subcommand
	}

	// Use the compiled pcs binary directly
	pcsCmd := exec.Command(pcsBinaryPath, args...)
	//pcsCmd.Env = os.Environ() // Inherit current environment
	oldPath := os.Getenv("PATH")
	pcsCmd.Env = []string{"PATH=" + tempDir + string(os.PathListSeparator) + oldPath}
	log.Printf("set Env=%s", pcsCmd.Env)

	var stdout, stderr bytes.Buffer
	pcsCmd.Stdout = &stdout
	pcsCmd.Stderr = &stderr

	if err := pcsCmd.Run(); err != nil {
		t.Fatalf("pcs command failed: %v\nStdout: %s\nStderr: %s", err, stdout.String(), stderr.String())
	}

	// 6. Verify output
	output := stdout.String()

	expectedEnvVars := []struct {
		Name  string
		Value string
	}{
		{EnvLedgerFile, expectedLedgerFile},
		{EnvDefaultCurrency, expectedDefaultCurrency},
		{EnvVerbose, strconv.FormatBool(expectedVerbose)},
	}

	for _, ev := range expectedEnvVars {
		expectedLine := fmt.Sprintf("%s=%s", ev.Name, ev.Value)
		if !strings.Contains(output, expectedLine) {
			t.Errorf("Expected output to contain %q, but got:\n%s", expectedLine, output)
		}
	}

	if stderr.Len() > 0 {
		t.Logf("Stderr from pcs command: %s", stderr.String())
	}
}

func TestFetchExtensionMechanism(t *testing.T) {
	// Arrange: Set up the test environment.
	// Create a temporary directory for the test artifacts.
	tempDir := t.TempDir()
	// Add the temporary directory to the PATH so the pcs binary can find the extension.
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", tempDir+string(os.PathListSeparator)+oldPath)
	defer os.Setenv("PATH", oldPath)

	// Create a dummy ledger file that declares a security, which will trigger a fetch request.
	ledgerContent := `{"command":"init","date":"2024-01-01","currency":"EUR"}
{"command":"declare","date":"2024-01-01","ticker":"DUMMY","id":"DUMMY.TICKER","currency":"EUR","memo":"A dummy security"}
{"command":"buy","date":"2024-01-02","security":"DUMMY","quantity":10,"amount":100}
`
	ledgerPath := filepath.Join(tempDir, "test_ledger.jsonl")
	if err := os.WriteFile(ledgerPath, []byte(ledgerContent), 0644); err != nil {
		t.Fatalf("Failed to write dummy ledger: %v", err)
	}

	// Define the source code for the dummy external provider.
	// This provider will check environment variables and return a hardcoded price.
	dummyProviderSource := `
package main

import (
	"encoding/json"
	"fmt"
	"os"
	
	"github.com/etnz/portfolio"
)

func main() {
	// For testing: print env vars to stderr so the main test can check them.
	fmt.Fprintf(os.Stderr, "PCS_LEDGER_FILE=%s\n", os.Getenv("PCS_LEDGER_FILE"))
	fmt.Fprintf(os.Stderr, "PCS_DEFAULT_CURRENCY=%s\n", os.Getenv("PCS_DEFAULT_CURRENCY"))

	// Read requests from stdin
	var requests map[portfolio.ID]portfolio.Range
	if err := json.NewDecoder(os.Stdin).Decode(&requests); err != nil {
		fmt.Fprintf(os.Stderr, "Go dummy: provider failed to decode requests: %v", err)
		os.Exit(1)
	}

	// For testing: print received requests to stderr
	for id := range requests {
		fmt.Fprintf(os.Stderr, "Go dummy: Received request for ID: %s\n", id)
	}

	// Prepare a hardcoded response
	resp := "{\"DUMMY.TICKER\":{ \"Prices\":{\"2024-01-03\":11.5}}}"
	fmt.Println(resp)
	// log it to stderr for testing
	fmt.Fprintln(os.Stderr, "Go dummy: Response:", resp)
}
`
	// Compile the dummy provider into an executable named 'pcs-fetch-dummy'.
	providerPath := filepath.Join(tempDir, "pcs-fetch-dummy")
	srcFile := providerPath + ".go"
	if err := os.WriteFile(srcFile, []byte(dummyProviderSource), 0644); err != nil {
		t.Fatalf("Failed to write pcs-fetch-dummy source: %v", err)
	}
	compileCmd := exec.Command("go", "build", "-o", providerPath, srcFile)
	compileCmd.Stderr = os.Stderr
	if err := compileCmd.Run(); err != nil {
		t.Fatalf("Failed to compile pcs-fetch-dummy: %v", err)
	}
	log.Println("Compiled pcs-fetch-dummy to", providerPath)

	// Compile the main pcs binary to be used in the test.
	pcsBinaryPath := filepath.Join(tempDir, "pcs")
	compileCmd = exec.Command("go", "build", "-o", pcsBinaryPath, "../pcs")
	compileCmd.Stderr = os.Stderr
	if err := compileCmd.Run(); err != nil {
		t.Fatalf("Failed to compile pcs binary: %v", err)
	}

	// Act: Execute the 'pcs fetch' command, which should invoke the dummy provider.
	args := []string{
		"--ledger-file", ledgerPath,
		"-v",
		"fetch",
		"dummy",
	}
	pcsCmd := exec.Command(pcsBinaryPath, args...)
	pcsCmd.Env = os.Environ() // Inherit PATH which was modified at the start

	var stdout, stderr bytes.Buffer
	pcsCmd.Stdout = &stdout
	pcsCmd.Stderr = &stderr

	if err := pcsCmd.Run(); err != nil {
		t.Fatalf("pcs command failed: %v\nStdout: %s\nStderr: %s", err, stdout.String(), stderr.String())
	}
	stderrOutput := stderr.String()

	log.Println("extension output:", stderrOutput)

	// Assert: Verify the outcomes.
	// Check that the provider received the correct env vars
	absLedgerPath, _ := filepath.Abs(ledgerPath)
	if !strings.Contains(stderrOutput, "PCS_LEDGER_FILE="+absLedgerPath) {
		t.Errorf("Expected stderr to contain ledger file env var, but it didn't. Stderr:\n%s", stderrOutput)
	}
	if !strings.Contains(stderrOutput, "PCS_DEFAULT_CURRENCY=EUR") {
		t.Errorf("Expected stderr to contain default currency env var, but it didn't. Stderr:\n%s", stderrOutput)
	}

	// Check that the ledger file was updated
	updatedContent, _ := os.ReadFile(ledgerPath)
	if !strings.Contains(string(updatedContent), `"command":"update-price","date":"2024-01-03"`) {
		t.Errorf("Expected ledger file to be updated with the new price. Content:\n%s", string(updatedContent))
	}
}
