package portfolio

import (
	"path/filepath"
	"testing"
)

func TestDocumentation(t *testing.T) {
	files, err := filepath.Glob("*.md")
	if err != nil {
		t.Fatal(err)
	}

	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			runTestableCommands(t, file)
		})
	}
}