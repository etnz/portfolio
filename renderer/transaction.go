package renderer

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/etnz/portfolio"
)

// Transactions renders a slice of transactions to a simple list string.
func Transactions(txs []portfolio.Transaction) string {
	var b strings.Builder
	var prevDate portfolio.Date
	for _, tx := range txs {
		dateStr := tx.When().String()
		if !prevDate.IsZero() && prevDate == tx.When() {
			// Use non-breaking spaces to maintain alignment
			dateStr = strings.Repeat("\u00A0", len(dateStr))
		}
		fmt.Fprintf(&b, "*   %s: %s\n", dateStr, Transaction(tx))
		prevDate = tx.When()
	}
	return b.String()
}

// Transaction renders a transaction to a string.
func Transaction(tx portfolio.Transaction) string {
	switch v := tx.(type) {
	case portfolio.Buy:
		return fmt.Sprintf("Buy %v of %q for %v", v.Quantity, v.Security, v.Amount)
	case portfolio.Sell:
		return fmt.Sprintf("Sell %v of %q for %v", v.Quantity, v.Security, v.Amount)
	case portfolio.Dividend:
		return fmt.Sprintf("Receive dividend of %v per share for %q", v.Amount, v.Security)
	case portfolio.Deposit:
		m := v.Amount
		return fmt.Sprintf("Deposit %v", m)
	case portfolio.Withdraw:
		m := v.Amount
		return fmt.Sprintf("Withdraw %v", m)
	case portfolio.Accrue:

		if v.Amount.IsPositive() {
			m := v.Amount
			return fmt.Sprintf("Accrue receivable %v from %q", m, v.Counterparty)
		}
		m := v.Amount.Neg()
		return fmt.Sprintf("Accrue payable %v to %q", m, v.Counterparty)
	case portfolio.Convert:
		return fmt.Sprintf("Convert %v to %v", v.FromAmount, v.ToAmount)
	case portfolio.Declare:
		return fmt.Sprintf("Declare %q as %q in %s", v.Ticker, v.ID, v.Currency)
	case portfolio.UpdatePrice:
		var buf strings.Builder
		keys := slices.Collect(maps.Keys(v.Prices))
		slices.Sort(keys)
		buf.WriteString("Update price for ")
		for i, k := range keys {
			buf.WriteString(strconv.Quote(k))
			buf.WriteString("=")
			buf.WriteString(v.Prices[k].StringFixed(4))
			if i < len(keys)-1 {
				buf.WriteString(", ")
			}
		}
		return buf.String()
	default:
		return string(tx.What())
	}
}
