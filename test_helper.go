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
type Command struct {
	Cmd      string
	Expected string
}

// buildPcs builds the pcs command and returns the path to the executable.
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

// parseTestableCommands parses a markdown file to extract commands and their expected outputs.
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

// runTestableCommands runs the testable commands from a given markdown file.
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