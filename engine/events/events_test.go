package events

import (
	"testing"

	"github.com/nathoo/questcore/engine/state"
	"github.com/nathoo/questcore/types"
)

func testDefs() *state.Defs {
	return &state.Defs{
		Game: types.GameDef{Start: "room1"},
		Rooms: map[string]types.RoomDef{
			"room1": {ID: "room1", Description: "A room.", Exits: map[string]string{}},
		},
		Entities: map[string]types.EntityDef{},
		Handlers: []types.EventHandler{
			{
				EventType: "item_taken",
				Effects: []types.Effect{
					{Type: "say", Params: map[string]any{"text": "You picked something up!"}},
				},
			},
			{
				EventType: "room_entered",
				Conditions: []types.Condition{
					{Type: "flag_set", Params: map[string]any{"flag": "visited_cave"}},
				},
				Effects: []types.Effect{
					{Type: "say", Params: map[string]any{"text": "Welcome back."}},
				},
			},
			{
				EventType: "item_taken",
				Effects: []types.Effect{
					{Type: "inc_counter", Params: map[string]any{"counter": "items_taken", "amount": 1}},
				},
			},
		},
	}
}

func TestDispatch_MatchesEventType(t *testing.T) {
	defs := testDefs()
	s := state.NewState(defs)

	events := []types.Event{
		{Type: "item_taken", Data: map[string]any{"item": "key"}},
	}

	effs := Dispatch(events, s, defs)
	if len(effs) != 2 {
		t.Fatalf("expected 2 effects from 2 matching handlers, got %d", len(effs))
	}
	if effs[0].Type != "say" {
		t.Errorf("expected say effect, got %q", effs[0].Type)
	}
	if effs[1].Type != "inc_counter" {
		t.Errorf("expected inc_counter effect, got %q", effs[1].Type)
	}
}

func TestDispatch_SkipsNonMatchingEventType(t *testing.T) {
	defs := testDefs()
	s := state.NewState(defs)

	events := []types.Event{
		{Type: "flag_changed", Data: map[string]any{}},
	}

	effs := Dispatch(events, s, defs)
	if len(effs) != 0 {
		t.Fatalf("expected 0 effects for non-matching event, got %d", len(effs))
	}
}

func TestDispatch_ConditionFails_Skipped(t *testing.T) {
	defs := testDefs()
	s := state.NewState(defs)
	// visited_cave flag NOT set, so room_entered handler should not fire.

	events := []types.Event{
		{Type: "room_entered", Data: map[string]any{"room": "room1"}},
	}

	effs := Dispatch(events, s, defs)
	if len(effs) != 0 {
		t.Fatalf("expected 0 effects when condition fails, got %d", len(effs))
	}
}

func TestDispatch_ConditionPasses(t *testing.T) {
	defs := testDefs()
	s := state.NewState(defs)
	s.Flags["visited_cave"] = true

	events := []types.Event{
		{Type: "room_entered", Data: map[string]any{"room": "room1"}},
	}

	effs := Dispatch(events, s, defs)
	if len(effs) != 1 {
		t.Fatalf("expected 1 effect when condition passes, got %d", len(effs))
	}
	text, _ := effs[0].Params["text"].(string)
	if text != "Welcome back." {
		t.Errorf("expected 'Welcome back.', got %q", text)
	}
}

func TestDispatch_NoHandlers(t *testing.T) {
	defs := &state.Defs{
		Game: types.GameDef{Start: "room1"},
		Rooms: map[string]types.RoomDef{
			"room1": {ID: "room1"},
		},
		Entities: map[string]types.EntityDef{},
		Handlers: nil,
	}
	s := state.NewState(defs)

	events := []types.Event{
		{Type: "item_taken", Data: map[string]any{}},
	}

	effs := Dispatch(events, s, defs)
	if len(effs) != 0 {
		t.Fatalf("expected 0 effects with no handlers, got %d", len(effs))
	}
}

func TestDispatch_MultipleEvents(t *testing.T) {
	defs := testDefs()
	s := state.NewState(defs)
	s.Flags["visited_cave"] = true

	events := []types.Event{
		{Type: "item_taken", Data: map[string]any{"item": "key"}},
		{Type: "room_entered", Data: map[string]any{"room": "room1"}},
	}

	effs := Dispatch(events, s, defs)
	// item_taken: say + inc_counter = 2, room_entered: say = 1. Total = 3.
	if len(effs) != 3 {
		t.Fatalf("expected 3 effects from multiple events, got %d", len(effs))
	}
}
