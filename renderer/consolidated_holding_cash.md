{{- if .Holdings }}

## Cash

| Ledger | Currency | Balance |
|:---|:---|---:|
{{- range $holding := .Holdings }}
{{- range .Cash }}
| {{ $holding.Name }} | {{ .Currency }} | {{ .Balance }} |
{{- end }}
{{- if .Cash }}| **Sub-total {{ $holding.Name }}** | | **{{ .TotalCashValue }}** |{{- end }}
{{- end }}
| **Consolidated Total** | | **{{ .ConsolidatedCashValue }}** |
{{- end -}}