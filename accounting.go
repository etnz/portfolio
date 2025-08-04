package portfolio

import (
	"fmt"

	"time"
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
	Ledger            *Ledger
	MarketData        *MarketData
	ReportingCurrency string
}

// NewAccountingSystem creates a new accounting system from a ledger and market data.
func NewAccountingSystem(ledger *Ledger, marketData *MarketData, reportingCurrency string) (*AccountingSystem, error) {
	if reportingCurrency != "" {
		if err := ValidateCurrency(reportingCurrency); err != nil {
			return nil, fmt.Errorf("invalid reporting currency: %w", err)
		}
	}
	return &AccountingSystem{
		Ledger:            ledger,
		MarketData:        marketData,
		ReportingCurrency: reportingCurrency,
	}, nil
}

// Validate checks a transaction for correctness and applies quick fixes where
// applicable (e.g., resolving "sell all"). It returns the validated (and
// potentially modified) transaction or an error detailing any validation failures.
func (as *AccountingSystem) Validate(tx Transaction) (Transaction, error) {
	var err error
	switch v := tx.(type) {
	case Buy:
		err =v.Validate(as)
		return v, err
	case Sell:
		err =v.Validate(as)
		return v, err
	case Dividend:
		err =v.Validate(as)
		return v, err
	case Deposit:
		err =v.Validate(as)
		return v, err
	case Withdraw:
		err =v.Validate(as)
		return v, err
	case Convert:
		err =v.Validate(as)
		return v, err
	default:
		return tx, fmt.Errorf("unsupported transaction type for validation: %T", tx)
	}
}

// TotalMarketValue calculates the total value of all security positions and cash balances on a given
// date, expressed in a single reporting currency. It uses the market data to find
// security prices and currency exchange rates.
func (as *AccountingSystem) TotalMarketValue(on date.Date) (float64, error) {
	if as.ReportingCurrency == "" {
		return 0, fmt.Errorf("reporting currency is not set in accounting system")
	}

	var totalValue float64

	// Calculate value of all security positions
	for ticker := range as.Ledger.AllSecurities() {
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
		convertedValue, err := as.convertCurrency(positionValue, sec.currency, as.ReportingCurrency, on)
		if err != nil {
			return 0, err
		}
		totalValue += convertedValue
	}

	// Add cash balances
	for currency := range as.Ledger.AllCurrencies() {
		balance := as.Ledger.CashBalance(currency, on)
		if balance == 0 {
			continue
		}

		convertedBalance, err := as.convertCurrency(balance, currency, as.ReportingCurrency, on)
		if err != nil {
			return 0, err
		}
		totalValue += convertedBalance
	}

	return totalValue, nil
}

// calculatePeriodPerformance is a helper to compute the return for a given period.
func (as *AccountingSystem) calculatePeriodPerformance(currentTMV float64, periodStartDate date.Date) (Performance, error) {
	// The value at the start of the period is the value at the end of the day before.
	startValueDate := periodStartDate.Add(-1)

	startValue, err := as.TotalMarketValue(startValueDate)
	if err != nil {
		return Performance{}, fmt.Errorf("could not get start value for date %s: %w", startValueDate, err)
	}

	if startValue == 0 {
		// Avoid division by zero. If start value was 0, return is not meaningful.
		return Performance{StartValue: 0, Return: 0}, nil
	}

	// Simple return calculation. This doesn't account for cash flows during the period.
	perfReturn := (currentTMV - startValue) / startValue
	return Performance{StartValue: startValue, Return: perfReturn}, nil
}

// NewSummary calculates and returns a comprehensive summary of the portfolio's
// state and performance on a given date.
func (as *AccountingSystem) NewSummary(on date.Date) (*Summary, error) {
	if as.ReportingCurrency == "" {
		return nil, fmt.Errorf("reporting currency is not set in accounting system")
	}

	summary := &Summary{
		Date:              on,
		ReportingCurrency: as.ReportingCurrency,
	}

	// 1. Calculate current total market value
	currentTMV, err := as.TotalMarketValue(on)
	if err != nil {
		return nil, fmt.Errorf("could not calculate summary: %w", err)
	}
	summary.TotalMarketValue = currentTMV

	// 2. Calculate performance for each period
	if summary.Daily, err = as.calculatePeriodPerformance(currentTMV, on); err != nil {
		return nil, fmt.Errorf("failed to calculate daily performance: %w", err)
	}
	if summary.WTD, err = as.calculatePeriodPerformance(currentTMV, date.StartOfWeek(on)); err != nil {
		return nil, fmt.Errorf("failed to calculate WTD performance: %w", err)
	}
	if summary.MTD, err = as.calculatePeriodPerformance(currentTMV, date.StartOfMonth(on)); err != nil {
		return nil, fmt.Errorf("failed to calculate MTD performance: %w", err)
	}
	if summary.QTD, err = as.calculatePeriodPerformance(currentTMV, date.StartOfQuarter(on)); err != nil {
		return nil, fmt.Errorf("failed to calculate QTD performance: %w", err)
	}
	if summary.YTD, err = as.calculatePeriodPerformance(currentTMV, date.StartOfYear(on)); err != nil {
		return nil, fmt.Errorf("failed to calculate YTD performance: %w", err)
	}

	// 3. Calculate performance since inception
	inceptionCostBasis, err := as.CostBasis(on)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate inception performance: %w", err)
	}
	summary.Inception.StartValue = inceptionCostBasis
	if inceptionCostBasis != 0 {
		summary.Inception.Return = (currentTMV - inceptionCostBasis) / inceptionCostBasis
	}

	return summary, nil
}
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

// CostBasis calculates the total net cash invested in the portfolio as of a
// specific date.
// It provides a stable cost basis by converting each cash flow
// (Deposit/Withdraw) to the reporting currency using the exchange rate on the
// day of the transaction. This method correctly separates investment performance
// from currency fluctuations.
func (as *AccountingSystem) CostBasis(on date.Date) (float64, error) {
	if as.ReportingCurrency == "" {
		return 0, fmt.Errorf("reporting currency is not set in accounting system")
	}

	var totalCostBasis float64

	for _, tx := range as.Ledger.Transactions() {
		if tx.When().After(on) {
			// The ledger is sorted, so we can stop iterating.
			break
		}
		if !tx.What().IsCashFlow() {
			continue
		}

		var amount float64
		var currency string

		switch v := tx.(type) {
		case Deposit:
			amount = v.Amount
			currency = v.Currency
		case Withdraw:
			amount = -v.Amount // Use negative amount for withdrawal
			currency = v.Currency
		}

		convertedAmount, err := as.convertCurrency(amount, currency, as.ReportingCurrency, tx.When())
		if err != nil {
			return 0, fmt.Errorf("could not convert cost basis for transaction on %s: %w", tx.When(), err)
		}
		totalCostBasis += convertedAmount

	}
	return totalCostBasis, nil
}
