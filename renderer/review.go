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
	ConditionalBlock(&b, func(w io.Writer) bool { return renderHoldingView(w, review) })
	ConditionalBlock(&b, func(w io.Writer) bool { return renderPerformanceView(w, review) })
	ConditionalBlock(&b, func(w io.Writer) bool { return renderDividendView(w, review) })
	ConditionalBlock(&b, func(w io.Writer) bool { return renderTaxView(w, review, method) })
	ConditionalBlock(&b, func(w io.Writer) bool { return renderTransactionsSection(w, review) })

	return b.String()
}

func renderReviewSummary(w io.Writer, review *portfolio.Review) bool {
	start, end := review.Start(), review.End()
	// --- Main Summary ---
	if p, ok := review.Range().Period(); ok {
		var title string
		if review.Range().To.After(portfolio.Today()) {
			title = p.ToDateName()
		} else {
			title = p.String()
		}
		identifier := review.Range().Identifier()

		fmt.Fprintf(w, "# %s Review for %s\n\n", strings.Title(title), identifier)
	} else {
		fmt.Fprintf(w, "# Review from %s to %s\n\n", review.Range().From, review.Range().To)
	}

	fmt.Fprintf(w, "*As of %s*\n\n", Now().Format("2006-01-02 15:04:05"))

	// Summary Table

	gain := review.PortfolioChange().Sub(review.CashFlow()).Sub(review.MarketGain())

	fmt.Fprintf(w, "| **%s** | **%s** |\n", "Total Portfolio Value", end.TotalPortfolio().String())
	fmt.Fprintln(w, "|---:|---:|")
	fmt.Fprintf(w, "| %s | %s |\n", "Previous Value", start.TotalPortfolio().String())
	fmt.Fprintln(w, "| | |")
	fmt.Fprintf(w, "| %s | %s |\n", "Cash Flow", review.CashFlow().SignedString())
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
	if !review.Dividends().IsZero() {
		fmt.Fprintln(w, "| | |")
		fmt.Fprintf(w, "| **%s** | **%s** |\n", "Dividends", review.Dividends().SignedString())
	}
	return true
}

func renderAccountsSection(w io.Writer, review *portfolio.Review) bool {
	start, end := review.Start(), review.End()
	fmt.Fprintf(w, "\n## Accounts\n\n")

	fmt.Fprintln(w, "|  **Cash Accounts** | Value | Forex % |")
	fmt.Fprintln(w, "|---:|---:|---:|")
	for cur := range end.Currencies() {
		// if end.Cash(cur).IsZero() && start.Cash(cur).IsZero() {
		// 	continue
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
		if end.Counterparty(acc).IsZero() && start.Counterparty(acc).IsZero() {
			continue
		}
		fmt.Fprintf(w, "| %s | %s |\n", acc, end.Counterparty(acc).String())
	}

	return true
}

func renderHoldingView(w io.Writer, review *portfolio.Review) bool {
	start, end := review.Start(), review.End()
	fmt.Fprintf(w, "\n## Holding View\n\n")

	fmt.Fprintln(w, "| Asset | Prev. Value | Flow | Gain | End Value |")
	fmt.Fprintln(w, "|:---|---:|---:|---:|---:|")
	for ticker := range end.Securities() {
		startValue := start.MarketValue(ticker)
		endValue := end.MarketValue(ticker)
		flow := review.AssetNetTradingFlow(ticker)
		gain := review.AssetMarketGain(ticker)

		if startValue.IsZero() && endValue.IsZero() && flow.IsZero() && gain.IsZero() {
			continue
		}
		fmt.Fprintf(w, "| %s | %s | %s | %s | %s |\n",
			ticker,
			startValue.String(),
			flow.SignedString(),
			gain.SignedString(),
			endValue.String(),
		)
	}

	fmt.Fprintf(w, "| **%s** | **%s** | **%s** | **%s** | **%s** |\n",
		"Total",
		start.TotalMarket().String(),
		review.NetTradingFlow().SignedString(),
		review.MarketGain().SignedString(),
		end.TotalMarket().String(),
	)
	return true
}

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

func renderDividendView(w io.Writer, review *portfolio.Review) bool {
	end := review.End()
	if review.Dividends().IsZero() {
		return false
	}

	fmt.Fprintf(w, "\n## Dividend View\n\n")

	fmt.Fprintln(w, "| Asset | Dividends | Dividend Return |")
	fmt.Fprintln(w, "|:---|---:|---:|")
	hasContent := false
	for ticker := range end.Securities() {
		dividends := review.AssetDividends(ticker)
		if dividends.IsZero() {
			continue
		}
		hasContent = true
		divReturn := review.AssetDividendReturn(ticker)
		fmt.Fprintf(w, "| %s | %s | %s |\n",
			ticker,
			dividends.SignedString(),
			divReturn.SignedString(),
		)
	}
	if !hasContent {
		return false
	}
	fmt.Fprintf(w, "| **%s** | **%s** | **%s** |\n",
		"Total",
		review.Dividends().SignedString(),
		review.DividendReturn().SignedString(),
	)
	return true
}

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

		if invested.IsZero() && dividends.IsZero() && realized.IsZero() && unrealized.IsZero() {
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
	fmt.Fprintln(w, "| Date | Type | Description |")
	fmt.Fprintln(w, "|:---|:---|:---|")
	for _, tx := range transactions {
		fmt.Fprintf(w, "| %s | %s | %s |\n", tx.When(), tx.What(), Transaction(tx))
	}
	return true
}
