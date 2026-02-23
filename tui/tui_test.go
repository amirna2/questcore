package tui

import (
	"strings"
	"testing"

	"github.com/nathoo/questcore/engine"
	"github.com/nathoo/questcore/engine/state"
	"github.com/nathoo/questcore/types"
)

func TestRoomDisplayName(t *testing.T) {
	tests := []struct {
		id   string
		want string
	}{
		{"hall", "Hall"},
		{"great_hall", "Great Hall"},
		{"castle_gates", "Castle Gates"},
		{"tower_top", "Tower Top"},
		{"secret_passage", "Secret Passage"},
	}
	for _, tt := range tests {
		got := roomDisplayName(tt.id)
		if got != tt.want {
			t.Errorf("roomDisplayName(%q) = %q, want %q", tt.id, got, tt.want)
		}
	}
}

func TestClassifyLine(t *testing.T) {
	tests := []struct {
		line string
		want lineKind
	}{
		{"You see: rusty key, old book.", kindYouSee},
		{"Exits: north, south, east.", kindExits},
		{"[Game saved to test.]", kindSystem},
		{"[trace] Effects: 2", kindTrace},
		{"You don't see that here.", kindError},
		{"You can't go that way.", kindError},
		{"You don't have that.", kindError},
		{"A grand hall with stone walls.", kindRoomDesc},
		{"Taken.", kindRoomDesc},
		{"", kindRoomDesc},
		{"'Ah, the adventurer. I wondered when they'd send someone competent.'", kindDialogue},
	}
	for _, tt := range tests {
		got := classifyLine(tt.line)
		if got != tt.want {
			t.Errorf("classifyLine(%q) = %v, want %v", tt.line, got, tt.want)
		}
	}
}

func TestContainsQuotedSpeech(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{"'Hello, adventurer. Welcome to the castle.'", true},
		{"It's a door.", false},            // short quote segment
		{"No quotes here.", false},         // no quotes at all
		{"'Hi'", false},                    // too short
		{"She says 'the crown is lost forever, you must find it.'", true},
	}
	for _, tt := range tests {
		got := containsQuotedSpeech(tt.line)
		if got != tt.want {
			t.Errorf("containsQuotedSpeech(%q) = %v, want %v", tt.line, got, tt.want)
		}
	}
}

func TestWordWrap(t *testing.T) {
	tests := []struct {
		text  string
		width int
		want  string
	}{
		{"short", 80, "short"},
		{"hello world", 5, "hello\nworld"},
		{"The great hall stretches before you with its vaulted ceiling.", 30,
			"The great hall stretches\nbefore you with its vaulted\nceiling."},
		{"", 80, ""},
		{"one", 80, "one"},
		{"a b c d e", 3, "a b\nc d\ne"},
	}
	for _, tt := range tests {
		got := wordWrap(tt.text, tt.width)
		if got != tt.want {
			t.Errorf("wordWrap(%q, %d) =\n  %q\nwant:\n  %q", tt.text, tt.width, got, tt.want)
		}
	}
}

func TestHistory_PushAndPrev(t *testing.T) {
	h := NewHistory(5)
	h.Push("look")
	h.Push("go north")
	h.Push("take key")

	prev, ok := h.Prev()
	if !ok || prev != "take key" {
		t.Errorf("expected 'take key', got %q (ok=%v)", prev, ok)
	}

	prev, ok = h.Prev()
	if !ok || prev != "go north" {
		t.Errorf("expected 'go north', got %q (ok=%v)", prev, ok)
	}

	prev, ok = h.Prev()
	if !ok || prev != "look" {
		t.Errorf("expected 'look', got %q (ok=%v)", prev, ok)
	}

	// At oldest, stays there.
	prev, ok = h.Prev()
	if !ok || prev != "look" {
		t.Errorf("expected 'look' at boundary, got %q (ok=%v)", prev, ok)
	}
}

func TestHistory_Next(t *testing.T) {
	h := NewHistory(5)
	h.Push("look")
	h.Push("go north")

	h.Prev() // "go north"
	h.Prev() // "look"

	next, ok := h.Next()
	if !ok || next != "go north" {
		t.Errorf("expected 'go north', got %q (ok=%v)", next, ok)
	}

	_, ok = h.Next()
	if ok {
		t.Error("expected false when past newest entry")
	}
}

func TestHistory_Empty(t *testing.T) {
	h := NewHistory(5)
	_, ok := h.Prev()
	if ok {
		t.Error("expected false on empty history")
	}
	_, ok = h.Next()
	if ok {
		t.Error("expected false on empty history")
	}
}

func TestHistory_MaxSize(t *testing.T) {
	h := NewHistory(2)
	h.Push("a")
	h.Push("b")
	h.Push("c") // "a" evicted

	prev, _ := h.Prev()
	if prev != "c" {
		t.Errorf("expected 'c', got %q", prev)
	}
	prev, _ = h.Prev()
	if prev != "b" {
		t.Errorf("expected 'b', got %q", prev)
	}
	// "a" is gone.
	prev, _ = h.Prev()
	if prev != "b" {
		t.Errorf("expected 'b' at boundary, got %q", prev)
	}
}

func TestHistory_NoDuplicates(t *testing.T) {
	h := NewHistory(5)
	h.Push("look")
	h.Push("look") // skipped
	h.Push("look") // skipped

	if len(h.entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(h.entries))
	}
}

func TestHistory_ResetCursor(t *testing.T) {
	h := NewHistory(5)
	h.Push("look")
	h.Push("go north")

	h.Prev() // "go north"
	h.ResetCursor()

	// After reset, Prev starts from the end again.
	prev, ok := h.Prev()
	if !ok || prev != "go north" {
		t.Errorf("expected 'go north' after reset, got %q", prev)
	}
}

// testDefs returns minimal game definitions for TUI testing.
func testDefs() *state.Defs {
	return &state.Defs{
		Game: types.GameDef{
			Title:   "Test Game",
			Author:  "Test",
			Version: "1.0",
			Start:   "hall",
			Intro:   "Welcome to the test.",
		},
		Rooms: map[string]types.RoomDef{
			"hall": {
				ID:          "hall",
				Description: "A grand hall.",
				Exits:       map[string]string{"north": "garden"},
			},
			"garden": {
				ID:          "garden",
				Description: "A peaceful garden.",
				Exits:       map[string]string{"south": "hall"},
			},
		},
		Entities: map[string]types.EntityDef{
			"key": {
				ID:   "key",
				Kind: "item",
				Props: map[string]any{
					"name":        "rusty key",
					"description": "An old key.",
					"location":    "hall",
					"takeable":    true,
				},
			},
		},
	}
}

func TestHandleMeta_Quit(t *testing.T) {
	defs := testDefs()
	eng := engine.New(defs)
	m := New(eng, defs)

	_, quit := m.handleMeta("/quit")
	if !quit {
		t.Error("expected quit=true for /quit")
	}

	_, quit = m.handleMeta("/exit")
	if !quit {
		t.Error("expected quit=true for /exit")
	}
}

func TestHandleMeta_Save(t *testing.T) {
	defs := testDefs()
	eng := engine.New(defs)
	m := New(eng, defs)
	m.saveDir = t.TempDir()

	output, quit := m.handleMeta("/save test")
	if quit {
		t.Error("save should not quit")
	}
	if len(output) == 0 || !strings.Contains(output[0], "Game saved") {
		t.Errorf("expected save confirmation, got %v", output)
	}
}

func TestHandleMeta_LoadNonexistent(t *testing.T) {
	defs := testDefs()
	eng := engine.New(defs)
	m := New(eng, defs)
	m.saveDir = t.TempDir()

	output, quit := m.handleMeta("/load nonexistent")
	if quit {
		t.Error("load should not quit")
	}
	if len(output) == 0 || !strings.Contains(output[0], "Load failed") {
		t.Errorf("expected load failure, got %v", output)
	}
}

func TestHandleMeta_Help(t *testing.T) {
	defs := testDefs()
	eng := engine.New(defs)
	m := New(eng, defs)

	output, quit := m.handleMeta("/help")
	if quit {
		t.Error("help should not quit")
	}

	joined := strings.Join(output, "\n")
	for _, expected := range []string{"/save", "/load", "/quit", "look", "inventory"} {
		if !strings.Contains(joined, expected) {
			t.Errorf("expected %q in help output", expected)
		}
	}
}

func TestHandleMeta_Trace(t *testing.T) {
	defs := testDefs()
	eng := engine.New(defs)
	m := New(eng, defs)

	output, _ := m.handleMeta("/trace")
	if !m.trace {
		t.Error("expected trace to be enabled")
	}
	if len(output) == 0 || !strings.Contains(output[0], "enabled") {
		t.Errorf("expected enabled message, got %v", output)
	}

	output, _ = m.handleMeta("/trace")
	if m.trace {
		t.Error("expected trace to be disabled")
	}
	if len(output) == 0 || !strings.Contains(output[0], "disabled") {
		t.Errorf("expected disabled message, got %v", output)
	}
}

func TestHandleMeta_Unknown(t *testing.T) {
	defs := testDefs()
	eng := engine.New(defs)
	m := New(eng, defs)

	output, quit := m.handleMeta("/bogus")
	if quit {
		t.Error("unknown command should not quit")
	}
	if len(output) == 0 || !strings.Contains(output[0], "Unknown command") {
		t.Errorf("expected unknown command message, got %v", output)
	}
}

func TestHandleMeta_State(t *testing.T) {
	defs := testDefs()
	eng := engine.New(defs)
	m := New(eng, defs)

	output, quit := m.handleMeta("/state")
	if quit {
		t.Error("state should not quit")
	}

	joined := strings.Join(output, "\n")
	if !strings.Contains(joined, "Location: hall") {
		t.Error("expected location in state output")
	}
	if !strings.Contains(joined, "Turn:") {
		t.Error("expected turn count in state output")
	}
}
