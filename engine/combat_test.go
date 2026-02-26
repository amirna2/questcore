package engine

import (
	"testing"

	"github.com/nathoo/questcore/engine/state"
	"github.com/nathoo/questcore/types"
)

func TestDamageCalc_Basic(t *testing.T) {
	// Fixed seed: we know what roll(6) produces.
	rng := NewRNG(42)

	// First roll with seed 42 on a d6.
	damage, roll := DamageCalc(5, 2, false, rng)

	// damage = max(1, roll + 5 - 2)
	expectedDamage := roll + 5 - 2
	if expectedDamage < 1 {
		expectedDamage = 1
	}
	if damage != expectedDamage {
		t.Errorf("expected damage %d, got %d (roll=%d)", expectedDamage, damage, roll)
	}
	if roll < 1 || roll > 6 {
		t.Errorf("roll out of range: %d", roll)
	}
}

func TestDamageCalc_MinimumOne(t *testing.T) {
	// High defense, low attack → should still be at least 1.
	rng := NewRNG(1)
	for i := 0; i < 100; i++ {
		damage, _ := DamageCalc(0, 20, false, rng)
		if damage < 1 {
			t.Fatalf("damage should be at least 1, got %d", damage)
		}
	}
}

func TestDamageCalc_DefendBonus(t *testing.T) {
	// Same seed, compare with and without defend.
	rng1 := NewRNG(42)
	rng2 := NewRNG(42)

	damageNormal, roll1 := DamageCalc(5, 2, false, rng1)
	damageDefend, roll2 := DamageCalc(5, 2, true, rng2)

	if roll1 != roll2 {
		t.Fatalf("same seed should produce same roll: %d vs %d", roll1, roll2)
	}

	// defend adds +2 to defense, so damage should be 2 less (clamped to 1).
	expectedDiff := 2
	if damageNormal-damageDefend != expectedDiff && damageDefend != 1 {
		t.Errorf("defend should reduce damage by 2: normal=%d, defend=%d", damageNormal, damageDefend)
	}
}

func TestDamageCalc_Deterministic(t *testing.T) {
	rng1 := NewRNG(99)
	rng2 := NewRNG(99)

	for i := 0; i < 50; i++ {
		d1, r1 := DamageCalc(4, 1, false, rng1)
		d2, r2 := DamageCalc(4, 1, false, rng2)
		if d1 != d2 || r1 != r2 {
			t.Fatalf("iteration %d: results differ: (%d,%d) vs (%d,%d)", i, d1, r1, d2, r2)
		}
	}
}

func combatDefs() *state.Defs {
	return &state.Defs{
		Game: types.GameDef{
			Start:       "cave",
			PlayerStats: map[string]int{"hp": 20, "max_hp": 20, "attack": 5, "defense": 2},
		},
		Rooms: map[string]types.RoomDef{
			"cave": {ID: "cave", Description: "A dark cave."},
			"hall": {ID: "hall", Description: "A grand hall.", Exits: map[string]string{"north": "cave"}},
		},
		Entities: map[string]types.EntityDef{
			"goblin": {
				ID:   "goblin",
				Kind: "enemy",
				Props: map[string]any{
					"name": "Cave Goblin", "location": "cave",
					"hp": 12, "max_hp": 12, "attack": 4, "defense": 1,
					"alive": true,
					"behavior": []types.BehaviorEntry{
						{Action: "attack", Weight: 70},
						{Action: "defend", Weight: 20},
						{Action: "flee", Weight: 10},
					},
				},
			},
			"skeleton": {
				ID:   "skeleton",
				Kind: "enemy",
				Props: map[string]any{
					"name": "Skeleton", "location": "hall",
					"hp": 8, "max_hp": 8, "attack": 3, "defense": 0,
					"alive": true,
					// No behavior — should default to attack-only.
				},
			},
		},
	}
}

func TestEnemyTurn_WeightedSelect(t *testing.T) {
	defs := combatDefs()
	s := state.NewState(defs)
	s.Combat = types.CombatState{Active: true, EnemyID: "goblin"}
	rng := NewRNG(42)

	// Run multiple turns and check distribution.
	counts := map[string]int{}
	for i := 0; i < 1000; i++ {
		intent := EnemyTurn(s, defs, rng)
		counts[intent.Verb]++
	}

	// Expect roughly 70% attack, 20% defend, 10% flee.
	if counts["attack"] < 600 || counts["attack"] > 800 {
		t.Errorf("expected ~700 attacks, got %d", counts["attack"])
	}
	if counts["defend"] < 100 || counts["defend"] > 300 {
		t.Errorf("expected ~200 defends, got %d", counts["defend"])
	}
	if counts["flee"] < 20 || counts["flee"] > 180 {
		t.Errorf("expected ~100 flees, got %d", counts["flee"])
	}
}

func TestEnemyTurn_NoBehavior_DefaultsToAttack(t *testing.T) {
	defs := combatDefs()
	s := state.NewState(defs)
	s.Combat = types.CombatState{Active: true, EnemyID: "skeleton"}
	rng := NewRNG(42)

	for i := 0; i < 10; i++ {
		intent := EnemyTurn(s, defs, rng)
		if intent.Verb != "attack" {
			t.Errorf("expected attack for enemy with no behavior, got %q", intent.Verb)
		}
	}
}

func TestEnemyTurn_Deterministic(t *testing.T) {
	defs := combatDefs()
	s := state.NewState(defs)
	s.Combat = types.CombatState{Active: true, EnemyID: "goblin"}

	rng1 := NewRNG(42)
	rng2 := NewRNG(42)

	for i := 0; i < 20; i++ {
		i1 := EnemyTurn(s, defs, rng1)
		i2 := EnemyTurn(s, defs, rng2)
		if i1.Verb != i2.Verb {
			t.Fatalf("turn %d: %q != %q", i, i1.Verb, i2.Verb)
		}
	}
}

func TestIsCombatVerb(t *testing.T) {
	tests := []struct {
		verb string
		want bool
	}{
		{"attack", true},
		{"defend", true},
		{"flee", true},
		{"use", true},
		{"inventory", true},
		{"look", true},
		{"go", false},
		{"take", false},
		{"talk", false},
		{"examine", false},
		{"drop", false},
	}
	for _, tt := range tests {
		if got := isCombatVerb(tt.verb); got != tt.want {
			t.Errorf("isCombatVerb(%q) = %v, want %v", tt.verb, got, tt.want)
		}
	}
}
