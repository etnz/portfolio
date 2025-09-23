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
	Date portfolio.Date
	ID   portfolio.ID
	Old  *decimal.Decimal
	New  decimal.Decimal
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
	Date   portfolio.Date
	ID     portfolio.ID
	Amount decimal.Decimal
}

func (c DividendChange) When() portfolio.Date { return c.Date }
func (c DividendChange) What() portfolio.ID   { return c.ID }

func Fetch(key string, ledger *portfolio.Ledger, inception bool) ([]Change, []portfolio.Transaction, error) {
	// For each asset in the ledger relative to MSSI
	// if inception: fetch all from inception day to today
	// otherwise: find the latest market data known for this asset, and the latest holding position
	//  fetch all from latest known market data to latest holding position.
	prices := make(map[point]PriceChange)
	splits := make(map[point]SplitChange)
	dividends := make(map[point]DividendChange)

	visited := make(map[portfolio.ID]struct{})
	id2Sec := make(map[portfolio.ID][]portfolio.Security)

	for sec := range ledger.AllSecurities() {
		id := sec.ID()
		id2Sec[id] = append(id2Sec[id], sec)

		if !(id.IsCurrencyPair() || id.IsISIN() || id.IsMSSI()) {
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

		if !to.After(from) {
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

		if err := findPrices(key, id, ticker, from, to, prices); err != nil {
			return nil, nil, err
		}

		if id.IsMSSI() {
			// Also fetch splits and dividends
			if err := fetchSplits(key, id, ticker, from, to, splits); err != nil {
				return nil, nil, err
			}
			if err := fetchDividends(key, id, ticker, from, to, dividends); err != nil {
				return nil, nil, err
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
			updates = append(updates, portfolio.NewDividend(v.Date, "", sec.Ticker(), portfolio.M(v.Amount, sec.Currency())))
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
	if inception {
		from = ledger.InceptionDate(sec.Ticker())
	} else {
		from = ledger.LastKnownMarketDataDate(sec.Ticker()).Add(1)
	}

	// Determine the to Date.
	to = portfolio.Today()
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
