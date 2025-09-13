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
