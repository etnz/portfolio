package renderer

import (
	"fmt"
	"io/fs"
	"strings"
	"text/template"
)

// ReviewRenderOptions holds configuration for rendering a review report.
type ReviewRenderOptions struct {
	SimplifiedView   bool // Use the simplified asset view instead of the consolidated one.
	SkipTransactions bool // Do not render the transactions section.
}

// RenderConsolidatedHolding renders the ConsolidatedHolding struct to a markdown string.
func RenderConsolidatedHolding(ch *ConsolidatedHolding) string {
	partials := map[string]string{
		"consolidated_holding_title":          "consolidated_holding_title.md",
		"consolidated_holding_securities":     "consolidated_holding_securities.md",
		"consolidated_holding_cash":           "consolidated_holding_cash.md",
		"consolidated_holding_counterparties": "consolidated_holding_counterparties.md",
	}
	return renderTemplate("consolidatedHolding", "consolidated_holding.md", partials, ch)
}

// ToMarkdown renders the Holding struct to a markdown string using a text/template.
func RenderHolding(h *Holding) string {
	partials := map[string]string{
		"holding_title":          "holding_title.md",
		"holding_securities":     "holding_securities.md",
		"holding_cash":           "holding_cash.md",
		"holding_counterparties": "holding_counterparties.md",
	}
	return renderTemplate("holding", "holding.md", partials, h)
}

// RenderReview renders the Review struct to a markdown string.
func RenderReview(r *Review, opts ReviewRenderOptions) string {
	// Phase 1: Declare template dependencies.
	// We define which partials are needed and how they are aliased in the main template.
	partials := map[string]string{
		"review_title":    "review_title.md",
		"review_summary":  "review_summary.md",
		"review_accounts": "review_accounts.md",
	}

	// Conditionally select the asset view template.
	if opts.SimplifiedView {
		partials["asset_view"] = "review_asset_view_simplified.md"
	} else {
		partials["asset_view"] = "review_asset_view_consolidated.md"
	}

	// Skip transactions if requested. An empty file name results in an empty template.
	if !opts.SkipTransactions {
		partials["review_transactions"] = "review_transactions.md"
	} else {
		partials["review_transactions"] = "review_transaction_skipped.md"
	}

	// Phase 2: Execute rendering with the generic utility.
	return renderTemplate("review", "review.md", partials, r)
}

// RenderConsolidatedReview renders the ConsolidatedReview struct to a markdown string.
func RenderConsolidatedReview(cr *ConsolidatedReview, opts ReviewRenderOptions) string {
	partials := map[string]string{
		"consolidated_review_title":    "consolidated_review_title.md",
		"consolidated_review_summary":  "consolidated_review_summary.md",
		"consolidated_review_accounts": "consolidated_review_accounts.md",
		"consolidated_asset_view":      "consolidated_review_asset_view.md",
	}

	// For now, we don't have a simplified view for consolidated review.
	if !opts.SkipTransactions {
		partials["consolidated_review_transactions"] = "consolidated_review_transactions.md"
	} else {
		partials["consolidated_review_transactions"] = "review_transaction_skipped.md" // Reuse the skipped template
	}

	return renderTemplate("consolidatedReview", "consolidated_review.md", partials, cr)
}

// renderTemplate is a generic utility to render a main template that depends on several partials.
func renderTemplate(templateName, mainFile string, partials map[string]string, data any) string {
	mainContent, err := fs.ReadFile(templates, mainFile)
	if err != nil {
		return fmt.Sprintf("error reading main template %q: %v", mainFile, err)
	}

	tmpl, err := template.New(templateName).Parse(string(mainContent))
	if err != nil {
		return fmt.Sprintf("error parsing main template %q: %v", mainFile, err)
	}

	for name, file := range partials {
		var content []byte
		// An empty file name is a valid case, resulting in an empty template.
		if file != "" {
			var readErr error
			content, readErr = fs.ReadFile(templates, file)
			if readErr != nil {
				return fmt.Sprintf("error reading partial template %q: %v", file, err)
			}
		}
		if _, err := tmpl.New(name).Parse(string(content)); err != nil {
			return fmt.Sprintf("error parsing partial template %q for %q: %v", file, name, err)
		}
	}

	var b strings.Builder
	if err := tmpl.ExecuteTemplate(&b, templateName, data); err != nil {
		return fmt.Sprintf("error executing template %q: %v", templateName, err)
	}
	return b.String()
}
