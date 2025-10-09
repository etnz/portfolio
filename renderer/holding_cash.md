{{- if .Cash }}

## Cash

| Currency | Balance |
|:---|---:|
{{- range .Cash }}
| {{ .Currency }} | {{ .Balance }} |
{{- end }}
| **Total** | **{{ .TotalCashValue }}** |
{{- end -}}