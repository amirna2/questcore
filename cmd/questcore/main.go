// QuestCore is a deterministic, data-driven game engine for text adventures.
// Usage: questcore [--version] [--plain] [--script <file>] [--trace] <game_directory>
package main

import (
	"fmt"
	"os"

	"github.com/nathoo/questcore/cli"
	"github.com/nathoo/questcore/engine"
	"github.com/nathoo/questcore/loader"
	"github.com/nathoo/questcore/tui"
)

// Set via -ldflags at build time.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	plain := false
	trace := false
	var gameDir string
	var scriptFile string

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--version":
			fmt.Printf("questcore %s (commit %s, built %s)\n", version, commit, date)
			return
		case "--plain":
			plain = true
		case "--trace":
			trace = true
		case "--script":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "--script requires a file path\n")
				os.Exit(1)
			}
			i++
			scriptFile = args[i]
		default:
			if gameDir == "" {
				gameDir = args[i]
			}
		}
	}

	if gameDir == "" {
		fmt.Fprintf(os.Stderr, "Usage: questcore [--version] [--plain] [--script <file>] [--trace] <game_directory>\n")
		os.Exit(1)
	}

	// Load and compile Lua game content.
	defs, err := loader.Load(gameDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading game: %v\n", err)
		os.Exit(1)
	}

	eng := engine.New(defs)

	// Script mode: open file, force plain, echo commands.
	if scriptFile != "" {
		f, err := os.Open(scriptFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening script: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		fmt.Printf("%s v%s by %s\n\n", defs.Game.Title, defs.Game.Version, defs.Game.Author)
		c := cli.New(eng, defs)
		c.In = f
		c.EchoInput = true
		c.Trace = trace
		c.Run()
		return
	}

	// Use plain CLI if --plain flag or stdout is not a terminal.
	if plain || !isTerminal() {
		fmt.Printf("%s v%s by %s\n\n", defs.Game.Title, defs.Game.Version, defs.Game.Author)
		c := cli.New(eng, defs)
		c.Trace = trace
		c.Run()
		return
	}

	if err := tui.Run(eng, defs); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// isTerminal returns true if stdout is a terminal (not piped/redirected).
func isTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
