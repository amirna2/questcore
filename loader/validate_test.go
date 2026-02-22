package loader

import (
	"testing"

	"github.com/nathoo/questcore/engine/state"
	"github.com/nathoo/questcore/types"
)

// validDefs returns a minimal valid Defs for testing.
func validDefs() *state.Defs {
	return &state.Defs{
		Game: types.GameDef{
			Title: "Test",
			Start: "hall",
		},
		Rooms: map[string]types.RoomDef{
			"hall": {
				ID:          "hall",
				Description: "A hall.",
			},
		},
		Entities: map[string]types.EntityDef{},
	}
}

func TestValidate_ValidDefs(t *testing.T) {
	defs := validDefs()
	if err := validate(defs); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidate_MissingStartRoom(t *testing.T) {
	defs := validDefs()
	defs.Game.Start = "nonexistent"

	err := validate(defs)
	if err == nil {
		t.Fatal("expected error for missing start room")
	}
	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected *ValidationError, got %T", err)
	}
	assertContains(t, ve.Errors, "start room")
}

func TestValidate_EmptyTitle(t *testing.T) {
	defs := validDefs()
	defs.Game.Title = ""

	err := validate(defs)
	if err == nil {
		t.Fatal("expected error for empty title")
	}
	ve := err.(*ValidationError)
	assertContains(t, ve.Errors, "Title")
}

func TestValidate_InvalidExitTarget(t *testing.T) {
	defs := validDefs()
	defs.Rooms["hall"] = types.RoomDef{
		ID:    "hall",
		Exits: map[string]string{"north": "void"},
	}

	err := validate(defs)
	if err == nil {
		t.Fatal("expected error for invalid exit target")
	}
	ve := err.(*ValidationError)
	assertContains(t, ve.Errors, "undefined room")
}

func TestValidate_DuplicateRuleID(t *testing.T) {
	defs := validDefs()
	defs.GlobalRules = []types.RuleDef{
		{ID: "dup", Scope: "global"},
		{ID: "dup", Scope: "global"},
	}

	err := validate(defs)
	if err == nil {
		t.Fatal("expected error for duplicate rule ID")
	}
	ve := err.(*ValidationError)
	assertContains(t, ve.Errors, "duplicate rule ID")
}

func TestValidate_UnknownEffectType(t *testing.T) {
	defs := validDefs()
	defs.GlobalRules = []types.RuleDef{
		{
			ID:      "r1",
			Scope:   "global",
			Effects: []types.Effect{{Type: "explode", Params: map[string]any{}}},
		},
	}

	err := validate(defs)
	if err == nil {
		t.Fatal("expected error for unknown effect type")
	}
	ve := err.(*ValidationError)
	assertContains(t, ve.Errors, "unknown effect type")
}

func TestValidate_UnknownConditionType(t *testing.T) {
	defs := validDefs()
	defs.GlobalRules = []types.RuleDef{
		{
			ID:    "r1",
			Scope: "global",
			Conditions: []types.Condition{
				{Type: "is_tuesday", Params: map[string]any{}},
			},
		},
	}

	err := validate(defs)
	if err == nil {
		t.Fatal("expected error for unknown condition type")
	}
	ve := err.(*ValidationError)
	assertContains(t, ve.Errors, "unknown condition type")
}

func TestValidate_UndefinedEntityInEffect(t *testing.T) {
	defs := validDefs()
	defs.GlobalRules = []types.RuleDef{
		{
			ID:    "r1",
			Scope: "global",
			Effects: []types.Effect{
				{Type: "give_item", Params: map[string]any{"item": "ghost_item"}},
			},
		},
	}

	err := validate(defs)
	if err == nil {
		t.Fatal("expected error for undefined entity in effect")
	}
	ve := err.(*ValidationError)
	assertContains(t, ve.Errors, "undefined entity")
}

func TestValidate_TemplateRefNotFlagged(t *testing.T) {
	defs := validDefs()
	defs.GlobalRules = []types.RuleDef{
		{
			ID:    "r1",
			Scope: "global",
			Effects: []types.Effect{
				{Type: "give_item", Params: map[string]any{"item": "{object}"}},
			},
		},
	}

	if err := validate(defs); err != nil {
		t.Fatalf("template refs should not be flagged, got: %v", err)
	}
}

func TestValidate_DanglingItemLocation_Warning(t *testing.T) {
	defs := validDefs()
	defs.Entities["key"] = types.EntityDef{
		ID:   "key",
		Kind: "item",
		Props: map[string]any{
			"location": "nonexistent_room",
		},
	}

	// Should not return error (only warning).
	if err := validate(defs); err != nil {
		t.Fatalf("dangling location should be warning only, got error: %v", err)
	}
}

func TestValidate_UnrecognizedVerb_Warning(t *testing.T) {
	defs := validDefs()
	defs.GlobalRules = []types.RuleDef{
		{
			ID:    "r1",
			Scope: "global",
			When:  types.MatchCriteria{Verb: "yeet"},
		},
	}

	// Should not return error (only warning).
	if err := validate(defs); err != nil {
		t.Fatalf("unrecognized verb should be warning only, got error: %v", err)
	}
}

func TestValidate_UndefinedRoomInEffect(t *testing.T) {
	defs := validDefs()
	defs.GlobalRules = []types.RuleDef{
		{
			ID:    "r1",
			Scope: "global",
			Effects: []types.Effect{
				{Type: "move_player", Params: map[string]any{"room": "abyss"}},
			},
		},
	}

	err := validate(defs)
	if err == nil {
		t.Fatal("expected error for undefined room in effect")
	}
	ve := err.(*ValidationError)
	assertContains(t, ve.Errors, "undefined room")
}

func TestValidate_UndefinedEntityInCondition(t *testing.T) {
	defs := validDefs()
	defs.GlobalRules = []types.RuleDef{
		{
			ID:    "r1",
			Scope: "global",
			Conditions: []types.Condition{
				{Type: "has_item", Params: map[string]any{"item": "ghost"}},
			},
		},
	}

	err := validate(defs)
	if err == nil {
		t.Fatal("expected error for undefined entity in condition")
	}
	ve := err.(*ValidationError)
	assertContains(t, ve.Errors, "undefined entity")
}

func TestValidate_UndefinedRoomInCondition(t *testing.T) {
	defs := validDefs()
	defs.GlobalRules = []types.RuleDef{
		{
			ID:    "r1",
			Scope: "global",
			Conditions: []types.Condition{
				{Type: "in_room", Params: map[string]any{"room": "nowhere"}},
			},
		},
	}

	err := validate(defs)
	if err == nil {
		t.Fatal("expected error for undefined room in condition")
	}
	ve := err.(*ValidationError)
	assertContains(t, ve.Errors, "undefined room")
}

// assertContains checks that at least one string in the slice contains substr.
func assertContains(t *testing.T, strs []string, substr string) {
	t.Helper()
	for _, s := range strs {
		if contains(s, substr) {
			return
		}
	}
	t.Errorf("expected one of %v to contain %q", strs, substr)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
