{{- if .Assets -}}
## Asset Performance

| Asset | Value | Gain | TWR |
|:---|---:|---:|---:|
{{- range .Assets }}
{{- if not .MarketGain.IsZero }}
| {{ .Ticker }} | {{ .EndValue }} | {{ .MarketGain.SignedString }} | {{ .TWR.SignedString }} |
{{- end }}
{{- end }}
| **Total** | **{{ .TotalEndMarketValue }}** | **{{ .MarketGains.SignedString }}** | **{{ .TotalTWR.SignedString }}** |
{{- end }}