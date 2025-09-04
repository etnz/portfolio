package renderer

import (
	"bytes"
	"fmt"

	"github.com/etnz/portfolio"
	md "github.com/nao1215/markdown"
)

func GainsMarkdown(r *portfolio.GainsReport) string {
	var buf bytes.Buffer
	doc := md.NewMarkdown(&buf)

	doc.H1("Capital Gains Report from " + r.Range.From.String() + " to " + r.Range.To.String())
	doc.PlainText(fmt.Sprintf("Method: %s", r.Method))

	doc.H2("Gains per Security")

	table := md.TableSet{
		Alignment: []md.TableAlignment{
			md.AlignLeft,
			md.AlignRight,
			md.AlignRight,
			md.AlignRight,
		},
		Header: []string{
			"Security",
			"Realized",
			"Unrealized",
			"Total",
		},
	}

	for _, s := range r.Securities {
		table.Rows = append(table.Rows, []string{
			s.Security,
			s.Realized.SignedString(),
			s.Unrealized.SignedString(),
			s.Total.SignedString(),
		})
	}
	table.Rows = append(table.Rows, []string{
		md.Bold("Total"),
		md.Bold(r.Realized.SignedString()),
		md.Bold(r.Unrealized.SignedString()),
		md.Bold(r.Total.SignedString()),
	})
	doc.Table(table)

	return doc.String()
}
