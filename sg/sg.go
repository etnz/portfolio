package sg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

const listURL = "https://www.societegeneralegestion.fr/product-services/tip/shares/v2/categories/public/search.json?filters%5B818%5D%5B0%5D=Action&filters%5B879%5D%5B0%5D=true&limit=2000&page=1&sort%5B0%5D%5Bfield%5D=label&sort%5B0%5D%5Border%5D=ASC&perimeter=501&searchText=&offset=0&grouping=false&category=public&partnerId=699&locale=fr-FR&id=282"

type Fund struct {
	ISIN      string `json:"isin"`
	Label     string `json:"label"`
	ShareCode string `json:"shareCode"`
}

type listResponse struct {
	Data struct {
		Response struct {
			Docs []Fund `json:"docs"`
		} `json:"response"`
	} `json:"data"`
}

// FetchFunds returns a map of ISIN to Fund details.
// It uses a daily caching client.
func FetchFunds() (map[string]Fund, error) {
	var result listResponse
	if err := jwget(newDailyCachingClient(), listURL, &result); err != nil {
		return nil, fmt.Errorf("failed to fetch SG funds list: %w", err)
	}

	funds := make(map[string]Fund)
	for _, f := range result.Data.Response.Docs {
		if f.ISIN != "" {
			funds[f.ISIN] = f
		}
	}
	return funds, nil
}

type NavPoint struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

type historyItem struct {
	NavHistory []NavPoint `json:"navHistory"`
}

// FetchHistory fetches historical NAV data for a given ISIN.
func FetchHistory(isin string) ([]NavPoint, error) {
	url := fmt.Sprintf("https://www.societegeneralegestion.fr/product-services/fdr/share/v3/full/%s", isin)
	body := []byte(`{"fields": ["navHistory"]}`)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json")

	client := newDailyCachingClient()
	resp, err := client.Do(req)
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

	return items[0].NavHistory, nil
}
