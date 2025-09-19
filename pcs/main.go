// Package main provides the entry point for the `pcs` command-line tool.
// It initializes the subcommand system, registers all available commands,
// and handles the execution of both built-in and external extension commands.
package main

import (
	"context"
	"flag"
	"io"
	"log"
	"maps"
	"os"
	"path"
	"slices"

	"github.com/etnz/portfolio/cmd"
	"github.com/google/subcommands"
	"github.com/posener/complete/v2"
	"github.com/posener/complete/v2/predict"
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

	complete.Complete("pcs", NewCommanderCompleter(commander))

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

func NewCommanderCompleter(cmd *subcommands.Commander) complete.Completer {
	sub := &completer{
		subcommands: make(map[string]complete.Completer),
		flags:       make(map[string]complete.Predictor),
		args:        predict.Nothing,
	}
	cmd.VisitCommands(func(g *subcommands.CommandGroup, c subcommands.Command) {
		sub.subcommands[c.Name()] = NewCommandCompleter(c)
	})
	cmd.VisitAll(func(f *flag.Flag) {
		sub.flags[f.Name] = NewFlagPredictor(f)
	})

	return sub
}

func NewCommandCompleter(cmd subcommands.Command) complete.Completer {
	sub := &completer{
		subcommands: make(map[string]complete.Completer),
		flags:       make(map[string]complete.Predictor),
		args:        predict.Nothing,
	}

	fs := flag.NewFlagSet(cmd.Name(), flag.ContinueOnError)
	cmd.SetFlags(fs)
	fs.VisitAll(func(f *flag.Flag) {
		sub.flags[f.Name] = NewFlagPredictor(f)
	})
	return sub
}

func NewFlagPredictor(f *flag.Flag) complete.Predictor {
	if p, ok := f.Value.(complete.Predictor); ok {
		return p
	}
	return predict.Nothing
}

type completer struct {
	subcommands map[string]complete.Completer
	flags       map[string]complete.Predictor
	args        complete.Predictor
}

// SubCmdList should return the list of all sub commands of the current command.
// We don't use it because complete either chose subcommands OR flags.
func (s *completer) SubCmdList() []string { return nil }

// return slices.Collect(maps.Keys(s.subcommands))
// }

// SubCmdGet should return a sub command of the current command for the given sub command name.
func (s *completer) SubCmdGet(cmd string) complete.Completer { return s.subcommands[cmd] }

// FlagList should return a list of all the flag names of the current command. The flag names
// should not have the dash prefix.
func (s *completer) FlagList() []string { return slices.Collect(maps.Keys(s.flags)) }

// FlagGet should return completion options for a given flag. It is invoked with the flag name
// without the dash prefix. The flag is not promised to be in the command flags. In that case,
// this method should return a nil predictor.
func (s *completer) FlagGet(flag string) complete.Predictor { return s.flags[flag] }

// ArgsGet should return predictor for positional arguments of the command line.
func (s *completer) ArgsGet() complete.Predictor {
	if len(s.subcommands) > 0 {
		return predict.Set(slices.Collect(maps.Keys(s.subcommands)))
	}
	if s.args != nil {
		return s.args
	}
	return predict.Nothing
}
