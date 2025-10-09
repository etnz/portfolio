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

func fix() bool {
	// return true
	return *fixDocs
}

//go:embed *.md
var testDocs embed.FS

const (
	bashSetup    = "bash setup"    // creates a new environment, run the bash, fail on error
	bashDemo     = "bash demo"     // creates a new environment, run the bash, fail on error, remember the output
	bashRun      = "bash run"      //                            run the bash, fail on error, remember the output
	consoleCheck = "console check" //                            compare content with the latest output.
	bashCheck    = "bash check"    //                            run the bash, fail on error
)

func TestFixModeIsOff(t *testing.T) {
	if fix() {
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
		t.Run("exist_"+topic, func(t *testing.T) {
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
		t.Run("declared_"+mdFile, func(t *testing.T) {
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
		})
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
	Type        string
	Content     string
	File        string
	Line        int
	EndLine     int
	StartOffset int
	EndOffset   int
	Padding     int
	Edited      bool
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
			lines := fcb.BaseBlock.Lines()
			startLine := lineNumber(content, fcb.Info.Segment.Start)
			startOffset := lines.At(0).Start
			endOffset := lines.At(lines.Len() - 1).Stop
			padding := padding(content, startOffset)

			switch lang {
			case bashCheck, bashSetup, bashRun, consoleCheck, bashDemo:
				blocks = append(blocks, &Block{
					Type:        lang,
					Content:     blockContent.String(),
					File:        file,
					Line:        startLine,
					EndLine:     startLine + lines.Len() - 1,
					StartOffset: startOffset,
					EndOffset:   endOffset,
					Padding:     padding,
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

// padding computes the indentation (number of spaces) for a given offset.
// It counts the number of space characters from the beginning of the line
// up to the given offset.
func padding(source []byte, offset int) int {
	if offset > len(source) {
		offset = len(source)
	}

	// Find the start of the current line.
	lineStart := 0
	if lastNewline := bytes.LastIndexByte(source[:offset], '\n'); lastNewline != -1 {
		lineStart = lastNewline + 1
	}

	// The indentation is the part of the line from its start to the offset.
	indentationSlice := source[lineStart:offset]

	return bytes.Count(indentationSlice, []byte(" "))
}

// blockRunner defines all that is need to run a test for a block
type blockRunner struct {
	env            []string // env use to execute commands
	previousOutput string
	cwd            string
}

func (r *blockRunner) runBlock(t *testing.T, block *Block) {
	t.Helper()

	// Check don't need execution.
	if block.Type == consoleCheck {
		want := block.Content
		got := trimEmptyLines(r.previousOutput)
		if strings.TrimSpace(want) != strings.TrimSpace(got) {
			if fix() {
				t.Logf("fixing %s:%d-%d padding=%d", block.File, block.Line, block.EndLine, block.Padding)
				block.Content = got
				block.Edited = true
			}
			// In normal mode, we report an error.
			t.Errorf("%s:%d: output mismatch:\ngot:\n%s\nwant:\n%s\n\n got=%q\nwant=%q\n", block.File, block.Line, got, want, got, want)
		}
		return
	}
	// Create a new execution folder on a new setup.
	if block.Type == bashSetup || block.Type == bashDemo {
		r.cwd = t.TempDir() // new scenario temp folder
	}

	// Execute bash.
	cmd := exec.Command("bash", "-c", "set -e; "+block.Content)
	cmd.Dir = r.cwd
	cmd.Env = append(r.env, "PORTFOLIO_PATH="+r.cwd)
	output, err := cmd.CombinedOutput()

	// Record last run output.
	if block.Type == bashRun || block.Type == bashDemo {
		r.previousOutput = strings.TrimSpace(string(output))
	}

	// Handling bash errors.
	if err != nil {
		switch block.Type {
		case bashSetup, bashRun:
			// other blocks may have dependencies on that execution, so it cannot be a simple "error"
			t.Fatalf("%s:%d: %s failed: %v with output:\n%s\n", block.File, block.Line, block.Type, err, output)
		case bashCheck, bashDemo:
			// nobody depends on those, so failure is an error, but other blocks can be tested.
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
	baseEnv := append(os.Environ(),
		newPath,
		"PORTFOLIO_TESTING_NOW=2006-01-02 15:04:05",
		"NO_RENDER=1",
	)

	blocks := parseMarkdown(t, file, md)
	if len(blocks) == 0 {
		return
	}

	cwd := t.TempDir()
	r := blockRunner{
		env: baseEnv,
		cwd: cwd,
	}
	for _, block := range blocks {
		r.runBlock(t, block)
	}

	if fix() {
		// file is relative to workspace for presentation reasons, but the test is
		// executed in ${workspace}/docs, hence the ".."
		rewriteMarkdown(t, filepath.Join("..", file), md, blocks)
	}
}

// rewriteMarkdown writes the updated content of the blocks back to the markdown file.
func rewriteMarkdown(t *testing.T, file string, originalContent string, blocks []*Block) {
	t.Helper()

	content := []byte(originalContent)
	var hasEdits bool

	// Iterate backwards to not invalidate offsets
	for i := len(blocks) - 1; i >= 0; i-- {
		block := blocks[i]
		if block.Edited {
			hasEdits = true
			// Replace the old block content with the new content.
			// This is done by concatenating the part of the file before the block,
			// the new block content, and the part of the file after the block.

			// New lines need to be padded by block.Padding amount (except the first one).
			padding := strings.Repeat(" ", block.Padding)
			lines := strings.Split(strings.TrimSpace(block.Content), "\n")

			paddedContent := strings.Join(lines, "\n"+padding)
			paddedContent += "\n"

			prefix := content[:block.StartOffset]
			suffix := content[block.EndOffset:]
			content = append(prefix, append([]byte(paddedContent), suffix...)...)
		}
	}

	if hasEdits {
		if err := os.WriteFile(file, content, 0644); err != nil {
			t.Fatalf("failed to write updated markdown to %s: %v", file, err)
		}
		t.Logf("updated %s", file)
	}
}

// trimEmptyLines takes a string, splits it into lines, and trims lines that
// consist only of whitespace to an empty string. This helps normalize command
// output for comparison against markdown blocks.
func trimEmptyLines(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			lines[i] = ""
		}
	}
	return strings.Join(lines, "\n")
}
