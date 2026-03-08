package yahoo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// SearchResult represents a single item found by Yahoo Finance search.
type SearchResult struct {
	Symbol    string `json:"symbol"`
	ShortName string `json:"shortname"`
	LongName  string `json:"longname"`
	Exchange  string `json:"exchange"`
	Type      string `json:"quoteType"`
	ExchDisp  string `json:"exchDisp"`
}

type searchResponse struct {
	Quotes []SearchResult `json:"quotes"`
}

// Search searches for securities via Yahoo Finance API.
func Search(query string) ([]SearchResult, error) {
	apiURL := fmt.Sprintf("https://query1.finance.yahoo.com/v1/finance/search?q=%s", url.QueryEscape(query))

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("yahoo search failed with status: %s", resp.Status)
	}

	var response searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode yahoo search response: %w", err)
	}

	return response.Quotes, nil
}
