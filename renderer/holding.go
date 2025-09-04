package renderer

import (
	"bytes"

	"github.com/etnz/portfolio"
	md "github.com/nao1215/markdown"
)

func HoldingMarkdown(r *portfolio.HoldingReport) string {
	var buf bytes.Buffer
	doc := md.NewMarkdown(&buf)

	doc.H1("Holding Report on " + r.Date.String())

	doc.PlainText("Total Portfolio Value: " + md.Bold(r.TotalValue.String()))

	if len(r.Securities) > 0 {
		doc.H2("Securities")
		table := md.TableSet{
			Alignment: []md.TableAlignment{
				md.AlignLeft,
				md.AlignRight,
				md.AlignRight,
				md.AlignRight,
			},
			Header: []string{
				"Ticker",
				"Quantity",
				"Price",
				"Market Value",
			},
		}
		for _, h := range r.Securities {
			table.Rows = append(table.Rows, []string{
				h.Ticker,
				h.Quantity.String(),
				h.Price.String(),
				h.MarketValue.String(),
			})
		}
		doc.Table(table)
	}

	if len(r.Cash) > 0 {
		doc.H2("Cash")
		table := md.TableSet{
			Alignment: []md.TableAlignment{
				md.AlignLeft,
				md.AlignRight,
				md.AlignRight,
			},
			Header: []string{
				"Currency",
				"Balance",
				"Value",
			},
		}
		for _, c := range r.Cash {
			table.Rows = append(table.Rows, []string{
				c.Currency,
				c.Balance.String(),
				c.Value.String(),
			})
		}
		doc.Table(table)
	}

	if len(r.Counterparties) > 0 {
		doc.H2("Counterparties")
		table := md.TableSet{
			Alignment: []md.TableAlignment{
				md.AlignLeft,
				md.AlignRight,
				md.AlignRight,
			},
			Header: []string{
				"Name",
				"Balance",
				"Value",
			},
		}
		for _, c := range r.Counterparties {
			table.Rows = append(table.Rows, []string{
				c.Name,
				c.Balance.SignedString(),
				c.Value.SignedString(),
			})
		}
		doc.Table(table)
	}

	return doc.String()
}
