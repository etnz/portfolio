package renderer

import (
	"fmt"
	"strings"

	"github.com/etnz/portfolio"
)

// ReviewMarkdown renders a ReviewReport to a markdown string.
func ReviewMarkdown(report *portfolio.ReviewReport) string {
	var b strings.Builder

	fmt.Fprint(&b, "# Review Report\n\n")
	fmt.Fprintf(&b, "Period from %s to %s\n\n", report.Range.From, report.Range.To)

	fmt.Fprintf(&b, "| **%s** | **%s** |\n", "Total Portfolio Value", report.PortfolioValue.End.String())
	fmt.Fprintln(&b, "|---:|---:|")
	fmt.Fprintf(&b, "| %s | %s |\n", "Previous Value", report.PortfolioValue.Start.String())
	fmt.Fprintf(&b, "| %s | %s |\n", "Cash Flow", report.CashFlow.SignedString())
	fmt.Fprintf(&b, "| %s | %s |\n", "Gains", report.NetGains().SignedString())

	fmt.Fprintln(&b, "|   |   |")
	fmt.Fprintf(&b, "| **%s** | **%s** |\n", "Net Change", report.PortfolioValue.Change().String())
	fmt.Fprintf(&b, "| %s | %s |\n", "Cash", report.Cash.Change().SignedString())
	fmt.Fprintf(&b, "| %s | %s |\n", "Counterparties", report.Counterparty.Change().SignedString())
	fmt.Fprintf(&b, "| %s | %s |\n", "Market Value", report.MarketChange().SignedString())
	// fmt.Fprintf(&b, "| %s | %s |\n", "Time-Weighted Return (TWR)**", report.PortfolioValue.Return.SignedString())

	fmt.Fprintln(&b, "|   |   |")
	fmt.Fprintln(&b, "|  **Cash Accounts** |  |")
	for _, acc := range report.CashAccounts {
		fmt.Fprintf(&b, "| %s | %s |\n", acc.Label, acc.Value.SignedString())
	}
	fmt.Fprintln(&b, "|   |   |")
	fmt.Fprintln(&b, "| **Counterparty Accounts**  |  |")
	for _, acc := range report.Counterparties {
		fmt.Fprintf(&b, "| %s | %s |\n", acc.Label, acc.Value.SignedString())
	}

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
		report.MarketValue.Start.String(),
		report.MarketValue.End.String(),
		report.Realized.SignedString(),
		report.Unrealized.SignedString(),
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
