package renderer

import (
	"bytes"
	"fmt"
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
	// Using a buffer to capture output of a nested template execution
	var renderedHoldings bytes.Buffer
	tmpl := template.Must(template.New("consolidatedHolding").Parse(consolidatedHoldingMarkdownTemplate))

	if err := tmpl.Execute(&renderedHoldings, ch); err != nil {
		return fmt.Sprintf("Error executing template: %v", err)
	}

	return renderedHoldings.String()
}

// ToMarkdown renders the Holding struct to a markdown string using a text/template.
func RenderHolding(h *Holding) string {
	tmpl := template.Must(template.New("holding").Parse(holdingMarkdownTemplate))
	var b strings.Builder
	if err := tmpl.Execute(&b, h); err != nil {
		// In a real application, you might want to handle this error more gracefully.
		return fmt.Sprintf("Error executing template: %v", err)
	}
	return b.String()
}

// RenderReview renders the Review struct to a markdown string.
func RenderReview(r *Review, opts ReviewRenderOptions) string {
	var b strings.Builder

	// Start with the main template and the common partials
	tmpl := template.Must(template.New("review").Parse(reviewMarkdownTemplate))
	template.Must(tmpl.Parse(reviewTitleTemplate))
	template.Must(tmpl.Parse(reviewSummaryTemplate))
	template.Must(tmpl.Parse(reviewAccountsTemplate))

	// asset view can be simplified of consolidated
	assetView := assetViewConsolidated
	if opts.SimplifiedView {
		assetView = assetViewSimplified
	}
	template.Must(tmpl.Parse(assetView))

	transactionView := reviewTransactionsTemplate
	if opts.SkipTransactions {
		transactionView = `{{define "review_transactions"}}{{end}}`
	}
	template.Must(tmpl.Parse(transactionView))

	if err := tmpl.ExecuteTemplate(&b, "review", r); err != nil {
		// In a real application, you might want to handle this error more gracefully.
		return fmt.Sprintf("Error executing template: %v", err)
	}
	return b.String()
}
