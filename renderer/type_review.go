package renderer

import (
	"embed"
	"os"
	"strings"
	"time"

	"github.com/etnz/portfolio"
)

//go:embed *.md
var templates embed.FS

// Now is the current time used in reports.
// it has to be a global variable so that tests can override it.
func Now() time.Time {
	if os.Getenv("PORTFOLIO_TESTING_NOW") != "" {
		t, err := time.Parse("2006-01-02 15:04:05", os.Getenv("PORTFOLIO_TESTING_NOW"))
		if err != nil {
			panic(err)
		}
		return t
	}
	return time.Now()
}

// Review is a struct to represent the review data for rendering.
type Review struct {
	Name                     string          `json:"name,omitempty"`
	AsOf                     string          `json:"asOf"`
	Range                    portfolio.Range `json:"range"`
	TotalPortfolioValue      portfolio.Money `json:"totalPortfolioValue"`
	TotalCashValue           portfolio.Money `json:"totalCashValue"`
	TotalCounterpartiesValue portfolio.Money `json:"totalCounterpartiesValue"`
	PreviousValue            portfolio.Money `json:"previousValue"`
	CapitalFlow              portfolio.Money `json:"capitalFlow"`
	MarketGains              portfolio.Money `json:"marketGains"`
	ForexGains               portfolio.Money `json:"forexGains"`
	NetChange                portfolio.Money `json:"netChange"`
	CashChange               portfolio.Money `json:"cashChange"`
	CounterpartiesChange     portfolio.Money `json:"counterpartiesChange"`
	MarketValueChange        portfolio.Money `json:"marketValueChange"`
	Dividends                portfolio.Money `json:"dividends"`
	TotalGains               portfolio.Money `json:"totalGains"`
	// Totals for the asset report
	TotalStartMarketValue portfolio.Money   `json:"totalStartMarketValue"`
	TotalEndMarketValue   portfolio.Money   `json:"totalEndMarketValue"`
	TotalNetTradingFlow   portfolio.Money   `json:"totalNetTradingFlow"`
	TotalRealizedGains    portfolio.Money   `json:"totalRealizedGains"`
	TotalUnrealizedGains  portfolio.Money   `json:"totalUnrealizedGains"`
	TotalTWR              portfolio.Percent `json:"totalTwr"`

	Accounts     Accounts      `json:"accounts"`
	Assets       []AssetReview `json:"assets"`
	Transactions []RenderableTransaction
}

// RenderableTransaction holds the data for a single transaction line in a report.
type RenderableTransaction struct {
	When   string
	Detail string
}

// Accounts holds the cash and counterparty account details for a review.
type Accounts struct {
	Cash         []CashAccount         `json:"cash"`
	Counterparty []CounterpartyAccount `json:"counterparty"`
}

// CashAccount represents a single cash account's state in a review.
type CashAccount struct {
	Currency    string            `json:"currency"`
	Value       portfolio.Money   `json:"value"`
	ForexReturn portfolio.Percent `json:"forexReturn"`
}

// CounterpartyAccount represents a single counterparty account's state in a review.
type CounterpartyAccount struct {
	Name  string          `json:"name"`
	Value portfolio.Money `json:"value"`
}

// AssetReview holds all the period metrics for a single asset.
type AssetReview struct {
	Ticker         string            `json:"ticker"`
	StartValue     portfolio.Money   `json:"startValue"`
	EndValue       portfolio.Money   `json:"endValue"`
	TradingFlow    portfolio.Money   `json:"tradingFlow"`
	MarketGain     portfolio.Money   `json:"marketGain"`
	RealizedGain   portfolio.Money   `json:"realizedGain"`
	UnrealizedGain portfolio.Money   `json:"unrealizedGain"`
	Dividends      portfolio.Money   `json:"dividends"`
	TWR            portfolio.Percent `json:"twr"`
}

// IsZero checks if all financial values in the AssetReview are zero.
func (ar AssetReview) IsZero() bool {
	return AllAreZero(ar.StartValue, ar.EndValue, ar.TradingFlow, ar.MarketGain, ar.RealizedGain, ar.UnrealizedGain, ar.Dividends)
}

// NewReview creates a new renderer.Review from a portfolio.Review.
func NewReview(pr *portfolio.Review, method portfolio.CostBasisMethod) *Review {
	start, end := pr.Start(), pr.End()
	forexGain := pr.PortfolioChange().Sub(pr.CashFlow()).Sub(pr.MarketGain())

	r := &Review{
		AsOf:                     Now().Format("2006-01-02 15:04:05"),
		Name:                     pr.Name(),
		Range:                    pr.Range(),
		TotalPortfolioValue:      end.TotalPortfolio(),
		TotalCashValue:           end.TotalCash(),
		TotalCounterpartiesValue: end.TotalCounterparty(),
		PreviousValue:            start.TotalPortfolio(),
		CapitalFlow:              pr.CashFlow(),
		MarketGains:              pr.MarketGain(),
		ForexGains:               forexGain,
		NetChange:                pr.PortfolioChange(),
		CashChange:               pr.CashChange(),
		CounterpartiesChange:     pr.CounterpartyChange(),
		MarketValueChange:        pr.TotalMarketChange(),
		Dividends:                pr.Dividends(),
		TotalGains:               pr.MarketGain().Add(forexGain).Add(pr.Dividends()),

		TotalStartMarketValue: pr.Start().TotalMarket(),
		TotalEndMarketValue:   pr.End().TotalMarket(),
		TotalNetTradingFlow:   pr.NetTradingFlow(),
		TotalRealizedGains:    pr.RealizedGains(method),
		TotalUnrealizedGains:  pr.End().TotalUnrealizedGains(method),
		TotalTWR:              pr.TimeWeightedReturn(),
	}

	// Populate Accounts
	for cur := range end.Currencies() {
		r.Accounts.Cash = append(r.Accounts.Cash, CashAccount{
			Currency:    cur,
			Value:       end.Cash(cur),
			ForexReturn: pr.CurrencyTimeWeightedReturn(cur),
		})
	}
	for acc := range end.Counterparties() {
		if AllAreZero(end.Counterparty(acc), start.Counterparty(acc)) {
			continue
		}
		r.Accounts.Counterparty = append(r.Accounts.Counterparty, CounterpartyAccount{
			Name:  acc,
			Value: end.Counterparty(acc),
		})
	}

	// Populate Assets
	for ticker := range end.Securities() {
		r.Assets = append(r.Assets, AssetReview{
			Ticker:         ticker,
			StartValue:     start.MarketValue(ticker),
			EndValue:       end.MarketValue(ticker),
			TradingFlow:    pr.AssetNetTradingFlow(ticker),
			MarketGain:     pr.AssetMarketGain(ticker),
			RealizedGain:   pr.AssetRealizedGains(ticker, method),
			UnrealizedGain: end.UnrealizedGains(ticker, method),
			Dividends:      pr.AssetDividends(ticker),
			TWR:            pr.AssetTimeWeightedReturn(ticker),
		})
	}

	// Populate Transactions
	txs := pr.Transactions()
	r.Transactions = make([]RenderableTransaction, len(txs))
	var prevDate portfolio.Date
	for i, tx := range txs {
		dateStr := tx.When().String()
		if !prevDate.IsZero() && prevDate == tx.When() {
			// Use non-breaking spaces to maintain alignment
			dateStr = strings.Repeat("\u00A0", len(dateStr))
		}
		r.Transactions[i] = RenderableTransaction{When: dateStr, Detail: Transaction(tx)}
		prevDate = tx.When()
	}

	return r
}

// --- Template Definitions ---

// TODO: template should be clearly matched to a single type in this package, and unit test should be provided for each.

const (
	// reviewMarkdownTemplate is the main layout template. It calls partials for each section.
	reviewMarkdownTemplate = `
{{- template "review_title" . -}}
{{- template "review_summary" . -}}
{{- template "review_accounts" . -}}
{{- template "asset_view" . -}}
{{- template "review_transactions" . -}}
`
)
