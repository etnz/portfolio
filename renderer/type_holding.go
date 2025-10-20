package renderer

import (
	"github.com/etnz/portfolio"
)

// Holding is a struct to represent the holding data in json.
// Numbers are handled using the exact decimal types (Money, Quantity, etc.)
// So that they already contain basics renderers (SignedString etc.)
type Holding struct {

	// Name of the ledger.
	Name string `json:"name,omitempty"`
	// Date of the holding
	Date portfolio.Date `json:"date"`
	// TotalPortfolioValue is the total value of the portfolio in the reporting currency.
	TotalPortfolioValue portfolio.Money `json:"totalPortfolioValue"`
	// TotalSecuritiesValue is the total value of all securities in the reporting currency.
	TotalSecuritiesValue portfolio.Money `json:"totalSecuritiesValue"`
	// TotalCashValue is the total value of all cash balances in the reporting currency.
	TotalCashValue portfolio.Money `json:"totalCashValue"`
	// TotalCounterpartiesValue is the total value of all counterparty accounts in the reporting currency.
	TotalCounterpartiesValue portfolio.Money `json:"totalCounterpartiesValue"`
	// Securities is a list of all securities held.
	Securities []HoldingSecurity `json:"securities"`
	// Cash is a list of all cash balances by currency.
	Cash []HoldingCash `json:"cash"`
	// Counterparties is a list of all counterparties balances.
	Counterparties []HoldingCounterparty `json:"counterparties"`
}

// HoldingSecurity represents a single security holding.
type HoldingSecurity struct {
	Ticker      string             `json:"ticker"`
	Quantity    portfolio.Quantity `json:"quantity"`
	Price       portfolio.Money    `json:"price"`
	MarketValue portfolio.Money    `json:"marketValue"`
	ID          portfolio.ID       `json:"id"`
	LastUpdate  portfolio.Date     `json:"lastUpdate"`
	Description string             `json:"description,omitempty"`
}

// HoldingCash represents a single cash balance.
type HoldingCash struct {
	Currency string          `json:"currency"`
	Balance  portfolio.Money `json:"balance"`
}

// HoldingCounterparty represents a single counterparty balance.
type HoldingCounterparty struct {
	Name    string          `json:"name"`
	Balance portfolio.Money `json:"balance"`
}

// NewHolding creates a new Holding struct from a portfolio snapshot.
// It populates the struct with all the necessary data for rendering a holding report.
func NewHolding(s *portfolio.Snapshot) *Holding {
	h := &Holding{
		Name:                     s.Name(),
		Date:                     s.On(),
		TotalPortfolioValue:      s.TotalPortfolio(),
		TotalSecuritiesValue:     s.TotalMarket(),
		TotalCashValue:           s.TotalCash(),
		TotalCounterpartiesValue: s.TotalCounterparty(),
		Securities:               make([]HoldingSecurity, 0),
		Cash:                     make([]HoldingCash, 0),
		Counterparties:           make([]HoldingCounterparty, 0),
	}

	// Populate Securities
	for ticker := range s.Securities() {
		pos := s.Position(ticker)
		if pos.IsZero() {
			continue
		}
		sec, _ := s.SecurityDetails(ticker)
		h.Securities = append(h.Securities, HoldingSecurity{
			Ticker:      ticker,
			Quantity:    pos,
			Price:       s.Price(ticker),
			MarketValue: s.MarketValue(ticker),
			ID:          sec.ID(),
			LastUpdate:  s.LastMarketDataDate(ticker),
			Description: sec.Description(),
		})
	}

	// Populate Cash
	for cur := range s.Currencies() {
		bal := s.Cash(cur)
		if bal.IsZero() {
			continue
		}
		h.Cash = append(h.Cash, HoldingCash{
			Currency: cur,
			Balance:  bal,
		})
	}

	// Populate Counterparties
	for acc := range s.Counterparties() {
		bal := s.Counterparty(acc)
		if bal.IsZero() {
			continue
		}
		h.Counterparties = append(h.Counterparties, HoldingCounterparty{
			Name:    acc,
			Balance: bal,
		})
	}

	return h
}
