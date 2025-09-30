package renderer

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/etnz/portfolio"
)

func HoldingMarkdown(s *portfolio.Snapshot) string {
	var b strings.Builder
	on := s.On()

	valDay := on.String()
	if on.IsToday() {
		valDay += " " + time.Now().Format("15:04:05")
	}
	fmt.Fprintf(&b, "# Holding Report on %s\n\n", valDay)

	fmt.Fprintf(&b, "Total Portfolio Value: **%s**\n\n", s.TotalPortfolio().String())

	// --- Securities Section ---
	securitiesSection := Header(func(w io.Writer) {
		fmt.Fprint(w, "## Securities\n\n")
		fmt.Fprintln(w, "| Ticker | Quantity | Price | Market Value |")
		fmt.Fprintln(w, "|:---|---:|---:|---:|")
	})

	for ticker := range s.Securities() {
		pos := s.Position(ticker)
		if pos.IsZero() {
			continue
		}
		securitiesSection.PrintHeader(&b)
		fmt.Fprintf(&b, "| %s | %s | %s | %s |\n",
			ticker,
			pos.String(),
			s.Price(ticker).String(),
			s.Convert(s.MarketValue(ticker)).String(),
		)
	}
	securitiesSection.PrintFooter(&b)

	// --- Cash Section ---
	cashSection := Header(func(w io.Writer) {
		fmt.Fprint(w, "\n## Cash\n\n")
		fmt.Fprintln(w, "| Currency | Balance | Value |")
		fmt.Fprintln(w, "|:---|---:|---:|")
	})

	for cur := range s.Currencies() {
		bal := s.Cash(cur)
		if bal.IsZero() {
			continue
		}
		cashSection.PrintHeader(&b)
		fmt.Fprintf(&b, "| %s | %s | %s |\n",
			cur,
			bal.String(),
			s.Convert(bal).String(),
		)
	}
	cashSection.PrintFooter(&b)

	// --- Counterparties Section ---
	if !s.TotalCounterparty().IsZero() {
		counterpartiesSection := Header(func(w io.Writer) {
			fmt.Fprint(w, "\n## Counterparties\n\n")
			fmt.Fprintln(w, "| Name | Balance | Value |")
			fmt.Fprintln(w, "|:---|---:|---:|")
		})
		for acc := range s.Counterparties() {
			bal := s.Counterparty(acc)
			if bal.IsZero() {
				continue
			}
			counterpartiesSection.PrintHeader(&b)
			fmt.Fprintf(&b, "| %s | %s | %s |\n",
				acc, bal.SignedString(), s.Convert(bal).SignedString())
		}
	}

	return b.String()
}

// DeclarationMarkdown renders the full declaration of stocks ticker.
func DeclarationMarkdown(s *portfolio.Snapshot) string {
	// use the snaphost to mark the asset as currently held or not
	var b strings.Builder
	fmt.Fprintf(&b, "# Securities\n\n")
	fmt.Fprintln(&b, "| Ticker | Held | Security ID | Currency | Description |")
	fmt.Fprintln(&b, "|:---|:---:|:---|:---|:---|")

	for t := range s.Securities() {
		sec, _ := s.SecurityDetails(t)
		held := " "
		if !s.Position(t).IsZero() {
			held = "X"
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n",
			sec.Ticker(),
			held,
			sec.ID(),
			sec.Currency(),
			sec.Description(),
		)
	}
	return b.String()

}
