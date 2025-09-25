package renderer

import (
	"fmt"
	"strings"

	"github.com/etnz/portfolio"
)

// LogMarkdown generates a markdown report from a slice of review blocks.
func LogMarkdown(reviews []*portfolio.Review, securities []portfolio.Security, method portfolio.CostBasisMethod) (string, error) {
	r := &logRenderer{
		Builder: &strings.Builder{},
		Method:  method,
	}

	if len(securities) > 0 {
		r.renderSecurities(securities)
	}

	for _, review := range reviews {
		renderReviewSummarylevel(r.Builder, review, 2, false)
		renderConsolidatedAssetReport(r.Builder, review, r.Method)
		r.Printf("\n")
	}
	return r.String(), nil
}

// logRenderer formats the output of the log generator into a markdown string.
type logRenderer struct {
	*strings.Builder
	Method   portfolio.CostBasisMethod
	deferred map[string]string
}

// Printf formats according to a format specifier and writes to the renderer's buffer.
func (r *logRenderer) Printf(format string, args ...any) {
	fmt.Fprintf(r, format, args...)
}

// DeferPrintf formats a string and stores it under a key, to be printed later.
// This is used to show the final state of a metric after all transactions in a period.
func (r *logRenderer) DeferPrintf(key, format string, args ...any) {
	if r.deferred == nil {
		r.deferred = make(map[string]string)
	}
	r.deferred[key] = fmt.Sprintf(format, args...)
}
func (r *logRenderer) renderSecurities(securities []portfolio.Security) {
	r.Printf("## Held Securities\n\n")
	r.Printf("| Ticker | ID | Currency | Description |\n")
	r.Printf("|:---|:---|:---|:---|\n")
	for _, sec := range securities {
		r.Printf("| %s | %s | %s | %s |\n", sec.Ticker(), sec.ID(), sec.Currency(), sec.Description())
	}
	r.Printf("\n")
}
