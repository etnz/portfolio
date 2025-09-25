package renderer

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/etnz/portfolio"
)

// Now is the current time used in reports.
// it has to be a global variable so that tests can override it.
func Now() time.Time {
	if os.Getenv("PORTFOLIO_TESTING_NOW") != "" {
		t, err := time.Parse("2006-01-02 15:04:05", os.Getenv("PORTFOLIO_TESTING_NOW"))
		if err != nil {
			panic(err)
		}
		return t
	}
	return time.Now()
}

// ReviewMarkdown renders a ReviewReport to a markdown string.
func ReviewMarkdown(review *portfolio.Review, method portfolio.CostBasisMethod) string {
	var b strings.Builder

	ConditionalBlock(&b, func(w io.Writer) bool { return renderReviewSummary(w, review) })
	ConditionalBlock(&b, func(w io.Writer) bool { return renderAccountsSection(w, review) })
	ConditionalBlock(&b, func(w io.Writer) bool { return renderConsolidatedAssetReport(w, review, method) })
	ConditionalBlock(&b, func(w io.Writer) bool { return renderTransactionsSection(w, review) })

	return b.String()
}

func renderReviewSummary(w io.Writer, review *portfolio.Review) bool {
	return renderReviewSummarylevel(w, review, 1, true)
}

// renderReviewSummarylevel renders the summary section of the review.
// level is the heading level to use for the title.
func renderReviewSummarylevel(w io.Writer, review *portfolio.Review, level int, asOf bool) bool {
	start, end := review.Start(), review.End()
	heading := strings.Repeat("#", level)
	// --- Main Summary ---
	if p, ok := review.Range().Period(); ok {
		var title string
		if review.Range().To.After(portfolio.Today()) {
			title = p.ToDateName()
		} else {
			title = p.String()
		}
		identifier := review.Range().Identifier()

		fmt.Fprintf(w, "%s %s Review for %s\n\n", heading, strings.Title(title), identifier)
	} else {
		fmt.Fprintf(w, "%s Review from %s to %s\n\n", heading, review.Range().From, review.Range().To)
	}

	if asOf {
		fmt.Fprintf(w, "*As of %s*\n\n", Now().Format("2006-01-02 15:04:05"))
	}

	// Summary Table

	gain := review.PortfolioChange().Sub(review.CashFlow()).Sub(review.MarketGain())

	fmt.Fprintf(w, "| **%s** | **%s** |\n", "Total Portfolio Value", end.TotalPortfolio().String())
	fmt.Fprintln(w, "|---:|---:|")
	fmt.Fprintf(w, "| %s | %s |\n", "Previous Value", start.TotalPortfolio().String())
	fmt.Fprintln(w, "| | |")
	fmt.Fprintf(w, "| %s | %s |\n", "Capital Flow", review.CashFlow().SignedString())
	fmt.Fprintf(w, "| %s | %s |\n", "+ Market Gains", review.MarketGain().SignedString())
	fmt.Fprintf(w, "| %s | %s |\n", "+ Forex Gains", gain.SignedString())
	fmt.Fprintf(w, "| **%s** | **%s** |\n", "= Net Change", review.PortfolioChange().String())

	rows := []bool{
		!review.CashChange().IsZero(),
		!review.CounterpartyChange().IsZero(),
		!review.TotalMarketChange().IsZero(),
	}
	var rowCount int
	for _, r := range rows {
		if r {
			rowCount++
		}
	}

	if rowCount > 1 {
		fmt.Fprintln(w, "| | |")
		fmt.Fprintf(w, "| %s | %s |\n", "Cash Change", review.CashChange().SignedString())
		fmt.Fprintf(w, "| %s | %s |\n", "+ Counterparties Change", review.CounterpartyChange().SignedString())
		fmt.Fprintf(w, "| %s | %s |\n", "+ Market Value Change", review.TotalMarketChange().SignedString()) // This is tmvChange inside MarketGainLoss
		fmt.Fprintf(w, "| **%s** | **%s** |\n", "= Net Change", review.PortfolioChange().String())
	}

	fmt.Fprintln(w, "| | |")
	fmt.Fprintf(w, "| %s | %s |\n", "Dividends", review.Dividends().SignedString())
	fmt.Fprintf(w, "| %s | %s |\n", "+ Market Gains", review.MarketGain().SignedString())
	fmt.Fprintf(w, "| %s | %s |\n", "+ Forex Gains", gain.SignedString())

	totalGains := review.MarketGain().Add(gain).Add(review.Dividends())
	if !totalGains.IsZero() {
		fmt.Fprintln(w, "| | |")
		fmt.Fprintf(w, "| **%s** | **%s** |\n", "=Total Gains", totalGains.SignedString())
	}
	return true
}

func renderAccountsSection(w io.Writer, review *portfolio.Review) bool {
	start, end := review.Start(), review.End()
	fmt.Fprintf(w, "\n## Accounts\n\n")

	fmt.Fprintln(w, "|  **Cash Accounts** | Value | Forex % |")
	fmt.Fprintln(w, "|---:|---:|---:|")
	for cur := range end.Currencies() {
		// if AllAreZero(end.Cash(cur), start.Cash(cur)) {
		//	continue
		// }
		var forexReturn string
		if cur != end.ReportingCurrency() {
			forexReturn = review.CurrencyTimeWeightedReturn(cur).SignedString()
		}
		fmt.Fprintf(w, "| %s | %s | %s |\n", cur, end.Cash(cur).String(), forexReturn)
	}

	fmt.Fprintln(w, "\n\n| **Counterparty Accounts**  | Value |")
	fmt.Fprintln(w, "|---:|---:|")
	for acc := range end.Counterparties() {
		if AllAreZero(end.Counterparty(acc), start.Counterparty(acc)) {
			continue
		}
		fmt.Fprintf(w, "| %s | %s |\n", acc, end.Counterparty(acc).String())
	}

	return true
}


// renderPerformanceView is now unused and can be removed.
func renderPerformanceView(w io.Writer, review *portfolio.Review) bool {
	end := review.End()
	fmt.Fprintf(w, "\n## Performance View\n\n")

	fmt.Fprintln(w, "| Asset | Value | Gain | TWR |")
	fmt.Fprintln(w, "|:---|---:|---:|---:|")
	for ticker := range end.Securities() {
		marketGain := review.AssetMarketGain(ticker)
		twr := review.AssetTimeWeightedReturn(ticker)
		endValue := end.MarketValue(ticker)

		if marketGain.IsZero() {
			continue
		}
		fmt.Fprintf(w, "| %s | %s | %s | %s |\n", ticker, endValue.String(), marketGain.SignedString(), twr.SignedString())
	}
	fmt.Fprintf(w, "| **%s** | **%s** | **%s** | **%s** |\n",
		"Total", end.TotalMarket().String(), review.MarketGain().SignedString(), review.TimeWeightedReturn().SignedString(),
	)
	return true
}

// renderConsolidatedAssetReport generates a single table with a comprehensive view of all assets.
func renderConsolidatedAssetReport(w io.Writer, review *portfolio.Review, method portfolio.CostBasisMethod) bool {
	start, end := review.Start(), review.End()

	fmt.Fprintf(w, "\n## Consolidated Asset Report\n\n")
	fmt.Fprintln(w, "| Asset | Start Value | End Value | Trading Flow | Market Gain | Realized Gain | Unrealized Gain | Dividends |")
	fmt.Fprintln(w, "|:---|---:|---:|---:|---:|---:|---:|---:|")

	for ticker := range end.Securities() {

		startValue := start.MarketValue(ticker)
		endValue := end.MarketValue(ticker)
		tradingFlow := review.AssetNetTradingFlow(ticker)
		marketGain := review.AssetMarketGain(ticker)
		realizedGain := review.AssetRealizedGains(ticker, method)
		unrealizedGain := review.End().UnrealizedGains(ticker, method)
		dividends := review.AssetDividends(ticker)

		if AllAreZero(startValue, endValue, tradingFlow, marketGain, realizedGain, unrealizedGain, dividends) {
			continue
		}

		fmt.Fprintf(w, "| %s | %s | %s | %s | %s | %s | %s | %s |\n", ticker, startValue.String(), endValue.String(), tradingFlow.SignedString(), marketGain.SignedString(), realizedGain.SignedString(), unrealizedGain.SignedString(), dividends.SignedString())
	}

	totalUnrealizedGain := review.UnrealizedGains(method)
	fmt.Fprintf(w, "| **Total** | **%s** | **%s** | **%s** | **%s** | **%s** | **%s** | **%s** |\n", start.TotalMarket().String(), end.TotalMarket().String(), review.NetTradingFlow().SignedString(), review.MarketGain().SignedString(), review.RealizedGains(method).SignedString(), totalUnrealizedGain.SignedString(), review.Dividends().SignedString())

	return true
}

// renderTaxView is now unused and can be removed.
func renderTaxView(w io.Writer, review *portfolio.Review, method portfolio.CostBasisMethod) bool {
	start, end := review.Start(), review.End()
	_ = start
	fmt.Fprintf(w, "\n## Tax View\n\n")

	fmt.Fprintf(w, "| Asset | Cost Basis (%s) | Dividends | Realized | Unrealized |\n", method)
	fmt.Fprintln(w, "|:---|---:|---:|---:|---:|")
	for ticker := range end.Securities() {
		invested := review.AssetCostBasis(ticker, method)
		dividends := review.AssetDividends(ticker)
		realized := review.AssetRealizedGains(ticker, method)
		unrealized := end.UnrealizedGains(ticker, method)

		if AllAreZero(invested, dividends, realized, unrealized) {
			continue
		}
		fmt.Fprintf(w, "| %s | %s | %s | %s | %s |\n",
			ticker,
			invested.String(),
			dividends.SignedString(),
			realized.SignedString(),
			unrealized.SignedString(),
		)
	}

	fmt.Fprintf(w, "| **%s** | **%s** | **%s** | **%s** | **%s** |\n",
		"Total",
		review.TotalCostBasis(method).String(),
		review.Dividends().SignedString(),
		review.RealizedGains(method).SignedString(),
		end.TotalUnrealizedGains(method).SignedString(),
	)
	return true
}

func renderTransactionsSection(w io.Writer, review *portfolio.Review) bool {
	transactions := review.Transactions()
	if len(transactions) == 0 || len(transactions) >= 40 {
		return false
	}

	fmt.Fprint(w, "\n## Transactions\n\n")
	fmt.Fprint(w, Transactions(transactions))
	return true
}
