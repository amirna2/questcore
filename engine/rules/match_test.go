package rules

import (
	"testing"

	"github.com/nathoo/questcore/engine/state"
	"github.com/nathoo/questcore/types"
)

func matchTestDefs() (*types.State, *state.Defs) {
	defs := &state.Defs{
		Game: types.GameDef{Start: "hall"},
		Rooms: map[string]types.RoomDef{
			"hall": {ID: "hall"},
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
		},
	}
	return state.NewState(defs), defs
}

func TestMatchesIntent(t *testing.T) {
	s, defs := matchTestDefs()

	tests := []struct {
		name     string
		when     types.MatchCriteria
		verb     string
		objectID string
		targetID string
		want     bool
	}{
		{
			name: "verb matches",
			when: types.MatchCriteria{Verb: "take"},
			verb: "take", objectID: "rusty_key",
			want: true,
		},
		{
			name: "verb mismatch",
			when: types.MatchCriteria{Verb: "drop"},
			verb: "take", objectID: "rusty_key",
			want: false,
		},
		{
			name: "object matches specific ID",
			when: types.MatchCriteria{Verb: "take", Object: "rusty_key"},
			verb: "take", objectID: "rusty_key",
			want: true,
		},
		{
			name: "object mismatch",
			when: types.MatchCriteria{Verb: "take", Object: "golden_key"},
			verb: "take", objectID: "rusty_key",
			want: false,
		},
		{
			name: "target matches",
			when: types.MatchCriteria{Verb: "use", Object: "rusty_key", Target: "iron_door"},
			verb: "use", objectID: "rusty_key", targetID: "iron_door",
			want: true,
		},
		{
			name: "target mismatch",
			when: types.MatchCriteria{Verb: "use", Object: "rusty_key", Target: "wooden_door"},
			verb: "use", objectID: "rusty_key", targetID: "iron_door",
			want: false,
		},
		{
			name: "object_kind matches",
			when: types.MatchCriteria{Verb: "take", ObjectKind: "item"},
			verb: "take", objectID: "rusty_key",
			want: true,
		},
		{
			name: "object_kind mismatch",
			when: types.MatchCriteria{Verb: "take", ObjectKind: "npc"},
			verb: "take", objectID: "rusty_key",
			want: false,
		},
		{
			name: "object prop matches",
			when: types.MatchCriteria{Verb: "take", ObjectProp: map[string]any{"takeable": true}},
			verb: "take", objectID: "rusty_key",
			want: true,
		},
		{
			name: "object prop mismatch",
			when: types.MatchCriteria{Verb: "take", ObjectProp: map[string]any{"takeable": false}},
			verb: "take", objectID: "rusty_key",
			want: false,
		},
		{
			name: "target prop matches",
			when: types.MatchCriteria{Verb: "use", TargetProp: map[string]any{"locked": true}},
			verb: "use", objectID: "rusty_key", targetID: "iron_door",
			want: true,
		},
		{
			name: "verb only, no object required",
			when: types.MatchCriteria{Verb: "look"},
			verb: "look",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchesIntent(tt.when, tt.verb, tt.objectID, tt.targetID, s, defs)
			if got != tt.want {
				t.Errorf("MatchesIntent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSpecificity(t *testing.T) {
	tests := []struct {
		name string
		rule types.RuleDef
		want int
	}{
		{
			name: "verb only",
			rule: types.RuleDef{When: types.MatchCriteria{Verb: "take"}},
			want: 0,
		},
		{
			name: "verb + object",
			rule: types.RuleDef{When: types.MatchCriteria{Verb: "take", Object: "key"}},
			want: 2,
		},
		{
			name: "verb + target",
			rule: types.RuleDef{When: types.MatchCriteria{Verb: "use", Target: "door"}},
			want: 4,
		},
		{
			name: "verb + object + target",
			rule: types.RuleDef{When: types.MatchCriteria{Verb: "use", Object: "key", Target: "door"}},
			want: 6,
		},
		{
			name: "verb + object + props",
			rule: types.RuleDef{When: types.MatchCriteria{Verb: "take", Object: "key", ObjectProp: map[string]any{"shiny": true}}},
			want: 3,
		},
		{
			name: "verb + object + target + props",
			rule: types.RuleDef{When: types.MatchCriteria{Verb: "use", Object: "key", Target: "door", TargetProp: map[string]any{"locked": true}}},
			want: 7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Specificity(tt.rule)
			if got != tt.want {
				t.Errorf("Specificity() = %d, want %d", got, tt.want)
			}
		})
	}
}
