package renderer

import (
	"fmt"
	"strings"

	"github.com/etnz/portfolio"
)

func GainsMarkdown(review *portfolio.Review, method portfolio.CostBasisMethod) string {
	var b strings.Builder
	end := review.End()

	fmt.Fprintf(&b, "# Capital Gains Report from %s to %s\n\n", review.Range().From.String(), review.Range().To.String())
	fmt.Fprintf(&b, "Method: %s\n\n", method)

	fmt.Fprint(&b, "## Gains per Security\n\n")
	fmt.Fprintln(&b, "| Security | Realized (Period) | Unrealized (at End) |")
	fmt.Fprintln(&b, "|:---|---:|---:|")

	for ticker := range end.Securities() {
		realized := review.AssetRealizedGains(ticker, method)
		unrealized := end.UnrealizedGains(ticker, method)

		if realized.IsZero() && unrealized.IsZero() && end.Position(ticker).IsZero() {
			continue
		}

		fmt.Fprintf(&b, "| %s | %s | %s |\n",
			ticker,
			realized.SignedString(),
			unrealized.SignedString(),
		)
	}
	fmt.Fprintf(&b, "| **%s** | **%s** | **%s** |\n",
		"Total",
		review.RealizedGains(method).SignedString(),
		end.TotalUnrealizedGains(method).SignedString(),
	)

	return b.String()
}
