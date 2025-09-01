package portfolio

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Command holds a command and its expected output.
// Command represents a shell command to be executed and its expected output,
// used for testing purposes.
type Command struct {
	Cmd      string
	Expected string
}

// buildPcs builds the `pcs` command-line executable and returns the absolute
// path to the compiled binary. It uses a temporary directory for the build
// output.
func buildPcs(t *testing.T, tmp string) string {
	t.Helper()

	output := filepath.Join(tmp, "pcs")

	// Build the pcs command
	buildCmd := exec.Command("go", "build", "-o", output, "./pcs/")
	err := buildCmd.Run()
	if err != nil {
		t.Fatalf("failed to build pcs command: %v", err)
	}

	return output
}

// parseTestableCommands parses a markdown file (e.g., README.md) to extract
// shell commands and their corresponding expected console outputs. These are
// used to create testable `Command` structs.
func parseTestableCommands(t *testing.T, file string) []Command {
	t.Helper()

	// Read the file
	content, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("failed to read %s: %v", file, err)
	}

	// Parse the file
	repo := string(content)
	re := regexp.MustCompile("(?m)```bash\\n(pcs.*?)\\n```\\n\\n```console\n((.|\\n)*?)```")
	matches := re.FindAllStringSubmatch(repo, -1)

	var commands []Command
	for _, match := range matches {
		commands = append(commands, Command{Cmd: match[1], Expected: match[2]})
	}

	return commands
}

// runTestableCommands executes a series of shell commands extracted from a
// markdown file and compares their actual output against the expected output
// defined in the markdown. This function is used for integration testing
// of the `pcs` command-line tool.
func runTestableCommands(t *testing.T, file string) {
	t.Helper()

	tmp := t.TempDir()
	pcsPath := buildPcs(t, tmp)

	commands := parseTestableCommands(t, file)

	for _, cmd := range commands {
		args := strings.Fields(cmd.Cmd)
		t.Log("Running command:", pcsPath, args)
		command := exec.Command(pcsPath, args[1:]...)
		command.Dir = tmp
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("failed to run command: %v, output: \n%s", err, output)
		}
		result := string(output)
		// replace tabs with spaces for consistent comparison
		result = strings.ReplaceAll(result, "\t", "        ")

		if cmd.Expected != result {
			t.Errorf("expected output:\n%q\nbut got:\n%q", cmd.Expected, result)
		}
	}
}