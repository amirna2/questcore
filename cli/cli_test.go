package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/nathoo/questcore/engine"
	"github.com/nathoo/questcore/engine/state"
	"github.com/nathoo/questcore/types"
)

// testDefs returns minimal game definitions for CLI testing.
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

func newTestCLI(t *testing.T, input string) (*CLI, *bytes.Buffer) {
	t.Helper()
	defs := testDefs()
	eng := engine.New(defs)
	var out bytes.Buffer
	c := &CLI{
		Engine:  eng,
		Defs:    defs,
		In:      strings.NewReader(input),
		Out:     &out,
		SaveDir: t.TempDir(),
	}
	return c, &out
}

func TestCLI_IntroAndStartingRoom(t *testing.T) {
	c, out := newTestCLI(t, "/quit\n")
	c.Run()

	output := out.String()
	if !strings.Contains(output, "Welcome to the test.") {
		t.Error("expected intro text in output")
	}
	if !strings.Contains(output, "A grand hall.") {
		t.Error("expected starting room description in output")
	}
}

func TestCLI_BasicGameplay(t *testing.T) {
	c, out := newTestCLI(t, "look\n/quit\n")
	c.Run()

	output := out.String()
	if !strings.Contains(output, "A grand hall.") {
		t.Error("expected room description from look command")
	}
}

func TestCLI_Navigation(t *testing.T) {
	c, out := newTestCLI(t, "go north\n/quit\n")
	c.Run()

	output := out.String()
	if !strings.Contains(output, "A peaceful garden.") {
		t.Error("expected garden description after going north")
	}
}

func TestCLI_HelpCommand(t *testing.T) {
	c, out := newTestCLI(t, "/help\n/quit\n")
	c.Run()

	output := out.String()
	if !strings.Contains(output, "/save") {
		t.Error("expected /save in help output")
	}
	if !strings.Contains(output, "/load") {
		t.Error("expected /load in help output")
	}
	if !strings.Contains(output, "/quit") {
		t.Error("expected /quit in help output")
	}
}

func TestCLI_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()

	// Play a bit and save.
	defs := testDefs()
	eng := engine.New(defs)
	var out bytes.Buffer
	c := &CLI{
		Engine:  eng,
		Defs:    defs,
		In:      strings.NewReader("go north\n/save test\n/quit\n"),
		Out:     &out,
		SaveDir: dir,
	}
	c.Run()

	saveOutput := out.String()
	if !strings.Contains(saveOutput, "Game saved to test.") {
		t.Error("expected save confirmation")
	}

	// Start fresh and load.
	eng2 := engine.New(defs)
	var out2 bytes.Buffer
	c2 := &CLI{
		Engine:  eng2,
		Defs:    defs,
		In:      strings.NewReader("/load test\n/quit\n"),
		Out:     &out2,
		SaveDir: dir,
	}
	c2.Run()

	loadOutput := out2.String()
	if !strings.Contains(loadOutput, "Game loaded from test") {
		t.Error("expected load confirmation")
	}
	// After loading, player should be in garden (from the saved state).
	if !strings.Contains(loadOutput, "A peaceful garden.") {
		t.Error("expected garden description after loading save")
	}
}

func TestCLI_UnknownMetaCommand(t *testing.T) {
	c, out := newTestCLI(t, "/bogus\n/quit\n")
	c.Run()

	output := out.String()
	if !strings.Contains(output, "Unknown command") {
		t.Error("expected unknown command message")
	}
}

func TestCLI_TraceToggle(t *testing.T) {
	c, out := newTestCLI(t, "/trace\nlook\n/trace\n/quit\n")
	c.Run()

	output := out.String()
	if !strings.Contains(output, "Trace output enabled") {
		t.Error("expected trace enabled message")
	}
	if !strings.Contains(output, "Trace output disabled") {
		t.Error("expected trace disabled message")
	}
}

func TestCLI_StateCommand(t *testing.T) {
	c, out := newTestCLI(t, "/state\n/quit\n")
	c.Run()

	output := out.String()
	if !strings.Contains(output, "Location: hall") {
		t.Error("expected location in state output")
	}
	if !strings.Contains(output, "Turn:") {
		t.Error("expected turn count in state output")
	}
}

func TestCLI_EmptyInput(t *testing.T) {
	c, out := newTestCLI(t, "\n\n/quit\n")
	c.Run()

	output := out.String()
	// Empty lines should be skipped (no "What do you want to do?" spam).
	count := strings.Count(output, "What do you want to do?")
	if count > 0 {
		t.Error("empty lines should be silently skipped by CLI")
	}
}

func TestCLI_LoadNonexistent(t *testing.T) {
	c, out := newTestCLI(t, "/load nonexistent\n/quit\n")
	c.SaveDir = t.TempDir()
	c.Run()

	output := out.String()
	if !strings.Contains(output, "Load failed") {
		t.Error("expected load failure message")
	}
}

func TestCLI_Again_RepeatsLastCommand(t *testing.T) {
	c, out := newTestCLI(t, "look\nagain\n/quit\n")
	c.Run()

	output := out.String()
	// "look" appears in the intro Step and twice from the two explicit look commands.
	// Count occurrences of the room description — should appear at least 3 times
	// (intro + first look + again).
	count := strings.Count(output, "A grand hall.")
	if count < 3 {
		t.Errorf("expected 'A grand hall.' at least 3 times (intro + look + again), got %d", count)
	}
}

func TestCLI_G_RepeatsLastCommand(t *testing.T) {
	c, out := newTestCLI(t, "look\ng\n/quit\n")
	c.Run()

	output := out.String()
	count := strings.Count(output, "A grand hall.")
	if count < 3 {
		t.Errorf("expected 'A grand hall.' at least 3 times, got %d", count)
	}
}

func TestCLI_Again_NothingToRepeat(t *testing.T) {
	c, out := newTestCLI(t, "again\n/quit\n")
	c.Run()

	output := out.String()
	if !strings.Contains(output, "Nothing to repeat") {
		t.Error("expected 'Nothing to repeat' when no prior command")
	}
}
