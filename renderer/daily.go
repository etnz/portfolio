package renderer

import (
	"bytes"
	"fmt"

	"github.com/etnz/portfolio"
	md "github.com/nao1215/markdown"
)

func DailyMarkdown(r *portfolio.DailyReport) string {
	var buf bytes.Buffer
	doc := md.NewMarkdown(&buf)

	doc.H1("Daily Report")

	table := md.TableSet{
		Alignment: []md.TableAlignment{md.AlignLeft, md.AlignRight},
	}
	// TODO: calculation in the report
	percentageGain := 0.0
	if r.ValueAtPrevClose != 0 {
		percentageGain = (r.TotalGain / r.ValueAtPrevClose) * 100
	}
	// TODO: in the report we shall use business types (like money for the Value, that has a decent String() and many different formats)

	valDay := "Value at Day's Close"
	if r.Date.IsToday() {
		valDay = fmt.Sprintf("Value at %s", r.Time.Format("15:04:05"))
	}

	table.Header = []string{md.Bold(valDay), md.Bold(fmt.Sprintf("%.2f", r.ValueAtClose))}
	table.Rows = append(table.Rows, []string{"Value at Prev. Close", fmt.Sprintf("%.2f", r.ValueAtPrevClose)})

	doc.Table(table)

	// TODO: calculation in the report struct
	nonZeroCount := 0
	if r.MarketGains != 0 {
		nonZeroCount++
	}
	if r.RealizedGains != 0 {
		nonZeroCount++
	}
	if r.NetCashFlow != 0 {
		nonZeroCount++
	}
	if nonZeroCount > 0 {
		doc.H2("Breakdown of Change")
		table := md.TableSet{
			Header:    []string{("Total Day's Gain"), fmt.Sprintf("%+.2f (%+.2f%%)", r.TotalGain, percentageGain)},
			Alignment: []md.TableAlignment{md.AlignLeft, md.AlignRight},
		}
		if r.MarketGains != 0 {
			table.Rows = append(table.Rows, []string{"Unrealized Market", fmt.Sprintf("%+.2f", r.MarketGains)})
		}
		if r.RealizedGains != 0 {
			table.Rows = append(table.Rows, []string{"Realized Market", fmt.Sprintf("%+.2f", r.RealizedGains)})
		}
		if r.NetCashFlow != 0 {
			table.Rows = append(table.Rows, []string{"Net Cash Flow", fmt.Sprintf("%+.2f", r.NetCashFlow)})
		}

		doc.Table(table)
	}

	if len(r.ActiveAssets) > 0 {
		doc.H2("Active Assets")
		table := md.TableSet{
			Header:    []string{"Ticker", "Gain / Loss", "Change"},
			Alignment: []md.TableAlignment{md.AlignLeft, md.AlignRight, md.AlignRight},
		}
		for _, asset := range r.ActiveAssets {
			if asset.Gain != 0 {
				table.Rows = append(table.Rows, []string{asset.Security, fmt.Sprintf("%.2f", asset.Gain), fmt.Sprintf("%+.2f%%", asset.Return*100)})
			}
		}
		doc.Table(table)
	}

	if len(r.Transactions) > 0 {
		doc.H2("Today's Transactions")
		var transactions []string
		for _, tx := range r.Transactions {
			// TODO: native transactions needs a proper renderer
			transactions = append(transactions, string(tx.What()))
		}
		doc.OrderedList(transactions...)
	}

	return doc.String()
}
