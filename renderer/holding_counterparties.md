{{- if .Counterparties }}

## Counterparties

| Name | Balance |
|:---|---:|
{{- range .Counterparties }}
| {{ .Name }} | {{ .Balance.SignedString }} |
{{- end }}
| **Total** | **{{ .TotalCounterpartiesValue.SignedString }}** |
{{- end -}}