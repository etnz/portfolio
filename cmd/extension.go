package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"syscall"
)

const (
	EnvMarketFile      = "PCS_MARKET_FILE"
	EnvLedgerFile      = "PCS_LEDGER_FILE"
	EnvDefaultCurrency = "PCS_DEFAULT_CURRENCY"
	EnvVerbose         = "PCS_VERBOSE"
)

// RunExtension attempts to find and execute an external pcs-<subcommand> binary.
// It returns (true, exitCode) if an extension was found and executed,
// and (false, 0) if no extension was found or executed.
func RunExtension(subcommand string, args []string) (bool, int) {
	externalCmdName := "pcs-" + subcommand

	// Look for the external command in PATH
	lp, err := exec.LookPath(externalCmdName)
	if err != nil {
		// Command not found in PATH
		log.Printf("External command %q not found in PATH: %v", externalCmdName, err)
		return false, 0
	}

	// Found external command, execute it
	cmd := exec.Command(lp, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Pass global flags as environment variables
	cmd.Env = os.Environ() // Start with existing environment variables
	cmd.Env = append(cmd.Env, EnvMarketFile+"="+*marketFile)
	cmd.Env = append(cmd.Env, EnvLedgerFile+"="+*ledgerFile)
	cmd.Env = append(cmd.Env, EnvDefaultCurrency+"="+*defaultCurrency)
	cmd.Env = append(cmd.Env, EnvVerbose+"="+strconv.FormatBool(*Verbose))

	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				return true, status.ExitStatus()
			}
		}
		// If it's not an ExitError or we can't get the status, report a generic error
		fmt.Fprintf(os.Stderr, "Error executing external command %q: %v\n", externalCmdName, err)

		return true, 1 // Indicate that an attempt was made, but it failed
	}

	return true, 0 // External command executed successfully with exit code 0
}
