// Package cli provides terminal I/O, output formatting, and meta-command
// dispatch for the QuestCore game engine.
package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/nathoo/questcore/engine"
	"github.com/nathoo/questcore/engine/save"
	"github.com/nathoo/questcore/engine/state"
	"github.com/nathoo/questcore/types"
)

// CLI handles terminal interaction with the player.
type CLI struct {
	Engine    *engine.Engine
	Defs      *state.Defs
	In        io.Reader
	Out       io.Writer
	SaveDir   string
	Trace     bool
	EchoInput bool   // echo each input line after the prompt (for script playback)
	lastCmd   string // for "again"/"g" repeat
}

// New creates a CLI wired to the given engine.
func New(eng *engine.Engine, defs *state.Defs) *CLI {
	home, _ := os.UserHomeDir()
	saveDir := filepath.Join(home, ".questcore", "saves")
	return &CLI{
		Engine:  eng,
		Defs:    defs,
		In:      os.Stdin,
		Out:     os.Stdout,
		SaveDir: saveDir,
	}
}

// Run starts the game loop. It shows the intro, describes the starting room,
// then loops: prompt → input → dispatch → output.
func (c *CLI) Run() {
	// Show intro.
	if c.Defs.Game.Intro != "" {
		c.printLine(c.Defs.Game.Intro)
		c.printLine("")
	}

	// Describe starting room.
	result := c.Engine.Step("look")
	c.printResult(result)

	scanner := bufio.NewScanner(c.In)
	for {
		c.print("> ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}
		// Skip comment lines (for script files).
		if strings.HasPrefix(input, "#") {
			continue
		}
		if c.EchoInput {
			c.printLine(input)
		}

		// Meta-commands start with '/'.
		if strings.HasPrefix(input, "/") {
			if c.handleMeta(input) {
				return // /quit
			}
			continue
		}

		// "again" / "g" repeats the last game command.
		lower := strings.ToLower(input)
		if lower == "again" || lower == "g" {
			if c.lastCmd == "" {
				c.printLine("Nothing to repeat.")
				continue
			}
			input = c.lastCmd
		} else {
			c.lastCmd = input
		}

		result := c.Engine.Step(input)
		c.printResult(result)

		if c.Trace {
			c.printTrace(result)
		}
	}
}

// handleMeta dispatches meta-commands. Returns true if the game should exit.
func (c *CLI) handleMeta(input string) bool {
	parts := strings.Fields(input)
	cmd := parts[0]
	var arg string
	if len(parts) > 1 {
		arg = parts[1]
	}

	switch cmd {
	case "/quit", "/exit":
		c.printSystem("Goodbye.")
		return true

	case "/save":
		c.cmdSave(arg)

	case "/load":
		c.cmdLoad(arg)

	case "/help":
		c.cmdHelp()

	case "/state":
		c.cmdState()

	case "/trace":
		c.Trace = !c.Trace
		if c.Trace {
			c.printSystem("Trace output enabled.")
		} else {
			c.printSystem("Trace output disabled.")
		}

	default:
		c.printSystem(fmt.Sprintf("Unknown command: %s. Type /help for available commands.", cmd))
	}

	return false
}

func (c *CLI) cmdSave(name string) {
	if name == "" {
		name = "quicksave"
	}

	data, err := save.Save(c.Engine.State, c.Defs)
	if err != nil {
		c.printSystem(fmt.Sprintf("Save failed: %v", err))
		return
	}

	if err := os.MkdirAll(c.SaveDir, 0o755); err != nil {
		c.printSystem(fmt.Sprintf("Save failed: %v", err))
		return
	}

	path := filepath.Join(c.SaveDir, name+".json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		c.printSystem(fmt.Sprintf("Save failed: %v", err))
		return
	}

	c.printSystem(fmt.Sprintf("Game saved to %s.", name))
}

func (c *CLI) cmdLoad(name string) {
	if name == "" {
		name = "quicksave"
	}

	path := filepath.Join(c.SaveDir, name+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		c.printSystem(fmt.Sprintf("Load failed: %v", err))
		return
	}

	sd, err := save.Load(data)
	if err != nil {
		c.printSystem(fmt.Sprintf("Load failed: %v", err))
		return
	}

	save.ApplySave(c.Engine.State, sd)
	c.printSystem(fmt.Sprintf("Game loaded from %s (turn %d).", name, sd.Turn))

	// Show current room after loading.
	result := c.Engine.Step("look")
	c.printResult(result)
}

func (c *CLI) cmdHelp() {
	help := []string{
		"System:",
		"  /save [name]  — Save game (default: quicksave)",
		"  /load [name]  — Load game (default: quicksave)",
		"  /quit         — Exit game",
		"  /help         — Show this help",
		"  /state        — Debug: dump current state",
		"  /trace        — Toggle debug trace output",
		"",
		"Game commands:",
		"  look (l)              — Describe the room",
		"  examine <thing> (x)   — Look closely at something",
		"  go/walk <dir>         — Move (or just type n/s/e/w/u/d)",
		"  take/get <item>       — Pick something up",
		"  drop <item>           — Put something down",
		"  use <item> on <thing> — Use an item on something",
		"  open / close          — Open or close something",
		"  talk/speak <npc>      — Talk to someone",
		"  ask <npc> about <topic>",
		"  give <item> to <npc>  — Give an item to someone",
		"  inventory (i)         — Check what you're carrying",
		"  wait (z)              — Let time pass",
		"  again (g)             — Repeat your last command",
	}
	for _, line := range help {
		c.printLine(line)
	}
}

func (c *CLI) cmdState() {
	s := c.Engine.State
	c.printSystem(fmt.Sprintf("Turn: %d", s.TurnCount))
	c.printSystem(fmt.Sprintf("Location: %s", s.Player.Location))
	c.printSystem(fmt.Sprintf("Inventory: %v", s.Player.Inventory))
	if len(s.Flags) > 0 {
		c.printSystem(fmt.Sprintf("Flags: %v", s.Flags))
	}
	if len(s.Counters) > 0 {
		c.printSystem(fmt.Sprintf("Counters: %v", s.Counters))
	}
}

func (c *CLI) printTrace(result types.Result) {
	if len(result.Effects) > 0 {
		c.printSystem(fmt.Sprintf("[trace] Effects: %d", len(result.Effects)))
		for _, e := range result.Effects {
			c.printSystem(fmt.Sprintf("[trace]   %s %v", e.Type, e.Params))
		}
	}
	if len(result.Events) > 0 {
		c.printSystem(fmt.Sprintf("[trace] Events: %d", len(result.Events)))
		for _, e := range result.Events {
			c.printSystem(fmt.Sprintf("[trace]   %s", e.Type))
		}
	}
}

func (c *CLI) printResult(result types.Result) {
	for _, line := range result.Output {
		c.printLine(line)
	}
}

func (c *CLI) printLine(text string) {
	fmt.Fprintln(c.Out, text)
}

func (c *CLI) print(text string) {
	fmt.Fprint(c.Out, text)
}

func (c *CLI) printSystem(text string) {
	fmt.Fprintf(c.Out, "[%s]\n", text)
}
