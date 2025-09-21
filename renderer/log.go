package renderer

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/etnz/portfolio"
)

// LogMarkdown generates a markdown report from a slice of review blocks.
func LogMarkdown(reviews []*portfolio.Review, securities []portfolio.Security, method portfolio.CostBasisMethod) (string, error) {
	r := &logRenderer{
		Builder: &strings.Builder{},
		Method:  method,
	}

	if len(securities) > 0 {
		r.renderSecurities(securities)
	}

	for _, review := range reviews {
		renderReviewSummarylevel(r.Builder, review, 2, false)
		renderConsolidatedAssetReport(r.Builder, review, r.Method)
		r.Printf("\n")
	}
	return r.String(), nil
}

// logRenderer formats the output of the log generator into a markdown string.
type logRenderer struct {
	*strings.Builder
	Method   portfolio.CostBasisMethod
	deferred map[string]string
}

// Printf formats according to a format specifier and writes to the renderer's buffer.
func (r *logRenderer) Printf(format string, args ...any) {
	fmt.Fprintf(r, format, args...)
}

// DeferPrintf formats a string and stores it under a key, to be printed later.
// This is used to show the final state of a metric after all transactions in a period.
func (r *logRenderer) DeferPrintf(key, format string, args ...any) {
	if r.deferred == nil {
		r.deferred = make(map[string]string)
	}
	r.deferred[key] = fmt.Sprintf(format, args...)
}
func (r *logRenderer) renderSecurities(securities []portfolio.Security) {
	r.Printf("## Held Securities\n\n")
	r.Printf("| Ticker | ID | Currency | Description |\n")
	r.Printf("|:---|:---|:---|:---|\n")
	for _, sec := range securities {
		r.Printf("| %s | %s | %s | %s |\n", sec.Ticker(), sec.ID(), sec.Currency(), sec.Description())
	}
	r.Printf("\n")
}

func (r *logRenderer) renderMain(review *portfolio.Review) {
	identifier := review.Range().Identifier()
	txs := review.Transactions()
	periodName := review.Range().Name()

	r.Printf("## %s\n\n", identifier)
	r.DeferPrintf("zz_total", "**End of %s Portfolio Value** | **%s**", periodName, review.End().TotalPortfolio().String())

	for _, tx := range txs {
		r.renderTransaction(tx, review)
	}
	r.flushDeferred()
	r.Printf("\n")
}

func (r *logRenderer) flushDeferred() {
	if len(r.deferred) == 0 {
		return
	}
	r.Printf("\n| | |\n|:---|---:|\n")
	keys := slices.Collect(maps.Keys(r.deferred))
	slices.Sort(keys)

	for _, k := range keys {
		r.Printf("| %s |\n", r.deferred[k])
	}
	clear(r.deferred) // Reset for the next block
}

func (r *logRenderer) renderTransaction(tx portfolio.Transaction, review *portfolio.Review) {
	snap := review.End()
	periodName := review.Range().Name()

	switch v := tx.(type) {
	case portfolio.Declare:
		r.Printf("*   **declare %s**: Mapped to %s (%s).\n", v.Ticker, v.ID, v.Currency)
	case portfolio.Deposit:
		if v.Settles != "" {
			r.Printf("*   **deposit**: %s to settle %s\n", v.Amount.SignedString(), v.Settles)
		} else {
			r.Printf("*   **deposit**: %s\n", v.Amount.SignedString())
			r.DeferPrintf("cashflow", "%s Total Cash Flow | %s", periodName, review.CashFlow().SignedString())
		}
		r.DeferPrintf("cash_"+v.Currency(), "Cash (%s) | %s", v.Currency(), snap.Cash(v.Currency()).String())
	case portfolio.Withdraw:
		if v.Settles != "" {
			r.Printf("*   **withdraw**: %s to settle %s\n", v.Amount.Neg().SignedString(), v.Settles)
		} else {
			r.Printf("*   **withdraw**: %s\n", v.Amount.Neg().SignedString())
			r.DeferPrintf("cashflow", "%s Total Cash Flow | %s", periodName, review.CashFlow().SignedString())
		}
		r.DeferPrintf("cash_"+v.Currency(), "Cash (%s) | %s", v.Currency(), snap.Cash(v.Currency()).String())
	case portfolio.Buy:
		r.Printf("*   **buy %s**: %s shares for %s\n", v.Security, v.Quantity, v.Amount)
		r.DeferPrintf("pos_"+v.Security, "%s Position | %s", v.Security, snap.Position(v.Security))
		r.DeferPrintf("cash_"+v.Currency(), "Cash (%s) | %s", v.Currency(), snap.Cash(v.Currency()).String())
	case portfolio.Sell:
		realizedGain := review.AssetRealizedGains(v.Security, r.Method)
		r.Printf("*   **sell %s**: %s shares for %s\n", v.Security, v.Quantity, v.Amount)
		r.DeferPrintf("pos_"+v.Security, "%s Position | %s", v.Security, snap.Position(v.Security))
		r.DeferPrintf("cash_"+v.Currency(), "Cash (%s) | %s", v.Currency(), snap.Cash(v.Currency()).String())
		r.DeferPrintf("gain_"+v.Security, "Total Realized Gain | %s", realizedGain.SignedString())
	case portfolio.Dividend:
		totalDividend := review.AssetDividends(v.Security)
		r.Printf("*   **dividend %s**: %s per share\n", v.Security, v.Amount)
		r.DeferPrintf("div_"+v.Security, "Total Dividend Received | %s", totalDividend.String())
	case portfolio.Accrue:
		var accrualType string
		if v.Amount.IsPositive() {
			accrualType = "Receivable"
		} else {
			accrualType = "Payable"
		}
		r.Printf("*   **accrue %s**: %s (%s)\n", v.Counterparty, v.Amount.SignedString(), accrualType)
		r.DeferPrintf("cpty_"+v.Counterparty, "Counterparty Balance | %s", snap.Counterparty(v.Counterparty).String())
	case portfolio.Split:
		r.Printf("*   **split %s**: %d-for-%d ratio\n", v.Security, v.Numerator, v.Denominator)
		r.DeferPrintf("pos_"+v.Security, "%s Position | %s", v.Security, snap.Position(v.Security))
	case portfolio.Convert:
		r.Printf("*   **convert**: %s to %s\n", v.FromAmount, v.ToAmount)
		r.DeferPrintf("cash_"+v.FromCurrency(), "Cash (%s) | %s", v.FromCurrency(), snap.Cash(v.FromCurrency()).String())
		r.DeferPrintf("cash_"+v.ToCurrency(), "Cash (%s) | %s", v.ToCurrency(), snap.Cash(v.ToCurrency()).String())
	case portfolio.UpdatePrice:
		// Per the spec, update-price is not rendered in the log.
	case portfolio.Init:
		// Init is not rendered.
	default:
		// For any other transaction type, just print a generic description.
		r.Printf("*   **%s**: %s\n", tx.What(), tx.When())
	}
	r.WriteString("\n")
}
func isVisible(tx portfolio.Transaction) bool {
	switch tx.(type) {
	case portfolio.UpdatePrice, portfolio.Init:
		return false
	default:
		return true
	}
}
