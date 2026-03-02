package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/nathoo/questcore/engine"
	"github.com/nathoo/questcore/engine/save"
	"github.com/nathoo/questcore/engine/state"
	"github.com/nathoo/questcore/types"
)

// rawLine stores an unstyled output line with its classification,
// so we can re-wrap and re-style when the terminal is resized.
type rawLine struct {
	text     string
	kind     lineKind
	isInput  bool // true for echoed player input
	isSystem bool // true for system messages
}

// Model is the Bubble Tea model for the QuestCore TUI.
type Model struct {
	engine *engine.Engine
	defs   *state.Defs

	viewport viewport.Model
	input    textinput.Model
	history  *History

	rawLines []rawLine // accumulated narrative lines (unstyled, for re-wrapping)

	width    int
	height   int
	ready    bool
	trace    bool
	quitting bool
	lastCmd  string
	saveDir  string
}

// gameOutputMsg carries output from the engine into the Update loop.
type gameOutputMsg struct {
	input    string   // echoed player input (empty for intro)
	lines    []string // output lines
	isSystem bool     // true for meta-command output
}

// New creates a TUI model wired to the given engine.
func New(eng *engine.Engine, defs *state.Defs) Model {
	ti := textinput.New()
	ti.Prompt = "> "
	ti.Focus()
	ti.CharLimit = 256
	ti.PromptStyle = styleInputPrompt

	home, _ := os.UserHomeDir()
	return Model{
		engine:  eng,
		defs:    defs,
		input:   ti,
		history: NewHistory(100),
		saveDir: filepath.Join(home, ".questcore", "saves"),
	}
}

// Run starts the Bubble Tea program.
func Run(eng *engine.Engine, defs *state.Defs) error {
	m := New(eng, defs)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}

// Init returns the initial command that produces intro text and first look.
func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.initialOutput())
}

func (m Model) initialOutput() tea.Cmd {
	return func() tea.Msg {
		var lines []string

		lines = append(lines, m.defs.Game.Title+" v"+m.defs.Game.Version+" by "+m.defs.Game.Author)
		lines = append(lines, "")

		if m.defs.Game.Intro != "" {
			lines = append(lines, m.defs.Game.Intro)
			lines = append(lines, "")
		}

		result := m.engine.Step("look")
		lines = append(lines, result.Output...)

		return gameOutputMsg{lines: lines}
	}
}

// Update handles messages (key presses, window resize, game output).
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		vpHeight := m.height - 2 // 1 status bar + 1 input line
		if vpHeight < 1 {
			vpHeight = 1
		}

		if !m.ready {
			m.viewport = viewport.New(m.width, vpHeight)
			m.viewport.KeyMap = viewportKeyMap()
			m.ready = true
		} else {
			m.viewport.Width = m.width
			m.viewport.Height = vpHeight
		}

		m.refreshViewport()

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			return m.handleEnter()

		case "up":
			if prev, ok := m.history.Prev(); ok {
				m.input.SetValue(prev)
				m.input.CursorEnd()
			}
			return m, nil

		case "down":
			if next, ok := m.history.Next(); ok {
				m.input.SetValue(next)
				m.input.CursorEnd()
			} else {
				m.input.SetValue("")
				m.history.ResetCursor()
			}
			return m, nil

		case "pgup", "pgdown":
			var vpCmd tea.Cmd
			m.viewport, vpCmd = m.viewport.Update(msg)
			return m, vpCmd
		}

	case gameOutputMsg:
		m = m.appendOutput(msg)
	}

	var inputCmd tea.Cmd
	m.input, inputCmd = m.input.Update(msg)
	cmds = append(cmds, inputCmd)

	return m, tea.Batch(cmds...)
}

// handleEnter processes the submitted input line.
func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	input := strings.TrimSpace(m.input.Value())
	m.input.SetValue("")

	if input == "" {
		return m, nil
	}

	m.history.Push(input)
	m.history.ResetCursor()

	// Handle "again" / "g".
	lower := strings.ToLower(input)
	if lower == "again" || lower == "g" {
		if m.lastCmd == "" {
			m = m.appendOutput(gameOutputMsg{
				input: input, lines: []string{"Nothing to repeat."}, isSystem: true,
			})
			return m, nil
		}
		input = m.lastCmd
	} else {
		m.lastCmd = input
	}

	// Meta-commands.
	if strings.HasPrefix(input, "/") {
		output, quit := m.handleMeta(input)
		m = m.appendOutput(gameOutputMsg{input: input, lines: output, isSystem: true})
		if quit {
			m.quitting = true
			return m, tea.Quit
		}
		m.updatePrompt()
		return m, nil
	}

	// Game over input blocking.
	if state.GetFlag(m.engine.State, "game_over") {
		m = m.appendOutput(gameOutputMsg{
			input: input,
			lines: []string{"Game over. Use /load to restore a save or /quit to exit."},
		})
		m.updatePrompt()
		return m, nil
	}

	// Snapshot combat state before step (combat may end during step).
	wasCombat := state.InCombat(m.engine.State)
	var preCombatEnemyID string
	if wasCombat {
		preCombatEnemyID = m.engine.State.Combat.EnemyID
	}

	// Game command.
	result := m.engine.Step(input)
	output := result.Output

	// Combat display injection.
	if state.InCombat(m.engine.State) {
		if box := m.renderCombatStatus(); box != "" {
			output = append(output, box)
		}
	} else if wasCombat && !state.GetFlag(m.engine.State, "game_over") {
		// Combat just ended with victory — show final result.
		output = append(output, m.renderVictory(preCombatEnemyID))
	}
	if state.GetFlag(m.engine.State, "game_over") {
		if wasCombat {
			output = append(output, m.renderDefeat(preCombatEnemyID))
		}
		output = append(output, m.renderGameOver(preCombatEnemyID))
	}

	if m.trace {
		output = append(output, m.formatTrace(result)...)
	}
	m = m.appendOutput(gameOutputMsg{input: input, lines: output})
	m.updatePrompt()
	return m, nil
}

// appendOutput adds lines to the narrative and refreshes the viewport.
func (m Model) appendOutput(msg gameOutputMsg) Model {
	if msg.input != "" {
		m.rawLines = append(m.rawLines, rawLine{
			text: "> " + msg.input, isInput: true,
		})
	}

	inCombat := state.InCombat(m.engine.State) || state.GetFlag(m.engine.State, "game_over")
	for _, line := range msg.lines {
		rl := rawLine{text: line, isSystem: msg.isSystem}
		if !msg.isSystem {
			// Detect pre-styled lines (lipgloss bordered boxes contain ANSI escapes).
			if strings.Contains(line, "\x1b[") {
				rl.kind = kindPreStyled
			} else {
				rl.kind = classifyLine(line)
				// In combat/game-over context, color plain narrative lines as combat.
				if inCombat && rl.kind == kindRoomDesc {
					rl.kind = kindCombat
				}
			}
		}
		m.rawLines = append(m.rawLines, rl)
	}

	// Blank line separator between turns.
	m.rawLines = append(m.rawLines, rawLine{})

	m.refreshViewport()

	return m
}

// refreshViewport re-wraps and re-styles all raw lines at the current width
// and updates the viewport content.
func (m *Model) refreshViewport() {
	if !m.ready {
		return
	}

	width := m.width
	if width < 10 {
		width = 10
	}

	var styled []string
	for _, rl := range m.rawLines {
		if rl.text == "" {
			styled = append(styled, "")
			continue
		}

		// Pre-styled lines (lipgloss boxes) skip word-wrap and re-styling.
		if rl.kind == kindPreStyled {
			styled = append(styled, rl.text)
			continue
		}

		wrapped := wordWrap(rl.text, width)

		switch {
		case rl.isInput:
			styled = append(styled, stylePlayerInput.Render(wrapped))
		case rl.isSystem:
			styled = append(styled, styledSystemMsg(wrapped))
		default:
			styled = append(styled, renderLineKind(wrapped, rl.kind))
		}
	}

	m.viewport.SetContent(strings.Join(styled, "\n"))
	m.viewport.GotoBottom()
}

// renderLineKind applies the style for a given lineKind.
func renderLineKind(line string, kind lineKind) string {
	switch kind {
	case kindYouSee:
		return styledYouSee(line)
	case kindExits:
		return styleExits.Render(line)
	case kindDialogue:
		return styleDialogue.Render(line)
	case kindSystem:
		return styleSystem.Render(line)
	case kindError:
		return styleError.Render(line)
	case kindTrace:
		return styleTrace.Render(line)
	case kindCombat:
		return styleCombat.Render(line)
	case kindCombatHeader:
		return styleCombatHeader.Render(line)
	case kindGameOver:
		return styleGameOver.Render(line)
	case kindPreStyled:
		return line
	default:
		return styleRoomDesc.Render(line)
	}
}

// wordWrap wraps text to fit within the given width, breaking at word
// boundaries. Preserves existing newlines within the text.
func wordWrap(text string, width int) string {
	if width <= 0 || len(text) <= width {
		return text
	}

	var result strings.Builder
	words := strings.Fields(text)
	lineLen := 0

	for i, word := range words {
		wLen := len(word)

		if i == 0 {
			result.WriteString(word)
			lineLen = wLen
			continue
		}

		if lineLen+1+wLen > width {
			result.WriteString("\n")
			result.WriteString(word)
			lineLen = wLen
		} else {
			result.WriteString(" ")
			result.WriteString(word)
			lineLen += 1 + wLen
		}
	}

	return result.String()
}

// View renders the full TUI layout: viewport + status bar + input.
func (m Model) View() string {
	if m.quitting {
		return ""
	}
	if !m.ready {
		return "Loading..."
	}

	return m.viewport.View() + "\n" + m.renderStatusBar() + "\n" + m.input.View()
}

// handleMeta dispatches meta-commands. Returns output lines and quit flag.
func (m *Model) handleMeta(input string) ([]string, bool) {
	parts := strings.Fields(input)
	cmd := parts[0]
	var arg string
	if len(parts) > 1 {
		arg = parts[1]
	}

	switch cmd {
	case "/quit", "/exit":
		return []string{"Goodbye."}, true

	case "/save":
		return m.cmdSave(arg), false

	case "/load":
		return m.cmdLoad(arg), false

	case "/help":
		return m.cmdHelp(), false

	case "/state":
		return m.cmdState(), false

	case "/trace":
		m.trace = !m.trace
		if m.trace {
			return []string{"Trace output enabled."}, false
		}
		return []string{"Trace output disabled."}, false

	default:
		return []string{fmt.Sprintf("Unknown command: %s. Type /help for available commands.", cmd)}, false
	}
}

func (m *Model) cmdSave(name string) []string {
	if name == "" {
		name = "quicksave"
	}

	data, err := save.Save(m.engine.State, m.defs)
	if err != nil {
		return []string{fmt.Sprintf("Save failed: %v", err)}
	}

	if err := os.MkdirAll(m.saveDir, 0o755); err != nil {
		return []string{fmt.Sprintf("Save failed: %v", err)}
	}

	path := filepath.Join(m.saveDir, name+".json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return []string{fmt.Sprintf("Save failed: %v", err)}
	}

	return []string{fmt.Sprintf("Game saved to %s.", name)}
}

func (m *Model) cmdLoad(name string) []string {
	if name == "" {
		name = "quicksave"
	}

	path := filepath.Join(m.saveDir, name+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return []string{fmt.Sprintf("Load failed: %v", err)}
	}

	sd, err := save.Load(data)
	if err != nil {
		return []string{fmt.Sprintf("Load failed: %v", err)}
	}

	save.ApplySave(m.engine.State, sd)
	m.engine.RestoreRNG(sd.RNGSeed, sd.RNGPosition)

	output := []string{fmt.Sprintf("Game loaded from %s (turn %d).", name, sd.Turn)}
	result := m.engine.Step("look")
	output = append(output, result.Output...)
	return output
}

func (m *Model) cmdHelp() []string {
	return []string{
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
		"",
		"Combat:",
		"  attack                — Attack the enemy",
		"  defend                — Defend (reduces damage taken)",
		"  flee                  — Attempt to flee combat",
		"",
		"Navigation: PgUp/PgDn to scroll, Up/Down for command history",
	}
}

func (m *Model) cmdState() []string {
	s := m.engine.State
	output := []string{
		fmt.Sprintf("Turn: %d", s.TurnCount),
		fmt.Sprintf("Location: %s", s.Player.Location),
		fmt.Sprintf("Inventory: %v", s.Player.Inventory),
	}
	if len(s.Flags) > 0 {
		output = append(output, fmt.Sprintf("Flags: %v", s.Flags))
	}
	if len(s.Counters) > 0 {
		output = append(output, fmt.Sprintf("Counters: %v", s.Counters))
	}
	if state.InCombat(s) {
		enemyHP, _ := state.GetStat(s, m.defs, s.Combat.EnemyID, "hp")
		enemyMaxHP, _ := state.GetStat(s, m.defs, s.Combat.EnemyID, "max_hp")
		output = append(output, fmt.Sprintf("Combat: enemy=%s round=%d defending=%v enemy_hp=%d/%d",
			s.Combat.EnemyID, s.Combat.RoundCount, s.Combat.Defending, enemyHP, enemyMaxHP))
	}
	return output
}

func (m *Model) formatTrace(result types.Result) []string {
	var lines []string
	if len(result.Effects) > 0 {
		lines = append(lines, fmt.Sprintf("[trace] Effects: %d", len(result.Effects)))
		for _, e := range result.Effects {
			lines = append(lines, fmt.Sprintf("[trace]   %s %v", e.Type, e.Params))
		}
	}
	if len(result.Events) > 0 {
		lines = append(lines, fmt.Sprintf("[trace] Events: %d", len(result.Events)))
		for _, e := range result.Events {
			lines = append(lines, fmt.Sprintf("[trace]   %s", e.Type))
		}
	}
	return lines
}

// updatePrompt sets the input prompt based on game state.
func (m *Model) updatePrompt() {
	switch {
	case state.GetFlag(m.engine.State, "game_over"):
		m.input.Prompt = "/load or /quit> "
		m.input.PromptStyle = styleGameOverPrompt
	case state.InCombat(m.engine.State):
		m.input.Prompt = "What do you do? (attack, defend, use <item>, flee) "
		m.input.PromptStyle = styleCombatPrompt
	default:
		m.input.Prompt = "> "
		m.input.PromptStyle = styleInputPrompt
	}
}

// viewportKeyMap returns a viewport keymap with Up/Down disabled
// (we use those for input history).
func viewportKeyMap() viewport.KeyMap {
	return viewport.KeyMap{
		PageDown:     key.NewBinding(key.WithKeys("pgdown")),
		PageUp:       key.NewBinding(key.WithKeys("pgup")),
		HalfPageDown: key.NewBinding(key.WithKeys("ctrl+d")),
		HalfPageUp:   key.NewBinding(key.WithKeys("ctrl+u")),
		Up:           key.NewBinding(key.WithDisabled()),
		Down:         key.NewBinding(key.WithDisabled()),
	}
}
