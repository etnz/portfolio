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
	readme := string(content)
	re := regexp.MustCompile("(?m)```bash\\n(pcs.*?)\\n```\\n\\n```console\\n((.|\\n)*?)```")
	matches := re.FindAllStringSubmatch(readme, -1)

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
		t.Run(cmd.Cmd, func(t *testing.T) {
			args := strings.Fields(cmd.Cmd)
			t.Log("Running command:", pcsPath, args)
			command := exec.Command(pcsPath, args[1:]...)
			output, err := command.CombinedOutput()
			if err != nil {
				t.Fatalf("failed to run command: %v, output: %s", err, output)
			}
			result := string(output)
			// replace tabs with spaces for consistent comparison
			result = strings.ReplaceAll(result, "\t", "        ")

			if cmd.Expected != result {
				t.Errorf("expected output:\n%q\nbut got:\n%q", cmd.Expected, result)
			}
		})
	}
}
