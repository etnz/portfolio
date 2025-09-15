package renderer

import (
	"fmt"
	"strings"

	"github.com/etnz/portfolio"
)

func DailyMarkdown(r *portfolio.DailyReport) string {
	var b strings.Builder

	fmt.Fprint(&b, "# Daily Report\n\n")

	genDate := portfolio.NewDate(r.Time.Year(), r.Time.Month(), r.Time.Day())
	if r.Date == genDate {
		fmt.Fprint(&b, "Report for "+r.Time.Format("2006-01-02 15:04:05")+"\n\n")

	} else {
		fmt.Fprint(&b, "Report for "+r.Date.String()+"\n\n")
	}

	fmt.Fprintf(&b, "| **%s** | **%s** |\n", "Value", r.ValueAtClose.String())
	fmt.Fprintln(&b, "|:---|---:|")
	fmt.Fprintf(&b, "| Value at Prev. Close | %s |\n", r.ValueAtPrevClose.String())

	if r.HasBreakdown() {
		fmt.Fprintf(&b, "\n## Breakdown of Change\n\n")
		fmt.Fprintf(&b, "| **%s** | **%s** |\n", "Total Day's Gain", r.TotalGain.SignedString())
		fmt.Fprintln(&b, "|:---|---:|")

		if !r.MarketGains.IsZero() {
			fmt.Fprintf(&b, "| Unrealized Market | %s |\n", r.MarketGains.SignedString())
		}
		if !r.RealizedGains.IsZero() {
			fmt.Fprintf(&b, "| Realized Market | %s |\n", r.RealizedGains.SignedString())
		}
		if !r.Dividends.IsZero() {
			fmt.Fprintf(&b, "| Dividends | %s |\n", r.Dividends.SignedString())
		}
		if !r.NetCashFlow.IsZero() {
			fmt.Fprintf(&b, "| Net Cash Flow | %s |\n", r.NetCashFlow.SignedString())
		}
	}

	if len(r.ActiveAssets) > 0 {
		fmt.Fprintf(&b, "\n## Active Assets\n\n")
		fmt.Fprintln(&b, "| Ticker | Gain / Loss | Change |")
		fmt.Fprintln(&b, "|:---|---:|---:|")

		for _, asset := range r.ActiveAssets {
			if !asset.Gain.IsZero() {
				fmt.Fprintf(&b, "| %s | %s | %s |\n",
					asset.Security,
					asset.Gain.String(),
					asset.Return.SignedString(),
				)
			}
		}
		// and the total row.
		fmt.Fprintf(&b, "| **%s** | **%s** | **%s** |\n",
			"Total",
			r.MarketGains.SignedString(),
			r.PercentageGain().SignedString(),
		)
	}

	if len(r.Transactions) > 0 {
		fmt.Fprintf(&b, "\n## Intraday's Transactions\n\n")
		for i, tx := range r.Transactions {
			fmt.Fprintf(&b, "%d. %s\n", i+1, Transaction(tx))
		}
	}

	return b.String()
}
