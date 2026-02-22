// Package tui provides a Bubble Tea terminal UI for the QuestCore game engine.
package tui

// History is a ring buffer for command history with cursor-based navigation.
type History struct {
	entries []string
	max     int
	cursor  int // -1 = not navigating, 0..len-1 = position in entries
}

// NewHistory creates a history buffer with the given maximum size.
func NewHistory(max int) *History {
	return &History{
		entries: make([]string, 0, max),
		max:     max,
		cursor:  -1,
	}
}

// Push adds a command to history. Consecutive duplicates are skipped.
func (h *History) Push(cmd string) {
	if len(h.entries) > 0 && h.entries[len(h.entries)-1] == cmd {
		return
	}
	h.entries = append(h.entries, cmd)
	if len(h.entries) > h.max {
		h.entries = h.entries[1:]
	}
}

// Prev returns the previous (older) history entry.
// Returns ("", false) if history is empty.
func (h *History) Prev() (string, bool) {
	if len(h.entries) == 0 {
		return "", false
	}
	if h.cursor == -1 {
		h.cursor = len(h.entries) - 1
	} else if h.cursor > 0 {
		h.cursor--
	}
	return h.entries[h.cursor], true
}

// Next returns the next (newer) history entry.
// Returns ("", false) when past the most recent entry (back to fresh input).
func (h *History) Next() (string, bool) {
	if h.cursor == -1 {
		return "", false
	}
	h.cursor++
	if h.cursor >= len(h.entries) {
		h.cursor = -1
		return "", false
	}
	return h.entries[h.cursor], true
}

// ResetCursor resets the navigation cursor to the "not navigating" state.
func (h *History) ResetCursor() {
	h.cursor = -1
}
