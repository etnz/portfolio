## Portfolio Summary

| Ledger | Portfolio Value | Previous Value | Capital Flow | Market Gains | Forex Gains | Net Change |
|:---|---:|---:|---:|---:|---:|---:|
{{- range .Reviews }}
| {{ .Name }} | {{ .TotalPortfolioValue }} | {{ .PreviousValue }} | {{ .CapitalFlow.SignedString }} | {{ .MarketGains.SignedString }} | {{ .ForexGains.SignedString }} | {{ .NetChange }} |
{{- end }}
| **Total** | **{{ .ConsolidatedTotalPortfolioValue }}** | **{{ .ConsolidatedPreviousValue }}** | **{{ .ConsolidatedCapitalFlow.SignedString }}** | **{{ .ConsolidatedMarketGains.SignedString }}** | **{{ .ConsolidatedForexGains.SignedString }}** | **{{ .ConsolidatedNetChange }}** |