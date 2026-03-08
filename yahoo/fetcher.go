package yahoo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/etnz/portfolio"
	"github.com/shopspring/decimal"
)

const (
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36"
)

type fetcher struct {
	client *http.Client
	once   sync.Once
	err    error
}

var globalFetcher = &fetcher{}

func (f *fetcher) init(ctx context.Context) error {
	f.once.Do(func() {
		jar, err := cookiejar.New(nil)
		if err != nil {
			f.err = fmt.Errorf("failed to create cookie jar: %w", err)
			return
		}
		f.client = &http.Client{
			Jar:     jar,
			Timeout: 30 * time.Second,
		}

		if err := establishSession(ctx, f.client); err != nil {
			f.err = err
			return
		}
	})
	return f.err
}

// Fetch retrieves historical prices, dividends, and splits for a given ticker from Yahoo Finance.
// It handles session cookies internally.
func Fetch(ctx context.Context, security portfolio.Security, from, to portfolio.Date) ([]portfolio.Transaction, error) {
	return globalFetcher.Fetch(ctx, security, from, to)
}

func (f *fetcher) Fetch(ctx context.Context, security portfolio.Security, from, to portfolio.Date) ([]portfolio.Transaction, error) {
	if err := f.init(ctx); err != nil {
		return nil, err
	}
	u, err := url.Parse(security.Fetcher())
	if err != nil {
		return nil, fmt.Errorf("invalid yahoo URI: %w", err)
	}
	yahooTicker := u.Host
	if yahooTicker == "" {
		yahooTicker = strings.Trim(u.Path, "/")
	}

	// Data Retrieval

	// URL: https://query1.finance.yahoo.com/v8/finance/chart/[TICKER]?period1=[START]&period2=[END]&interval=1d&includeAdjustedClose=true
	apiURL := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?period1=%d&period2=%d&interval=1d&includeAdjustedClose=true&events=div%%7Csplit", yahooTicker, from.Unix(), to.Unix())

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("yahoo api returned status: %s", resp.Status)
	}

	// Response Schema
	var data struct {
		Chart struct {
			Result []struct {
				Meta struct {
					Currency string `json:"currency"`
					Symbol   string `json:"symbol"`
				} `json:"meta"`
				Timestamp  []int64 `json:"timestamp"`
				Indicators struct {
					Quote []struct {
						Close []*float64 `json:"close"`
					} `json:"quote"`
				} `json:"indicators"`
				Events *struct {
					Dividends map[string]struct {
						Amount float64 `json:"amount"`
						Date   int64   `json:"date"`
					} `json:"dividends"`
					Splits map[string]struct {
						Numerator   float64 `json:"numerator"`
						Denominator float64 `json:"denominator"`
						SplitRatio  string  `json:"splitRatio"`
						Date        int64   `json:"date"`
					} `json:"splits"`
				} `json:"events"`
			} `json:"result"`
			Error *struct {
				Code        string `json:"code"`
				Description string `json:"description"`
			} `json:"error"`
		} `json:"chart"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode json response: %w", err)
	}

	if data.Chart.Error != nil {
		return nil, fmt.Errorf("yahoo api error: %s - %s", data.Chart.Error.Code, data.Chart.Error.Description)
	}
	if len(data.Chart.Result) == 0 {
		return nil, fmt.Errorf("yahoo api returned no result")
	}

	result := data.Chart.Result[0]
	if len(result.Timestamp) == 0 {
		return nil, nil // No data
	}

	// Validation: Check if quotes exist
	if len(result.Indicators.Quote) == 0 || len(result.Indicators.Quote[0].Close) != len(result.Timestamp) {
		// Mismatch or empty quotes
		return nil, nil
	}

	currency := security.Currency()
	var txs []portfolio.Transaction
	quotes := result.Indicators.Quote[0].Close

	for i, ts := range result.Timestamp {
		if quotes[i] == nil {
			continue
		}

		t := time.Unix(ts, 0).UTC()
		pDate := portfolio.NewDate(t.Year(), t.Month(), t.Day())

		val := decimal.NewFromFloat(*quotes[i])
		txs = append(txs, portfolio.NewUpdatePrice(pDate, security.Ticker(), portfolio.M(val, currency)))
	}

	if result.Events != nil {
		for _, div := range result.Events.Dividends {
			t := time.Unix(div.Date, 0).UTC()
			pDate := portfolio.NewDate(t.Year(), t.Month(), t.Day())
			val := decimal.NewFromFloat(div.Amount)
			txs = append(txs, portfolio.NewDividend(pDate, "Yahoo Finance", security.Ticker(), portfolio.M(val, currency)))
		}
		for _, split := range result.Events.Splits {
			t := time.Unix(split.Date, 0).UTC()
			pDate := portfolio.NewDate(t.Year(), t.Month(), t.Day())
			txs = append(txs, portfolio.NewSplit(pDate, security.Ticker(), int64(split.Numerator), int64(split.Denominator)))
		}
	}

	// Sort transactions by date
	sort.Slice(txs, func(i, j int) bool {
		return txs[i].When().Before(txs[j].When())
	})

	return txs, nil
}

func establishSession(ctx context.Context, client *http.Client) error {
	// Phase A: Session Initialization (Warm-up)
	// Target URL: https://finance.yahoo.com/quote/[TICKER]
	// We use AAPL as a generic ticker to establish the session.
	consentPage := "https://finance.yahoo.com/quote/AAPL"
	req, err := http.NewRequestWithContext(ctx, "GET", consentPage, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch quote page for cookies: %w", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch quote page for cookies: status %s for url: %s", resp.Status, consentPage)
	}
	return nil
}
