package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/etnz/portfolio/docs"
	"github.com/google/subcommands"
)

type topicCmd struct{}

func (*topicCmd) Name() string     { return "topic" }
func (*topicCmd) Synopsis() string { return "show documentation" }
func (*topicCmd) Usage() string {
	return `topic <topic>

Show documentation for a given topic.
`
}

func (c *topicCmd) SetFlags(f *flag.FlagSet) {}

func (c *topicCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	topics := f.Args()
	if len(topics) == 0 {
		topics = []string{"readme"}
	}

	doc, err := docs.GetTopics(topics...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading doc: %v\n", err)
		return subcommands.ExitFailure
	}
	printMarkdown(doc)

	return subcommands.ExitSuccess
}
