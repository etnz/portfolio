package portfolio

import (
	"strings"
	"testing"
)

// TestImport creates a very basic check that imports is working as expected.
func TestImport(t *testing.T) {
	sample1 := `
{"ticker":"AAPL","id":"US0378331005","history":{"2025-07-29":195.5,"2025-07-30":196.25,"2025-07-31":198.1}}
{"ticker":"NVDA","id":"US67066G1040","history":{"2025-07-29":175.51,"2025-07-30":178.9,"2025-07-31":177.85}}
`

	sample1 = strings.Trim(sample1, "\n\t")

	securities := NewSecurities()
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
