## Transactions
{{- range .Reviews }}
### {{ .Name }}
{{- if .Transactions }}
{{ range .Transactions -}}
* {{ .When }}: {{ .Detail }}
{{ end }}
{{- else }}
*No transactions in this period.*
{{- end }}
{{ end }}