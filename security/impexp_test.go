package security

import (
	"strings"
	"testing"
)

// TestImport creates a very basic check that imports is working as expected.
func TestImport(t *testing.T) {
	sample1 := `
	{"ticker1":{"id":"LU345346445.XTRA","history":{"2025-01-01":123.12,"2025-01-02":124.12}}}
	`
	sample1 = strings.Trim(sample1, "\n\t")

	securities := New()
	if err := securities.Import(strings.NewReader(sample1)); err != nil {
		t.Errorf("cannot import sample 1: %v", err)
	}

	sb := strings.Builder{}
	if err := securities.Export(&sb); err != nil {
		t.Errorf("Export() has error %v", err)
	}
	got := sb.String()
	got = strings.Trim(got, "\n\t")

	if got != sample1 {
		t.Errorf("export/import sequence is not stable got \n%s\n want \n%s\n", got, sample1)
	}
}
