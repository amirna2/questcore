package rules

import (
	"testing"

	"github.com/nathoo/questcore/engine/state"
	"github.com/nathoo/questcore/types"
)

func pipelineDefs() *state.Defs {
	return &state.Defs{
		Game: types.GameDef{Start: "hall"},
		Rooms: map[string]types.RoomDef{
			"hall": {
				ID:          "hall",
				Description: "A grand hall.",
				Exits:       map[string]string{"south": "entrance"},
				Rules: []types.RuleDef{
					{
						ID:    "room_take_key",
						Scope: "room:hall",
						When:  types.MatchCriteria{Verb: "take", Object: "rusty_key"},
						Effects: []types.Effect{
							{Type: "say", Params: map[string]any{"text": "You carefully pick up the rusty key."}},
						},
						SourceOrder: 0,
					},
				},
				Fallbacks: map[string]string{
					"push":    "Nothing in this hall can be pushed.",
					"default": "Your footsteps echo through the grand hall.",
				},
			},
		},
		Entities: map[string]types.EntityDef{
			"rusty_key": {
				ID:   "rusty_key",
				Kind: "item",
				Props: map[string]any{
					"name":     "Rusty Key",
					"location": "hall",
					"takeable": true,
				},
				Rules: []types.RuleDef{
					{
						ID:    "entity_examine_key",
						Scope: "entity:rusty_key",
						When:  types.MatchCriteria{Verb: "examine", Object: "rusty_key"},
						Effects: []types.Effect{
							{Type: "say", Params: map[string]any{"text": "It's covered in rust."}},
						},
						SourceOrder: 0,
					},
				},
			},
			"iron_door": {
				ID:   "iron_door",
				Kind: "entity",
				Props: map[string]any{
					"name":     "Iron Door",
					"location": "hall",
					"locked":   true,
					"fallbacks": map[string]any{
						"push": "The door doesn't budge.",
					},
				},
				Rules: []types.RuleDef{
					{
						ID:    "entity_use_key_on_door",
						Scope: "entity:iron_door",
						When:  types.MatchCriteria{Verb: "use", Object: "rusty_key", Target: "iron_door"},
						Conditions: []types.Condition{
							{Type: "has_item", Params: map[string]any{"item": "rusty_key"}},
						},
						Effects: []types.Effect{
							{Type: "say", Params: map[string]any{"text": "The door unlocks!"}},
						},
						SourceOrder: 0,
					},
				},
			},
		},
		GlobalRules: []types.RuleDef{
			{
				ID:    "global_take",
				Scope: "global",
				When:  types.MatchCriteria{Verb: "take", ObjectKind: "item"},
				Effects: []types.Effect{
					{Type: "say", Params: map[string]any{"text": "Taken."}},
				},
				SourceOrder: 0,
			},
			{
				ID:    "global_look",
				Scope: "global",
				When:  types.MatchCriteria{Verb: "look"},
				Effects: []types.Effect{
					{Type: "say", Params: map[string]any{"text": "You look around."}},
				},
				SourceOrder: 1,
			},
		},
	}
}

func TestEvaluate_RoomRuleBeatsGlobal(t *testing.T) {
	defs := pipelineDefs()
	s := state.NewState(defs)
	intent := types.Intent{Verb: "take", Object: "rusty_key"}

	effects, matched := Evaluate(s, defs, intent, "rusty_key", "")
	if !matched {
		t.Fatal("expected matched=true")
	}
	if len(effects) != 1 {
		t.Fatalf("expected 1 effect, got %d", len(effects))
	}
	text, _ := effects[0].Params["text"].(string)
	if text != "You carefully pick up the rusty key." {
		t.Errorf("expected room rule text, got %q", text)
	}
}

func TestEvaluate_TargetEntityRule(t *testing.T) {
	defs := pipelineDefs()
	s := state.NewState(defs)
	s.Player.Inventory = []string{"rusty_key"}
	intent := types.Intent{Verb: "use", Object: "rusty_key", Target: "iron_door"}

	effects, matched := Evaluate(s, defs, intent, "rusty_key", "iron_door")
	if !matched {
		t.Fatal("expected matched=true")
	}
	if len(effects) != 1 {
		t.Fatalf("expected 1 effect, got %d", len(effects))
	}
	text, _ := effects[0].Params["text"].(string)
	if text != "The door unlocks!" {
		t.Errorf("expected target entity rule text, got %q", text)
	}
}

func TestEvaluate_ConditionFails_SkipsRule(t *testing.T) {
	defs := pipelineDefs()
	s := state.NewState(defs)
	// Player does NOT have rusty_key — condition should fail.
	intent := types.Intent{Verb: "use", Object: "rusty_key", Target: "iron_door"}

	effects, matched := Evaluate(s, defs, intent, "rusty_key", "iron_door")
	if matched {
		t.Fatal("expected matched=false for fallback")
	}
	// Should fall through to fallback since no rule's conditions pass.
	if len(effects) != 1 {
		t.Fatalf("expected 1 fallback effect, got %d", len(effects))
	}
	if effects[0].Type != "say" {
		t.Errorf("expected say effect, got %q", effects[0].Type)
	}
}

func TestEvaluate_GlobalRule(t *testing.T) {
	defs := pipelineDefs()
	s := state.NewState(defs)
	intent := types.Intent{Verb: "look"}

	effects, matched := Evaluate(s, defs, intent, "", "")
	if !matched {
		t.Fatal("expected matched=true")
	}
	if len(effects) != 1 {
		t.Fatalf("expected 1 effect, got %d", len(effects))
	}
	text, _ := effects[0].Params["text"].(string)
	if text != "You look around." {
		t.Errorf("expected global look text, got %q", text)
	}
}

func TestEvaluate_ObjectEntityRule(t *testing.T) {
	defs := pipelineDefs()
	s := state.NewState(defs)
	intent := types.Intent{Verb: "examine", Object: "rusty_key"}

	effects, matched := Evaluate(s, defs, intent, "rusty_key", "")
	if !matched {
		t.Fatal("expected matched=true")
	}
	if len(effects) != 1 {
		t.Fatalf("expected 1 effect, got %d", len(effects))
	}
	text, _ := effects[0].Params["text"].(string)
	if text != "It's covered in rust." {
		t.Errorf("expected entity examine text, got %q", text)
	}
}

func TestEvaluate_Fallback_EntityFallback(t *testing.T) {
	defs := pipelineDefs()
	s := state.NewState(defs)
	intent := types.Intent{Verb: "push", Object: "iron_door"}

	effects, matched := Evaluate(s, defs, intent, "iron_door", "")
	if matched {
		t.Fatal("expected matched=false for fallback")
	}
	if len(effects) != 1 {
		t.Fatalf("expected 1 effect, got %d", len(effects))
	}
	text, _ := effects[0].Params["text"].(string)
	if text != "The door doesn't budge." {
		t.Errorf("expected entity fallback text, got %q", text)
	}
}

func TestEvaluate_Fallback_RoomVerbFallback(t *testing.T) {
	defs := pipelineDefs()
	s := state.NewState(defs)
	// Push without a specific entity — should hit room fallback for "push".
	intent := types.Intent{Verb: "push"}

	effects, matched := Evaluate(s, defs, intent, "", "")
	if matched {
		t.Fatal("expected matched=false for fallback")
	}
	if len(effects) != 1 {
		t.Fatalf("expected 1 effect, got %d", len(effects))
	}
	text, _ := effects[0].Params["text"].(string)
	if text != "Nothing in this hall can be pushed." {
		t.Errorf("expected room push fallback, got %q", text)
	}
}

func TestEvaluate_Fallback_RoomDefaultFallback(t *testing.T) {
	defs := pipelineDefs()
	s := state.NewState(defs)
	intent := types.Intent{Verb: "dance"}

	effects, matched := Evaluate(s, defs, intent, "", "")
	if matched {
		t.Fatal("expected matched=false for fallback")
	}
	if len(effects) != 1 {
		t.Fatalf("expected 1 effect, got %d", len(effects))
	}
	text, _ := effects[0].Params["text"].(string)
	if text != "Your footsteps echo through the grand hall." {
		t.Errorf("expected room default fallback, got %q", text)
	}
}

func TestEvaluate_Fallback_GlobalDefault(t *testing.T) {
	defs := &state.Defs{
		Game: types.GameDef{Start: "empty_room"},
		Rooms: map[string]types.RoomDef{
			"empty_room": {
				ID:          "empty_room",
				Description: "Nothing here.",
				Exits:       map[string]string{},
			},
		},
		Entities: map[string]types.EntityDef{},
	}
	s := state.NewState(defs)
	intent := types.Intent{Verb: "dance"}

	effects, matched := Evaluate(s, defs, intent, "", "")
	if matched {
		t.Fatal("expected matched=false for fallback")
	}
	if len(effects) != 1 {
		t.Fatalf("expected 1 effect, got %d", len(effects))
	}
	text, _ := effects[0].Params["text"].(string)
	if text != "You can't do that." {
		t.Errorf("expected global default fallback, got %q", text)
	}
}

func TestEvaluate_SpecificityRanking(t *testing.T) {
	defs := &state.Defs{
		Game: types.GameDef{Start: "room"},
		Rooms: map[string]types.RoomDef{
			"room": {
				ID:    "room",
				Exits: map[string]string{},
				Rules: []types.RuleDef{
					{
						ID:          "generic_take",
						When:        types.MatchCriteria{Verb: "take"},
						Effects:     []types.Effect{{Type: "say", Params: map[string]any{"text": "generic"}}},
						SourceOrder: 0,
					},
					{
						ID:          "specific_take",
						When:        types.MatchCriteria{Verb: "take", Object: "gem"},
						Effects:     []types.Effect{{Type: "say", Params: map[string]any{"text": "specific"}}},
						SourceOrder: 1,
					},
				},
			},
		},
		Entities: map[string]types.EntityDef{
			"gem": {ID: "gem", Kind: "item", Props: map[string]any{"location": "room"}},
		},
	}
	s := state.NewState(defs)

	effects, _ := Evaluate(s, defs, types.Intent{Verb: "take", Object: "gem"}, "gem", "")
	text, _ := effects[0].Params["text"].(string)
	if text != "specific" {
		t.Errorf("expected specific rule to win, got %q", text)
	}
}

func TestEvaluate_PriorityBreaksTie(t *testing.T) {
	defs := &state.Defs{
		Game: types.GameDef{Start: "room"},
		Rooms: map[string]types.RoomDef{
			"room": {
				ID:    "room",
				Exits: map[string]string{},
				Rules: []types.RuleDef{
					{
						ID:          "low_priority",
						When:        types.MatchCriteria{Verb: "look"},
						Effects:     []types.Effect{{Type: "say", Params: map[string]any{"text": "low"}}},
						Priority:    0,
						SourceOrder: 0,
					},
					{
						ID:          "high_priority",
						When:        types.MatchCriteria{Verb: "look"},
						Effects:     []types.Effect{{Type: "say", Params: map[string]any{"text": "high"}}},
						Priority:    10,
						SourceOrder: 1,
					},
				},
			},
		},
		Entities: map[string]types.EntityDef{},
	}
	s := state.NewState(defs)

	effects, _ := Evaluate(s, defs, types.Intent{Verb: "look"}, "", "")
	text, _ := effects[0].Params["text"].(string)
	if text != "high" {
		t.Errorf("expected high priority to win, got %q", text)
	}
}

func TestEvaluate_SourceOrderBreaksTie(t *testing.T) {
	defs := &state.Defs{
		Game: types.GameDef{Start: "room"},
		Rooms: map[string]types.RoomDef{
			"room": {
				ID:    "room",
				Exits: map[string]string{},
				Rules: []types.RuleDef{
					{
						ID:          "first",
						When:        types.MatchCriteria{Verb: "look"},
						Effects:     []types.Effect{{Type: "say", Params: map[string]any{"text": "first"}}},
						Priority:    0,
						SourceOrder: 0,
					},
					{
						ID:          "second",
						When:        types.MatchCriteria{Verb: "look"},
						Effects:     []types.Effect{{Type: "say", Params: map[string]any{"text": "second"}}},
						Priority:    0,
						SourceOrder: 1,
					},
				},
			},
		},
		Entities: map[string]types.EntityDef{},
	}
	s := state.NewState(defs)

	effects, _ := Evaluate(s, defs, types.Intent{Verb: "look"}, "", "")
	text, _ := effects[0].Params["text"].(string)
	if text != "first" {
		t.Errorf("expected earlier source order to win, got %q", text)
	}
}
