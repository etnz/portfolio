package renderer

import (
	"fmt"
	"strings"

	"github.com/etnz/portfolio"
)

// ReviewMarkdown renders a ReviewReport to a markdown string.
func ReviewMarkdown(report *portfolio.ReviewReport) string {
	var b strings.Builder

	// --- Main Summary ---
	fmt.Fprint(&b, "# Review Report\n\n")
	if p, ok := report.Range.Period(); ok {
		name := p.String()
		fmt.Fprintf(&b, "%s Report for %s\n\n", strings.Title(name), report.Range.Identifier())
	} else {
		fmt.Fprintf(&b, "Period from %s to %s\n\n", report.Range.From, report.Range.To)
	}

	fmt.Fprintf(&b, "| **%s** | **%s** |\n", "Total Portfolio Value", report.PortfolioValue.End.String())
	fmt.Fprintln(&b, "|---:|---:|")
	fmt.Fprintf(&b, "| %s | %s |\n", "Previous Value", report.PortfolioValue.Start.String())
	fmt.Fprintf(&b, "| %s | %s |\n", "Cash Flow", report.CashFlow.SignedString())
	fmt.Fprintf(&b, "| %s | %s |\n", "Market Gains", report.NetGains().SignedString())
	if !report.Total.Dividends.IsZero() {
		fmt.Fprintf(&b, "| %s | %s |\n", "Dividends", report.Total.Dividends.SignedString())
		fmt.Fprintln(&b, "|   |   |")
		fmt.Fprintf(&b, "| **%s** | **%s** |\n", "Total Return", report.TotalReturn().SignedString())
	}

	fmt.Fprintln(&b, "|   |   |")
	fmt.Fprintf(&b, "| **%s** | **%s** |\n", "Net Change", report.PortfolioValue.Change().String())
	fmt.Fprintf(&b, "| %s | %s |\n", "Cash", report.Cash.Change().SignedString())
	fmt.Fprintf(&b, "| %s | %s |\n", "Counterparties", report.Counterparty.Change().SignedString())
	fmt.Fprintf(&b, "| %s | %s |\n", "Market Value", report.Total.Value.Change().SignedString())

	if len(report.Counterparties) > 0 {
		fmt.Fprintln(&b, "|   |   |")
		fmt.Fprintln(&b, "| **Counterparty Accounts**  |  |")
		for _, acc := range report.Counterparties {
			fmt.Fprintf(&b, "| %s | %s |\n", acc.Label, acc.Value.String())
		}
	}

	fmt.Fprintf(&b, "\n## Cash Accounts\n\n")
	fmt.Fprintln(&b, "|  **Cash Accounts** | Value | Forex Return % |")
	fmt.Fprintln(&b, "|---:|---:|---:|")
	for _, acc := range report.CashAccounts {
		fmt.Fprintf(&b, "| %s | %s | %s |\n", acc.Label, acc.Value.String(), acc.Return.SignedString())
	}

	// --- Holding View ---
	fmt.Fprintf(&b, "\n## Holding View\n\n")
	fmt.Fprintln(&b, "| Asset | Prev. Value | Flow | Gain | End Value |")
	fmt.Fprintln(&b, "|:---|---:|---:|---:|---:|")
	for _, asset := range report.Assets {
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n",
			asset.Security,
			asset.Value.Start.String(),
			asset.Flow().SignedString(),
			asset.Gain().SignedString(),
			asset.Value.End.String(),
		)
	}
	fmt.Fprintf(&b, "| **%s** | **%s** | **%s** | **%s** | **%s** |\n",
		"Total",
		report.Total.Value.Start.String(),
		report.Total.Flow().SignedString(),
		report.Total.Gain().SignedString(),
		report.Total.Value.End.String(),
	)

	// --- Performance View ---
	fmt.Fprintf(&b, "\n## Performance View\n\n")
	fmt.Fprintln(&b, "| Asset | Gain | Dividends | Total Return | Return % |")
	fmt.Fprintln(&b, "|:---|---:|---:|---:|---:|")
	for _, asset := range report.Assets {
		if !asset.Gain().IsZero() || asset.Value.Return != 0 {
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n",
				asset.Security,
				asset.Gain().SignedString(),
				asset.Dividends.SignedString(),
				asset.TotalReturn().SignedString(),
				asset.Value.Return.SignedString(),
			)
		}
	}
	fmt.Fprintf(&b, "| **%s** | **%s** | **%s** | **%s** | **%s** |\n",
		"Total",
		report.Total.Gain().SignedString(),
		report.Total.Dividends.SignedString(),
		report.Total.TotalReturn().SignedString(),
		report.PortfolioValue.Return.SignedString(),
	)

	// --- Tax View ---
	fmt.Fprintf(&b, "\n## Tax View\n\n")
	fmt.Fprintln(&b, "| Asset | Invested | Dividends | Realized | Unrealized |")
	fmt.Fprintln(&b, "|:---|---:|---:|---:|---:|")
	for _, asset := range report.Assets {
		if !asset.Buys.IsZero() || !asset.RealizedGains.IsZero() || !asset.UnrealizedGains.IsZero() {
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n",
				asset.Security,
				asset.Buys.String(),
				asset.Dividends.SignedString(),
				asset.RealizedGains.SignedString(),
				asset.UnrealizedGains.SignedString(),
			)
		}
	}
	fmt.Fprintf(&b, "| **%s** | **%s** | **%s** | **%s** | **%s** |\n",
		"Total",
		report.Total.Buys.String(),
		report.Total.Dividends.SignedString(),
		report.Total.RealizedGains.SignedString(),
		report.Total.UnrealizedGains.SignedString(),
	)

	// --- Transactions ---
	// only print section if not too big
	if len(report.Transactions) < 20 {
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
	}

	return b.String()
}
