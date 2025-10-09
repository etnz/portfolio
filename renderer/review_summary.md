| **Total Portfolio Value** | **{{ .TotalPortfolioValue }}** |
|---:|---:|
| Previous Value | {{ .PreviousValue }} |
| | |
|   Capital Flow | {{ .CapitalFlow.SignedString }} |
| + Market Gains | {{ .MarketGains.SignedString }} |
| + Forex Gains | {{ .ForexGains.SignedString }} |
| **= Net Change** | **{{ .NetChange }}** |
{{- if or (not .CashChange.IsZero) (not .CounterpartiesChange.IsZero) (not .MarketValueChange.IsZero) }}
| | |
| Cash Change | {{ .CashChange.SignedString }} |
| + Counterparties Change | {{ .CounterpartiesChange.SignedString }} |
| + Market Value Change | {{ .MarketValueChange.SignedString }} |
| **= Net Change** | **{{ .NetChange }}** |
{{- end }}
| | |
|   Dividends | {{ .Dividends.SignedString }} |
| + Market Gains | {{ .MarketGains.SignedString }} |
| + Forex Gains | {{ .ForexGains.SignedString }} |
| **=Total Gains** | **{{ .TotalGains.SignedString }}** |