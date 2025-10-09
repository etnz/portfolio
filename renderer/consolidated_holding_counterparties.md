{{- if .Holdings }}

## Counterparties

| Ledger | Name | Balance |
|:---|:---|---:|
{{- range $holding := .Holdings }}
{{- range .Counterparties }}
| {{ $holding.Name }} | {{ .Name }} | {{ .Balance.SignedString }} |
{{- end }}
{{- if .Counterparties }}| **Sub-total {{ $holding.Name }}** | | **{{ $holding.TotalCounterpartiesValue.SignedString }}** |{{- end }}
{{- end }}
| **Consolidated Total** | | **{{ .ConsolidatedCounterpartiesValue.SignedString }}** |
{{- end -}}