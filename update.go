package portfolio

import (
	"errors"
	"fmt"
	"log"

	"github.com/etnz/portfolio/date"
)

// This file contains functions to update the database with latest prices.
// updatable are:
// - security available at EODHD. If the security has currently no data, a default starting date is used.
// - forex pairs.
//
// In order to figure that out, the security.ID is used to identify the security ISIN and MIC where applicable,
// or the forex pair.
// there might be other thypes of securities, but they are not supported by update, yet (like privately traded assets)

// defaultPriceHistoryStartDate is the date from which to start fetching price history if a security has no prices yet.
var defaultPriceHistoryStartDate = date.New(2020, 01, 01)

// updateSecurityPrices attempts to fetch and update prices for a single security.
func updateSecurityPrices(sec *Security, from, to date.Date) error {
	apiKey := eodhdApiKey()
	if apiKey == "" {
		return errors.New("EODHD API key is not set. Use -eodhd-api-key flag or EODHD_API_KEY environment variable")
	}

	var prices date.History[float64]
	var err error

	// Determine security type and fetch prices accordingly.
	if isin, mic, mssiErr := sec.ID().MSSI(); mssiErr == nil {
		// This is an MSSI security.
		prices, err = eodhdDailyISIN(apiKey, isin, mic, from, to)
		if err != nil {
			return fmt.Errorf("failed to get prices for MSSI %s (%s): %w", sec.Ticker(), sec.ID(), err)
		}
	} else if base, quote, cpErr := sec.ID().CurrencyPair(); cpErr == nil {
		// This is a CurrencyPair.
		prices, err = eodhdDailyFrom(apiKey, base, quote, from, to)
		if err != nil {
			return fmt.Errorf("failed to get prices for CurrencyPair %s (%s): %w", sec.Ticker(), sec.ID(), err)
		}
	} else {
		// This is a private or unsupported security type for updates.
		return nil // Not an error, just nothing to do.
	}

	if prices.Len() == 0 {
		log.Printf("no new prices found for security %q (%v) between %s and %s", sec.Ticker(), sec.ID(), from, to)
		return nil
	}

	// Append all new prices to the security.
	for day, price := range prices.Values() {
		sec.Prices().Append(day, price)
	}
	return nil
}

// Update iterates through all securities in the market data and fetches the latest
// prices for each updatable security (i.e., those with an MSSI or CurrencyPair ID).
// It fetches prices from the day after the last known price up to yesterday.
// It returns a joined error if any updates fail.
func (m *MarketData) Update() error {

	yesterday := date.Today().Add(-1)
	origin := defaultPriceHistoryStartDate

	var errs error

	for _, sec := range m.securities {
		latest, _ := sec.Prices().Latest()

		// If we already have yesterday's price, we are up-to-date.
		if !latest.Before(yesterday) {
			continue
		}

		// Determine the start date for fetching new prices.
		// If no prices exist, use the default origin. Otherwise, start from the day after the latest price.
		fetchFrom := origin
		if !latest.Before(origin) {
			fetchFrom = latest.Add(1)
		}

		// Don't try to fetch from the future.
		if !fetchFrom.Before(yesterday) {
			continue
		}

		if err := updateSecurityPrices(sec, fetchFrom, yesterday); err != nil {
			errs = errors.Join(errs, err)
			continue
		}
	}
	return errs
}
