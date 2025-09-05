package portfolio

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestDocumentation(t *testing.T) {
	files, err := filepath.Glob("docs/*.md")
	if err != nil {
		t.Fatal(err)
	}
	files = append(files, "README.md")

	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			runTestableCommands(t, file)
		})
	}
}

func TestTopics(t *testing.T) {
	// This test ensures that the documentation is in sync with the code.
	// It checks two things:
	// 1. Every topic listed in docs/readme.md can be successfully loaded by the pcs topic <topic_name> command.
	// 2. Every .md file in the docs directory (excluding readme.md itself) is present in the list of topics extracted from docs/readme.md.

	// Read docs/readme.md line by line and extract topics using regex.
	file, err := os.Open("docs/readme.md")
	if err != nil {
		t.Fatalf("failed to open docs/readme.md: %v", err)
	}
	defer file.Close()

	var topicsInReadme []string
	scanner := bufio.NewScanner(file)
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
		t.Fatalf("error scanning docs/readme.md: %v", err)
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
	files, err := filepath.Glob("docs/*.md")
	if err != nil {
		t.Fatalf("failed to glob docs/*.md: %v", err)
	}

	var mdFiles []string
	for _, file := range files {
		base := filepath.Base(file)
		if base != "readme.md" {
			mdFiles = append(mdFiles, strings.TrimSuffix(base, ".md"))
		}
	}

	for _, mdFile := range mdFiles {
		found := false
		for _, topic := range topicsInReadme {
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