{{- if .Assets }}

## Consolidated Asset Report

| Asset | Start Value | End Value | Trading Flow | Market Gain | Realized Gain | Unrealized Gain | Dividends | TWR |
|:---|---:|---:|---:|---:|---:|---:|---:|---:|
{{- range .Assets }}
{{- if not .IsZero }}
| {{ .Ticker }} | {{ .StartValue }} | {{ .EndValue }} | {{ .TradingFlow.SignedString }} | {{ .MarketGain.SignedString }} | {{ .RealizedGain.SignedString }} | {{ .UnrealizedGain.SignedString }} | {{ .Dividends.SignedString }} | {{ .TWR.SignedString }} |
{{- end }}
{{- end }}
| **Total** | **{{ .TotalStartMarketValue }}** | **{{ .TotalEndMarketValue }}** | **{{ .TotalNetTradingFlow.SignedString }}** | **{{ .MarketGains.SignedString }}** | **{{ .TotalRealizedGains.SignedString }}** | **{{ .TotalUnrealizedGains.SignedString }}** | **{{ .Dividends.SignedString }}** | **{{ .TotalTWR.SignedString }}** |
{{- end }}