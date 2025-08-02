package main

import (
	"context"
	"flag"
	"os"
	"path"

	"github.com/etnz/portfolio/cmd"
	"github.com/google/subcommands"
)

func main() {
	commander := subcommands.NewCommander(flag.CommandLine, path.Base(os.Args[0]))

	commander.Register(commander.HelpCommand(), "")
	commander.Register(commander.FlagsCommand(), "")
	commander.Register(commander.CommandsCommand(), "")

	cmd.Register(commander)

	flag.Parse()
	os.Exit(int(commander.Execute(context.Background())))
}
