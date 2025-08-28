package portfolio

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// This file contains the logic to test the examples in the README.md file.
//
// To add a new testable example to the README.md file, you need to follow these steps:
//
// 1.  Add the command to the README.md file, wrapped in a ```bash ... ``` block.
// 2.  Add the expected output of the command, wrapped in a ```console ... ``` block.
//
// The test will automatically parse the README.md file, run the commands, and compare the output with the expected output.

// Command holds a command and its expected output.
type Command struct {
	Cmd      string
	Expected string
}

// buildPcs builds the pcs command and returns the path to the executable and a cleanup function.
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

// parseReadme parses the README.md file to extract commands and their expected outputs.
func parseReadme(t *testing.T) []Command {
	t.Helper()

	// Read the README.md file
	content, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("failed to read README.md: %v", err)
	}

	// Parse the README.md file
	repo := string(content)
	re := regexp.MustCompile("(?m)```bash\\n(pcs.*?)\n```\\n\\n```console\n((.|\\n)*?)```")
	matches := re.FindAllStringSubmatch(repo, -1)

	var commands []Command
	for _, match := range matches {
		commands = append(commands, Command{Cmd: match[1], Expected: match[2]})
	}

	return commands
}

func TestReadme(t *testing.T) {
	tmp := t.TempDir()
	pcsPath := buildPcs(t, tmp)

	commands := parseReadme(t)

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
