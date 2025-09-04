package renderer

import (
	"bytes"
	"fmt"

	"github.com/etnz/portfolio"
	md "github.com/nao1215/markdown"
)

func SummaryMarkdown(s *portfolio.Summary) string {
	var buf bytes.Buffer
	doc := md.NewMarkdown(&buf)

	doc.H1(fmt.Sprintf("Portfolio Summary on %s", s.Date))
	doc.PlainText(fmt.Sprintf("Total Market Value: %.2f %s", s.TotalMarketValue, s.ReportingCurrency))

	doc.H2("Performance")

	_, week := s.Date.ISOWeek()
	quarter := (s.Date.Month()-1)/3 + 1

	dayLabel := fmt.Sprintf("Day %d", s.Date.Day())
	weekLabel := fmt.Sprintf("Week %d", week)
	monthLabel := fmt.Sprintf("%s", s.Date.Month())
	quarterLabel := fmt.Sprintf("Q%d", quarter)
	yearLabel := fmt.Sprintf("%d", s.Date.Year())

	formatPerf := func(p portfolio.Performance) string {
		return fmt.Sprintf("%+.2f%%", p.Return*100)
	}

	table := md.TableSet{
		Header: []string{"Period", "Return"},
		Rows: [][]string{
			{dayLabel, formatPerf(s.Daily)},
			{weekLabel, formatPerf(s.WTD)},
			{monthLabel, formatPerf(s.MTD)},
			{quarterLabel, formatPerf(s.QTD)},
			{yearLabel, formatPerf(s.YTD)},
			{"Inception", formatPerf(s.Inception)},
		},
	}
	doc.Table(table)

	return doc.String()
}
