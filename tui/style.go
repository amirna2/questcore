package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Styles used throughout the TUI.
var (
	styleStatusBar = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("252")).
			Bold(true)

	styleInputPrompt = lipgloss.NewStyle().
				Foreground(lipgloss.Color("34"))

	styleRoomDesc = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255"))

	styleYouSee = lipgloss.NewStyle().
			Bold(true)

	styleExits = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	styleDialogue = lipgloss.NewStyle().
			Foreground(lipgloss.Color("228"))

	styleSystem = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	styleError = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	stylePlayerInput = lipgloss.NewStyle().
				Foreground(lipgloss.Color("34"))

	styleTrace = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))
)

// lineKind identifies the type of an output line for styling.
type lineKind int

const (
	kindRoomDesc lineKind = iota
	kindYouSee
	kindExits
	kindDialogue
	kindSystem
	kindError
	kindTrace
)

// classifyLine determines what kind of output line this is.
func classifyLine(line string) lineKind {
	switch {
	case strings.HasPrefix(line, "[trace]"):
		return kindTrace
	case strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]"):
		return kindSystem
	case strings.HasPrefix(line, "You see:"):
		return kindYouSee
	case strings.HasPrefix(line, "Exits:"):
		return kindExits
	case strings.HasPrefix(line, "You don't see"),
		strings.HasPrefix(line, "You can't"),
		strings.HasPrefix(line, "You don't have"):
		return kindError
	case containsQuotedSpeech(line):
		return kindDialogue
	default:
		return kindRoomDesc
	}
}

// containsQuotedSpeech checks if a line contains NPC dialogue in single quotes.
func containsQuotedSpeech(line string) bool {
	inQuote := false
	quoteLen := 0
	for _, r := range line {
		if r == '\'' {
			if inQuote && quoteLen > 5 {
				return true
			}
			inQuote = !inQuote
			quoteLen = 0
		} else if inQuote {
			quoteLen++
		}
	}
	return false
}

// styledYouSee renders "You see: item1, item2." with item names bold.
func styledYouSee(line string) string {
	const prefix = "You see: "
	if !strings.HasPrefix(line, prefix) {
		return styleRoomDesc.Render(line)
	}
	return styleRoomDesc.Render(prefix) + styleYouSee.Render(line[len(prefix):])
}

// styledPlayerInput renders the echoed player input in green with "> " prefix.
func styledPlayerInput(input string) string {
	return stylePlayerInput.Render("> " + input)
}

// styledSystemMsg renders a system message in gray with brackets.
func styledSystemMsg(text string) string {
	return styleSystem.Render("[" + text + "]")
}
