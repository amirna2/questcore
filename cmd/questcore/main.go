// QuestCore is a deterministic, data-driven game engine for text adventures.
// Usage: questcore [--version] [--plain] <game_directory>
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
	var gameDir string

	for _, arg := range os.Args[1:] {
		switch arg {
		case "--version":
			fmt.Printf("questcore %s (commit %s, built %s)\n", version, commit, date)
			return
		case "--plain":
			plain = true
		default:
			if gameDir == "" {
				gameDir = arg
			}
		}
	}

	if gameDir == "" {
		fmt.Fprintf(os.Stderr, "Usage: questcore [--version] [--plain] <game_directory>\n")
		os.Exit(1)
	}

	// Load and compile Lua game content.
	defs, err := loader.Load(gameDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading game: %v\n", err)
		os.Exit(1)
	}

	eng := engine.New(defs)

	// Use plain CLI if --plain flag or stdout is not a terminal.
	if plain || !isTerminal() {
		fmt.Printf("%s v%s by %s\n\n", defs.Game.Title, defs.Game.Version, defs.Game.Author)
		c := cli.New(eng, defs)
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
