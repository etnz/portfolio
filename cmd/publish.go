package cmd

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"text/template"

	"github.com/etnz/portfolio"
	"github.com/etnz/portfolio/renderer"
	"github.com/google/subcommands"
)

type reportTask struct {
	Period portfolio.Range
	Report string
}

type publishCmd struct {
	outputDir      string
	frontMatterTpl string
}

func (*publishCmd) Name() string { return "publish" }

func (*publishCmd) Synopsis() string { return "generates all historical reports for the portfolio" }

func (*publishCmd) Usage() string {
	return `publish [-o <dir>] [-frontmatter <file>]

  Generates all historical reports (review, holding, etc.) for all periods
  (daily, weekly, monthly, quarterly, and yearly) and saves them to a
  structured directory tree.
`
}

func (c *publishCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.outputDir, "o", "reports", "Root directory for the generated reports")
	f.StringVar(&c.frontMatterTpl, "frontmatter", "", "Path to a Go template file for the report front matter")
}

func (c *publishCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	var frontMatterTpl *template.Template
	if c.frontMatterTpl != "" {
		var err error
		frontMatterTpl, err = template.ParseFiles(c.frontMatterTpl)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to parse front matter template: %v\n", err)
			return subcommands.ExitFailure
		}
	}

	if err := os.MkdirAll(c.outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create output directory: %v\n", err)
		return subcommands.ExitFailure
	}

	as, err := DecodeAccountingSystem()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create accounting system: %v\n", err)
		return subcommands.ExitFailure
	}

	startDate := as.Ledger.OldestTransactionDate()
	if startDate.IsZero() {
		fmt.Println("Ledger is empty, nothing to publish.")
		return subcommands.ExitSuccess
	}
	endDate := portfolio.Today().Add(-1)

	periods, err := generatePeriods(startDate, endDate)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to generate periods: %v\n", err)
		return subcommands.ExitFailure
	}
	if len(periods) == 0 {
		fmt.Println("No periods to publish.")
		return subcommands.ExitSuccess
	}

	// Prepare tasks to compute and generate each report

	tasks := make([]reportTask, 0)

	// Fill the task with review and holding reports
	for _, period := range periods {
		tasks = append(tasks, reportTask{Period: period, Report: "review"})
		tasks = append(tasks, reportTask{Period: period, Report: "holding"})
	}

	// Run the tasks
	for _, task := range tasks {
		var md string

		switch task.Report {
		case "review":
			reviewReport, err := as.NewReviewReport(task.Period)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to generate review report for %s: %v\n", task.Period, err)
				continue
			}
			md = renderer.ReviewMarkdown(reviewReport)
		case "holding":
			holdingReport, err := as.NewHoldingReport(task.Period.To)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to generate holding report for %s: %v\n", task.Period.To, err)
				continue
			}
			md = renderer.HoldingMarkdown(holdingReport)
		}

		// Generate frontmatter if template is provided
		if frontMatterTpl != nil {
			fm, err := renderFrontMatter(frontMatterTpl, task)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to render front matter for %s report %s: %v\n", task.Report, task.Period.Identifier(), err)
				continue
			}
			md = fm + "\n" + md // Prepend front matter to markdown
		}

		filePath := path.Join(task.Report, task.Period.Name(), task.Period.Identifier()+".md")
		fullPath := filepath.Join(c.outputDir, filePath)

		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			fmt.Fprintf(os.Stderr, "failed to create output directory for file %s: %v\n", filePath, err)
			return subcommands.ExitFailure
		}

		if err := os.WriteFile(fullPath, []byte(md), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "failed to write file %s: %v\n", filePath, err)
			return subcommands.ExitFailure
		}
		log.Printf("Generated %s report for period %s", task.Report, task.Period.Identifier())
	}

	return subcommands.ExitSuccess
}

func generatePeriods(startDate, endDate portfolio.Date) ([]portfolio.Range, error) {
	if startDate.IsZero() {
		// no transactions
		return []portfolio.Range{}, nil
	}

	var ranges []portfolio.Range

	// Daily, Weekly, Monthly, Quarterly, Yearly
	for _, periodType := range []portfolio.Period{portfolio.Daily, portfolio.Weekly, portfolio.Monthly, portfolio.Quarterly, portfolio.Yearly} {
		for r := portfolio.NewRange(startDate, periodType); !r.From.After(endDate); r = portfolio.NewRange(r.To.Add(1), periodType) {
			ranges = append(ranges, r)
		}
	}

	return ranges, nil
}

func renderFrontMatter(tpl *template.Template, task reportTask) (string, error) {
	var fmBuffer bytes.Buffer
	if err := tpl.Execute(&fmBuffer, task); err != nil {
		return "", err
	}
	return fmBuffer.String(), nil
}
