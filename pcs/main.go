package main

import (
	"context"
	"flag"
	"os"
	"path"

	"github.com/etnz/porfolio/cmd"
	"github.com/google/subcommands"
)

func main() {
	commander := subcommands.NewCommander(flag.CommandLine, path.Base(os.Args[0]))

	for _, c := range cmd.Commands {
		commander.Register(c, "")
	}

	flag.Parse()
	os.Exit(int(commander.Execute(context.Background())))
}
