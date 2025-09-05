package renderer

import (
	"fmt"
	"strings"

	"github.com/etnz/portfolio"
)

// ReviewMarkdown renders a ReviewReport to a markdown string.
func ReviewMarkdown(report *portfolio.ReviewReport) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# Review Report for %s to %s\n\n", report.Range.From, report.Range.To)

	// Summary
	fmt.Fprint(&b, "## Summary\n\n")

	fmt.Fprintln(&b, "|    |    |")
	fmt.Fprintln(&b, "|---:|---:|")

	fmt.Fprintf(&b, "| %s | %s | |\n", "**Previous Total Value**", report.PrevPortfolioValue.String())
	//fmt.Fprintf(&b, "| %s | %s |\n", "Time-Weighted Return (TWR)**", report.Performance.Return.SignedString())
	fmt.Fprintf(&b, "| %s | %s |\n", "Cash Flow", report.CashFlow.SignedString())
	fmt.Fprintf(&b, "| %s | %s |\n", "Cash Change", report.CashChange.SignedString())
	fmt.Fprintf(&b, "| %s | %s |\n", "Counterparties Change", report.CounterpartyChange.SignedString())
	fmt.Fprintf(&b, "| %s | %s |\n", "Realized Gains", report.Gains.Realized.SignedString())
	fmt.Fprintf(&b, "| %s | %s |\n", "Market Gains", report.MarketChange().SignedString())
	fmt.Fprintf(&b, "| %s | %s |\n", "Total Change", report.Gains.Total.SignedString())
	fmt.Fprintf(&b, "| **%s** | **%s** |\n", "New Total Value", report.TotalPortfolioValue.String())

	// Asset Reviews
	fmt.Fprintf(&b, "\n## Asset Performance\n\n")
	fmt.Fprintln(&b, "| Asset | Start Value | End Value | Realized Gains | Unrealized Gains |")
	fmt.Fprintln(&b, "|:---|---:|---:|---:|---:|")
	for _, asset := range report.Assets {
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n",
			asset.Security,
			asset.StartingValue.String(),
			asset.EndingValue.String(),
			asset.RealizedGains.SignedString(),
			asset.UnrealizedGains.SignedString(),
		)
	}
	fmt.Fprintf(&b, "| **%s** | **%s** | **%s** | **%s** | **%s** |\n",
		"Total",
		report.PrevMarketValue.String(),
		report.TotalMarketValue.String(),
		report.Gains.Realized.SignedString(),
		report.Gains.Unrealized.SignedString(),
	)

	// Transactions
	fmt.Fprintf(&b, "\n## Transactions\n\n")
	if len(report.Transactions) == 0 {
		fmt.Fprintln(&b, "No transactions in this period.")
	} else {
		fmt.Fprintln(&b, "| Date | Type | Description |")
		fmt.Fprintln(&b, "|:---|:---|:---|")
		for _, tx := range report.Transactions {
			fmt.Fprintf(&b, "| %s | %s | %s |\n", tx.When(), tx.What(), Transaction(tx))
		}
	}

	return b.String()
}
