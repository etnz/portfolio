// Package main provides the entry point for the `pcs` command-line tool.
// It initializes the subcommand system, registers all available commands,
// and handles the execution of both built-in and external extension commands.
package main

import (
	"context"
	"flag"
	"io"
	"log"
	"os"
	"path"

	"github.com/etnz/portfolio/cmd"
	"github.com/google/subcommands"
)

// main is the entry point of the `pcs` application. It sets up the command
// line parser, registers all subcommands, and executes the requested command.
// It also handles the execution of external commands if a matching built-in
// command is not found.
func main() {
	commander := subcommands.NewCommander(flag.CommandLine, path.Base(os.Args[0]))

	commander.Register(commander.HelpCommand(), "")
	commander.Register(commander.FlagsCommand(), "")
	commander.Register(commander.CommandsCommand(), "")

	cmd.Register(commander)

	flag.Parse()

	if !*cmd.Verbose {
		log.SetOutput(io.Discard)
	}

	// Check if a subcommand is provided
	if flag.NArg() > 0 {
		subcommand := flag.Arg(0)
		isBuiltIn := false

		// Iterate through registered built-in commands to check for a match
		commander.VisitCommands(func(g *subcommands.CommandGroup, c subcommands.Command) {
			if c.Name() == subcommand {
				isBuiltIn = true
			}
		})

		// If it's not a built-in command, attempt to run as an extension
		if !isBuiltIn {
			extensionExecuted, exitCode := cmd.RunExtension(subcommand, os.Args[1:])
			if extensionExecuted {
				os.Exit(exitCode)
			}
		}
	}

	// If no extension was executed (either not found, or it was a built-in command),
	// proceed with built-in commands execution.
	os.Exit(int(commander.Execute(context.Background())))
}
