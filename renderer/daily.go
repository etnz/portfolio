package renderer

import (
	"io"
	"strings"

	"github.com/etnz/portfolio"
)

func DailyMarkdown(review *portfolio.Review, method portfolio.CostBasisMethod) string {
	var b strings.Builder
	ConditionalBlock(&b, func(w io.Writer) bool { return renderReviewSummary(w, review) })
	ConditionalBlock(&b, func(w io.Writer) bool { return renderPerformanceView(w, review) })
	return b.String()
}
