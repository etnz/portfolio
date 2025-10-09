{{- if .Holdings }}

## Securities

| Ledger | Ticker | Quantity | Price | Market Value |
|:---|:---|---:|---:|---:|
{{- range $holding := .Holdings }}
{{- range .Securities }}
| {{ $holding.Name }} | {{ .Ticker }} | {{ .Quantity }} | {{ .Price }} | {{ .MarketValue }} |
{{- end }}
{{- if .Securities }}| **Sub-total {{ $holding.Name }}** | | | | **{{ $holding.TotalSecuritiesValue }}** |{{- end }}
{{- end }}
| **Consolidated Total** | | | | **{{ .ConsolidatedSecuritiesValue }}** |
{{- end -}}