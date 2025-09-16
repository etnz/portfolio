package renderer

import (
	"fmt"
	"io"
	"strings"

	"github.com/etnz/portfolio"
)

// ReviewMarkdown renders a ReviewReport to a markdown string.
func ReviewMarkdown(review *portfolio.Review, method portfolio.CostBasisMethod) string {
	var b strings.Builder
	start, end := review.Start(), review.End()

	// --- Main Summary ---
	fmt.Fprint(&b, "# Review Report\n\n")
	if p, ok := review.Range().Period(); ok {
		name := p.String()
		fmt.Fprintf(&b, "%s Report for %s\n\n", strings.Title(name), review.Range().Identifier())
	} else {
		fmt.Fprintf(&b, "Period from %s to %s\n\n", review.Range().From, review.Range().To)
	}

	fmt.Fprintf(&b, "| **%s** | **%s** |\n", "Total Portfolio Value", end.TotalPortfolio().String())
	fmt.Fprintln(&b, "|---:|---:|")
	fmt.Fprintf(&b, "| %s | %s |\n", "Previous Value", start.TotalPortfolio().String())
	fmt.Fprintf(&b, "| %s | %s |\n", "Cash Flow", review.CashFlow().SignedString())
	fmt.Fprintf(&b, "| %s | %s |\n", "Market Gains", review.MarketGainLoss().SignedString())
	if !review.Dividends().IsZero() {
		fmt.Fprintf(&b, "| %s | %s |\n", "Dividends", review.Dividends().SignedString())
		fmt.Fprintln(&b, "|   |   |")
		fmt.Fprintf(&b, "| **%s** | **%s** |\n", "Total Return", review.TotalReturn().SignedString())
	}

	fmt.Fprintln(&b, "|   |   |")
	fmt.Fprintf(&b, "| **%s** | **%s** |\n", "Net Change", review.PortfolioChange().String())
	fmt.Fprintf(&b, "| %s | %s |\n", "Cash", review.CashChange().SignedString())
	fmt.Fprintf(&b, "| %s | %s |\n", "Counterparties", review.CounterpartyChange().SignedString())
	fmt.Fprintf(&b, "| %s | %s |\n", "Market Value", end.TotalMarket().Sub(start.TotalMarket()).SignedString()) // This is tmvChange inside MarketGainLoss

	// --- Counterparty Accounts ---
	counterpartiesSection := Header(func(w io.Writer) {
		fmt.Fprintln(w, "|   |   |")
		fmt.Fprintln(w, "| **Counterparty Accounts**  |  |")
	})
	for acc := range end.Counterparties() {
		if end.Counterparty(acc).IsZero() && start.Counterparty(acc).IsZero() {
			continue
		}
		counterpartiesSection.PrintHeader(&b)
		fmt.Fprintf(&b, "| %s | %s |\n", acc, end.Counterparty(acc).String())
	}
	counterpartiesSection.PrintFooter(&b)

	fmt.Fprintf(&b, "\n## Cash Accounts\n\n")
	fmt.Fprintln(&b, "|  **Cash Accounts** | Value | Forex Return % |")
	fmt.Fprintln(&b, "|---:|---:|---:|")
	for cur := range end.Currencies() {
		if end.Cash(cur).IsZero() && start.Cash(cur).IsZero() {
			continue
		}
		startRate := start.ExchangeRate(cur)
		endRate := end.ExchangeRate(cur)
		var forexReturn portfolio.Percent
		if !startRate.IsZero() {
			forexReturn = portfolio.Percent(100 * (endRate.AsFloat()/startRate.AsFloat() - 1))
		}
		fmt.Fprintf(&b, "| %s | %s | %s |\n", cur, end.Cash(cur).String(), forexReturn.SignedString())
	}

	// --- Holding View ---
	fmt.Fprintf(&b, "\n## Holding View\n\n")
	fmt.Fprintln(&b, "| Asset | Prev. Value | Flow | Gain | End Value |")
	fmt.Fprintln(&b, "|:---|---:|---:|---:|---:|")
	for ticker := range end.Securities() {
		startValue := start.MarketValue(ticker)
		endValue := end.MarketValue(ticker)
		flow, _ := review.AssetNetTradingFlow(ticker)
		gain, _ := review.AssetMarketGainLoss(ticker)

		if startValue.IsZero() && endValue.IsZero() && flow.IsZero() && gain.IsZero() {
			continue
		}

		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n",
			ticker,
			startValue.String(),
			flow.SignedString(),
			gain.SignedString(),
			endValue.String(),
		)
	}
	fmt.Fprintf(&b, "| **%s** | **%s** | **%s** | **%s** | **%s** |\n",
		"Total",
		start.TotalMarket().String(),
		review.NetTradingFlow().SignedString(),
		review.MarketGainLoss().SignedString(),
		end.TotalMarket().String(),
	)

	// --- Performance View ---
	fmt.Fprintf(&b, "\n## Performance View\n\n")
	fmt.Fprintln(&b, "| Asset | Gain | Dividends | Total Return |")
	fmt.Fprintln(&b, "|:---|---:|---:|---:|")
	for ticker := range end.Securities() {
		gain, _ := review.AssetMarketGainLoss(ticker)
		dividends, _ := review.AssetDividends(ticker)
		totalReturn := gain.Add(dividends)
		if gain.IsZero() && dividends.IsZero() {
			continue
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s |\n",
			ticker,
			gain.SignedString(),
			dividends.SignedString(),
			totalReturn.SignedString(),
		)
	}
	fmt.Fprintf(&b, "| **%s** | **%s** | **%s** | **%s** |\n",
		"Total",
		review.MarketGainLoss().SignedString(),
		review.Dividends().SignedString(),
		review.TotalReturn().SignedString(),
	)

	// --- Tax View ---
	fmt.Fprintf(&b, "\n## Tax View\n\n")
	fmt.Fprintln(&b, "| Asset | Invested | Dividends | Realized | Unrealized |")
	fmt.Fprintln(&b, "|:---|---:|---:|---:|---:|")
	for ticker := range end.Securities() {
		invested := end.NetTradingFlow(ticker) // This is total invested since inception
		dividends, _ := review.AssetDividends(ticker)
		realized, _ := review.AssetRealizedGains(ticker, method)
		unrealized := end.UnrealizedGains(ticker, method)

		if invested.IsZero() && dividends.IsZero() && realized.IsZero() && unrealized.IsZero() && start.Position(ticker).IsZero() {
			continue
		}

		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n",
			ticker,
			invested.String(),
			dividends.SignedString(),
			realized.SignedString(),
			unrealized.SignedString(),
		)
	}
	fmt.Fprintf(&b, "| **%s** | **%s** | **%s** | **%s** | **%s** |\n",
		"Total",
		end.TotalNetTradingFlow().String(),
		review.Dividends().SignedString(),
		review.RealizedGains(method).SignedString(),
		end.TotalUnrealizedGains(method).SignedString(),
	)

	// --- Transactions ---
	transactionsSection := Header(func(w io.Writer) {
		fmt.Fprintf(w, "\n## Transactions\n\n")
		fmt.Fprintln(w, "| Date | Type | Description |")
		fmt.Fprintln(w, "|:---|:---|:---|")
	})

	transactions := review.Transactions()
	if len(transactions) < 20 {
		for _, tx := range transactions {
			transactionsSection.PrintHeader(&b)
			fmt.Fprintf(&b, "| %s | %s | %s |\n", tx.When(), tx.What(), Transaction(tx))
		}
	}

	return b.String()
}
