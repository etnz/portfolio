{{- if .Transactions }}

## Transactions

{{ range .Transactions -}}
* {{ .When }}: {{ .Detail }}
{{ end }}
{{- end }}