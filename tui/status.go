package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nathoo/questcore/engine/state"
)

// roomDisplayName derives a human-readable name from a room ID.
// "great_hall" -> "Great Hall", "castle_gates" -> "Castle Gates".
func roomDisplayName(id string) string {
	words := strings.Split(id, "_")
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

// renderStatusBar produces a full-width inverted status line showing
// current room, exits, inventory, and turn count.
func (m Model) renderStatusBar() string {
	s := m.engine.State

	roomName := roomDisplayName(s.Player.Location)

	exits := state.RoomExits(s, m.defs, s.Player.Location)
	dirs := make([]string, 0, len(exits))
	for dir := range exits {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)
	exitStr := strings.Join(dirs, ",")

	invCount := len(s.Player.Inventory)

	left := fmt.Sprintf(" %s | Exits: %s", roomName, exitStr)
	right := fmt.Sprintf("T:%d ", s.TurnCount)

	// Show inventory items if they fit, otherwise just count.
	if invCount > 0 {
		var names []string
		for _, id := range s.Player.Inventory {
			name := id
			if n, ok := state.GetEntityProp(s, m.defs, id, "name"); ok {
				if ns, ok := n.(string); ok {
					name = ns
				}
			}
			names = append(names, name)
		}
		invStr := strings.Join(names, ", ")
		candidate := fmt.Sprintf("Inv: %s | T:%d ", invStr, s.TurnCount)
		if lipgloss.Width(left)+lipgloss.Width(candidate)+2 < m.width {
			right = candidate
		} else {
			right = fmt.Sprintf("Inv: %d | T:%d ", invCount, s.TurnCount)
		}
	}

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}

	bar := left + strings.Repeat(" ", gap) + right
	return styleStatusBar.Width(m.width).Render(bar)
}
