## Consolidated Asset Report

| Ledger | Asset | Start Value | End Value | Trading Flow | Market Gain | Realized Gain | Unrealized Gain | Dividends | TWR |
|:---|:---|---:|---:|---:|---:|---:|---:|---:|---:|
{{- range $review := .Reviews }}
{{- range .Assets }}
{{- if not .IsZero }}
| {{ $review.Name }} | {{ .Ticker }} | {{ .StartValue }} | {{ .EndValue }} | {{ .TradingFlow.SignedString }} | {{ .MarketGain.SignedString }} | {{ .RealizedGain.SignedString }} | {{ .UnrealizedGain.SignedString }} | {{ .Dividends.SignedString }} | {{ .TWR.SignedString }} |
{{- end }}
{{- end }}
| **Sub-total {{ $review.Name }}** | | **{{ $review.TotalStartMarketValue }}** | **{{ $review.TotalEndMarketValue }}** | **{{ $review.TotalNetTradingFlow.SignedString }}** | **{{ $review.MarketGains.SignedString }}** | **{{ $review.TotalRealizedGains.SignedString }}** | **{{ $review.TotalUnrealizedGains.SignedString }}** | **{{ $review.Dividends.SignedString }}** | **{{ $review.TotalTWR.SignedString }}** |
{{- end }}
| **Consolidated Total** | | **{{ .ConsolidatedTotalStartMarketValue }}** | **{{ .ConsolidatedTotalEndMarketValue }}** | **{{ .ConsolidatedTotalNetTradingFlow.SignedString }}** | **{{ .ConsolidatedMarketGains.SignedString }}** | **{{ .ConsolidatedTotalRealizedGains.SignedString }}** | **{{ .ConsolidatedTotalUnrealizedGains.SignedString }}** | **{{ .ConsolidatedDividends.SignedString }}** | |