package dialogue

import (
	"sort"
	"testing"

	"github.com/nathoo/questcore/engine/state"
	"github.com/nathoo/questcore/types"
)

func testDefs() *state.Defs {
	return &state.Defs{
		Game: types.GameDef{Start: "tavern"},
		Rooms: map[string]types.RoomDef{
			"tavern": {ID: "tavern", Description: "A dimly lit tavern."},
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
						Text: "Welcome to the tavern, stranger!",
						Effects: []types.Effect{
							{Type: "set_flag", Params: map[string]any{"flag": "met_barkeep", "value": true}},
						},
					},
					"rumors": {
						Text:     "I heard there's treasure in the caves...",
						Requires: []types.Condition{{Type: "flag_set", Params: map[string]any{"flag": "met_barkeep"}}},
					},
					"secret": {
						Text:     "The dragon guards the north passage.",
						Requires: []types.Condition{{Type: "has_item", Params: map[string]any{"item": "gold_coin"}}},
						Effects: []types.Effect{
							{Type: "set_flag", Params: map[string]any{"flag": "knows_secret", "value": true}},
						},
					},
				},
			},
			"chest": {
				ID:    "chest",
				Kind:  "item",
				Props: map[string]any{"name": "Chest", "location": "tavern"},
			},
		},
	}
}

func TestAvailableTopics_AllAvailable(t *testing.T) {
	defs := testDefs()
	s := state.NewState(defs)
	s.Flags["met_barkeep"] = true
	s.Player.Inventory = []string{"gold_coin"}

	topics := AvailableTopics("barkeep", s, defs)
	sort.Strings(topics)
	if len(topics) != 3 {
		t.Fatalf("expected 3 topics, got %d: %v", len(topics), topics)
	}
	expected := []string{"greeting", "rumors", "secret"}
	for i, exp := range expected {
		if topics[i] != exp {
			t.Errorf("expected %q, got %q", exp, topics[i])
		}
	}
}

func TestAvailableTopics_GatedByCondition_Hidden(t *testing.T) {
	defs := testDefs()
	s := state.NewState(defs)
	// met_barkeep not set → rumors hidden. No gold_coin → secret hidden.

	topics := AvailableTopics("barkeep", s, defs)
	if len(topics) != 1 {
		t.Fatalf("expected 1 topic (greeting only), got %d: %v", len(topics), topics)
	}
	if topics[0] != "greeting" {
		t.Errorf("expected 'greeting', got %q", topics[0])
	}
}

func TestAvailableTopics_GatedByCondition_Visible(t *testing.T) {
	defs := testDefs()
	s := state.NewState(defs)
	s.Flags["met_barkeep"] = true

	topics := AvailableTopics("barkeep", s, defs)
	sort.Strings(topics)
	if len(topics) != 2 {
		t.Fatalf("expected 2 topics, got %d: %v", len(topics), topics)
	}
	expected := []string{"greeting", "rumors"}
	for i, exp := range expected {
		if topics[i] != exp {
			t.Errorf("expected %q, got %q", exp, topics[i])
		}
	}
}

func TestSelectTopic_Valid(t *testing.T) {
	defs := testDefs()
	s := state.NewState(defs)

	text, effs := SelectTopic("barkeep", "greeting", s, defs)
	if text != "Welcome to the tavern, stranger!" {
		t.Errorf("expected greeting text, got %q", text)
	}
	if len(effs) != 1 || effs[0].Type != "set_flag" {
		t.Errorf("expected 1 set_flag effect, got %v", effs)
	}
}

func TestSelectTopic_ConditionFails(t *testing.T) {
	defs := testDefs()
	s := state.NewState(defs)
	// met_barkeep not set → rumors condition fails.

	text, effs := SelectTopic("barkeep", "rumors", s, defs)
	if text != "" {
		t.Errorf("expected empty text, got %q", text)
	}
	if effs != nil {
		t.Errorf("expected nil effects, got %v", effs)
	}
}

func TestSelectTopic_Nonexistent(t *testing.T) {
	defs := testDefs()
	s := state.NewState(defs)

	text, effs := SelectTopic("barkeep", "weather", s, defs)
	if text != "" {
		t.Errorf("expected empty text, got %q", text)
	}
	if effs != nil {
		t.Errorf("expected nil effects, got %v", effs)
	}
}

func TestAvailableTopics_NoTopics(t *testing.T) {
	defs := testDefs()
	s := state.NewState(defs)

	topics := AvailableTopics("chest", s, defs)
	if topics != nil {
		t.Errorf("expected nil for entity with no topics, got %v", topics)
	}
}

func TestAvailableTopics_UnknownEntity(t *testing.T) {
	defs := testDefs()
	s := state.NewState(defs)

	topics := AvailableTopics("nobody", s, defs)
	if topics != nil {
		t.Errorf("expected nil for unknown entity, got %v", topics)
	}
}
