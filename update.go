package portfolio

import (
	"errors"
	"fmt"
	"log"

	"github.com/etnz/portfolio/date"
)

// This file contains functions to update the database with latest prices.

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
		latest, _ := prices.Latest()
		sec := m.Get(id)

		// If we already have yesterday's price, we are up-to-date.
		if !latest.Before(end) {
			continue
		}

		// Determine the start date for fetching new prices.
		// If no prices exist, use the default origin. Otherwise, start from the day after the latest price.
		fetchFrom := start
		if latest.Before(start) {
			fetchFrom = latest.Add(1)
		}

		// Don't try to fetch from the future.
		if fetchFrom.After(end) {
			continue
		}

		if err := updateSecurityPrices(sec, prices, fetchFrom, end); err != nil {
			errs = errors.Join(errs, err)
			continue
		}
	}
	return errs
}

// updateSecuritySplits attempts to fetch and update splits for a single security.
// updateSecuritySplits attempts to fetch and update splits for a single security.
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
