# Consolidated Holding Report on {{ .Date.DayString }}

| Ledger | Portfolio Value |
|:---|---:|
{{- range .Holdings }}
| {{ .Name }} | {{ .TotalPortfolioValue }} |
{{- end }}
| **Total** | **{{ .ConsolidatedPortfolioValue }}** |