package portfolio

import (
	"errors"
	"fmt"
	"iter"
	"log"

	"github.com/etnz/portfolio/date"
)

// Split represents a stock split event.
type Split struct {
	Date        date.Date `json:"date"`
	Numerator   int64     `json:"num"`
	Denominator int64     `json:"den"`
}

// MarketData holds all the market data, including security definitions and their price histories.
type MarketData struct {
	securities map[ID]Security
	tickers    map[string]ID
	prices     map[ID]*date.History[float64]
	splits     map[ID][]Split
}

// NewMarketData creates an empty MarketData store.
func NewMarketData() *MarketData {
	return &MarketData{
		securities: make(map[ID]Security),
		tickers:    make(map[string]ID),
		prices:     make(map[ID]*date.History[float64]),
		splits:     make(map[ID][]Split),
	}
}

// Securities returns an iterator over all securities in the market data.
func (m *MarketData) Securities() iter.Seq[Security] {
	return func(yield func(Security) bool) {
		for _, sec := range m.securities {
			if !yield(sec) {
				break
			}
		}
	}
}

// Add adds a security to the market data. It also initializes an empty price history for it.
func (m *MarketData) Add(s Security) {
	if _, ok := m.securities[s.ID()]; ok {
		return
	}
	m.securities[s.ID()] = s
	m.tickers[s.Ticker()] = s.ID()
	m.prices[s.ID()] = &date.History[float64]{}
	m.splits[s.ID()] = []Split{}
}

// Get retrieves a security by its ID. It returns zero if the security is not found.
func (m *MarketData) Get(id ID) Security { return m.securities[id] }

// Resolve converts a ticker to a security ID.
func (m *MarketData) Resolve(ticker string) ID {
	return m.tickers[ticker]
}

// PriceAsOf returns the price of a security on a given date.
func (m *MarketData) PriceAsOf(id ID, on date.Date) (float64, bool) {
	if prices, ok := m.prices[id]; ok {
		return prices.ValueAsOf(on)
	}
	return 0, false
}

func (m *MarketData) Append(id ID, day date.Date, price float64) bool {
	if prices, ok := m.prices[id]; ok {
		prices.Append(day, price)
		return true
	}
	return false
}

// SetPrice sets the price for a security on a specific date.
// This is used for manual price adjustments.
func (m *MarketData) SetPrice(id ID, day date.Date, price float64) error {
	// check that the security exists
	if _, ok := m.securities[id]; !ok {
		return fmt.Errorf("security with ID %q not found", id)
	}
	if prices, ok := m.prices[id]; ok {
		prices.Append(day, price)
		return nil
	}
	// this should not happen if the security exists
	return fmt.Errorf("price history not found for security with ID %q", id)
}

// Values return a iterator on date and prices for the given ID (or nil)
func (m *MarketData) Prices(id ID) iter.Seq2[date.Date, float64] {
	prices, ok := m.prices[id]
	if !ok {
		return func(yield func(date.Date, float64) bool) {}
	}
	return prices.Values()

}

// Has checks if a security with the given ticker exists in the market data.
func (m *MarketData) Has(ticker string) bool {
	_, ok := m.tickers[ticker]
	return ok
}

// read retrieves the price for a given security on a specific day.
// It returns the price and true if found, otherwise it returns 0.0 and false.
func (m *MarketData) read(id ID, day date.Date) (float64, bool) {
	prices, ok := m.prices[id]
	if !ok {
		return 0.0, false
	}
	return prices.Get(day)
}

// AddSplit adds a split to the market data for a given security.
func (m *MarketData) AddSplit(id ID, split Split) {
	m.splits[id] = append(m.splits[id], split)
}

// Splits returns all splits for a given security.
func (m *MarketData) Splits(id ID) []Split {
	return m.splits[id]
}

// SetSplits sets the splits for a given security, replacing any existing ones.
func (m *MarketData) SetSplits(id ID, splits []Split) {
	m.splits[id] = splits
}

// updateSecurityPrices attempts to fetch and update prices for a single security.
func updateSecurityPrices(sec Security, prices *date.History[float64], from, to date.Date) error {
	apiKey := eodhdApiKey()
	if apiKey == "" {
		return errors.New("EODHD API key is not set. Use -eodhd-api-key flag or EODHD_API_KEY environment variable")
	}

	var newPrices date.History[float64]
	var err error

	// Determine security type and fetch prices accordingly.
	if isin, mic, mssiErr := sec.ID().MSSI(); mssiErr == nil {
		// This is an MSSI security.
		newPrices, err = eodhdDailyMSSI(apiKey, isin, mic, from, to)
		if err != nil {
			return fmt.Errorf("failed to get prices for MSSI %s (%s): %w", sec.Ticker(), sec.ID(), err)
		}
	} else if base, quote, cpErr := sec.ID().CurrencyPair(); cpErr == nil {
		// This is a CurrencyPair.
		newPrices, err = eodhdDailyCurrencyPair(apiKey, base, quote, from, to)
		if err != nil {
			return fmt.Errorf("failed to get prices for CurrencyPair %s (%s): %w", sec.Ticker(), sec.ID(), err)
		}
	} else if isin, fundErr := sec.ID().ISIN(); fundErr == nil {
		// This is a Fund.
		newPrices, err = eodhdDailyISIN(apiKey, isin, from, to)
		if err != nil {
			return fmt.Errorf("failed to get prices for ISIN %s (%s): %w", sec.Ticker(), sec.ID(), err)
		}

	} else {
		// This is a private or unsupported security type for updates.
		return nil // Not an error, just nothing to do.
	}

	if newPrices.Len() == 0 {
		log.Printf("no new prices found for security %q (%v) between %s and %s", sec.Ticker(), sec.ID(), from, to)
		return nil
	}

	// Append all new prices to the security.
	for day, price := range newPrices.Values() {
		prices.Append(day, price)
	}
	return nil
}

// UpdatePrices iterates through all securities in the market data and fetches the latest
// prices for each updatable security (i.e., those with an MSSI or CurrencyPair ID).
// It fetches prices from the day after the last known price up to yesterday.
// It returns a joined error if any updates fail.
func (m *MarketData) UpdatePrices(start, end date.Date) error {

	var errs error

	for id, prices := range m.prices {
		//latest, _ := prices.Latest()
		sec := m.Get(id)

		// Don't try to fetch from the future.
		if start.After(end) {
			continue
		}

		if err := updateSecurityPrices(sec, prices, start, end); err != nil {
			errs = errors.Join(errs, err)
			continue
		}
	}
	return errs
}

func updateSecuritySplits(sec Security) ([]Split, error) {
	apiKey := eodhdApiKey()
	if apiKey == "" {
		return nil, errors.New("EODHD API key is not set. Use -eodhd-api-key flag or EODHD_API_KEY environment variable")
	}

	var ticker string
	var err error

	// Determine security type and fetch ticker accordingly.
	if isin, mic, mssiErr := sec.ID().MSSI(); mssiErr == nil {
		// This is an MSSI security.
		ticker, err = eodhdSearchByMSSI(apiKey, isin, mic)
		if err != nil {
			return nil, fmt.Errorf("failed to get ticker for MSSI %s (%s): %w", sec.Ticker(), sec.ID(), err)
		}
	} else if isin, fundErr := sec.ID().ISIN(); fundErr == nil {
		// This is a Fund.
		ticker, err = eodhdSearchByISIN(apiKey, isin)
		if err != nil {
			return nil, fmt.Errorf("failed to get ticker for ISIN %s (%s): %w", sec.Ticker(), sec.ID(), err)
		}
	} else {
		// This is a private or unsupported security type for updates.
		return nil, nil // Not an error, just nothing to do.
	}

	if ticker == "" {
		return nil, nil // No ticker found, nothing to do.
	}

	return eodhdSplits(apiKey, ticker)
}

// UpdateSplits iterates through all securities in the market data and fetches the latest
// splits for each updatable security.
func (m *MarketData) UpdateSplits() error {
	var errs error

	for id, sec := range m.securities {
		newSplits, err := updateSecuritySplits(sec)
		if err != nil {
			log.Printf("failed to update splits for security %q (%v): %v", sec.Ticker(), sec.ID(), err)
			errs = errors.Join(errs, err)
			continue
		}

		if len(newSplits) > 0 {
			log.Printf("found %d splits for security %q (%v)", len(newSplits), sec.Ticker(), sec.ID())
			m.SetSplits(id, newSplits)
		}
	}
	return errs
}

func (m *MarketData) UpdateIntraday() error {

	// Update the EURUSD ticker
	val, err := tradegateLatestEURperUSD()
	if err != nil {
		return err
	}
	id, _ := NewCurrencyPair("USD", "EUR")
	m.Append(id, date.Today(), 1/val)

	// then update stocks
	for id, sec := range m.securities {

		var latest float64
		var err error

		// If it's a stock
		if isin, _, mssiErr := id.MSSI(); mssiErr == nil {
			latest, err = tradegateLatest(sec.Ticker(), isin)
		} else if isin, fundErr := id.ISIN(); fundErr == nil {
			latest, err = tradegateLatest(sec.Ticker(), isin)
		} else {
			continue
		}
		if err != nil {
			log.Printf("warning error reading intraday value for %s: %v", sec.Ticker(), err)
		} else {
			if sec.Currency() == "USD" {
				// all assets in tradegate are in eur (so far) so convert back to USD if needed.
				m.Append(id, date.Today(), latest*val)
			}
			if sec.Currency() == "EUR" {
				m.Append(id, date.Today(), latest)
			}
		}
	}
	return nil

}
