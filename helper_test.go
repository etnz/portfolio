package portfolio

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

const (
	bashSetup    = "bash setup"
	bashRun      = "bash run"
	consoleCheck = "console check"
	bashCheck    = "bash check"
)

// Block represents a fenced code block in the markdown file.
type Block struct {
	Type    string
	Content string
	File    string
	Line    int
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

// parseMarkdown parses a markdown file and returns a list of scenarios.
func parseMarkdown(t *testing.T, file string) []*Block {
	t.Helper()

	content, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("failed to read %s: %v", file, err)
	}

	mdParser := goldmark.DefaultParser()
	root := mdParser.Parse(text.NewReader(content))

	// Read all blocks.

	var blocks []*Block

	ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		if fcb, ok := n.(*ast.FencedCodeBlock); ok {
			if fcb.Info == nil {
				return ast.WalkContinue, nil
			}
			lang := string(fcb.Info.Segment.Value(content))

			// lang := string(fcb.Language(content))
			var blockContent strings.Builder
			for i := 0; i < fcb.Lines().Len(); i++ {
				line := fcb.Lines().At(i)
				blockContent.WriteString(string(line.Value(content)))
			}

			// Get the line number of the block
			startOffset := fcb.Info.Segment.Start

			switch lang {
			case bashCheck, bashSetup, bashRun, consoleCheck:
				blocks = append(blocks, &Block{
					Type:    lang,
					Content: blockContent.String(),
					File:    file,
					Line:    lineNumber(content, startOffset),
				})
			}
		}
		return ast.WalkContinue, nil
	})

	return blocks
}

// lineNumber computes the lineNumber for a given offset AST offset
func lineNumber(source []byte, offset int) (lineNumber int) {
	newline := []byte{'\n'}
	// Create a slice of the source from the beginning to the node's offset.
	sourceToNode := source[:offset]

	// Count the number of newlines in that slice.
	lineCount := bytes.Count(sourceToNode, newline)

	// The line number is the number of newlines + 1.
	return lineCount + 1
}

type runner struct {
	env            []string // env use to execute commands
	previousOutput string
	tmpFolder      string
}

func (r *runner) runBlock(t *testing.T, block *Block) {
	t.Helper()

	// Check don't need execution.
	if block.Type == consoleCheck {
		want := strings.TrimSpace(block.Content)
		got := strings.TrimSpace(r.previousOutput)
		// replace tabs with spaces for consistent comparison
		got = strings.ReplaceAll(got, "\t", "        ")
		if want != got {
			// Print out the diffs.
			t.Errorf("%s:%d: output mismatch:\ngot:\n\n%s\n\nwant:\n\n%s\n\ngot :%q\nwant:%q\n", block.File, block.Line, got, want, got, want)
		}
		return
	}
	// Create a new execution folder on a new setup.
	if block.Type == bashSetup {
		r.tmpFolder = t.TempDir() // new scenario temp folder
	}

	// Execute bash.
	cmd := exec.Command("bash", "-c", "set -e; "+block.Content)
	cmd.Dir = r.tmpFolder
	cmd.Env = r.env
	output, err := cmd.CombinedOutput()

	// Record last run output.
	if block.Type == bashRun {
		r.previousOutput = string(output)
	}

	// Handling bash errors.
	if err != nil {
		switch block.Type {
		case bashSetup, bashRun:
			t.Fatalf("%s:%d: %s failed: %v with output:\n%s\n", block.File, block.Line, block.Type, err, output)
		case bashCheck:
			t.Errorf("%s:%d: %s failed: %v with output:\n%s\n", block.File, block.Line, block.Type, err, output)
			return
		default:
			t.Fatalf("%s:%d: unknown block type: %s", block.File, block.Line, block.Type)
		}
	}
}

// runTestableCommands executes a series of scenarios extracted from a
// markdown file.
func runTestableCommands(t *testing.T, file string) {
	t.Helper()

	globalTmp := t.TempDir()
	pcsPath := buildPcs(t, globalTmp)
	pcsDir := filepath.Dir(pcsPath)

	newPath := fmt.Sprintf("PATH=%s%c%s", pcsDir, os.PathListSeparator, os.Getenv("PATH"))
	baseEnv := append(os.Environ(), newPath)

	blocks := parseMarkdown(t, file)
	if len(blocks) == 0 {
		return
	}

	r := runner{
		env:       baseEnv,
		tmpFolder: t.TempDir(),
	}
	for _, block := range blocks {
		r.runBlock(t, block)
	}
}

func must[T any](a T, err error) T {
	if err != nil {
		panic(err)
	}
	return a
}
