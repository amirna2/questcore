package engine

import (
	"strings"
	"testing"

	"github.com/nathoo/questcore/engine/state"
	"github.com/nathoo/questcore/types"
)

// testDefs builds a small test game: 2 rooms, a key, a book, a statue, and some rules.
// Entity names are single words to work with the parser's word splitting.
func testDefs() *state.Defs {
	return &state.Defs{
		Game: types.GameDef{
			Title:   "Test Game",
			Version: "1.0",
			Start:   "hall",
		},
		Rooms: map[string]types.RoomDef{
			"hall": {
				ID:          "hall",
				Description: "A grand hall with stone walls.",
				Exits:       map[string]string{"north": "garden"},
				Rules: []types.RuleDef{
					{
						ID:    "hall_take_key",
						Scope: "room:hall",
						When:  types.MatchCriteria{Verb: "take", Object: "key"},
						Effects: []types.Effect{
							{Type: "say", Params: map[string]any{"text": "You carefully lift the key from the pedestal."}},
							{Type: "give_item", Params: map[string]any{"item": "key"}},
						},
						SourceOrder: 0,
					},
				},
			},
			"garden": {
				ID:          "garden",
				Description: "A beautiful garden with flowers.",
				Exits:       map[string]string{"south": "hall"},
			},
		},
		Entities: map[string]types.EntityDef{
			"key": {
				ID:   "key",
				Kind: "item",
				Props: map[string]any{
					"name":        "Key",
					"description": "A gleaming silver key.",
					"location":    "hall",
					"takeable":    true,
				},
			},
			"book": {
				ID:   "book",
				Kind: "item",
				Props: map[string]any{
					"name":        "Book",
					"description": "A dusty old book.",
					"location":    "hall",
					"takeable":    true,
				},
			},
			"statue": {
				ID:   "statue",
				Kind: "entity",
				Props: map[string]any{
					"name":        "Statue",
					"description": "A weathered statue of a knight.",
					"location":    "hall",
				},
			},
		},
		Handlers: []types.EventHandler{
			{
				EventType: "item_taken",
				Conditions: []types.Condition{
					{Type: "flag_not", Params: map[string]any{"flag": "first_item_msg"}},
				},
				Effects: []types.Effect{
					{Type: "say", Params: map[string]any{"text": "Your first treasure!"}},
					{Type: "set_flag", Params: map[string]any{"flag": "first_item_msg", "value": true}},
				},
			},
		},
	}
}

func outputContains(output []string, substr string) bool {
	for _, line := range output {
		if strings.Contains(line, substr) {
			return true
		}
	}
	return false
}

func TestStep_GoNorth_MovesPlayer(t *testing.T) {
	e := New(testDefs())
	result := e.Step("go north")

	if e.State.Player.Location != "garden" {
		t.Errorf("expected player in 'garden', got %q", e.State.Player.Location)
	}
	if !outputContains(result.Output, "beautiful garden") {
		t.Errorf("expected room description, got %v", result.Output)
	}
}

func TestStep_GoInvalidDirection(t *testing.T) {
	e := New(testDefs())
	result := e.Step("go east")

	if e.State.Player.Location != "hall" {
		t.Errorf("expected player still in 'hall', got %q", e.State.Player.Location)
	}
	if !outputContains(result.Output, "can't go that way") {
		t.Errorf("expected 'can't go that way', got %v", result.Output)
	}
}

func TestStep_DirectionShortcut(t *testing.T) {
	e := New(testDefs())
	result := e.Step("n")

	if e.State.Player.Location != "garden" {
		t.Errorf("expected player in 'garden', got %q", e.State.Player.Location)
	}
	if !outputContains(result.Output, "beautiful garden") {
		t.Errorf("expected garden description, got %v", result.Output)
	}
	_ = result
}

func TestStep_TakeItem_BuiltIn(t *testing.T) {
	e := New(testDefs())
	// book has no room rule override, so built-in take kicks in.
	result := e.Step("take book")

	if len(e.State.Player.Inventory) != 1 || e.State.Player.Inventory[0] != "book" {
		t.Errorf("expected book in inventory, got %v", e.State.Player.Inventory)
	}
	if !outputContains(result.Output, "You take the Book") {
		t.Errorf("expected take message, got %v", result.Output)
	}
}

func TestStep_TakeItem_RuleOverride(t *testing.T) {
	e := New(testDefs())
	// key has a room rule that overrides built-in take.
	result := e.Step("take key")

	if !outputContains(result.Output, "carefully lift the key") {
		t.Errorf("expected room rule text, got %v", result.Output)
	}
	// The rule's give_item effect should add it to inventory.
	if len(e.State.Player.Inventory) != 1 || e.State.Player.Inventory[0] != "key" {
		t.Errorf("expected key in inventory, got %v", e.State.Player.Inventory)
	}
}

func TestStep_Look_DescribesRoom(t *testing.T) {
	e := New(testDefs())
	result := e.Step("look")

	if !outputContains(result.Output, "grand hall") {
		t.Errorf("expected room description, got %v", result.Output)
	}
	if !outputContains(result.Output, "You see:") {
		t.Errorf("expected entity listing, got %v", result.Output)
	}
	if !outputContains(result.Output, "Exits:") {
		t.Errorf("expected exits listing, got %v", result.Output)
	}
}

func TestStep_Inventory_Empty(t *testing.T) {
	e := New(testDefs())
	result := e.Step("inventory")

	if !outputContains(result.Output, "carrying nothing") {
		t.Errorf("expected empty inventory message, got %v", result.Output)
	}
}

func TestStep_Inventory_WithItems(t *testing.T) {
	e := New(testDefs())
	e.State.Player.Inventory = []string{"key"}
	result := e.Step("i")

	if !outputContains(result.Output, "Key") {
		t.Errorf("expected Key in inventory, got %v", result.Output)
	}
}

func TestStep_Examine(t *testing.T) {
	e := New(testDefs())
	result := e.Step("examine statue")

	if !outputContains(result.Output, "weathered statue") {
		t.Errorf("expected statue description, got %v", result.Output)
	}
}

func TestStep_EventHandler_Fires(t *testing.T) {
	e := New(testDefs())
	result := e.Step("take book")

	// Event handler should fire on first item_taken event.
	if !outputContains(result.Output, "first treasure") {
		t.Errorf("expected event handler output, got %v", result.Output)
	}
	if !e.State.Flags["first_item_msg"] {
		t.Error("expected first_item_msg flag to be set")
	}
}

func TestStep_EventHandler_DoesNotFireTwice(t *testing.T) {
	e := New(testDefs())
	e.Step("take book")

	// Drop and re-take.
	e.Step("drop book")
	result := e.Step("take book")

	// Handler condition (flag_not first_item_msg) should fail now.
	if outputContains(result.Output, "first treasure") {
		t.Error("event handler should not fire twice")
	}
}

func TestStep_TurnCounter_Increments(t *testing.T) {
	e := New(testDefs())

	if e.State.TurnCount != 0 {
		t.Fatalf("expected initial turn 0, got %d", e.State.TurnCount)
	}

	e.Step("look")
	if e.State.TurnCount != 1 {
		t.Errorf("expected turn 1 after first step, got %d", e.State.TurnCount)
	}

	e.Step("go north")
	if e.State.TurnCount != 2 {
		t.Errorf("expected turn 2 after second step, got %d", e.State.TurnCount)
	}
}

func TestStep_CommandLogged(t *testing.T) {
	e := New(testDefs())
	e.Step("look")
	e.Step("go north")

	if len(e.State.CommandLog) != 2 {
		t.Fatalf("expected 2 commands logged, got %d", len(e.State.CommandLog))
	}
	if e.State.CommandLog[0] != "look" {
		t.Errorf("expected 'look', got %q", e.State.CommandLog[0])
	}
	if e.State.CommandLog[1] != "go north" {
		t.Errorf("expected 'go north', got %q", e.State.CommandLog[1])
	}
}

func TestStep_EmptyInput(t *testing.T) {
	e := New(testDefs())
	result := e.Step("")

	if !outputContains(result.Output, "What do you want to do?") {
		t.Errorf("expected prompt for empty input, got %v", result.Output)
	}
	// Empty input should not increment turn.
	if e.State.TurnCount != 0 {
		t.Errorf("expected turn 0 for empty input, got %d", e.State.TurnCount)
	}
}

func TestStep_UnknownEntity(t *testing.T) {
	e := New(testDefs())
	result := e.Step("take dragon")

	if !outputContains(result.Output, "don't see") {
		t.Errorf("expected not found error, got %v", result.Output)
	}
}

func TestStep_Drop(t *testing.T) {
	e := New(testDefs())
	e.State.Player.Inventory = []string{"key"}

	result := e.Step("drop key")

	if len(e.State.Player.Inventory) != 0 {
		t.Errorf("expected empty inventory after drop, got %v", e.State.Player.Inventory)
	}
	if !outputContains(result.Output, "You drop the Key") {
		t.Errorf("expected drop message, got %v", result.Output)
	}
}

func TestStep_DropNotHeld(t *testing.T) {
	e := New(testDefs())
	result := e.Step("drop key")

	if !outputContains(result.Output, "don't have that") {
		t.Errorf("expected 'don't have that', got %v", result.Output)
	}
}

func talkTestDefs() *state.Defs {
	return &state.Defs{
		Game: types.GameDef{
			Title:   "Talk Test",
			Version: "1.0",
			Start:   "tavern",
		},
		Rooms: map[string]types.RoomDef{
			"tavern": {
				ID:          "tavern",
				Description: "A dimly lit tavern.",
				Exits:       map[string]string{},
			},
		},
		Entities: map[string]types.EntityDef{
			"barkeep": {
				ID:   "barkeep",
				Kind: "npc",
				Props: map[string]any{
					"name":     "Barkeep",
					"location": "tavern",
				},
				Topics: map[string]types.TopicDef{
					"greeting": {
						Text: "Welcome to the tavern!",
						Effects: []types.Effect{
							{Type: "set_flag", Params: map[string]any{"flag": "met_barkeep", "value": true}},
						},
					},
					"rumors": {
						Text:     "I heard there's treasure in the caves.",
						Requires: []types.Condition{{Type: "flag_set", Params: map[string]any{"flag": "met_barkeep"}}},
					},
				},
			},
			"chair": {
				ID:   "chair",
				Kind: "entity",
				Props: map[string]any{
					"name":     "Chair",
					"location": "tavern",
				},
			},
		},
	}
}

func TestStep_Talk_NoTarget(t *testing.T) {
	e := New(talkTestDefs())
	result := e.Step("talk")

	if !outputContains(result.Output, "Talk to whom?") {
		t.Errorf("expected 'Talk to whom?', got %v", result.Output)
	}
}

func TestStep_Talk_NonNPC(t *testing.T) {
	e := New(talkTestDefs())
	result := e.Step("talk chair")

	if !outputContains(result.Output, "can't talk to that") {
		t.Errorf("expected 'can't talk to that', got %v", result.Output)
	}
}

func TestStep_Talk_AutoPlayFirstTopic(t *testing.T) {
	e := New(talkTestDefs())
	result := e.Step("talk barkeep")

	if !outputContains(result.Output, "Welcome to the tavern!") {
		t.Errorf("expected greeting text, got %v", result.Output)
	}
	if !e.State.Flags["met_barkeep"] {
		t.Error("expected met_barkeep flag to be set")
	}
}

func TestStep_Talk_SpecificTopic(t *testing.T) {
	e := New(talkTestDefs())
	e.State.Flags["met_barkeep"] = true
	result := e.Step("ask barkeep about rumors")

	if !outputContains(result.Output, "treasure in the caves") {
		t.Errorf("expected rumors text, got %v", result.Output)
	}
}

func TestStep_Talk_TopicConditionFails(t *testing.T) {
	e := New(talkTestDefs())
	// met_barkeep not set, so "rumors" condition fails.
	// But "greeting" is still available, so the hint should list it.
	result := e.Step("ask barkeep about rumors")

	if !outputContains(result.Output, "nothing to say about that") {
		t.Errorf("expected 'nothing to say about that', got %v", result.Output)
	}
	if !outputContains(result.Output, "ask about") {
		t.Errorf("expected topic hint, got %v", result.Output)
	}
}

func TestStep_Talk_TopicNotFound_ShowsAvailable(t *testing.T) {
	e := New(talkTestDefs())
	e.State.Flags["met_barkeep"] = true
	result := e.Step("ask barkeep about weather")

	if !outputContains(result.Output, "nothing to say about that") {
		t.Errorf("expected 'nothing to say about that', got %v", result.Output)
	}
	// Should hint at available topics: greeting and rumors.
	if !outputContains(result.Output, "greeting") {
		t.Errorf("expected 'greeting' in topic hint, got %v", result.Output)
	}
	if !outputContains(result.Output, "rumors") {
		t.Errorf("expected 'rumors' in topic hint, got %v", result.Output)
	}
}

func TestStep_Wait(t *testing.T) {
	e := New(testDefs())
	result := e.Step("wait")

	if !outputContains(result.Output, "Time passes.") {
		t.Errorf("expected 'Time passes.', got %v", result.Output)
	}
	if e.State.TurnCount != 1 {
		t.Errorf("expected turn 1 after wait, got %d", e.State.TurnCount)
	}
}

func TestStep_WaitAlias_Z(t *testing.T) {
	e := New(testDefs())
	result := e.Step("z")

	if !outputContains(result.Output, "Time passes.") {
		t.Errorf("expected 'Time passes.' from z alias, got %v", result.Output)
	}
}

func TestStep_WalkNorth(t *testing.T) {
	e := New(testDefs())
	result := e.Step("walk north")

	if e.State.Player.Location != "garden" {
		t.Errorf("expected player in 'garden', got %q", e.State.Player.Location)
	}
	if !outputContains(result.Output, "beautiful garden") {
		t.Errorf("expected room description, got %v", result.Output)
	}
}

func sceneryTestDefs() *state.Defs {
	return &state.Defs{
		Game: types.GameDef{
			Title:   "Scenery Test",
			Version: "1.0",
			Start:   "hall",
		},
		Rooms: map[string]types.RoomDef{
			"hall": {
				ID:          "hall",
				Description: "A grand hall with a massive fireplace and faded tapestries on the walls.",
				Exits:       map[string]string{},
			},
		},
		Entities: map[string]types.EntityDef{
			"key": {
				ID:   "key",
				Kind: "item",
				Props: map[string]any{
					"name":     "Key",
					"location": "hall",
					"takeable": true,
				},
			},
		},
		GlobalRules: []types.RuleDef{
			{
				ID:    "push_wall",
				Scope: "global",
				When:  types.MatchCriteria{Verb: "push", Object: "wall"},
				Effects: []types.Effect{
					{Type: "say", Params: map[string]any{"text": "The wall doesn't budge."}},
				},
			},
		},
	}
}

func TestStep_SceneryFallback_ExamineFireplace(t *testing.T) {
	e := New(sceneryTestDefs())
	result := e.Step("examine fireplace")

	if !outputContains(result.Output, "nothing special about the fireplace") {
		t.Errorf("expected scenery response for fireplace, got %v", result.Output)
	}
}

func TestStep_SceneryFallback_TakeTapestries(t *testing.T) {
	e := New(sceneryTestDefs())
	result := e.Step("take tapestries")

	if !outputContains(result.Output, "can't take the tapestries") {
		t.Errorf("expected scenery take response, got %v", result.Output)
	}
}

func TestStep_SceneryFallback_NotInDescription(t *testing.T) {
	e := New(sceneryTestDefs())
	result := e.Step("examine dragon")

	// "dragon" is not in the room description — should get "don't see" error.
	if !outputContains(result.Output, "don't see") {
		t.Errorf("expected not-found error for dragon, got %v", result.Output)
	}
}

func TestStep_UnresolvedNoun_RuleStillFires(t *testing.T) {
	e := New(sceneryTestDefs())
	// "wall" is not an entity, but there's a global rule for "push wall".
	result := e.Step("push wall")

	if !outputContains(result.Output, "doesn't budge") {
		t.Errorf("expected rule to fire for 'push wall', got %v", result.Output)
	}
}
