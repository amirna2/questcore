package save

import (
	"encoding/json"
	"testing"

	"github.com/nathoo/questcore/engine/state"
	"github.com/nathoo/questcore/types"
)

func testDefs() *state.Defs {
	return &state.Defs{
		Game: types.GameDef{
			Title:   "Test Game",
			Version: "1.0",
			Start:   "hall",
		},
		Rooms: map[string]types.RoomDef{
			"hall": {ID: "hall", Description: "A hall.", Exits: map[string]string{"north": "garden"}},
		},
		Entities: map[string]types.EntityDef{
			"key": {ID: "key", Kind: "item", Props: map[string]any{"name": "Key", "location": "hall"}},
		},
	}
}

func TestRoundTrip(t *testing.T) {
	defs := testDefs()
	s := state.NewState(defs)

	// Modify state.
	s.Player.Inventory = []string{"key"}
	s.Player.Location = "garden"
	s.Player.Stats["strength"] = 10
	s.Flags["door_open"] = true
	s.Counters["visits"] = 3
	s.TurnCount = 7
	s.RNGSeed = 42
	s.CommandLog = []string{"go north", "take key"}
	s.Entities["key"] = types.EntityState{
		Location: " ",
		Props:    map[string]any{"shiny": true},
	}

	// Save.
	data, err := Save(s, defs)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load.
	sd, err := Load(data)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Apply to fresh state.
	s2 := state.NewState(defs)
	ApplySave(s2, sd)

	// Verify.
	if s2.Player.Location != "garden" {
		t.Errorf("expected location 'garden', got %q", s2.Player.Location)
	}
	if len(s2.Player.Inventory) != 1 || s2.Player.Inventory[0] != "key" {
		t.Errorf("expected inventory [key], got %v", s2.Player.Inventory)
	}
	if s2.Player.Stats["strength"] != 10 {
		t.Errorf("expected strength 10, got %d", s2.Player.Stats["strength"])
	}
	if !s2.Flags["door_open"] {
		t.Error("expected door_open flag true")
	}
	if s2.Counters["visits"] != 3 {
		t.Errorf("expected visits 3, got %d", s2.Counters["visits"])
	}
	if s2.TurnCount != 7 {
		t.Errorf("expected turn 7, got %d", s2.TurnCount)
	}
	if s2.RNGSeed != 42 {
		t.Errorf("expected seed 42, got %d", s2.RNGSeed)
	}
	if len(s2.CommandLog) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(s2.CommandLog))
	}
	if s2.CommandLog[0] != "go north" || s2.CommandLog[1] != "take key" {
		t.Errorf("command log mismatch: %v", s2.CommandLog)
	}
	es := s2.Entities["key"]
	if es.Location != " " {
		t.Errorf("expected entity location ' ', got %q", es.Location)
	}
	if shiny, ok := es.Props["shiny"]; !ok || shiny != true {
		t.Errorf("expected shiny=true, got %v", es.Props)
	}
}

func TestSave_ProducesValidJSON(t *testing.T) {
	defs := testDefs()
	s := state.NewState(defs)

	data, err := Save(s, defs)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if !json.Valid(data) {
		t.Fatal("Save output is not valid JSON")
	}

	// Verify game metadata.
	var raw map[string]any
	json.Unmarshal(data, &raw)
	if raw["version"] != "1.0" {
		t.Errorf("expected version '1.0', got %v", raw["version"])
	}
	if raw["game"] != "Test Game" {
		t.Errorf("expected game 'Test Game', got %v", raw["game"])
	}
}

func TestLoad_MissingOptionalFields(t *testing.T) {
	// Minimal JSON — only required fields.
	data := []byte(`{"version":"1.0","game":"Test","turn":0,"player":{"Location":"hall"}}`)

	sd, err := Load(data)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if sd.Flags == nil {
		t.Error("expected non-nil flags")
	}
	if sd.Counters == nil {
		t.Error("expected non-nil counters")
	}
	if sd.EntityState == nil {
		t.Error("expected non-nil entity_state")
	}
	if sd.Player.Inventory == nil {
		t.Error("expected non-nil inventory")
	}
	if sd.Player.Stats == nil {
		t.Error("expected non-nil stats")
	}
	if sd.CommandLog == nil {
		t.Error("expected non-nil command_log")
	}
}

func TestRoundTrip_WithCombatState(t *testing.T) {
	defs := testDefs()
	s := state.NewState(defs)

	s.Combat = types.CombatState{
		Active:           true,
		EnemyID:          "goblin",
		RoundCount:       3,
		Defending:        true,
		PreviousLocation: "hall",
	}

	data, err := Save(s, defs)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	sd, err := Load(data)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	s2 := state.NewState(defs)
	ApplySave(s2, sd)

	if !s2.Combat.Active {
		t.Error("expected combat active")
	}
	if s2.Combat.EnemyID != "goblin" {
		t.Errorf("expected enemy 'goblin', got %q", s2.Combat.EnemyID)
	}
	if s2.Combat.RoundCount != 3 {
		t.Errorf("expected round 3, got %d", s2.Combat.RoundCount)
	}
	if !s2.Combat.Defending {
		t.Error("expected defending true")
	}
	if s2.Combat.PreviousLocation != "hall" {
		t.Errorf("expected previous location 'hall', got %q", s2.Combat.PreviousLocation)
	}
}

func TestRoundTrip_WithRNGPosition(t *testing.T) {
	defs := testDefs()
	s := state.NewState(defs)
	s.RNGSeed = 42
	s.RNGPosition = 17

	data, err := Save(s, defs)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	sd, err := Load(data)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if sd.RNGPosition != 17 {
		t.Errorf("expected RNGPosition 17, got %d", sd.RNGPosition)
	}

	s2 := state.NewState(defs)
	ApplySave(s2, sd)

	if s2.RNGPosition != 17 {
		t.Errorf("expected RNGPosition 17 after apply, got %d", s2.RNGPosition)
	}
}

func TestLoad_MissingCombat_DefaultsToInactive(t *testing.T) {
	// JSON without combat or rng_position fields (old save format).
	data := []byte(`{"version":"1.0","game":"Test","turn":5,"player":{"Location":"hall"},"rng_seed":42}`)

	sd, err := Load(data)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if sd.Combat.Active {
		t.Error("expected combat inactive for old save")
	}
	if sd.Combat.EnemyID != "" {
		t.Errorf("expected empty enemy ID, got %q", sd.Combat.EnemyID)
	}
	if sd.RNGPosition != 0 {
		t.Errorf("expected RNGPosition 0, got %d", sd.RNGPosition)
	}
}

func TestEntityState_PreservedThroughRoundTrip(t *testing.T) {
	defs := testDefs()
	s := state.NewState(defs)
	s.Entities["key"] = types.EntityState{
		Location: "garden",
		Props:    map[string]any{"visible": false},
	}

	data, err := Save(s, defs)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	sd, err := Load(data)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	es := sd.EntityState["key"]
	if es.Location != "garden" {
		t.Errorf("expected location 'garden', got %q", es.Location)
	}
	if es.Props["visible"] != false {
		t.Errorf("expected visible=false, got %v", es.Props["visible"])
	}
}
