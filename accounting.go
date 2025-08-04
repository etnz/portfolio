package portfolio

import (
	"fmt"

	"github.com/etnz/portfolio/date"
)

// Performance holds the starting value and the calculated return for a specific period.
type Performance struct {
	StartValue float64
	Return     float64 // Return is a ratio (e.g., 0.05 for 5%)
}

// Summary provides a comprehensive, at-a-glance overview of the portfolio's
// state and performance on a given date.
type Summary struct {
	Date              date.Date
	ReportingCurrency string
	TotalMarketValue  float64
	Daily             Performance
	WTD               Performance // Week-to-Date
	MTD               Performance // Month-to-Date
	QTD               Performance // Quarter-to-Date
	YTD               Performance // Year-to-Date
	Inception         Performance
}

// AccountingSystem encapsulates all the data required for portfolio management,
// combining transactional data with market data. It serves as a central point
// of access for querying portfolio state, such as positions and cash balances,
// and for validating new transactions.
//
// By holding both the Ledger (the record of all transactions) and the MarketData
// (the repository of security information and prices), it provides the complete
// context needed for most portfolio operations.
type AccountingSystem struct {
	Ledger     *Ledger
	MarketData *MarketData
}

// NewAccountingSystem creates a new accounting system from a ledger and market data.
func NewAccountingSystem(ledger *Ledger, marketData *MarketData) *AccountingSystem {
	return &AccountingSystem{
		Ledger:     ledger,
		MarketData: marketData,
	}
}

// Validate checks a transaction for correctness and applies quick fixes where
// applicable (e.g., resolving "sell all"). It returns the validated (and
// potentially modified) transaction or an error detailing any validation failures.
func (as *AccountingSystem) Validate(tx Transaction) (Transaction, error) {
	var err error
	// The type switch creates a copy (v) of the transaction struct.
	// We must call Validate on a pointer to this copy (&v) to allow modifications.
	// We then return the (potentially modified) copy.
	switch v := tx.(type) {
	case Buy:
		err = (&v).Validate(as)
		return v, err
	case Sell:
		err = (&v).Validate(as)
		return v, err
	case Dividend:
		err = (&v).Validate(as)
		return v, err
	case Deposit:
		err = (&v).Validate(as)
		return v, err
	case Withdraw:
		err = (&v).Validate(as)
		return v, err
	case Convert:
		err = (&v).Validate(as)
		return v, err
	default:
		return tx, fmt.Errorf("unsupported transaction type for validation: %T", tx)
	}
}

// TotalMarketValue calculates the total value of all security positions and cash balances on a given
// date, expressed in a single reporting currency. It uses the market data to find
// security prices and currency exchange rates.
func (as *AccountingSystem) TotalMarketValue(on date.Date, reportingCurrency string) (float64, error) {
	if err := ValidateCurrency(reportingCurrency); err != nil {
		return 0, fmt.Errorf("invalid reporting currency: %w", err)
	}

	var totalValue float64

	// Calculate value of all security positions
	for _, ticker := range as.Ledger.AllSecurities() {
		position := as.Ledger.Position(ticker, on)
		if position <= 0 {
			continue
		}

		sec := as.MarketData.Get(ticker)
		if sec == nil {
			// This should not happen if validation is correct
			return 0, fmt.Errorf("security %q found in ledger but not in market data", ticker)
		}

		price, ok := as.MarketData.PriceAsOf(ticker, on)
		if !ok {
			return 0, fmt.Errorf("could not find price for security %q as of %s", ticker, on)
		}

		positionValue := position * price

		// Convert to reporting currency if necessary
		convertedValue, err := as.convertCurrency(positionValue, sec.currency, reportingCurrency, on)
		if err != nil {
			return 0, err
		}
		totalValue += convertedValue
	}

	// Add cash balances
	for _, currency := range as.Ledger.AllCurrencies() {
		balance := as.Ledger.CashBalance(currency, on)
		if balance == 0 {
			continue
		}

		convertedBalance, err := as.convertCurrency(balance, currency, reportingCurrency, on)
		if err != nil {
			return 0, err
		}
		totalValue += convertedBalance
	}

	return totalValue, nil
}

// convertCurrency converts an amount from a source currency to a target currency as of a given date.
func (as *AccountingSystem) convertCurrency(amount float64, fromCurrency, toCurrency string, on date.Date) (float64, error) {
	if fromCurrency == toCurrency {
		return amount, nil
	}

	// To convert from fromCurrency to toCurrency, we need the pair fromCurrency + toCurrency.
	pairTicker := fromCurrency + toCurrency
	rate, ok := as.MarketData.PriceAsOf(pairTicker, on)
	if !ok {
		// If the direct pair is not found, try the inverse pair.
		inversePairTicker := toCurrency + fromCurrency
		inverseRate, ok := as.MarketData.PriceAsOf(inversePairTicker, on)
		if !ok {
			return 0, fmt.Errorf("could not find exchange rate for %s to %s as of %s", fromCurrency, toCurrency, on)
		}
		if inverseRate == 0 {
			return 0, fmt.Errorf("inverse exchange rate for %s is zero, cannot convert", inversePairTicker)
		}
		rate = 1.0 / inverseRate
	}
	return amount * rate, nil
}
