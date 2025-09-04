package renderer

import (
	"fmt"

	"github.com/etnz/portfolio"
)

// Transaction renders a transaction to a string.
func Transaction(tx portfolio.Transaction) string {
	switch v := tx.(type) {
	case portfolio.Buy:
		return fmt.Sprintf("Bought %.2f of %s for %.2f", v.Quantity, v.Security, v.Amount)
	case portfolio.Sell:
		return fmt.Sprintf("Sold %.2f of %s for %.2f", v.Quantity, v.Security, v.Amount)
	case portfolio.Dividend:
		return fmt.Sprintf("Dividend of %.2f for %s", v.Amount, v.Security)
	case portfolio.Deposit:
		return fmt.Sprintf("Deposited %.2f %s", v.Amount, v.Currency)
	case portfolio.Withdraw:
		return fmt.Sprintf("Withdrew %.2f %s", v.Amount, v.Currency)
	case portfolio.Accrue:
		return fmt.Sprintf("Accrued %.2f %s for %s", v.Amount, v.Currency, v.Counterparty)
	case portfolio.Convert:
		return fmt.Sprintf("Converted %.2f %s to %.2f %s", v.FromAmount, v.FromCurrency, v.ToAmount, v.ToCurrency)
	case portfolio.Declare:
		return fmt.Sprintf("Declared %s as %s in %s", v.Ticker, v.ID, v.Currency)
	default:
		return string(tx.What())
	}
}
