## Accounts

### Cash Accounts
| Ledger | Currency | Value | Forex % |
|:---|:---|---:|---:|
{{- range $review := .Reviews }}
{{- range .Accounts.Cash }}
| {{ $review.Name }} | {{ .Currency }} | {{ .Value }} | {{ .ForexReturn.SignedString }} |
{{- end }}
| **Sub-total {{ $review.Name }}** | | **{{ $review.TotalCashValue }}** | |
{{- end }}
| **Consolidated Total** | | **{{ .ConsolidatedTotalCashValue }}** | |

### Counterparty Accounts
| Ledger | Name | Value |
|:---|:---|---:|
{{- range $review := .Reviews }}
{{- range .Accounts.Counterparty }}
| {{ $review.Name }} | {{ .Name }} | {{ .Value.SignedString }} |
{{- end }}
| **Sub-total {{ $review.Name }}** | | **{{ $review.TotalCounterpartiesValue.SignedString }}** |
{{- end }}
| **Consolidated Total** | | **{{ .ConsolidatedTotalCounterpartiesValue.SignedString }}** |