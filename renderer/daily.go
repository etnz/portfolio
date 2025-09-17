package renderer

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/etnz/portfolio"
)

func DailyMarkdown(review *portfolio.Review, method portfolio.CostBasisMethod) string {
	var b strings.Builder
	start, end := review.Start(), review.End()
	on := end.On()

	fmt.Fprint(&b, "# Daily Report\n\n")

	if on.IsToday() {
		fmt.Fprint(&b, "Report for "+time.Now().Format("2006-01-02 15:04:05")+"\n\n")

	} else {
		fmt.Fprint(&b, "Report for "+on.String()+"\n\n")
	}

	fmt.Fprintf(&b, "| **%s** | **%s** |\n", "Value", end.TotalPortfolio().String())
	fmt.Fprintln(&b, "|:---|---:|")
	fmt.Fprintf(&b, "| Value at Prev. Close | %s |\n", start.TotalPortfolio().String())

	totalGain := review.PortfolioChange()
	marketGains := review.MarketGain()
	realizedGains := review.RealizedGains(method)
	dividends := review.Dividends()
	netCashFlow := review.CashFlow()

	fmt.Fprintf(&b, "\n## Breakdown of Change\n\n")
	fmt.Fprintf(&b, "| **%s** | **%s** |\n", "Total Day's Gain", totalGain.SignedString()) //nolint:all
	fmt.Fprintln(&b, "|:---|---:|")

	if !marketGains.IsZero() {
		fmt.Fprintf(&b, "| Unrealized Market | %s |\n", marketGains.SignedString())
	}
	if !realizedGains.IsZero() {
		fmt.Fprintf(&b, "| Realized Market | %s |\n", realizedGains.SignedString())
	}
	if !dividends.IsZero() {
		fmt.Fprintf(&b, "| Dividends | %s |\n", dividends.SignedString())
	}
	if !netCashFlow.IsZero() {
		fmt.Fprintf(&b, "| Net Cash Flow | %s |\n", netCashFlow.SignedString())
	}

	activeAssetsSection := Header(func(w io.Writer) {
		fmt.Fprintf(w, "\n## Active Assets\n\n")
		fmt.Fprintln(w, "| Ticker | Gain / Loss | Change |")
		fmt.Fprintln(w, "|:---|---:|---:|")
	}).Footer(func(w io.Writer) {
		var percentageGain portfolio.Percent
		if !start.TotalMarket().IsZero() {
			percentageGain = portfolio.Percent(100 * marketGains.AsFloat() / start.TotalMarket().AsFloat())
		}
		// and the total row.
		fmt.Fprintf(w, "| **%s** | **%s** | **%s** |\n",
			"Total",
			marketGains.SignedString(),
			percentageGain.SignedString(),
		)
	})

	for ticker := range end.Securities() {
		gain := review.AssetMarketGain(ticker)
		if gain.IsZero() {
			continue
		}
		activeAssetsSection.PrintHeader(&b)
		twr := review.AssetTimeWeightedReturn(ticker)
		fmt.Fprintf(&b, "| %s | %s | %s |\n",
			ticker,
			gain.String(),
			twr.SignedString(),
		)
	}
	activeAssetsSection.PrintFooter(&b)

	transactions := review.Transactions()
	if len(transactions) > 0 {
		fmt.Fprintf(&b, "\n## Intraday's Transactions\n\n")
		for i, tx := range transactions {
			fmt.Fprintf(&b, "%d. %s\n", i+1, Transaction(tx))
		}
	}

	return b.String()
}
