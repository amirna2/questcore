package resolve

import (
	"testing"

	"github.com/nathoo/questcore/engine/state"
	"github.com/nathoo/questcore/types"
)

func testDefs() *state.Defs {
	return &state.Defs{
		Game: types.GameDef{Start: "hall"},
		Rooms: map[string]types.RoomDef{
			"hall":     {ID: "hall", Exits: map[string]string{"south": "entrance"}},
			"entrance": {ID: "entrance", Exits: map[string]string{"north": "hall"}},
		},
		Entities: map[string]types.EntityDef{
			"rusty_key": {
				ID:   "rusty_key",
				Kind: "item",
				Props: map[string]any{
					"name":     "Rusty Key",
					"location": "hall",
				},
			},
			"golden_key": {
				ID:   "golden_key",
				Kind: "item",
				Props: map[string]any{
					"name":     "Golden Key",
					"location": "entrance",
				},
			},
			"iron_door": {
				ID:   "iron_door",
				Kind: "entity",
				Props: map[string]any{
					"name":     "Iron Door",
					"location": "hall",
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
}

func TestResolve_ExactID(t *testing.T) {
	defs := testDefs()
	s := state.NewState(defs)

	res, err := Resolve(s, defs, types.Intent{Verb: "take", Object: "rusty_key"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ObjectID != "rusty_key" {
		t.Errorf("expected rusty_key, got %q", res.ObjectID)
	}
}

func TestResolve_ByName_CaseInsensitive(t *testing.T) {
	defs := testDefs()
	s := state.NewState(defs)

	res, err := Resolve(s, defs, types.Intent{Verb: "examine", Object: "rusty key"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ObjectID != "rusty_key" {
		t.Errorf("expected rusty_key, got %q", res.ObjectID)
	}
}

func TestResolve_RoomScoped(t *testing.T) {
	defs := testDefs()
	s := state.NewState(defs)
	// Player is in "hall" (start room). Golden key is in "entrance".

	_, err := Resolve(s, defs, types.Intent{Verb: "take", Object: "golden key"})
	if err == nil {
		t.Fatal("expected not-found error for entity in different room")
	}
	if _, ok := err.(*NotFoundError); !ok {
		t.Errorf("expected NotFoundError, got %T: %v", err, err)
	}
}

func TestResolve_Inventory(t *testing.T) {
	defs := testDefs()
	s := state.NewState(defs)
	s.Player.Inventory = []string{"golden_key"}

	// Golden key is in entrance (different room), but player has it.
	res, err := Resolve(s, defs, types.Intent{Verb: "use", Object: "golden key"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ObjectID != "golden_key" {
		t.Errorf("expected golden_key, got %q", res.ObjectID)
	}
}

func TestResolve_ObjectAndTarget(t *testing.T) {
	defs := testDefs()
	s := state.NewState(defs)
	s.Player.Inventory = []string{"rusty_key"}

	res, err := Resolve(s, defs, types.Intent{Verb: "use", Object: "rusty_key", Target: "iron_door"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ObjectID != "rusty_key" {
		t.Errorf("expected rusty_key, got %q", res.ObjectID)
	}
	if res.TargetID != "iron_door" {
		t.Errorf("expected iron_door, got %q", res.TargetID)
	}
}

func TestResolve_NotFound(t *testing.T) {
	defs := testDefs()
	s := state.NewState(defs)

	_, err := Resolve(s, defs, types.Intent{Verb: "take", Object: "sword"})
	if err == nil {
		t.Fatal("expected error")
	}
	nf, ok := err.(*NotFoundError)
	if !ok {
		t.Fatalf("expected NotFoundError, got %T: %v", err, err)
	}
	if nf.Name != "sword" {
		t.Errorf("expected name 'sword', got %q", nf.Name)
	}
}

func TestResolve_Ambiguity(t *testing.T) {
	defs := testDefs()
	// Add a second entity named "Rusty Key" in the same room.
	defs.Entities["rusty_key_2"] = types.EntityDef{
		ID:   "rusty_key_2",
		Kind: "item",
		Props: map[string]any{
			"name":     "Rusty Key",
			"location": "hall",
		},
	}
	s := state.NewState(defs)

	_, err := Resolve(s, defs, types.Intent{Verb: "take", Object: "rusty key"})
	if err == nil {
		t.Fatal("expected ambiguity error")
	}
	ae, ok := err.(*AmbiguityError)
	if !ok {
		t.Fatalf("expected AmbiguityError, got %T: %v", err, err)
	}
	if len(ae.Candidates) != 2 {
		t.Errorf("expected 2 candidates, got %d", len(ae.Candidates))
	}
}

func TestResolve_NoObjectOrTarget(t *testing.T) {
	defs := testDefs()
	s := state.NewState(defs)

	res, err := Resolve(s, defs, types.Intent{Verb: "look"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ObjectID != "" || res.TargetID != "" {
		t.Errorf("expected empty result, got %+v", res)
	}
}

func TestResolve_RuntimeLocationOverride(t *testing.T) {
	defs := testDefs()
	s := state.NewState(defs)
	// Move guard from hall to entrance at runtime.
	s.Entities["guard"] = types.EntityState{Location: "entrance"}

	// Player is in hall; guard should no longer be visible.
	_, err := Resolve(s, defs, types.Intent{Verb: "talk", Object: "old guard"})
	if err == nil {
		t.Fatal("expected not-found error for moved entity")
	}
	if _, ok := err.(*NotFoundError); !ok {
		t.Errorf("expected NotFoundError, got %T: %v", err, err)
	}
}

func TestResolve_ByEntityID_CaseInsensitive(t *testing.T) {
	defs := testDefs()
	s := state.NewState(defs)

	// "guard" is an entity ID, should resolve even though the name is "Old Guard".
	res, err := Resolve(s, defs, types.Intent{Verb: "talk", Object: "guard"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ObjectID != "guard" {
		t.Errorf("expected guard, got %q", res.ObjectID)
	}
}

func TestResolve_PartialNameMatch(t *testing.T) {
	defs := testDefs()
	s := state.NewState(defs)

	// "key" should match "Rusty Key" (word match) when only one key is visible.
	res, err := Resolve(s, defs, types.Intent{Verb: "take", Object: "key"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ObjectID != "rusty_key" {
		t.Errorf("expected rusty_key, got %q", res.ObjectID)
	}
}

func TestResolve_PartialNameMatch_Ambiguity(t *testing.T) {
	defs := testDefs()
	// Move golden_key to same room as rusty_key.
	defs.Entities["golden_key"] = types.EntityDef{
		ID:   "golden_key",
		Kind: "item",
		Props: map[string]any{
			"name":     "Golden Key",
			"location": "hall",
		},
	}
	s := state.NewState(defs)

	// "key" matches both "Rusty Key" and "Golden Key" — ambiguity.
	_, err := Resolve(s, defs, types.Intent{Verb: "take", Object: "key"})
	if err == nil {
		t.Fatal("expected ambiguity error for partial match")
	}
	if _, ok := err.(*AmbiguityError); !ok {
		t.Errorf("expected AmbiguityError, got %T: %v", err, err)
	}
}

func TestResolve_UnderscoreNormalization(t *testing.T) {
	defs := testDefs()
	s := state.NewState(defs)

	// "rusty key" (with space) should match entity ID "rusty_key" (with underscore).
	res, err := Resolve(s, defs, types.Intent{Verb: "take", Object: "rusty key"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ObjectID != "rusty_key" {
		t.Errorf("expected rusty_key, got %q", res.ObjectID)
	}
}

func TestResolve_UnderscoreNormalization_Target(t *testing.T) {
	defs := testDefs()
	s := state.NewState(defs)
	s.Player.Inventory = []string{"rusty_key"}

	// "iron door" should match "iron_door" via underscore normalization.
	res, err := Resolve(s, defs, types.Intent{Verb: "use", Object: "rusty key", Target: "iron door"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ObjectID != "rusty_key" {
		t.Errorf("expected rusty_key, got %q", res.ObjectID)
	}
	if res.TargetID != "iron_door" {
		t.Errorf("expected iron_door, got %q", res.TargetID)
	}
}
