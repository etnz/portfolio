package docs

import (
	"bufio"
	"bytes"
	"embed"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

var fixDocs = flag.Bool("fix-docs", false, "if true, update docs instead of checking them")

//go:embed *.md
var testDocs embed.FS

const (
	bashSetup    = "bash setup"
	bashRun      = "bash run"
	consoleCheck = "console check"
	bashCheck    = "bash check"
)

func TestFixModeIsOff(t *testing.T) {
	if *fixDocs {
		t.Fatal("-fix-docs is enabled. This flag should only be used for updating documentation and must be disabled for regular tests.")
	}
}

func TestTopics(t *testing.T) {
	// This test ensures that the documentation is in sync with the code.
	// It checks two things:
	// 1. Every topic listed in docs/readme.md can be successfully loaded by the pcs topic <topic_name> command.
	// 2. Every .md file in the docs directory (excluding readme.md itself) is present in the list of topics extracted from docs/readme.md.

	// Read readme.md from embedded fs and extract topics using regex.
	content, err := testDocs.ReadFile("readme.md")
	if err != nil {
		t.Fatalf("failed to read readme.md from embed.FS: %v", err)
	}
	scanner := bufio.NewScanner(bytes.NewReader(content))

	var topicsInReadme []string
	topicRegex := regexp.MustCompile(`^\*\s+([^:]+):.*$`)

	for scanner.Scan() {
		line := scanner.Text()
		matches := topicRegex.FindStringSubmatch(line)
		if len(matches) > 1 {
			topic := strings.TrimSpace(matches[1])
			topicsInReadme = append(topicsInReadme, topic)
		}
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("error scanning readme.md: %v", err)
	}

	// Check 1: Every topic listed in docs/readme.md can be successfully loaded.
	for _, topic := range topicsInReadme {
		t.Run("load_"+topic, func(t *testing.T) {
			_, err := GetTopic(topic)
			if err != nil {
				t.Errorf("failed to get topic %q: %v", topic, err)
			}
		})
	}

	// Check 2: Every .md file in the docs directory (excluding readme.md itself) is present in the list of topics extracted from docs/readme.md.
	mdFiles, err := GetAllTopics()
	if err != nil {
		t.Fatalf("failed to get all topics: %v", err)
	}

	for _, mdFile := range mdFiles {
		found := false
		for _, topic := range topicsInReadme { // TODO: this is O(n^2), should be O(n) with a map
			if topic == mdFile {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("topic %q is not listed in docs/readme.md", mdFile)
		}
	}
}

func TestCodeBlocks(t *testing.T) {
	topics, err := GetAllTopics()
	if err != nil {
		t.Fatal(err)
	}
	topics = append(topics, "readme") // Add readme to the list of files to test

	for _, topic := range topics {
		t.Run(topic, func(t *testing.T) {
			content, err := GetTopic(topic)
			if err != nil {
				t.Fatalf("failed to get topic %q: %v", topic, err)
			}
			filePath := filepath.Join("docs", topic+".md")
			runBlocks(t, filePath, content)
		})
	}

	// Handle ../README.md separately
	readmeContent, err := os.ReadFile("../README.md")
	if err != nil {
		t.Fatalf("failed to read ../README.md: %v", err)
	}
	runBlocks(t, "README.md", string(readmeContent))
}

// HELPER

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
	buildCmd := exec.Command("go", "build", "-o", output, "../pcs/")
	err := buildCmd.Run()
	if err != nil {
		t.Fatalf("failed to build pcs command: %v", err)
	}

	return output
}

// parseMarkdown parses a markdown file and returns a list of Blocks.
func parseMarkdown(t *testing.T, file string, md string) []*Block {
	t.Helper()

	content := []byte(md)

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

// lineNumber computes the lineNumber for a given offset AST offset.
// the markdown parser we use does not support that feature so we
// have to implement it.
func lineNumber(source []byte, offset int) (lineNumber int) {
	newline := []byte{'\n'}
	// Create a slice of the source from the beginning to the node's offset.
	sourceToNode := source[:offset]

	// Count the number of newlines in that slice.
	lineCount := bytes.Count(sourceToNode, newline)

	// The line number is the number of newlines + 1.
	return lineCount + 1
}

// blockRunner defines all that is need to run a test for a block
type blockRunner struct {
	env            []string // env use to execute commands
	previousOutput string
	tmpFolder      string
}

func (r *blockRunner) runBlock(t *testing.T, block *Block) {
	t.Helper()

	// Check don't need execution.
	if block.Type == consoleCheck {
		want := block.Content
		got := r.previousOutput
		if strings.TrimSpace(want) != strings.TrimSpace(got) {
			if *fixDocs {
				t.Logf("fixing %s:%d", block.File, block.Line)
				// In fix mode, we update the file.
				// file path are relative to the workspace not the test
				filepath := filepath.Join("..", block.File)
				content, err := os.ReadFile(filepath)
				if err != nil {
					t.Fatalf("failed to read file for fixing: %v", err)
				}
				newContent := strings.Replace(string(content), "```console check\n"+want+"```", "```console check\n"+got+"```", 1)
				if err := os.WriteFile(filepath, []byte(newContent), 0644); err != nil {
					t.Fatalf("failed to write fixed file: %v", err)
				}
			} else {
				// In normal mode, we report an error.
				t.Errorf("%s:%d: output mismatch:\ngot:\n%s\nwant:\n%s", block.File, block.Line, got, want)
			}
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

// runBlocks executes a series of scenarios extracted from a
// markdown file.
func runBlocks(t *testing.T, file string, md string) {
	t.Helper()
	// override the Now function in renderer to have a stable fake date.

	globalTmp := t.TempDir()
	pcsPath := buildPcs(t, globalTmp)
	pcsDir := filepath.Dir(pcsPath)

	newPath := fmt.Sprintf("PATH=%s%c%s", pcsDir, os.PathListSeparator, os.Getenv("PATH"))
	baseEnv := append(os.Environ(), newPath, "PORTFOLIO_TESTING_NOW=2006-01-02 15:04:05")

	blocks := parseMarkdown(t, file, md)
	if len(blocks) == 0 {
		return
	}

	r := blockRunner{
		env:       baseEnv,
		tmpFolder: t.TempDir(),
	}
	for _, block := range blocks {
		r.runBlock(t, block)
	}
}
