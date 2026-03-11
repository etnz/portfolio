package amundi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/etnz/portfolio"
)

type navPoint struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

type historyItem struct {
	NavHistory []navPoint `json:"navHistory"`
}

// Fetch implements the Fetcher interface for Amundi.
// It retrieves historical NAV data for a given ISIN from Amundi's website.
func Fetch(ctx context.Context, security portfolio.Security, from, to portfolio.Date) ([]portfolio.Transaction, error) {
	u, err := url.Parse(security.Fetcher())
	if err != nil {
		return nil, fmt.Errorf("invalid fetcher URI: %w", err)
	}
	isin := u.Host
	if isin == "" {
		isin = u.Path
	}

	apiURL := fmt.Sprintf("https://www.amundi.fr/product-services/fdr/share/v3/full/%s", isin)
	body := []byte(`{"fields": ["navHistory"]}`)

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	// Using a browser-like User-Agent to avoid being blocked.
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch history for %s: %w", isin, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status code %d for %s", resp.StatusCode, isin)
	}

	var items []historyItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("failed to decode history for %s: %w", isin, err)
	}

	if len(items) == 0 {
		return nil, nil
	}

	var txs []portfolio.Transaction
	// The API returns an array, the first element contains the history.
	for _, point := range items[0].NavHistory {
		date, err := portfolio.ParseDate(point.Date)
		if err != nil {
			continue
		}
		if date.Before(from) || date.After(to) {
			continue
		}

		price := portfolio.M(point.Value, security.Currency())
		tx := portfolio.NewUpdatePrice(date, security.Ticker(), price)
		txs = append(txs, tx)
	}
	return txs, nil
}
