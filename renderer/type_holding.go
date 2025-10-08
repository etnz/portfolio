package renderer

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

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

// holdingMarkdownTemplate is the template for rendering a Holding report in Markdown.
const holdingMarkdownTemplate = `# Holding Report on {{ .Date.DayString }}

Total {{ if .Name }}{{ .Name }} {{ end }}Portfolio Value: **{{ .TotalPortfolioValue }}**

{{- if .Securities }}

## Securities

| Ticker | Quantity | Price | Market Value |
|:---|---:|---:|---:|
{{- range .Securities }}
| {{ .Ticker }} | {{ .Quantity }} | {{ .Price }} | {{ .MarketValue }} |
{{- end }}
| **Total** | | | **{{ .TotalSecuritiesValue }}** |
{{- end -}}

{{- if .Cash }}

## Cash

| Currency | Balance |
|:---|---:|
{{- range .Cash }}
| {{ .Currency }} | {{ .Balance }} |
{{- end }}
{{- end -}}

{{- if .Counterparties }}

## Counterparties

| Name | Balance |
|:---|---:|
{{- range .Counterparties }}
| {{ .Name }} | {{ .Balance.SignedString }} |
{{- end }}
| **Total** | **{{ .TotalCounterpartiesValue.SignedString }}** |
{{- end -}}
`

// ToMarkdown renders the Holding struct to a markdown string using a text/template.
func RenderHolding(h *Holding) string {
	tmpl := template.Must(template.New("holding").Parse(holdingMarkdownTemplate))
	var b strings.Builder
	if err := tmpl.Execute(&b, h); err != nil {
		// In a real application, you might want to handle this error more gracefully.
		return fmt.Sprintf("Error executing template: %v", err)
	}
	return b.String()
}

// ConsolidatedHolding represents a consolidated view of multiple holdings.
type ConsolidatedHolding struct {
	Date                            portfolio.Date  `json:"date"`
	ConsolidatedPortfolioValue      portfolio.Money `json:"consolidatedPortfolioValue"`
	ConsolidatedSecuritiesValue     portfolio.Money `json:"consolidatedSecuritiesValue"`
	ConsolidatedCashValue           portfolio.Money `json:"consolidatedCashValue"`
	ConsolidatedCounterpartiesValue portfolio.Money `json:"consolidatedCounterpartiesValue"`
	Holdings                        []*Holding      `json:"holdings"`
	ReportingCurrency               string          `json:"reportingCurrency"`
}

// NewConsolidatedHolding creates a new ConsolidatedHolding from a list of snapshots.
// It assumes the reporting currency of the first snapshot for consolidation.
func NewConsolidatedHolding(snapshots []*portfolio.Snapshot) *ConsolidatedHolding {
	if len(snapshots) == 0 {
		return &ConsolidatedHolding{}
	}

	// Use the first snapshot to determine the date and reporting currency for the whole consolidation.
	firstSnap := snapshots[0]
	on := firstSnap.On()
	reportingCurrency := firstSnap.ReportingCurrency()

	ch := &ConsolidatedHolding{
		Date:                            on,
		Holdings:                        make([]*Holding, 0, len(snapshots)),
		ReportingCurrency:               reportingCurrency,
		ConsolidatedPortfolioValue:      portfolio.M(0, reportingCurrency),
		ConsolidatedSecuritiesValue:     portfolio.M(0, reportingCurrency),
		ConsolidatedCashValue:           portfolio.M(0, reportingCurrency),
		ConsolidatedCounterpartiesValue: portfolio.M(0, reportingCurrency),
	}

	for _, s := range snapshots {
		h := NewHolding(s)
		ch.Holdings = append(ch.Holdings, h)

		// Convert and aggregate totals to the consolidated reporting currency.
		ch.ConsolidatedPortfolioValue = ch.ConsolidatedPortfolioValue.Add(firstSnap.Convert(h.TotalPortfolioValue))
		ch.ConsolidatedCashValue = ch.ConsolidatedCashValue.Add(firstSnap.Convert(h.TotalCashValue))
		ch.ConsolidatedSecuritiesValue = ch.ConsolidatedSecuritiesValue.Add(firstSnap.Convert(h.TotalSecuritiesValue))
		ch.ConsolidatedCounterpartiesValue = ch.ConsolidatedCounterpartiesValue.Add(firstSnap.Convert(h.TotalCounterpartiesValue))
	}

	return ch
}

const consolidatedHoldingMarkdownTemplate = `# Consolidated Holding Report on {{ .Date.DayString }}

Consolidated Portfolio Value: **{{ .ConsolidatedPortfolioValue }}**

## Securities

| Ledger | Ticker | Quantity | Price | Market Value |
|:---|:---|---:|---:|---:|
{{- range $holding := .Holdings }}
{{- if .Securities }}
{{- range .Securities }}
| {{ $holding.Name }} | {{ .Ticker }} | {{ .Quantity }} | {{ .Price }} | {{ .MarketValue }} |
{{- end }}
| **Sub-total {{ $holding.Name }}** | | | | **{{ $holding.TotalSecuritiesValue }}** |
{{- end }}
{{- end }}
| **Consolidated Total** | | | | **{{ .ConsolidatedSecuritiesValue }}** |

## Cash

| Ledger | Currency | Balance |
|:---|:---|---:|
{{- range $holding := .Holdings }}
{{- if .Cash }}
{{- range .Cash }}
| {{ $holding.Name }} | {{ .Currency }} | {{ .Balance }} |
{{- end }}
| **Sub-total {{ $holding.Name }}** | | **{{ .TotalCashValue }}** |
{{- end }}
{{- end }}
| **Consolidated Total** | | **{{ .ConsolidatedCashValue }}** |

## Counterparties

| Ledger | Name | Balance |
|:---|:---|---:|
{{- range $holding := .Holdings }}
{{- if .Counterparties }}
{{- range .Counterparties }}
| {{ $holding.Name }} | {{ .Name }} | {{ .Balance.SignedString }} |
{{- end }}
| **Sub-total {{ $holding.Name }}** | | **{{ $holding.TotalCounterpartiesValue.SignedString }}** |
{{- end }}
{{- end }}
| **Consolidated Total** | | **{{ .ConsolidatedCounterpartiesValue.SignedString }}** |
`

// RenderConsolidatedHolding renders the ConsolidatedHolding struct to a markdown string.
func RenderConsolidatedHolding(ch *ConsolidatedHolding) string {
	// Using a buffer to capture output of a nested template execution
	var renderedHoldings bytes.Buffer
	tmpl := template.Must(template.New("consolidatedHolding").Parse(consolidatedHoldingMarkdownTemplate))

	if err := tmpl.Execute(&renderedHoldings, ch); err != nil {
		return fmt.Sprintf("Error executing template: %v", err)
	}

	return renderedHoldings.String()
}
