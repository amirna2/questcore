package rules

import (
	"testing"

	"github.com/nathoo/questcore/engine/state"
	"github.com/nathoo/questcore/types"
)

func condTestState() (*types.State, *state.Defs) {
	defs := &state.Defs{
		Game: types.GameDef{Start: "hall"},
		Rooms: map[string]types.RoomDef{
			"hall": {ID: "hall"},
		},
		Entities: map[string]types.EntityDef{
			"door": {
				ID:   "door",
				Kind: "entity",
				Props: map[string]any{
					"locked":   true,
					"location": "hall",
				},
			},
		},
	}
	s := state.NewState(defs)
	s.Player.Inventory = []string{"rusty_key"}
	s.Flags["quest_started"] = true
	s.Counters["score"] = 50
	return s, defs
}

func TestEvalCondition(t *testing.T) {
	s, defs := condTestState()

	tests := []struct {
		name string
		cond types.Condition
		want bool
	}{
		{
			name: "has_item: player has item",
			cond: types.Condition{Type: "has_item", Params: map[string]any{"item": "rusty_key"}},
			want: true,
		},
		{
			name: "has_item: player lacks item",
			cond: types.Condition{Type: "has_item", Params: map[string]any{"item": "sword"}},
			want: false,
		},
		{
			name: "flag_set: flag is true",
			cond: types.Condition{Type: "flag_set", Params: map[string]any{"flag": "quest_started"}},
			want: true,
		},
		{
			name: "flag_set: flag is unset",
			cond: types.Condition{Type: "flag_set", Params: map[string]any{"flag": "door_open"}},
			want: false,
		},
		{
			name: "flag_not: flag is unset",
			cond: types.Condition{Type: "flag_not", Params: map[string]any{"flag": "door_open"}},
			want: true,
		},
		{
			name: "flag_not: flag is true",
			cond: types.Condition{Type: "flag_not", Params: map[string]any{"flag": "quest_started"}},
			want: false,
		},
		{
			name: "flag_is: matches value",
			cond: types.Condition{Type: "flag_is", Params: map[string]any{"flag": "quest_started", "value": true}},
			want: true,
		},
		{
			name: "flag_is: does not match",
			cond: types.Condition{Type: "flag_is", Params: map[string]any{"flag": "quest_started", "value": false}},
			want: false,
		},
		{
			name: "counter_gt: passes",
			cond: types.Condition{Type: "counter_gt", Params: map[string]any{"counter": "score", "value": 10}},
			want: true,
		},
		{
			name: "counter_gt: fails (equal)",
			cond: types.Condition{Type: "counter_gt", Params: map[string]any{"counter": "score", "value": 50}},
			want: false,
		},
		{
			name: "counter_gt: fails (greater)",
			cond: types.Condition{Type: "counter_gt", Params: map[string]any{"counter": "score", "value": 100}},
			want: false,
		},
		{
			name: "counter_lt: passes",
			cond: types.Condition{Type: "counter_lt", Params: map[string]any{"counter": "score", "value": 100}},
			want: true,
		},
		{
			name: "counter_lt: fails",
			cond: types.Condition{Type: "counter_lt", Params: map[string]any{"counter": "score", "value": 10}},
			want: false,
		},
		{
			name: "in_room: matches",
			cond: types.Condition{Type: "in_room", Params: map[string]any{"room": "hall"}},
			want: true,
		},
		{
			name: "in_room: does not match",
			cond: types.Condition{Type: "in_room", Params: map[string]any{"room": "entrance"}},
			want: false,
		},
		{
			name: "prop_is: matches",
			cond: types.Condition{Type: "prop_is", Params: map[string]any{"entity": "door", "prop": "locked", "value": true}},
			want: true,
		},
		{
			name: "prop_is: does not match",
			cond: types.Condition{Type: "prop_is", Params: map[string]any{"entity": "door", "prop": "locked", "value": false}},
			want: false,
		},
		{
			name: "not: negates true → false",
			cond: types.Condition{
				Type:  "not",
				Inner: &types.Condition{Type: "has_item", Params: map[string]any{"item": "rusty_key"}},
			},
			want: false,
		},
		{
			name: "not: negates false → true",
			cond: types.Condition{
				Type:  "not",
				Inner: &types.Condition{Type: "has_item", Params: map[string]any{"item": "sword"}},
			},
			want: true,
		},
		{
			name: "unknown condition type: false",
			cond: types.Condition{Type: "bogus"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvalCondition(tt.cond, s, defs)
			if got != tt.want {
				t.Errorf("EvalCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvalAllConditions_AllPass(t *testing.T) {
	s, defs := condTestState()
	conds := []types.Condition{
		{Type: "has_item", Params: map[string]any{"item": "rusty_key"}},
		{Type: "flag_set", Params: map[string]any{"flag": "quest_started"}},
		{Type: "in_room", Params: map[string]any{"room": "hall"}},
	}
	if !EvalAllConditions(conds, s, defs) {
		t.Error("expected all conditions to pass")
	}
}

func TestEvalAllConditions_OneFails(t *testing.T) {
	s, defs := condTestState()
	conds := []types.Condition{
		{Type: "has_item", Params: map[string]any{"item": "rusty_key"}},
		{Type: "has_item", Params: map[string]any{"item": "sword"}}, // fails
		{Type: "in_room", Params: map[string]any{"room": "hall"}},
	}
	if EvalAllConditions(conds, s, defs) {
		t.Error("expected conditions to fail")
	}
}

func TestEvalAllConditions_Empty(t *testing.T) {
	s, defs := condTestState()
	if !EvalAllConditions(nil, s, defs) {
		t.Error("expected empty conditions to pass")
	}
}

// --- Combat condition tests ---

func combatCondTestState() (*types.State, *state.Defs) {
	defs := &state.Defs{
		Game: types.GameDef{
			Start:       "cave",
			PlayerStats: map[string]int{"hp": 20, "max_hp": 20, "attack": 5, "defense": 2},
		},
		Rooms: map[string]types.RoomDef{
			"cave": {ID: "cave"},
		},
		Entities: map[string]types.EntityDef{
			"goblin": {
				ID:   "goblin",
				Kind: "enemy",
				Props: map[string]any{
					"hp": 12, "max_hp": 12, "attack": 4, "defense": 1,
				},
			},
		},
	}
	s := state.NewState(defs)
	s.Combat.Active = true
	s.Combat.EnemyID = "goblin"
	return s, defs
}

func TestCombatConditions(t *testing.T) {
	s, defs := combatCondTestState()

	tests := []struct {
		name string
		cond types.Condition
		want bool
	}{
		{
			name: "in_combat: active",
			cond: types.Condition{Type: "in_combat"},
			want: true,
		},
		{
			name: "in_combat_with: correct enemy",
			cond: types.Condition{Type: "in_combat_with", Params: map[string]any{"entity": "goblin"}},
			want: true,
		},
		{
			name: "in_combat_with: wrong enemy",
			cond: types.Condition{Type: "in_combat_with", Params: map[string]any{"entity": "dragon"}},
			want: false,
		},
		{
			name: "stat_gt: player hp > 10",
			cond: types.Condition{Type: "stat_gt", Params: map[string]any{"entity": "player", "stat": "hp", "value": 10}},
			want: true,
		},
		{
			name: "stat_gt: player hp > 20 (not greater)",
			cond: types.Condition{Type: "stat_gt", Params: map[string]any{"entity": "player", "stat": "hp", "value": 20}},
			want: false,
		},
		{
			name: "stat_lt: goblin hp < 20",
			cond: types.Condition{Type: "stat_lt", Params: map[string]any{"entity": "goblin", "stat": "hp", "value": 20}},
			want: true,
		},
		{
			name: "stat_lt: goblin hp < 5 (not less)",
			cond: types.Condition{Type: "stat_lt", Params: map[string]any{"entity": "goblin", "stat": "hp", "value": 5}},
			want: false,
		},
		{
			name: "stat_gt: missing stat returns false",
			cond: types.Condition{Type: "stat_gt", Params: map[string]any{"entity": "goblin", "stat": "magic", "value": 0}},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvalCondition(tt.cond, s, defs)
			if got != tt.want {
				t.Errorf("EvalCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCombatConditions_NotInCombat(t *testing.T) {
	s, defs := combatCondTestState()
	s.Combat.Active = false

	if EvalCondition(types.Condition{Type: "in_combat"}, s, defs) {
		t.Error("expected in_combat to be false when not in combat")
	}
	if EvalCondition(types.Condition{Type: "in_combat_with", Params: map[string]any{"entity": "goblin"}}, s, defs) {
		t.Error("expected in_combat_with to be false when not in combat")
	}
}
