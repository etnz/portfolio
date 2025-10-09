## Accounts

|  **Cash Accounts** | Value | Forex % |
|---:|---:|---:|
{{- range .Accounts.Cash }}
| {{ .Currency }} | {{ .Value }} | {{ .ForexReturn.SignedString }} |
{{- end }}
| **Total** | **{{ .TotalCashValue }}** | |

|  **Counterparty Accounts** | Value |
|---:|---:|
{{- range .Accounts.Counterparty }}
| {{ .Name }} | {{ .Value.SignedString }} |
{{- end }}
| **Total** | **{{ .TotalCounterpartiesValue.SignedString }}** |