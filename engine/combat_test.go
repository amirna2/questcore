package engine

import (
	"strings"
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
					"loot_items": []types.LootEntry{
						{ItemID: "goblin_blade", Chance: 50},
					},
					"loot_gold": 5,
				},
			},
			"goblin_blade": {
				ID:   "goblin_blade",
				Kind: "item",
				Props: map[string]any{
					"name": "Rusty Goblin Blade", "location": "cave",
					"takeable": true,
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

// --- Integration tests: full combat through Step() ---

// combatEngine creates an engine with combat-ready defs and starts combat.
func combatEngine() *Engine {
	defs := combatDefs()
	eng := New(defs)
	// Start combat with the goblin.
	eng.State.Combat = types.CombatState{
		Active:           true,
		EnemyID:          "goblin",
		RoundCount:       0,
		PreviousLocation: "cave",
	}
	// Initialize enemy runtime stats.
	eng.State.Entities["goblin"] = types.EntityState{
		Props: map[string]any{
			"hp": 12, "max_hp": 12, "attack": 4, "defense": 1,
			"alive": true,
		},
	}
	return eng
}

func TestStep_CombatEndsOnEnemyDefeat(t *testing.T) {
	eng := combatEngine()
	// Set goblin HP to 1 so a single attack kills it.
	es := eng.State.Entities["goblin"]
	es.Props["hp"] = 1
	eng.State.Entities["goblin"] = es

	result := eng.Step("attack goblin")

	// Combat should have ended.
	if state.InCombat(eng.State) {
		t.Error("expected combat to end after defeating the enemy")
	}

	// Should have enemy_defeated event.
	foundDefeated := false
	foundEnded := false
	for _, e := range result.Events {
		if e.Type == "enemy_defeated" {
			foundDefeated = true
		}
		if e.Type == "combat_ended" {
			foundEnded = true
		}
	}
	if !foundDefeated {
		t.Error("expected enemy_defeated event")
	}
	if !foundEnded {
		t.Error("expected combat_ended event")
	}

	// Enemy should be marked dead.
	alive, _ := state.GetEntityProp(eng.State, eng.Defs, "goblin", "alive")
	if alive != false {
		t.Errorf("expected goblin alive=false, got %v", alive)
	}
}

func TestStep_NoEnemyTurnAfterDefeat(t *testing.T) {
	eng := combatEngine()
	// Set goblin HP to 1 so the player's attack kills it.
	es := eng.State.Entities["goblin"]
	es.Props["hp"] = 1
	eng.State.Entities["goblin"] = es

	result := eng.Step("attack goblin")

	// The enemy should NOT get a turn after being killed.
	// Check that no enemy attack output appears.
	for _, line := range result.Output {
		if contains(line, "attacks you") {
			t.Errorf("dead enemy should not attack, but got: %q", line)
		}
	}
}

func TestStep_EnemyGetsNormalTurnWhileAlive(t *testing.T) {
	eng := combatEngine()
	// Goblin has full HP — should survive the player's attack and get a turn.

	result := eng.Step("attack goblin")

	// Goblin should still be alive (12 HP, max single hit is 1d6+5=11).
	if !state.InCombat(eng.State) {
		// It's possible the goblin died in one hit with a max roll,
		// but with 12 HP and max damage 11, it shouldn't happen.
		t.Error("expected combat to still be active after one attack vs 12 HP enemy")
	}

	// The result should contain both player attack and enemy action output.
	if len(result.Output) < 2 {
		t.Errorf("expected at least 2 lines of output (player + enemy), got %d", len(result.Output))
	}
}

func TestStep_CombatBlocksNonCombatVerbs(t *testing.T) {
	eng := combatEngine()

	result := eng.Step("take sword")

	found := false
	for _, line := range result.Output {
		if contains(line, "middle of a fight") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected combat restriction message, got: %v", result.Output)
	}
}

func TestStep_AttackAfterCombatEnds(t *testing.T) {
	eng := combatEngine()
	// Kill the goblin.
	es := eng.State.Entities["goblin"]
	es.Props["hp"] = 1
	eng.State.Entities["goblin"] = es

	eng.Step("attack goblin")

	// Now try attacking again — should not be in combat.
	result := eng.Step("attack goblin")

	// Should not re-enter combat or deal damage.
	if state.InCombat(eng.State) {
		t.Error("should not be in combat after enemy is already dead")
	}

	// Should not contain attack output.
	for _, line := range result.Output {
		if contains(line, "You strike") {
			t.Errorf("should not be able to attack dead enemy, got: %q", line)
		}
	}
}

func TestStep_GoRewrittenToFleeDuringCombat(t *testing.T) {
	eng := combatEngine()

	result := eng.Step("go south")

	// "go" should be rewritten to "flee" during combat.
	foundFlee := false
	for _, line := range result.Output {
		if contains(line, "run") || contains(line, "escape") || contains(line, "flee") {
			foundFlee = true
		}
	}
	if !foundFlee {
		t.Errorf("expected flee output when using 'go' during combat, got: %v", result.Output)
	}
}

// --- ProcessLoot unit tests ---

func TestProcessLoot_Deterministic(t *testing.T) {
	defs := combatDefs()
	s := state.NewState(defs)

	// Run with two identical RNGs — should produce identical results.
	rng1 := NewRNG(42)
	rng2 := NewRNG(42)

	effs1, out1 := ProcessLoot(s, defs, "goblin", rng1)
	effs2, out2 := ProcessLoot(s, defs, "goblin", rng2)

	if len(effs1) != len(effs2) {
		t.Fatalf("effect counts differ: %d vs %d", len(effs1), len(effs2))
	}
	if len(out1) != len(out2) {
		t.Fatalf("output counts differ: %d vs %d", len(out1), len(out2))
	}
	for i := range out1 {
		if out1[i] != out2[i] {
			t.Errorf("output[%d] differs: %q vs %q", i, out1[i], out2[i])
		}
	}
}

func TestProcessLoot_GoldAlwaysDrops(t *testing.T) {
	defs := combatDefs()
	s := state.NewState(defs)
	rng := NewRNG(1)

	effs, output := ProcessLoot(s, defs, "goblin", rng)

	// Gold (5) should always drop regardless of roll.
	foundGold := false
	for _, eff := range effs {
		if eff.Type == "inc_counter" {
			if eff.Params["counter"] == "gold" && eff.Params["amount"] == 5 {
				foundGold = true
			}
		}
	}
	if !foundGold {
		t.Errorf("expected gold drop effect, got effects: %v", effs)
	}

	foundGoldMsg := false
	for _, line := range output {
		if contains(line, "5 gold") {
			foundGoldMsg = true
		}
	}
	if !foundGoldMsg {
		t.Errorf("expected gold message in output, got: %v", output)
	}
}

func TestProcessLoot_ItemDropRolled(t *testing.T) {
	defs := combatDefs()
	s := state.NewState(defs)

	// Run many trials — with 50% chance, we should see some drops and some misses.
	drops := 0
	trials := 100
	for i := 0; i < trials; i++ {
		rng := NewRNG(int64(i))
		effs, _ := ProcessLoot(s, defs, "goblin", rng)
		for _, eff := range effs {
			if eff.Type == "give_item" && eff.Params["item"] == "goblin_blade" {
				drops++
			}
		}
	}

	// With 50% chance over 100 trials, expect roughly 30-70 drops.
	if drops < 20 || drops > 80 {
		t.Errorf("expected ~50%% item drop rate, got %d/%d", drops, trials)
	}
}

func TestProcessLoot_NoLootTable(t *testing.T) {
	defs := combatDefs()
	s := state.NewState(defs)
	rng := NewRNG(42)

	// Skeleton has no loot.
	effs, output := ProcessLoot(s, defs, "skeleton", rng)

	if len(effs) != 0 {
		t.Errorf("expected no effects for enemy with no loot, got %v", effs)
	}
	if len(output) != 0 {
		t.Errorf("expected no output for enemy with no loot, got %v", output)
	}
}

func TestProcessLoot_UnknownEnemy(t *testing.T) {
	defs := combatDefs()
	s := state.NewState(defs)
	rng := NewRNG(42)

	effs, output := ProcessLoot(s, defs, "nonexistent", rng)

	if len(effs) != 0 || len(output) != 0 {
		t.Errorf("expected nil for unknown enemy, got effs=%v, output=%v", effs, output)
	}
}

// --- Loot integration test through Step() ---

func TestStep_LootDropsOnEnemyDefeat(t *testing.T) {
	eng := combatEngine()
	// Set goblin HP to 1 so one attack kills it.
	es := eng.State.Entities["goblin"]
	es.Props["hp"] = 1
	eng.State.Entities["goblin"] = es

	result := eng.Step("attack goblin")

	// Gold should always drop.
	if eng.State.Counters["gold"] != 5 {
		t.Errorf("expected 5 gold after defeating goblin, got %d", eng.State.Counters["gold"])
	}

	// Should see gold message in output.
	foundGoldMsg := false
	for _, line := range result.Output {
		if contains(line, "5 gold") {
			foundGoldMsg = true
		}
	}
	if !foundGoldMsg {
		t.Errorf("expected gold drop message in output, got: %v", result.Output)
	}
}

func TestStep_NoLootWhileEnemyAlive(t *testing.T) {
	eng := combatEngine()
	// Goblin at full HP — won't die this turn.

	eng.Step("attack goblin")

	// No gold should be awarded.
	if eng.State.Counters["gold"] != 0 {
		t.Errorf("expected 0 gold while enemy alive, got %d", eng.State.Counters["gold"])
	}
}

// contains checks if s contains substr (case-insensitive).
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
