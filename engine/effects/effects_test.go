package effects

import (
	"testing"

	"github.com/nathoo/questcore/engine/state"
	"github.com/nathoo/questcore/types"
)

func testSetup() (*types.State, *state.Defs, Context) {
	defs := &state.Defs{
		Game: types.GameDef{Start: "hall"},
		Rooms: map[string]types.RoomDef{
			"hall": {
				ID:          "hall",
				Description: "A grand hall with marble columns.",
				Exits:       map[string]string{"south": "entrance"},
			},
			"entrance": {
				ID:          "entrance",
				Description: "The entrance.",
				Exits:       map[string]string{"north": "hall"},
			},
		},
		Entities: map[string]types.EntityDef{
			"rusty_key": {
				ID:   "rusty_key",
				Kind: "item",
				Props: map[string]any{
					"name":        "Rusty Key",
					"description": "An old iron key.",
					"location":    "hall",
				},
			},
			"iron_door": {
				ID:   "iron_door",
				Kind: "entity",
				Props: map[string]any{
					"name":     "Iron Door",
					"location": "hall",
					"locked":   true,
				},
			},
			"guard": {
				ID:   "guard",
				Kind: "npc",
				Props: map[string]any{
					"name":     "Old Guard",
					"location": "hall",
				},
			},
		},
	}
	s := state.NewState(defs)
	ctx := Context{Verb: "use", ObjectID: "rusty_key", TargetID: "iron_door"}
	return s, defs, ctx
}

func TestApply_Say(t *testing.T) {
	s, defs, ctx := testSetup()
	effects := []types.Effect{
		{Type: "say", Params: map[string]any{"text": "Hello, world!"}},
	}

	_, output := Apply(s, defs, effects, ctx)
	if len(output) != 1 || output[0] != "Hello, world!" {
		t.Errorf("expected [Hello, world!], got %v", output)
	}
}

func TestApply_Say_TemplateInterpolation(t *testing.T) {
	s, defs, ctx := testSetup()
	effects := []types.Effect{
		{Type: "say", Params: map[string]any{"text": "You use {object.name} on {target.name}."}},
	}

	_, output := Apply(s, defs, effects, ctx)
	expected := "You use Rusty Key on Iron Door."
	if len(output) != 1 || output[0] != expected {
		t.Errorf("expected %q, got %v", expected, output)
	}
}

func TestApply_Say_RoomDescription(t *testing.T) {
	s, defs, ctx := testSetup()
	effects := []types.Effect{
		{Type: "say", Params: map[string]any{"text": "{room.description}"}},
	}

	_, output := Apply(s, defs, effects, ctx)
	expected := "A grand hall with marble columns."
	if len(output) != 1 || output[0] != expected {
		t.Errorf("expected %q, got %v", expected, output)
	}
}

func TestApply_Say_PlayerInventory_Empty(t *testing.T) {
	s, defs, ctx := testSetup()
	effects := []types.Effect{
		{Type: "say", Params: map[string]any{"text": "{player.inventory}"}},
	}

	_, output := Apply(s, defs, effects, ctx)
	expected := "You are carrying nothing."
	if len(output) != 1 || output[0] != expected {
		t.Errorf("expected %q, got %v", expected, output)
	}
}

func TestApply_Say_PlayerInventory_WithItems(t *testing.T) {
	s, defs, ctx := testSetup()
	s.Player.Inventory = []string{"rusty_key"}
	effects := []types.Effect{
		{Type: "say", Params: map[string]any{"text": "{player.inventory}"}},
	}

	_, output := Apply(s, defs, effects, ctx)
	expected := "Rusty Key"
	if len(output) != 1 || output[0] != expected {
		t.Errorf("expected %q, got %v", expected, output)
	}
}

func TestApply_GiveItem(t *testing.T) {
	s, defs, ctx := testSetup()
	effects := []types.Effect{
		{Type: "give_item", Params: map[string]any{"item": "rusty_key"}},
	}

	events, _ := Apply(s, defs, effects, ctx)

	// Item should be in inventory.
	if len(s.Player.Inventory) != 1 || s.Player.Inventory[0] != "rusty_key" {
		t.Errorf("expected rusty_key in inventory, got %v", s.Player.Inventory)
	}
	// Entity location should be cleared.
	es := s.Entities["rusty_key"]
	if es.Location == "" {
		t.Error("expected entity location to be set to sentinel (not empty)")
	}
	// Should emit item_taken event.
	if len(events) != 1 || events[0].Type != "item_taken" {
		t.Errorf("expected item_taken event, got %v", events)
	}
}

func TestApply_GiveItem_TemplateObject(t *testing.T) {
	s, defs, ctx := testSetup()
	effects := []types.Effect{
		{Type: "give_item", Params: map[string]any{"item": "{object}"}},
	}

	Apply(s, defs, effects, ctx)
	if len(s.Player.Inventory) != 1 || s.Player.Inventory[0] != "rusty_key" {
		t.Errorf("expected rusty_key in inventory, got %v", s.Player.Inventory)
	}
}

func TestApply_RemoveItem(t *testing.T) {
	s, defs, ctx := testSetup()
	s.Player.Inventory = []string{"rusty_key", "sword"}
	effects := []types.Effect{
		{Type: "remove_item", Params: map[string]any{"item": "rusty_key"}},
	}

	events, _ := Apply(s, defs, effects, ctx)

	if len(s.Player.Inventory) != 1 || s.Player.Inventory[0] != "sword" {
		t.Errorf("expected [sword], got %v", s.Player.Inventory)
	}
	if len(events) != 1 || events[0].Type != "item_dropped" {
		t.Errorf("expected item_dropped event, got %v", events)
	}
}

func TestApply_SetFlag(t *testing.T) {
	s, defs, ctx := testSetup()
	effects := []types.Effect{
		{Type: "set_flag", Params: map[string]any{"flag": "door_unlocked", "value": true}},
	}

	events, _ := Apply(s, defs, effects, ctx)

	if !s.Flags["door_unlocked"] {
		t.Error("expected door_unlocked to be true")
	}
	if len(events) != 1 || events[0].Type != "flag_changed" {
		t.Errorf("expected flag_changed event, got %v", events)
	}
}

func TestApply_IncCounter(t *testing.T) {
	s, defs, ctx := testSetup()
	s.Counters["score"] = 10
	effects := []types.Effect{
		{Type: "inc_counter", Params: map[string]any{"counter": "score", "amount": 5}},
	}

	Apply(s, defs, effects, ctx)

	if s.Counters["score"] != 15 {
		t.Errorf("expected score 15, got %d", s.Counters["score"])
	}
}

func TestApply_SetCounter(t *testing.T) {
	s, defs, ctx := testSetup()
	s.Counters["score"] = 10
	effects := []types.Effect{
		{Type: "set_counter", Params: map[string]any{"counter": "score", "value": 0}},
	}

	Apply(s, defs, effects, ctx)

	if s.Counters["score"] != 0 {
		t.Errorf("expected score 0, got %d", s.Counters["score"])
	}
}

func TestApply_SetProp(t *testing.T) {
	s, defs, ctx := testSetup()
	effects := []types.Effect{
		{Type: "set_prop", Params: map[string]any{"entity": "iron_door", "prop": "locked", "value": false}},
	}

	Apply(s, defs, effects, ctx)

	val, ok := state.GetEntityProp(s, defs, "iron_door", "locked")
	if !ok {
		t.Fatal("expected to find locked prop")
	}
	if val != false {
		t.Errorf("expected locked=false, got %v", val)
	}
}

func TestApply_MoveEntity(t *testing.T) {
	s, defs, ctx := testSetup()
	effects := []types.Effect{
		{Type: "move_entity", Params: map[string]any{"entity": "guard", "room": "entrance"}},
	}

	events, _ := Apply(s, defs, effects, ctx)

	loc := state.EntityLocation(s, defs, "guard")
	if loc != "entrance" {
		t.Errorf("expected entrance, got %q", loc)
	}
	if len(events) != 1 || events[0].Type != "entity_moved" {
		t.Errorf("expected entity_moved event, got %v", events)
	}
}

func TestApply_MovePlayer(t *testing.T) {
	s, defs, ctx := testSetup()
	effects := []types.Effect{
		{Type: "move_player", Params: map[string]any{"room": "entrance"}},
	}

	events, _ := Apply(s, defs, effects, ctx)

	if s.Player.Location != "entrance" {
		t.Errorf("expected entrance, got %q", s.Player.Location)
	}
	if len(events) != 1 || events[0].Type != "room_entered" {
		t.Errorf("expected room_entered event, got %v", events)
	}
}

func TestApply_OpenExit(t *testing.T) {
	s, defs, ctx := testSetup()
	effects := []types.Effect{
		{Type: "open_exit", Params: map[string]any{"room": "hall", "direction": "west", "target": "treasury"}},
	}

	Apply(s, defs, effects, ctx)

	exits := state.RoomExits(s, defs, "hall")
	if exits["west"] != "treasury" {
		t.Errorf("expected west→treasury, got %v", exits)
	}
	// Original exit should still exist.
	if exits["south"] != "entrance" {
		t.Errorf("expected south→entrance to remain, got %v", exits)
	}
}

func TestApply_CloseExit(t *testing.T) {
	s, defs, ctx := testSetup()
	effects := []types.Effect{
		{Type: "close_exit", Params: map[string]any{"room": "hall", "direction": "south"}},
	}

	Apply(s, defs, effects, ctx)

	exits := state.RoomExits(s, defs, "hall")
	if _, ok := exits["south"]; ok {
		t.Error("expected south exit to be closed")
	}
}

func TestApply_EmitEvent(t *testing.T) {
	s, defs, ctx := testSetup()
	effects := []types.Effect{
		{Type: "emit_event", Params: map[string]any{"event": "puzzle_solved"}},
	}

	events, _ := Apply(s, defs, effects, ctx)

	if len(events) != 1 || events[0].Type != "puzzle_solved" {
		t.Errorf("expected puzzle_solved event, got %v", events)
	}
}

func TestApply_Stop(t *testing.T) {
	s, defs, ctx := testSetup()
	effects := []types.Effect{
		{Type: "say", Params: map[string]any{"text": "first"}},
		{Type: "stop"},
		{Type: "say", Params: map[string]any{"text": "second"}},
	}

	_, output := Apply(s, defs, effects, ctx)

	if len(output) != 1 || output[0] != "first" {
		t.Errorf("expected [first] (stop should halt), got %v", output)
	}
}

func TestApply_UnknownEffect_NoError(t *testing.T) {
	s, defs, ctx := testSetup()
	effects := []types.Effect{
		{Type: "bogus_effect", Params: map[string]any{}},
		{Type: "say", Params: map[string]any{"text": "still works"}},
	}

	_, output := Apply(s, defs, effects, ctx)

	if len(output) != 1 || output[0] != "still works" {
		t.Errorf("expected [still works], got %v", output)
	}
}

func TestApply_MultipleEffects(t *testing.T) {
	s, defs, ctx := testSetup()
	effects := []types.Effect{
		{Type: "give_item", Params: map[string]any{"item": "rusty_key"}},
		{Type: "set_flag", Params: map[string]any{"flag": "has_key", "value": true}},
		{Type: "set_prop", Params: map[string]any{"entity": "iron_door", "prop": "locked", "value": false}},
		{Type: "say", Params: map[string]any{"text": "You take the key and unlock the door."}},
	}

	events, output := Apply(s, defs, effects, ctx)

	// Verify all mutations.
	if len(s.Player.Inventory) != 1 {
		t.Errorf("expected 1 item in inventory, got %d", len(s.Player.Inventory))
	}
	if !s.Flags["has_key"] {
		t.Error("expected has_key flag to be true")
	}
	val, _ := state.GetEntityProp(s, defs, "iron_door", "locked")
	if val != false {
		t.Error("expected iron_door locked=false")
	}
	if len(output) != 1 {
		t.Errorf("expected 1 output, got %d", len(output))
	}
	// item_taken + flag_changed = 2 events.
	if len(events) != 2 {
		t.Errorf("expected 2 events, got %d", len(events))
	}
}
