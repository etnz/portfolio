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
	as := &AccountingSystem{
		Ledger:            ledger,
		MarketData:        marketData,
		ReportingCurrency: reportingCurrency,
	}
	// if err := as.declareSecurities(); err != nil {
	// 	return nil, fmt.Errorf("could not declare securities from ledger to the market data: %w", err)
	// }

	return as, nil
}

// declareSecurities scan all securities (and currencies) in the ledger and
// make sure they are declared in the marketdata
func DeclareSecurities(ledger *Ledger, marketData *MarketData, defaultCurrency string) error {
	if err := ValidateCurrency(defaultCurrency); err != nil {
		return fmt.Errorf("invalid default currency: %w", err)
	}

	for sec := range ledger.AllSecurities() {
		marketData.Add(sec)
	}
	for currency := range ledger.AllCurrencies() {
		if currency == defaultCurrency {
			// skip absurd self currency
			continue
		}
		id, err := NewCurrencyPair(currency, defaultCurrency)
		if err != nil {
			return fmt.Errorf("could not create currency pair: %w", err)
		}
		marketData.Add(NewSecurity(id, id.String(), currency))
	}
	return nil
}

// Validate checks a transaction for correctness and applies quick fixes where
// applicable (e.g., resolving "sell all"). It returns the validated (and
// potentially modified) transaction or an error detailing any validation failures.
func (as *AccountingSystem) Validate(tx Transaction) (Transaction, error) {
	var err error
	switch v := tx.(type) {
	case Buy:
		err = v.Validate(as)
		return v, err
	case Sell:
		err = v.Validate(as)
		return v, err
	case Dividend:
		err = v.Validate(as)
		return v, err
	case Deposit:
		err = v.Validate(as)
		return v, err
	case Withdraw:
		err = v.Validate(as)
		return v, err
	case Convert:
		err = v.Validate(as)
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
	for sec := range as.Ledger.AllSecurities() {
		position := as.Ledger.Position(sec.Ticker(), on)
		if position <= 0 {
			continue
		}

		price, ok := as.MarketData.PriceAsOf(sec.ID(), on)
		if !ok {
			return 0, fmt.Errorf("could not find price for security %q as of %s", sec.Ticker(), on)
		}

		positionValue := position * price

		// Convert to reporting currency if necessary
		convertedValue, err := as.ConvertCurrency(positionValue, sec.currency, as.ReportingCurrency, on)
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

		convertedBalance, err := as.ConvertCurrency(balance, currency, as.ReportingCurrency, on)
		if err != nil {
			return 0, err
		}
		totalValue += convertedBalance
	}

	return totalValue, nil
}

// getCashFlows retrieves all cash flow transactions (deposits and withdrawals)
// within a given date range and returns them as a map of date to the total
// net flow amount in the reporting currency for that date. The transactions are
// assumed to be in chronological order.
func (as *AccountingSystem) getCashFlows(start, end date.Date) (date.History[float64], error) {
	var flows date.History[float64]
	for _, tx := range as.Ledger.transactions {
		txDate := tx.When()
		if txDate.Before(start) {
			continue
		}
		if txDate.After(end) {
			break // The ledger is sorted, so we can stop.
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
			amount = -v.Amount
			currency = v.Currency
		default:
			// This path is unreachable due to the IsCashFlow check.
			continue
		}

		convertedAmount, err := as.ConvertCurrency(amount, currency, as.ReportingCurrency, txDate)
		if err != nil {
			return date.History[float64]{}, fmt.Errorf("could not convert cash flow on %s: %w", txDate, err)
		}
		flows.AppendAdd(txDate, convertedAmount)
	}
	return flows, nil
}

// calculatePeriodPerformance computes the Time-Weighted Return (TWR) for a given period.
// TWR measures the compound growth rate of a portfolio, removing the distorting
// effects of cash flows. This is the standard for comparing investment manager performance.
func (as *AccountingSystem) calculatePeriodPerformance(startDate, endDate date.Date) (Performance, error) {
	// The value at the start of the period is the value at the end of the day before.
	startValueDate := startDate.Add(-1)
	startValue, err := as.TotalMarketValue(startValueDate)
	if err != nil {
		return Performance{}, fmt.Errorf("could not get start value for date %s: %w", startValueDate, err)
	}

	// 1. Get all cash flows within the period.
	cashFlows, err := as.getCashFlows(startDate, endDate)
	if err != nil {
		return Performance{}, fmt.Errorf("failed to get cash flows for TWR: %w", err)
	}

	if last, _ := cashFlows.Latest(); last != endDate {
		cashFlows.Append(endDate, 0)
	}

	// 3. Geometrically link the Holding Period Return (HPR) of each sub-period.
	linkedReturn := 1.0
	lastValue := startValue

	// Iterate through each valuation date, which marks the end of a sub-period.
	for d, cashFlowOnDate := range cashFlows.Values() {
		valueAfterCF, err := as.TotalMarketValue(d)
		if err != nil {
			return Performance{}, fmt.Errorf("could not get market value for date %s: %w", d, err)
		}

		valueBeforeCF := valueAfterCF - cashFlowOnDate

		if lastValue != 0 {
			hpr := (valueBeforeCF - lastValue) / lastValue
			linkedReturn *= (1.0 + hpr)
		}
		lastValue = valueAfterCF
	}

	twr := linkedReturn - 1.0
	return Performance{StartValue: startValue, Return: twr}, nil
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
	if summary.Daily, err = as.calculatePeriodPerformance(on, on); err != nil {
		return nil, fmt.Errorf("failed to calculate daily performance: %w", err)
	}
	if summary.WTD, err = as.calculatePeriodPerformance(date.StartOfWeek(on), on); err != nil {
		return nil, fmt.Errorf("failed to calculate WTD performance: %w", err)
	}
	if summary.MTD, err = as.calculatePeriodPerformance(date.StartOfMonth(on), on); err != nil {
		return nil, fmt.Errorf("failed to calculate MTD performance: %w", err)
	}
	if summary.QTD, err = as.calculatePeriodPerformance(date.StartOfQuarter(on), on); err != nil {
		return nil, fmt.Errorf("failed to calculate QTD performance: %w", err)
	}
	if summary.YTD, err = as.calculatePeriodPerformance(date.StartOfYear(on), on); err != nil {
		return nil, fmt.Errorf("failed to calculate YTD performance: %w", err)
	}

	// 3. Calculate performance since inception
	var inceptionDate date.Date
	if len(as.Ledger.transactions) > 0 {
		inceptionDate = as.Ledger.transactions[0].When()
	} else {
		inceptionDate = on
	}
	if summary.Inception, err = as.calculatePeriodPerformance(inceptionDate, on); err != nil {
		return nil, fmt.Errorf("failed to calculate inception performance: %w", err)
	}

	return summary, nil
}

// ConvertCurrency converts an amount from a source currency to a target currency as of a given date.
func (as *AccountingSystem) ConvertCurrency(amount float64, fromCurrency, toCurrency string, on date.Date) (float64, error) {
	if fromCurrency == toCurrency {
		return amount, nil
	}

	// To convert from fromCurrency to toCurrency, we need the pair fromCurrency + toCurrency.
	pairTicker, err := NewCurrencyPair(fromCurrency, toCurrency)
	if err != nil {
		return 0, fmt.Errorf("could not create currency pair: %w", err)
	}

	rate, ok := as.MarketData.PriceAsOf(pairTicker, on)
	if !ok {
		// If the direct pair is not found, try the inverse pair.
		inversePairTicker, err := NewCurrencyPair(toCurrency, fromCurrency)
		if err != nil {
			return 0, fmt.Errorf("could not create currency pair: %w", err)
		}
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

		convertedAmount, err := as.ConvertCurrency(amount, currency, as.ReportingCurrency, tx.When())
		if err != nil {
			return 0, fmt.Errorf("could not convert cost basis for transaction on %s: %w", tx.When(), err)
		}
		totalCostBasis += convertedAmount

	}
	return totalCostBasis, nil
}
