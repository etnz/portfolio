{{- if .Holdings }}

## Securities

| Ledger | Ticker | Quantity | Price | Market Value | Last Update |
|:---|:---|---:|---:|---:|:---|
{{- range $holding := .Holdings }}
{{- range .Securities }}
| {{ $holding.Name }} | {{ .Ticker }} | {{ .Quantity }} | {{ .Price }} | {{ .MarketValue }} | {{ if not .LastUpdate.IsZero }}{{ .LastUpdate.Format "2006-01-02" }}{{ end }} |
{{- end }}
{{- if .Securities }}| **Sub-total {{ $holding.Name }}** | | | | **{{ $holding.TotalSecuritiesValue }}** | |{{- end }}
{{- end }}
| **Consolidated Total** | | | | **{{ .ConsolidatedSecuritiesValue }}** | |
{{- end -}}