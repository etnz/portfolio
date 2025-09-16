package renderer

import (
	"fmt"
	"strings"

	"github.com/etnz/portfolio"
)

func HistoryMarkdown(snapshots []*portfolio.Snapshot, security, currency string) string {
	var b strings.Builder

	if security != "" {
		fmt.Fprintf(&b, "# History for %s\n\n", security)
		fmt.Fprintln(&b, "| Date | Position | Price | Value |")
		fmt.Fprintln(&b, "|:---|---:|---:|---:|")
		for _, s := range snapshots {
			fmt.Fprintf(&b, "| %s | %s | %s | %s |\n",
				s.On().String(),
				s.Position(security).String(),
				s.Price(security).String(),
				s.Convert(s.MarketValue(security)).String(),
			)
		}
	} else {
		fmt.Fprintf(&b, "# History for %s\n\n", currency)
		fmt.Fprintln(&b, "| Date | Value |")
		fmt.Fprintln(&b, "|:---|---:|")
		for _, s := range snapshots {
			fmt.Fprintf(&b, "| %s | %s |\n",
				s.On().String(),
				s.Convert(s.Cash(currency)).String(),
			)
		}
	}

	return b.String()
}
