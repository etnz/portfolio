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

	valDay := "Value at Day's Close"
	genDate := portfolio.NewDate(r.Time.Year(), r.Time.Month(), r.Time.Day())
	if r.Date == genDate {
		valDay = fmt.Sprintf("Value at %s", r.Time.Format("15:04:05"))
	}
	doc.Table(md.TableSet{
		Alignment: []md.TableAlignment{
			md.AlignLeft,
			md.AlignRight,
		},
		Header: []string{
			md.Bold(valDay),
			md.Bold(r.ValueAtClose.String()),
		},
		Rows: [][]string{
			{"Value at Prev. Close", r.ValueAtPrevClose.String()},
		},
	})

	if r.HasBreakdown() {
		doc.H2("Breakdown of Change")
		table := md.TableSet{
			Alignment: []md.TableAlignment{md.AlignLeft,
				md.AlignRight,
			},
			Header: []string{
				md.Bold("Total Day's Gain"),
				md.Bold(r.TotalGain.SignedString()),
				r.PercentageGain().SignedString()},
		}
		if !r.MarketGains.IsZero() {
			table.Rows = append(table.Rows, []string{
				"Unrealized Market",
				r.MarketGains.SignedString(),
				"",
			})
		}
		if !r.RealizedGains.IsZero() {
			table.Rows = append(table.Rows, []string{
				"Realized Market",
				r.RealizedGains.SignedString(),
				"",
			})
		}
		if !r.NetCashFlow.IsZero() {
			table.Rows = append(table.Rows, []string{
				"Net Cash Flow",
				r.NetCashFlow.SignedString(),
				"",
			})
		}
		doc.Table(table)
	}

	if len(r.ActiveAssets) > 0 {
		doc.H2("Active Assets")
		table := md.TableSet{
			Alignment: []md.TableAlignment{md.AlignLeft,
				md.AlignRight,
				md.AlignRight,
			},
			Header: []string{
				"Ticker",
				"Gain / Loss",
				"Change",
			},
		}
		for _, asset := range r.ActiveAssets {
			if !asset.Gain.IsZero() {
				table.Rows = append(table.Rows, []string{
					asset.Security,
					asset.Gain.String(),
					asset.Return.SignedString(),
				})
			}
		}
		doc.Table(table)
	}

	if len(r.Transactions) > 0 {
		doc.H2("Today's Transactions")
		var transactions []string
		for _, tx := range r.Transactions {
			transactions = append(transactions, Transaction(tx))
		}
		doc.OrderedList(transactions...)
	}

	return doc.String()
}
