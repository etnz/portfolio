package renderer

import (
	"io"
	"strings"

	"github.com/etnz/portfolio"
)

// PeriodicMarkdown renders a short, summary-focused review for a given period.
func PeriodicMarkdown(review *portfolio.Review, method portfolio.CostBasisMethod) string {
	var b strings.Builder
	ConditionalBlock(&b, func(w io.Writer) bool { return renderReviewSummary(w, review) })
	ConditionalBlock(&b, func(w io.Writer) bool { return renderPerformanceView(w, review) })
	return b.String()
}
