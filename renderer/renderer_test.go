package renderer

import (
	"bytes"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
)

//go:embed testdata/*.json
var testcasesFS embed.FS

//go:embed testdata/*.md
var testcasesGoldenFS embed.FS

var fixPartials = flag.Bool("fix-partials", false, "if true, update failing partial test case .md files with the received output")

func TestFixPartialsIsOff(t *testing.T) {
	if *fixPartials {
		t.Fatal("-fix-partials is enabled. This flag should only be used for updating test fixtures and must be disabled for regular tests.")
	}
}

func TestTemplatePartials(t *testing.T) {
	testCases := []struct {
		name       string
		structFile string
		goldenFile string
		dataType   any
	}{
		{
			name:       "review_title",
			structFile: "testdata/review_title.json",
			goldenFile: "testdata/review_title.md",
			dataType:   &Review{},
		},
		{
			name:       "review_summary",
			structFile: "testdata/review_summary.json",
			goldenFile: "testdata/review_summary.md",
			dataType:   &Review{},
		},
		{
			name:       "review_accounts",
			structFile: "testdata/review_accounts.json",
			goldenFile: "testdata/review_accounts.md",
			dataType:   &Review{},
		},
		{
			name:       "review_transactions",
			structFile: "testdata/review_transactions.json",
			goldenFile: "testdata/review_transactions.md",
			dataType:   &Review{},
		},
		{
			name:       "review_asset_view_consolidated",
			structFile: "testdata/review_asset_view_consolidated.json",
			goldenFile: "testdata/review_asset_view_consolidated.md",
			dataType:   &Review{},
		},
		{
			name:       "review_asset_view_simplified",
			structFile: "testdata/review_asset_view_simplified.json",
			goldenFile: "testdata/review_asset_view_simplified.md",
			dataType:   &Review{},
		},
		{
			name:       "review_transaction_skipped",
			structFile: "testdata/review_transaction_skipped.json",
			goldenFile: "testdata/review_transaction_skipped.md",
			dataType:   &Review{},
		},
		{
			name:       "consolidated_review_title",
			structFile: "testdata/consolidated_review_title.json",
			goldenFile: "testdata/consolidated_review_title.md",
			dataType:   &ConsolidatedReview{},
		},
		{
			name:       "consolidated_review_summary",
			structFile: "testdata/consolidated_review_summary.json",
			goldenFile: "testdata/consolidated_review_summary.md",
			dataType:   &ConsolidatedReview{},
		},
		{
			name:       "consolidated_review_accounts",
			structFile: "testdata/consolidated_review_accounts.json",
			goldenFile: "testdata/consolidated_review_accounts.md",
			dataType:   &ConsolidatedReview{},
		},
		{
			name:       "consolidated_review_asset_view",
			structFile: "testdata/consolidated_review_asset_view.json",
			goldenFile: "testdata/consolidated_review_asset_view.md",
			dataType:   &ConsolidatedReview{},
		},
		{
			name:       "consolidated_review_transactions",
			structFile: "testdata/consolidated_review_transactions.json",
			goldenFile: "testdata/consolidated_review_transactions.md",
			dataType:   &ConsolidatedReview{},
		},
		{
			name:       "holding_title",
			structFile: "testdata/holding.json",
			goldenFile: "testdata/holding_title.md",
			dataType:   &Holding{},
		},
		{
			name:       "holding_securities",
			structFile: "testdata/holding.json",
			goldenFile: "testdata/holding_securities.md",
			dataType:   &Holding{},
		},
		{
			name:       "holding_cash",
			structFile: "testdata/holding.json",
			goldenFile: "testdata/holding_cash.md",
			dataType:   &Holding{},
		},
		{
			name:       "holding_counterparties",
			structFile: "testdata/holding.json",
			goldenFile: "testdata/holding_counterparties.md",
			dataType:   &Holding{},
		},
		{
			name:       "consolidated_holding_title",
			structFile: "testdata/consolidated_holding.json",
			goldenFile: "testdata/consolidated_holding_title.md",
			dataType:   &ConsolidatedHolding{},
		},
		{
			name:       "consolidated_holding_securities",
			structFile: "testdata/consolidated_holding.json",
			goldenFile: "testdata/consolidated_holding_securities.md",
			dataType:   &ConsolidatedHolding{},
		},
		{
			name:       "consolidated_holding_cash",
			structFile: "testdata/consolidated_holding.json",
			goldenFile: "testdata/consolidated_holding_cash.md",
			dataType:   &ConsolidatedHolding{},
		},
		{
			name:       "consolidated_holding_counterparties",
			structFile: "testdata/consolidated_holding.json",
			goldenFile: "testdata/consolidated_holding_counterparties.md",
			dataType:   &ConsolidatedHolding{},
		},
	}

	// --- Coverage Check ---
	set := parseTemplates(t)
	testedPartialsMap := make(map[string]struct{})
	for _, tc := range testCases {
		testedPartialsMap[tc.name+".md"] = struct{}{}
	}
	for _, partialFile := range set.partials {
		if _, ok := testedPartialsMap[partialFile]; !ok {
			t.Errorf("untested template partial found: %s. Please add a test case to TestTemplatePartials.", partialFile)
		}
	}

	// --- Orphan Check ---
	usedStructs := make(map[string]struct{})
	usedGoldens := make(map[string]struct{})
	for _, tc := range testCases {
		usedStructs[tc.structFile] = struct{}{}
		usedGoldens[tc.goldenFile] = struct{}{}
	}

	for _, structFile := range set.partialStructs {
		if _, ok := usedStructs["testdata/"+structFile]; !ok {
			if *fixPartials {
				path := filepath.Join("testdata", structFile)
				os.Remove(path)
				t.Logf("removed unused partial struct file: %s", path)
			} else {
				t.Errorf("unused partial struct file found: %s. Please remove it or add a test case.", structFile)
			}
		}
	}
	for _, goldenFile := range set.partialGoldens {
		if _, ok := usedGoldens["testdata/"+goldenFile]; !ok {
			if *fixPartials {
				path := filepath.Join("testdata", goldenFile)
				os.Remove(path)
				t.Logf("removed unused partial golden file: %s", path)
			} else {
				t.Errorf("unused partial golden file found: %s. Please remove it or add a test case.", goldenFile)
			}
		}
	}
	for _, f := range set.orphanStructs {
		if *fixPartials {
			path := filepath.Join("testdata", f)
			os.Remove(path)
			t.Logf("removed orphan struct file: %s", path)
		} else {
			t.Errorf("orphan struct file found: %s. It does not match any known template.", f)
		}
	}
	for _, f := range set.orphanGoldens {
		if *fixPartials {
			path := filepath.Join("testdata", f)
			os.Remove(path)
			t.Logf("removed orphan golden file: %s", path)
		} else {
			t.Errorf("orphan golden file found: %s. It does not match any known template.", f)
		}
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 1. Read the input struct from JSON
			// The tc.dataType is a pointer to a zero value of the target struct.
			// Unmarshal will populate it.
			jsonData, err := testcasesFS.ReadFile(tc.structFile)
			if err != nil {
				t.Fatalf("failed to read struct file %q: %v", tc.structFile, err)
			}
			if err := json.Unmarshal(jsonData, tc.dataType); err != nil {
				t.Fatalf("failed to unmarshal struct data from %q: %v", tc.structFile, err)
			}

			// 2. Read the template partial
			templateFile := tc.name + ".md"
			templateContent, err := fs.ReadFile(templates, templateFile)
			if err != nil {
				t.Fatalf("failed to read template file %q: %v", templateFile, err)
			}

			// 3. Execute the template
			tmpl, err := template.New(tc.name).Parse(string(templateContent))
			if err != nil {
				t.Fatalf("failed to parse template %q: %v", templateFile, err)
			}

			var renderedOutput bytes.Buffer
			if err := tmpl.Execute(&renderedOutput, tc.dataType); err != nil {
				t.Fatalf("failed to execute template %q: %v", templateFile, err)
			}

			// 4. Read the expected output (golden file)
			goldenData, err := fs.ReadFile(testcasesGoldenFS, tc.goldenFile)
			if err != nil {
				// If the file doesn't exist and we're in fix mode, create it.
				if os.IsNotExist(err) && *fixPartials {
					// In fix mod we don't want to fail so we return an empty string.
					// Do not return the actual renderedOutput otherwise the test pass
					// and the golden will never get fixed.
					goldenData = []byte{} // Start with empty content
				} else {
					t.Fatalf("failed to read golden file %q: %v", tc.goldenFile, err)
				}
			}

			// 5. Compare and potentially fix
			got := renderedOutput.String()
			want := string(goldenData)

			if got != want {
				if *fixPartials {
					// Ensure testdata directory exists
					if err := os.MkdirAll(filepath.Dir(tc.goldenFile), 0755); err != nil {
						t.Fatalf("failed to create testdata directory: %v", err)
					}
					// Write the new "golden" output
					if err := os.WriteFile(tc.goldenFile, []byte(got), 0644); err != nil {
						t.Fatalf("failed to write updated golden file %q: %v", tc.goldenFile, err)
					}
					t.Logf("updated golden file %s", tc.goldenFile)
				} else {
					// In normal mode, report an error with a diff-like output.
					t.Errorf("output mismatch for %s:\n--- want\n+++ got\n%s",
						tc.name,
						createDiff(want, got),
					)
				}
			}
		})
	}
}

func TestReportRendering(t *testing.T) {
	testCases := []struct {
		name       string
		structFile string
		goldenFile string
		dataType   any
		renderFunc func(t *testing.T, data any) string
	}{
		{
			name:       "review",
			structFile: "testdata/review.json",
			goldenFile: "testdata/review_assembly.md",
			dataType:   &Review{},
			renderFunc: func(t *testing.T, data any) string {
				return RenderReview(data.(*Review), ReviewRenderOptions{SimplifiedView: false, SkipTransactions: false})
			},
		},
		{
			name:       "consolidated_review",
			structFile: "testdata/consolidated_review.json",
			goldenFile: "testdata/consolidated_review_assembly.md",
			dataType:   &ConsolidatedReview{},
			renderFunc: func(t *testing.T, data any) string {
				return RenderConsolidatedReview(data.(*ConsolidatedReview), ReviewRenderOptions{SkipTransactions: false})
			},
		},
		{
			name:       "holding",
			structFile: "testdata/holding.json",
			goldenFile: "testdata/holding_assembly.md",
			dataType:   &Holding{},
			renderFunc: func(t *testing.T, data any) string {
				return RenderHolding(data.(*Holding))
			},
		},
		{
			name:       "consolidated_holding",
			structFile: "testdata/consolidated_holding.json",
			goldenFile: "testdata/consolidated_holding_assembly.md",
			dataType:   &ConsolidatedHolding{},
			renderFunc: func(t *testing.T, data any) string {
				return RenderConsolidatedHolding(data.(*ConsolidatedHolding))
			},
		},
	}

	// --- Coverage Check ---
	set := parseTemplates(t)
	testedAssembliesMap := make(map[string]struct{})
	for _, tc := range testCases {
		// The test case name should correspond to the assembly file name without the extension.
		testedAssembliesMap[tc.name+".md"] = struct{}{}
	}

	for _, assemblyFile := range set.assemblies {
		if _, ok := testedAssembliesMap[assemblyFile]; !ok {
			t.Errorf("untested assembly template found: %s. Please add a test case to TestReportRendering.", assemblyFile)
		}
	}

	// --- Orphan Check ---
	usedStructs := make(map[string]struct{})
	usedGoldens := make(map[string]struct{})
	for _, tc := range testCases {
		usedStructs[tc.structFile] = struct{}{}
		usedGoldens[tc.goldenFile] = struct{}{}
	}

	for _, structFile := range set.assemblyStructs {
		if _, ok := usedStructs["testdata/"+structFile]; !ok {
			if *fixPartials {
				path := filepath.Join("testdata", structFile)
				os.Remove(path)
				t.Logf("removed unused assembly struct file: %s", path)
			} else {
				t.Errorf("unused assembly struct file found: %s. Please remove it or add a test case.", structFile)
			}
		}
	}
	for _, goldenFile := range set.assemblyGoldens {
		if _, ok := usedGoldens["testdata/"+goldenFile]; !ok {
			if *fixPartials {
				path := filepath.Join("testdata", goldenFile)
				os.Remove(path)
				t.Logf("removed unused assembly golden file: %s", path)
			} else {
				t.Errorf("unused assembly golden file found: %s. Please remove it or add a test case.", goldenFile)
			}
		}
	}
	for _, f := range set.orphanStructs {
		if *fixPartials {
			path := filepath.Join("testdata", f)
			os.Remove(path)
			t.Logf("removed orphan struct file: %s", path)
		} else {
			t.Errorf("orphan struct file found: %s. It does not match any known template.", f)
		}
	}
	for _, f := range set.orphanGoldens {
		if *fixPartials {
			path := filepath.Join("testdata", f)
			os.Remove(path)
			t.Logf("removed orphan golden file: %s", path)
		} else {
			t.Errorf("orphan golden file found: %s. It does not match any known template.", f)
		}
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 1. Read the input struct from JSON
			jsonData, err := testcasesFS.ReadFile(tc.structFile)
			if err != nil {
				t.Fatalf("failed to read struct file %q: %v", tc.structFile, err)
			}
			if err := json.Unmarshal(jsonData, tc.dataType); err != nil {
				t.Fatalf("failed to unmarshal struct data from %q: %v", tc.structFile, err)
			}

			// 2. Execute the render function
			got := tc.renderFunc(t, tc.dataType)

			// 3. Read the expected output (golden file)
			goldenData, err := fs.ReadFile(testcasesGoldenFS, tc.goldenFile)
			if err != nil {
				if os.IsNotExist(err) && *fixPartials {
					goldenData = []byte{}
				} else {
					t.Fatalf("failed to read golden file %q: %v", tc.goldenFile, err)
				}
			}
			want := string(goldenData)

			// 4. Compare and potentially fix
			if got != want {
				if *fixPartials {
					if err := os.MkdirAll(filepath.Dir(tc.goldenFile), 0755); err != nil {
						t.Fatalf("failed to create testdata directory: %v", err)
					}
					if err := os.WriteFile(tc.goldenFile, []byte(got), 0644); err != nil {
						t.Fatalf("failed to write updated golden file %q: %v", tc.goldenFile, err)
					}
					t.Logf("updated golden file %s", tc.goldenFile)
				} else {
					t.Errorf("output mismatch for %s:\n--- want\n+++ got\n%s",
						tc.name,
						createDiff(want, got),
					)
				}
			}
		})
	}
}

func createDiff(want, got string) string {
	// A simple diff-like representation for clearer test failures.
	return fmt.Sprintf("-%s\n+%s", strings.ReplaceAll(want, "\n", "\n-"), strings.ReplaceAll(got, "\n", "\n+"))
}

// --- Coverage Helper Functions ---

// templateSet describes the discovered templates from the filesystem.
type templateSet struct {
	// assemblies is a list of all discovered assembly template files (e.g., "review.md").
	assemblies []string
	// partials is a list of all discovered partial template files (e.g., "review_title.md").
	partials []string

	// --- Test Data Files ---

	// Files for partial tests
	partialGoldens []string
	partialStructs []string

	// Files for assembly tests
	assemblyGoldens []string
	assemblyStructs []string

	// Files that don't match any known template
	orphanGoldens []string
	orphanStructs []string
}

// parseTemplates scans the embedded filesystem for .md files and categorizes them
// as either assembly templates or partial templates.
func parseTemplates(t *testing.T) templateSet {
	t.Helper()

	templateFiles, err := templates.ReadDir(".")
	if err != nil {
		t.Fatalf("failed to read embedded templates: %v", err)
	}

	set := templateSet{
		assemblies:      []string{},
		partials:        []string{},
		partialGoldens:  []string{},
		partialStructs:  []string{},
		assemblyGoldens: []string{},
		assemblyStructs: []string{},
		orphanGoldens:   []string{},
		orphanStructs:   []string{},
	}

	// --- 1. Classify *.md templates in the root directory ---
	var allTemplateNames []string
	for _, file := range templateFiles {
		fileName := file.Name()
		if file.IsDir() || !strings.HasSuffix(fileName, ".md") {
			continue
		}
		allTemplateNames = append(allTemplateNames, fileName)
	}

	partialBaseNames := make(map[string]struct{})
	assemblyBaseNames := make(map[string]struct{})

	for _, name1 := range allTemplateNames {
		isPartial := false
		base1 := strings.TrimSuffix(name1, ".md")
		for _, name2 := range allTemplateNames {
			if name1 == name2 {
				continue
			}
			base2 := strings.TrimSuffix(name2, ".md")
			if strings.HasPrefix(base1, base2+"_") {
				isPartial = true
				break
			}
		}
		if isPartial {
			set.partials = append(set.partials, name1)
			partialBaseNames[base1] = struct{}{}
		} else {
			set.assemblies = append(set.assemblies, name1)
			assemblyBaseNames[base1] = struct{}{}
		}
	}

	// --- 2. Classify testdata files based on template classification ---
	testDataFiles, _ := testcasesFS.ReadDir("testdata")
	for _, f := range testDataFiles {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".json") {
			continue
		}
		fileName := f.Name()
		baseName := strings.TrimSuffix(fileName, ".json")

		if _, ok := partialBaseNames[baseName]; ok {
			set.partialStructs = append(set.partialStructs, fileName)
		} else if _, ok := assemblyBaseNames[baseName]; ok {
			set.assemblyStructs = append(set.assemblyStructs, fileName)
		} else {
			set.orphanStructs = append(set.orphanStructs, fileName)
		}
	}

	testGoldenFiles, _ := testcasesGoldenFS.ReadDir("testdata")
	for _, f := range testGoldenFiles {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".md") {
			continue
		}
		fileName := f.Name()
		baseName := strings.TrimSuffix(fileName, ".md")

		// Assembly golden files have a `_assembly` suffix.
		assemblyBaseName := strings.TrimSuffix(baseName, "_assembly")

		if _, ok := partialBaseNames[baseName]; ok {
			set.partialGoldens = append(set.partialGoldens, fileName)
		} else if _, ok := assemblyBaseNames[assemblyBaseName]; ok {
			set.assemblyGoldens = append(set.assemblyGoldens, fileName)
		} else {
			set.orphanGoldens = append(set.orphanGoldens, fileName)
		}
	}

	return set
}
