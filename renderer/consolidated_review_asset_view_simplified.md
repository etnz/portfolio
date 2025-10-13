## Consolidated Asset Performance

| Ledger | Asset | Value | Gain | TWR |
|:---|:---|---:|---:|---:|
{{- range $review := .Reviews }}
{{- range .Assets }}
{{- if not .MarketGain.IsZero }}
| {{ $review.Name }} | {{ .Ticker }} | {{ .EndValue }} | {{ .MarketGain.SignedString }} | {{ .TWR.SignedString }} |
{{- end }}
{{- end }}
| **Sub-total {{ $review.Name }}** | | **{{ $review.TotalEndMarketValue }}** | **{{ $review.MarketGains.SignedString }}** | **{{ $review.TotalTWR.SignedString }}** |
{{- end }}
| **Consolidated Total** | | **{{ .ConsolidatedTotalEndMarketValue }}** | **{{ .ConsolidatedMarketGains.SignedString }}** | |