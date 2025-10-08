package renderer

import (
	"fmt"
	"strings"

	"github.com/etnz/portfolio"
)

// Still used in `assist` because that is the only way to let Gemini know the tickers (ID and description are key)
// We'll keep it until we have a better way to do it (maybe passing Holding as a json would be better)

// DeclarationMarkdown renders the full declaration of stocks ticker.
func DeclarationMarkdown(s *portfolio.Snapshot) string {
	// use the snaphost to mark the asset as currently held or not
	var b strings.Builder
	fmt.Fprintf(&b, "# Securities\n\n")
	fmt.Fprintln(&b, "| Ticker | Held | Security ID | Currency | Description |")
	fmt.Fprintln(&b, "|:---|:---:|:---|:---|:---|")

	for t := range s.Securities() {
		sec, _ := s.SecurityDetails(t)
		held := " "
		if !s.Position(t).IsZero() {
			held = "X"
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n",
			sec.Ticker(),
			held,
			sec.ID(),
			sec.Currency(),
			sec.Description(),
		)
	}
	return b.String()

}
