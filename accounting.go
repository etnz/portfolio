package portfolio

import (
	"fmt"
	"time"

	"github.com/etnz/portfolio/date"
	"github.com/shopspring/decimal"
)

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
	journal           *Journal
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

// DeclareSecurities scans all securities (and currencies) in the ledger and
// ensures they are declared in the marketdata. This function is crucial for
// maintaining consistency between the ledger's transactional records and the
// market data's security definitions.
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
	case Declare:
		err = v.Validate(as)
		return v, err
	case Accrue:
		err = v.Validate(as)
		return v, err
	default:
		return tx, fmt.Errorf("unsupported transaction type for validation: %T", tx)
	}
}
func (as *AccountingSystem) getJournal() (*Journal, error) {
	var err error
	if as.journal == nil {
		as.journal, err = as.newJournal()
	}
	if err != nil {
		return nil, err
	}
	return as.journal, nil
}

// Balance computes the Balance on a given day.
func (as *AccountingSystem) Balance(on date.Date) (*Balance, error) {
	j, err := as.getJournal()
	if err != nil {
		return nil, fmt.Errorf("could not get journal: %w", err)
	}
	balance, err := NewBalance(j, on, FIFO) // TODO: Make cost basis method configurable
	if err != nil {
		return nil, fmt.Errorf("could not create balance from journal: %w", err)
	}
	return balance, nil
}

// TotalPortfolioValue calculates the total value of all security positions and cash balances on a given
// date, expressed in a single reporting currency. It uses the market data to find
// security prices and currency exchange rates.
func (as *AccountingSystem) TotalPortfolioValue(on date.Date) (float64, error) {
	balance, err := as.Balance(on)
	if err != nil {
		return 0, fmt.Errorf("could not get balance for %s: %w", on, err)
	}
	return balance.TotalPortfolioValue().InexactFloat64(), nil
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

	endBalance, err := as.Balance(on)
	if err != nil {
		return nil, err
	}

	yesterdayBalance, err := as.Balance(on.Add(-1))
	if err != nil {
		return nil, err
	}

	weekBalance, err := as.Balance(date.StartOfWeek(on).Add(-1))
	if err != nil {
		return nil, err
	}

	monthBalance, err := as.Balance(date.StartOfMonth(on).Add(-1))
	if err != nil {
		return nil, err
	}

	quarterBalance, err := as.Balance(date.StartOfQuarter(on).Add(-1))
	if err != nil {
		return nil, err
	}

	yearBalance, err := as.Balance(date.StartOfYear(on).Add(-1))
	if err != nil {
		return nil, err
	}

	// 1. Calculate current total market value
	summary.TotalMarketValue = endBalance.TotalPortfolioValue().InexactFloat64()

	// 2. Calculate performance for each period
	periodTWR := func(start *Balance) (perf Performance) {
		return Performance{
			StartValue: start.TotalPortfolioValue().InexactFloat64(),
			Return:     endBalance.linkedTWR/start.linkedTWR - 1,
		}
	}

	summary.Daily = periodTWR(yesterdayBalance)
	summary.WTD = periodTWR(weekBalance)
	summary.MTD = periodTWR(monthBalance)
	summary.QTD = periodTWR(quarterBalance)
	summary.YTD = periodTWR(yearBalance)
	summary.Inception = Performance{
		StartValue: 0,
		Return:     endBalance.linkedTWR - 1,
	}
	return summary, nil
}

// NewDailyReport calculates and returns a summary of the portfolio's performance for a single day.
func (as *AccountingSystem) NewDailyReport(on date.Date) (*DailyReport, error) {
	endBalance, err := as.Balance(on)
	if err != nil {
		return nil, err
	}
	// 1. Calculate value at previous day's close
	startBalance, err := as.Balance(on.Add(-1))
	if err != nil {
		return nil, err
	}

	report := &DailyReport{
		Date:              on,
		Time:              time.Now(), // Generation time
		ReportingCurrency: as.ReportingCurrency,
		ActiveAssets:      []AssetGain{},
		Transactions:      []Transaction{},
	}

	valueAtClose := endBalance.TotalPortfolioValue()
	valueAtPrevClose := startBalance.TotalPortfolioValue()
	report.ValueAtClose = NewMoney(valueAtClose, as.ReportingCurrency)
	report.ValueAtPrevClose = NewMoney(valueAtPrevClose, as.ReportingCurrency)

	// 3. Get all transactions for the specified day
	for _, tx := range as.Ledger.transactions {
		if tx.When() == on {
			report.Transactions = append(report.Transactions, tx)
		}
	}

	// 4. Calculate Net Cash Flow and Realized Gains for the day
	netCashFlow := endBalance.TotalCash().Sub(startBalance.TotalCash())
	report.NetCashFlow = NewMoney(netCashFlow, as.ReportingCurrency)
	// Gains have to be computed per security (in fact per securities currency)
	// then converted to the reporting currency.
	totalRealized := decimal.Zero
	for sec := range endBalance.Securities() {
		ticker := sec.Ticker()
		gain := endBalance.RealizedGain(ticker).Sub(startBalance.RealizedGain(ticker))
		gain = endBalance.Convert(gain, sec.Currency())
		totalRealized = totalRealized.Add(gain)
	}
	report.RealizedGains = NewMoney(totalRealized, as.ReportingCurrency)

	// 5. Calculate Total Gain and Market Gains
	report.TotalGain = NewMoney(valueAtClose.Sub(valueAtPrevClose), as.ReportingCurrency)
	report.MarketGains = NewMoney(valueAtClose.Sub(valueAtPrevClose).Sub(totalRealized).Sub(netCashFlow), as.ReportingCurrency)

	// 6. Calculate Active Asset Gains
	for sec := range endBalance.Securities() {

		if endBalance.Position(sec.Ticker()).IsZero() {
			continue // ignore assets not held today
		}

		valueToday := endBalance.MarketValue(sec.Ticker())
		valuePrev := startBalance.MarketValue(sec.Ticker())
		gain := valueToday.Sub(valuePrev)

		// Adjust for buys/sells during the day
		for _, tx := range report.Transactions {
			switch v := tx.(type) {
			case Buy:
				if v.Security == sec.Ticker() {
					cost := decimal.NewFromFloat(v.Amount)
					gain = gain.Sub(cost)
				}
			case Sell:
				if v.Security == sec.Ticker() {
					proceeds := decimal.NewFromFloat(v.Amount)
					gain = gain.Add(proceeds)
				}
			}
		}

		yield := 0.0
		if !valuePrev.IsZero() { // if there was an initial value
			yield = (gain.Div(valuePrev).InexactFloat64()) * 100
		}

		// convert to reporting currency
		gain = endBalance.Convert(gain, sec.Currency())
		assetGain := AssetGain{
			Security: sec.Ticker(),
			Gain:     NewMoney(gain, as.ReportingCurrency),
			Return:   Percent(yield),
		}

		report.ActiveAssets = append(report.ActiveAssets, assetGain)

	}

	return report, nil
}

// CostBasis calculates the total net cash invested in the portfolio as of a
// specific date.
// It provides a stable cost basis by converting each cash flow
// (Deposit/Withdraw) to the reporting currency using the exchange rate on the
// day of the transaction. This method correctly separates investment performance
// from currency fluctuations.
func (as *AccountingSystem) CostBasis(on date.Date) (float64, error) {
	bal, err := as.Balance(on)
	if err != nil {
		return 0, err
	}
	return bal.TotalCostBasis().InexactFloat64(), nil
}

// CalculateGains computes the realized and unrealized gains for all securities
// over a given period, using a specified cost basis accounting method.
func (as *AccountingSystem) CalculateGains(period date.Range, method CostBasisMethod) (*GainsReport, error) {
	report := &GainsReport{
		Range:             period,
		Method:            method,
		ReportingCurrency: as.ReportingCurrency,
		Securities:        []SecurityGains{},
	}

	journal, err := as.getJournal()
	if err != nil {
		return nil, fmt.Errorf("could not get journal: %w", err)
	}

	endBalance, err := NewBalance(journal, period.To, method)
	if err != nil {
		return nil, fmt.Errorf("could not create balance from journal: %w", err)
	}
	startBalance, err := NewBalance(journal, period.From.Add(-1), method)
	if err != nil {
		return nil, fmt.Errorf("could not create balance from journal: %w", err)
	}

	for sec := range as.Ledger.AllSecurities() {

		realizedGainEnd := endBalance.RealizedGain(sec.Ticker())
		realizedGainStart := startBalance.RealizedGain(sec.Ticker())

		realizedGain := realizedGainEnd.Sub(realizedGainStart)

		// Unrealized Gain
		costBasisEnd := endBalance.CostBasis(sec.Ticker())
		marketValueEnd := endBalance.MarketValue(sec.Ticker())
		unrealizedGainEnd := marketValueEnd.Sub(costBasisEnd)

		costBasisStart := startBalance.CostBasis(sec.Ticker())
		marketValueStart := startBalance.MarketValue(sec.Ticker())
		unrealizedGainStart := marketValueStart.Sub(costBasisStart)

		unrealizedGain := unrealizedGainEnd.Sub(unrealizedGainStart)

		if realizedGain.IsZero() && unrealizedGain.IsZero() {
			continue
		}

		report.Securities = append(report.Securities, SecurityGains{
			Security:    sec.Ticker(),
			Realized:    realizedGain.InexactFloat64(),
			Unrealized:  unrealizedGain.InexactFloat64(),
			Total:       realizedGain.Add(unrealizedGain).InexactFloat64(),
			CostBasis:   costBasisEnd.InexactFloat64(),
			MarketValue: marketValueEnd.InexactFloat64(),
			Quantity:    endBalance.Position(sec.Ticker()).InexactFloat64(),
		})
	}

	return report, nil
}

// NewHoldingReport calculates and returns a detailed holdings report for a given date.
func (as *AccountingSystem) NewHoldingReport(on date.Date) (*HoldingReport, error) {
	report := &HoldingReport{
		Date:              on,
		Time:              time.Now(), // Generation time
		ReportingCurrency: as.ReportingCurrency,
		Securities:        []SecurityHolding{},
		Cash:              []CashHolding{},
		Counterparties:    []CounterpartyHolding{},
	}

	balance, err := as.Balance(on)
	if err != nil {
		return nil, err
	}

	// Securities
	for sec := range balance.Securities() {
		ticker := sec.Ticker()
		id := sec.ID()
		currency := sec.Currency()
		position := balance.Position(ticker)
		if position.IsZero() {
			continue
		}
		price := balance.Price(ticker)
		value := balance.MarketValue(ticker)
		convertedValue := balance.Convert(value, currency)
		report.Securities = append(report.Securities, SecurityHolding{
			Ticker:      ticker,
			ID:          id.String(),
			Currency:    currency,
			Quantity:    NewQuantity(position),
			Price:       NewMoney(price, currency),
			MarketValue: NewMoney(convertedValue, as.ReportingCurrency),
		})
	}

	// Cash
	for currency := range balance.Currencies() {
		bal := balance.Cash(currency)
		if bal.IsZero() {
			continue
		}
		convertedBalance := balance.Convert(bal, currency)
		report.Cash = append(report.Cash, CashHolding{
			Currency: currency,
			Balance:  NewMoney(bal, currency),
			Value:    NewMoney(convertedBalance, as.ReportingCurrency),
		})
	}

	// Counterparties
	for account := range balance.Counterparties() {
		bal, currency := balance.Counterparty(account), balance.CounterpartyCurrency(account)
		if bal.IsZero() {
			continue
		}
		convertedBalance := balance.Convert(bal, currency)
		report.Counterparties = append(report.Counterparties, CounterpartyHolding{
			Name:     account,
			Currency: currency,
			Balance:  NewMoney(bal, currency),
			Value:    NewMoney(convertedBalance, as.ReportingCurrency),
		})
	}

	report.TotalValue = NewMoney(balance.TotalPortfolioValue(), as.ReportingCurrency)

	return report, nil
}
