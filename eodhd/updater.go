package eodhd

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"sort"

	"github.com/etnz/portfolio"
	"github.com/shopspring/decimal"
)

// point is a private struct to hold together the date and ID of a price update.
type point struct {
	Date portfolio.Date
	ID   portfolio.ID
}

type priceChange struct {
	Date     portfolio.Date
	ID       portfolio.ID
	Old      *decimal.Decimal
	New      decimal.Decimal
	Currency string
}

func (c priceChange) When() portfolio.Date { return c.Date }
func (c priceChange) What() portfolio.ID   { return c.ID }

type splitChange struct {
	Date        portfolio.Date
	ID          portfolio.ID
	Numerator   int64
	Denominator int64
}

func (c splitChange) When() portfolio.Date { return c.Date }
func (c splitChange) What() portfolio.ID   { return c.ID }

type dividendChange struct {
	Date     portfolio.Date
	ID       portfolio.ID
	Amount   decimal.Decimal
	Currency string
}

func (c dividendChange) When() portfolio.Date { return c.Date }
func (c dividendChange) What() portfolio.ID   { return c.ID }

// FetchSecurity fetches market data for a single security using the EODHD API.
// It implements the cmd.Fetcher interface.
func FetchSecurity(ctx context.Context, security portfolio.Security, from, to portfolio.Date) ([]portfolio.Transaction, error) {
	uri := security.Fetcher()
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("invalid eodhd URI: %w", err)
	}
	ticker := u.Host
	if ticker == "" {
		ticker = u.Path
	}

	apiKey := os.Getenv("EODHD_API_TOKEN")
	if apiKey == "" {
		apiKey = os.Getenv("EODHD_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("EODHD_API_TOKEN environment variable is not set")
	}

	prices := make(map[point]priceChange)
	splits := make(map[point]splitChange)
	dividends := make(map[point]dividendChange)

	id := security.ID()

	if err := findPrices(apiKey, id, ticker, from, to, prices); err != nil {
		return nil, err
	}

	if id.IsMSSI() {
		if err := fetchSplits(apiKey, id, ticker, from, to, splits); err != nil {
			return nil, err
		}
		if err := fetchDividends(apiKey, id, ticker, from, to, dividends); err != nil {
			return nil, err
		}
	}

	updates := make([]portfolio.Transaction, 0, len(prices)+len(splits)+len(dividends))

	for _, v := range splits {
		updates = append(updates, portfolio.NewSplit(v.Date, security.Ticker(), v.Numerator, v.Denominator))
	}
	for _, v := range dividends {
		updates = append(updates, portfolio.NewDividend(v.Date, "fetched from eodhd.com", security.Ticker(), portfolio.M(v.Amount, v.Currency)))
	}
	for _, v := range prices {
		updates = append(updates, portfolio.NewUpdatePrice(v.Date, security.Ticker(), portfolio.M(v.New, security.Currency())))
	}

	sort.Slice(updates, func(i, j int) bool {
		return updates[i].When().Before(updates[j].When())
	})

	return updates, nil
}
