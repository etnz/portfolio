package renderer

import (
	"fmt"
	"strings"

	"github.com/etnz/portfolio"
)

// SummaryData holds all the calculated information needed by the summary renderer.
type SummaryData struct {
	Date             portfolio.Date
	TotalMarketValue portfolio.Money
	Daily            portfolio.Percent
	WTD              portfolio.Percent // Week-to-Date
	MTD              portfolio.Percent // Month-to-Date
	QTD              portfolio.Percent // Quarter-to-Date
	YTD              portfolio.Percent // Year-to-Date
	Inception        portfolio.Percent
}

func SummaryMarkdown(s *SummaryData) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# Portfolio Summary on %s\n\n", s.Date)
	fmt.Fprintf(&b, "Total Market Value: %s\n\n", s.TotalMarketValue.String())

	fmt.Fprintln(&b, "## Performance\n")
	fmt.Fprintln(&b, "| Period | Return |")
	fmt.Fprintln(&b, "|:---|---:|")
	fmt.Fprintf(&b, "| Day %d | %s |\n", s.Date.Day(), s.Daily.SignedString())
	_, week := s.Date.ISOWeek()
	fmt.Fprintf(&b, "| Week %d | %s |\n", week, s.WTD.SignedString())
	fmt.Fprintf(&b, "| %s | %s |\n", s.Date.Month().String(), s.MTD.SignedString())
	quarter := (s.Date.Month()-1)/3 + 1
	fmt.Fprintf(&b, "| Q%d | %s |\n", quarter, s.QTD.SignedString())
	fmt.Fprintf(&b, "| %d | %s |\n", s.Date.Year(), s.YTD.SignedString())
	fmt.Fprintf(&b, "| Inception | %s |\n", s.Inception.SignedString())

	return b.String()
}
