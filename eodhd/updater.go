package eodhd

import (
	"fmt"
	"log"

	"github.com/etnz/portfolio"
	"github.com/shopspring/decimal"
)

// point is a private struct to hold together the date and ID of a price update.
type point struct {
	Date portfolio.Date
	ID   portfolio.ID
}

type Change interface {
	When() portfolio.Date
	What() portfolio.ID
}

type PriceChange struct {
	Date     portfolio.Date
	ID       portfolio.ID
	Old      *decimal.Decimal
	New      decimal.Decimal
	Currency string
}

func (c PriceChange) When() portfolio.Date { return c.Date }
func (c PriceChange) What() portfolio.ID   { return c.ID }

type SplitChange struct {
	Date        portfolio.Date
	ID          portfolio.ID
	Numerator   int64
	Denominator int64
}

func (c SplitChange) When() portfolio.Date { return c.Date }
func (c SplitChange) What() portfolio.ID   { return c.ID }

type DividendChange struct {
	Date     portfolio.Date
	ID       portfolio.ID
	Amount   decimal.Decimal
	Currency string
}

func (c DividendChange) When() portfolio.Date { return c.Date }
func (c DividendChange) What() portfolio.ID   { return c.ID }

// FetchOptions is a bitmask to control what data to fetch.
type FetchOptions uint

const (
	// FetchForex fetches data for currency pairs.
	FetchForex FetchOptions = 1 << iota
	// FetchMSSI fetches data for securities identified by MSSI.
	FetchMSSI
	// FetchISIN fetches data for securities identified by ISIN.
	FetchISIN
	FetchPrices
	FetchSplits
	FetchDividends

	// FetchAll is a convenience constant to fetch all data types.
	FetchAll = FetchForex | FetchMSSI | FetchISIN | FetchPrices | FetchSplits | FetchDividends
)

// Fetch retrieves market data from the EODHD API for securities in the provided ledger.
// It can fetch historical prices, splits, and dividends based on the specified options.
//
// The function determines the date range for fetching data for each security. If 'inception'
// is true, it fetches from the security's first appearance in the ledger. Otherwise, it
// performs an incremental update, fetching from the date of the last known market data
// up to either today or the last day the asset was held.
//
// Parameters:
//   - key: The EODHD API key.
//   - ledger: The portfolio ledger containing the securities to update.
//   - inception: If true, fetches data from the security's inception date. If false, fetches incrementally.
//   - options: A bitmask of FetchOptions (e.g., FetchPrices, FetchSplits) to specify what data to retrieve. If 0, it defaults to FetchAll.
//   - tickers: An optional list of security tickers to update. If empty, it updates all relevant securities in the ledger.
//
// Returns:
//   - A slice of Change interfaces, providing a detailed log of each data point fetched.
//   - A slice of portfolio.Transaction objects (UpdatePrice, Split, Dividend) ready to be applied to a ledger.
//   - An error if the API request or data processing fails.
func Fetch(key string, ledger *portfolio.Ledger, inception bool, options FetchOptions, tickers ...string) ([]Change, []portfolio.Transaction, error) {
	if options == 0 {
		options = FetchAll
	}
	prices := make(map[point]PriceChange)
	splits := make(map[point]SplitChange)
	dividends := make(map[point]DividendChange)

	tickersToUpdate := make(map[string]struct{})
	if len(tickers) > 0 {
		for _, t := range tickers {
			tickersToUpdate[t] = struct{}{}
		}
	}

	visited := make(map[portfolio.ID]struct{})
	id2Sec := make(map[portfolio.ID][]portfolio.Security)

	for sec := range ledger.AllSecurities() {
		// If a list of tickers is provided, only update those.
		if len(tickersToUpdate) > 0 {
			if _, shouldUpdate := tickersToUpdate[sec.Ticker()]; !shouldUpdate {
				continue
			}
		}
		id := sec.ID()
		id2Sec[id] = append(id2Sec[id], sec)
		// Check if we should fetch data for this security type based on options
		shouldFetch := false
		if (options&FetchForex != 0) && id.IsCurrencyPair() {
			shouldFetch = true
		}
		if (options&FetchMSSI != 0) && id.IsMSSI() {
			shouldFetch = true
		}
		if (options&FetchISIN != 0) && id.IsISIN() {
			shouldFetch = true
		}
		if !shouldFetch {
			continue
		}

		// Skip ID already visited.
		// This is possible because it is possible to have
		// two securities with a same ID and different tickers.
		if _, ok := visited[id]; ok {
			continue
		}
		visited[id] = struct{}{}

		// Compute the fetch bounds.
		from, to, err := computeBounds(sec, ledger, inception)
		if err != nil {
			return nil, nil, err
		}

		if from.After(to) { // from <= to
			// empty range, skip it
			continue
		}

		// Compute the ticker for this security
		ticker, err := findTicker(key, sec)
		if err != nil {
			// temp, ignore that error
			log.Println("warning", err)
			continue
			// return nil, nil, err
		}

		if options&FetchPrices != 0 {
			if err := findPrices(key, id, ticker, from, to, prices); err != nil {
				return nil, nil, err
			}
		}

		if id.IsMSSI() {
			if options&FetchSplits != 0 {
				if err := fetchSplits(key, id, ticker, from, to, splits); err != nil {
					return nil, nil, err
				}
			}
			if options&FetchDividends != 0 {
				if err := fetchDividends(key, id, ticker, from, to, dividends); err != nil {
					return nil, nil, err
				}
			}
		}
	}

	// Format the prices received in to
	changes := make([]Change, 0, len(prices)+len(splits)+len(dividends))
	updates := make([]portfolio.Transaction, 0, len(prices)+len(splits)+len(dividends))

	for _, v := range splits {
		changes = append(changes, v)
		for _, sec := range id2Sec[v.ID] {
			updates = append(updates, portfolio.NewSplit(v.Date, sec.Ticker(), v.Numerator, v.Denominator))
		}
	}
	for _, v := range dividends {
		changes = append(changes, v)
		for _, sec := range id2Sec[v.ID] {
			// The dividend currency from the API is the source of truth, as companies
			// pay dividends in their home currency, which may differ from the
			// security's trading currency on a specific exchange (e.g., a US company
			// paying in USD for shares traded in EUR on XETRA).
			// The portfolio ledger will correctly credit the cash to the corresponding currency account.
			updates = append(updates, portfolio.NewDividend(v.Date, "fetched from eodhd.com", sec.Ticker(), portfolio.M(v.Amount, v.Currency)))
		}
	}
	for _, v := range prices {
		changes = append(changes, v)
		for _, sec := range id2Sec[v.ID] {
			updates = append(updates, portfolio.NewUpdatePrice(v.Date, sec.Ticker(), portfolio.M(v.New, sec.Currency())))
		}
	}
	return changes, updates, nil
}

func computeBounds(sec portfolio.Security, ledger *portfolio.Ledger, inception bool) (from, to portfolio.Date, err error) {
	id := sec.ID()
	if id.IsCurrencyPair() {
		return forexBounds(sec, ledger, inception)
	}
	if id.IsISIN() || id.IsMSSI() {
		return assetBounds(sec, ledger, inception)
	}
	return from, to, fmt.Errorf("security %s is not a traded asset in eodhd", id)

}

// forexBounds computes the bounds to query for that security.
func forexBounds(sec portfolio.Security, ledger *portfolio.Ledger, inception bool) (from, to portfolio.Date, err error) {
	id := sec.ID()
	_, _, err = id.CurrencyPair()
	if err != nil {
		return from, to, err
	}

	// Determine the from Date.
	if inception {
		from = ledger.InceptionDate(sec.Ticker())
	} else {
		from = ledger.LastKnownMarketDataDate(sec.Ticker())
	}

	// Determine the to Date.
	to = portfolio.Today().Add(-1)
	return from, to, nil
}

// assetBounds computes the time bounds to query for that security.
func assetBounds(sec portfolio.Security, ledger *portfolio.Ledger, inception bool) (from, to portfolio.Date, err error) {
	// Compute all significant parts of the ID
	id := sec.ID()

	if !(id.IsISIN() || id.IsMSSI()) {
		// It is not a valid ID for eodhd, skipping it
		return from, to, fmt.Errorf("security %s is not a traded asset in eodhd", id)
	}

	// Determine the from Date.
	// The oldest possible value is inception date.
	from = ledger.InceptionDate(sec.Ticker())
	if !inception {
		// If we want incremental updates, start from the last known date + 1
		lastKnown := ledger.LastKnownMarketDataDate(sec.Ticker()).Add(1)
		// caveat: if there are no market data known (it's the case for new assets),
		// lastKnown will be the zero date, which is before inception date.
		// so we need to take the max of both.
		if lastKnown.After(from) {
			from = lastKnown
		}
	}

	// Determine the to Date.
	to = portfolio.Today().Add(-1)
	// For forex it's always today.
	// For Asset, it could be the last day the asset was held.
	// It's an Asset, the current holding could be 0. check it out.
	lastOperationDate := ledger.LastOperationDate(sec.Ticker())
	pos := ledger.Position(lastOperationDate, sec.Ticker())
	if pos.IsZero() {
		// Latest position is 0, so we don't need updates beyond this.
		to = lastOperationDate
		// we include the day of selling because: if we try to fetch for a too small range
		// there is the risk that there is no data for this range, and we will try for every.
		// The sell day is very likely an open day with actual data. (but it is not guaranteed)
	}
	return from, to, nil
}
