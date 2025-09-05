package renderer

import (
	"bytes"
	"fmt"

	"github.com/etnz/portfolio"
	md "github.com/nao1215/markdown"
)

func HistoryMarkdown(r *portfolio.HistoryReport) string {
	var buf bytes.Buffer
	doc := md.NewMarkdown(&buf)

	if r.Security != "" {
		doc.H1(fmt.Sprintf("History for %s", r.Security))
	} else {
		doc.H1(fmt.Sprintf("History for %s", r.Currency))
	}

	if r.Security != "" {
		table := md.TableSet{
			Alignment: []md.TableAlignment{
				md.AlignLeft,
				md.AlignRight,
				md.AlignRight,
				md.AlignRight,
			},
			Header: []string{"Date", "Position", "Price", "Value"},
			Rows:   [][]string{},
		}
		for _, entry := range r.Entries {
			table.Rows = append(table.Rows, []string{
				entry.Date.String(),
				entry.Position.String(),
				entry.Price.String(),
				entry.Value.String(),
			})
		}
		doc.Table(table)
	} else {
		table := md.TableSet{
			Alignment: []md.TableAlignment{
				md.AlignLeft,
				md.AlignRight,
			},
			Header: []string{"Date", "Value"},
			Rows:   [][]string{},
		}
		for _, entry := range r.Entries {
			table.Rows = append(table.Rows, []string{
				entry.Date.String(),
				entry.Value.String(),
			})
		}
		doc.Table(table)
	}

	return doc.String()
}
