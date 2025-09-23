package renderer

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/etnz/portfolio"
)

// Transaction renders a transaction to a string.
func Transaction(tx portfolio.Transaction) string {
	switch v := tx.(type) {
	case portfolio.Buy:
		return fmt.Sprintf("Bought %v of %s for %v", v.Quantity, v.Security, v.Amount)
	case portfolio.Sell:
		return fmt.Sprintf("Sold %v of %s for %v", v.Quantity, v.Security, v.Amount)
	case portfolio.Dividend:
		return fmt.Sprintf("Dividend of %v for %s", v.Amount, v.Security)
	case portfolio.Deposit:
		m := v.Amount
		return fmt.Sprintf("Deposited %v", m)
	case portfolio.Withdraw:
		m := v.Amount
		return fmt.Sprintf("Withdrew %v", m)
	case portfolio.Accrue:

		if v.Amount.IsPositive() {
			m := v.Amount
			return fmt.Sprintf("Accrued receivable %v from %s", m, v.Counterparty)
		}
		m := v.Amount.Neg()
		return fmt.Sprintf("Accrued payable %v to %s", m, v.Counterparty)
	case portfolio.Convert:
		return fmt.Sprintf("Converted %v to %v", v.FromAmount, v.ToAmount)
	case portfolio.Declare:
		return fmt.Sprintf("Declared %s as %s in %s", v.Ticker, v.ID, v.Currency)
	case portfolio.UpdatePrice:
		var buf strings.Builder
		keys := slices.Collect(maps.Keys(v.Prices))
		slices.Sort(keys)
		buf.WriteString("Price update: ")
		for i, k := range keys {
			buf.WriteString(k)
			buf.WriteString(" ")
			buf.WriteString(v.Prices[k].StringFixed(4))
			if i < len(keys) {
				buf.WriteString(", ")
			}
		}
		return buf.String()
	default:
		return string(tx.What())
	}
}
