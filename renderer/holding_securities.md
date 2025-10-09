{{- if .Securities }}

## Securities

| Ticker | Quantity | Price | Market Value |
|:---|---:|---:|---:|
{{- range .Securities }}
| {{ .Ticker }} | {{ .Quantity }} | {{ .Price }} | {{ .MarketValue }} |
{{- end }}
| **Total** | | | **{{ .TotalSecuritiesValue }}** |
{{- end -}}