package eodhd

import (
	"fmt"
	"net/url"

	"github.com/etnz/portfolio"
)

// SearchResult matches the structure of a single item in the EODHD search API response.
type SearchResult struct {
	Code              string         `json:"Code"`
	Exchange          string         `json:"Exchange"`
	Name              string         `json:"Name"`
	Type              string         `json:"Type"`
	Country           string         `json:"Country"`
	Currency          string         `json:"Currency"`
	ISIN              string         `json:"ISIN"`
	PreviousClose     float64        `json:"previousClose"`
	PreviousCloseDate portfolio.Date `json:"previousCloseDate"`
	MIC               string         `json:"-"` // Populated by Search, not from API directly.
}

// Search searches for securities via EOD Historical Data API.
func Search(apiKey string, searchTerm string) ([]SearchResult, error) {
	apiURL := fmt.Sprintf("https://eodhd.com/api/search/%s?api_token=%s&fmt=json", url.PathEscape(searchTerm), url.QueryEscape(apiKey))

	var results []SearchResult
	if err := jwget(newDailyCachingClient(), apiURL, &results); err != nil {
		return nil, err
	}
	// Search results reference an exchange code that could match multiple MIC (only for the US apparently).
	mic2Exchange, err := fetchMicToExchangeCode(apiKey)
	if err != nil {
		return nil, err
	}
	// Reverse the map.
	exchange2mic := make(map[string][]string)
	for k, v := range mic2Exchange {
		exchange2mic[v] = append(exchange2mic[v], k)
	}

	// Now we fully rebuild the search result list with potentially different MIC
	newResults := make([]SearchResult, 0, len(results))
	for _, result := range results {
		for _, mic := range exchange2mic[result.Exchange] {
			r := result
			r.MIC = mic
			newResults = append(newResults, r)
		}
	}
	return newResults, nil
}
