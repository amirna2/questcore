package engine

import (
	"fmt"

	"github.com/nathoo/questcore/engine/state"
	"github.com/nathoo/questcore/types"
)

// combatVerbs are the commands allowed during combat.
var combatVerbs = map[string]bool{
	"attack":    true,
	"defend":    true,
	"flee":      true,
	"use":       true,
	"inventory": true,
	"look":      true,
}

// isCombatVerb returns true if the verb is allowed during combat.
func isCombatVerb(verb string) bool {
	return combatVerbs[verb]
}

// DamageCalc computes damage: max(1, roll(1d6) + attack - defense).
// If defending, defense gets +2 bonus. Returns (damage, dieRoll).
func DamageCalc(attackerAttack, defenderDefense int, defending bool, rng *RNG) (damage, roll int) {
	roll = rng.Roll(6)
	def := defenderDefense
	if defending {
		def += 2
	}
	damage = roll + attackerAttack - def
	if damage < 1 {
		damage = 1
	}
	return damage, roll
}

// EnemyTurn selects an action for the enemy based on weighted behavior.
// Returns an Intent for the enemy's action.
func EnemyTurn(s *types.State, defs *state.Defs, rng *RNG) types.Intent {
	enemyID := s.Combat.EnemyID
	behavior := getEnemyBehavior(defs, enemyID)

	if len(behavior) == 0 {
		// No behavior defined — default to attack.
		return types.Intent{Verb: "attack"}
	}

	weights := make([]int, len(behavior))
	for i, b := range behavior {
		weights[i] = b.Weight
	}

	idx := rng.WeightedSelect(weights)
	return types.Intent{Verb: behavior[idx].Action}
}

// getEnemyBehavior retrieves the behavior table from entity props.
func getEnemyBehavior(defs *state.Defs, enemyID string) []types.BehaviorEntry {
	def, ok := defs.Entities[enemyID]
	if !ok {
		return nil
	}
	behavior, ok := def.Props["behavior"]
	if !ok {
		return nil
	}
	if b, ok := behavior.([]types.BehaviorEntry); ok {
		return b
	}
	return nil
}

// defaultCombatAttack produces effects for a default attack action.
// actor is "player" or an enemy entity ID.
func (e *Engine) defaultCombatAttack(actor string) ([]types.Effect, []string) {
	var attackerID, defenderID string
	if actor == "player" {
		attackerID = "player"
		defenderID = e.State.Combat.EnemyID
	} else {
		attackerID = actor
		defenderID = "player"
	}

	attackStat, _ := state.GetStat(e.State, e.Defs, attackerID, "attack")
	defenseStat, _ := state.GetStat(e.State, e.Defs, defenderID, "defense")

	// Check if defender is defending this round.
	defending := false
	if defenderID == "player" {
		defending = e.State.Combat.Defending
	} else {
		if v, ok := state.GetEntityProp(e.State, e.Defs, defenderID, "defending"); ok {
			defending, _ = v.(bool)
		}
	}

	damage, roll := DamageCalc(attackStat, defenseStat, defending, e.RNG)

	attackerName := e.combatantName(attackerID)
	defenderName := e.combatantName(defenderID)

	var output []string
	if actor == "player" {
		output = append(output, fmt.Sprintf("You strike the %s!", defenderName))
	} else {
		output = append(output, fmt.Sprintf("The %s attacks you!", attackerName))
	}

	defDisplay := defenseStat
	if defending {
		defDisplay += 2
	}
	output = append(output, fmt.Sprintf("  Roll: 1d6+%d → [%d]+%d = %d vs defense %d → %d damage",
		attackStat, roll, attackStat, roll+attackStat, defDisplay, damage))

	effs := []types.Effect{
		{Type: "damage", Params: map[string]any{"target": defenderID, "amount": damage}},
	}

	return effs, output
}

// defaultCombatDefend produces effects for a default defend action.
func (e *Engine) defaultCombatDefend(actor string) ([]types.Effect, []string) {
	if actor == "player" {
		e.State.Combat.Defending = true
		return nil, []string{"You brace yourself. (+2 defense this round)"}
	}
	// Enemy defending.
	enemyID := actor
	return []types.Effect{
		{Type: "set_prop", Params: map[string]any{"entity": enemyID, "prop": "defending", "value": true}},
	}, []string{fmt.Sprintf("The %s braces for your attack.", e.combatantName(enemyID))}
}

// defaultCombatFlee handles flee attempts. On 4+: escape. On fail: enemy free attack.
func (e *Engine) defaultCombatFlee(actor string) ([]types.Effect, []string) {
	roll := e.RNG.Roll(6)

	if actor == "player" {
		if roll >= 4 {
			// Escape successful.
			prevRoom := e.State.Combat.PreviousLocation
			if prevRoom == "" {
				prevRoom = e.State.Player.Location
			}
			effs := []types.Effect{
				{Type: "end_combat"},
				{Type: "move_player", Params: map[string]any{"room": prevRoom}},
			}
			output := []string{
				fmt.Sprintf("You turn and run! Roll: 1d6 → [%d] — you escape!", roll),
			}
			return effs, output
		}
		// Flee failed — enemy gets a free attack.
		output := []string{
			fmt.Sprintf("You try to run but can't escape! Roll: 1d6 → [%d]", roll),
		}
		return nil, output
	}

	// Enemy flee.
	enemyID := actor
	enemyName := e.combatantName(enemyID)
	if roll >= 4 {
		effs := []types.Effect{
			{Type: "end_combat"},
			{Type: "move_entity", Params: map[string]any{"entity": enemyID, "room": ""}},
		}
		return effs, []string{fmt.Sprintf("The %s turns and flees! Roll: 1d6 → [%d]", enemyName, roll)}
	}
	return nil, []string{fmt.Sprintf("The %s tries to flee but fails! Roll: 1d6 → [%d]", enemyName, roll)}
}

// ProcessLoot rolls for each item in the enemy's loot table and produces
// effects for successful drops (give_item) and gold (inc_counter).
func ProcessLoot(s *types.State, defs *state.Defs, enemyID string, rng *RNG) ([]types.Effect, []string) {
	def, ok := defs.Entities[enemyID]
	if !ok {
		return nil, nil
	}

	var effs []types.Effect
	var output []string

	// Roll for each loot item.
	if lootItems, ok := def.Props["loot_items"].([]types.LootEntry); ok {
		for _, item := range lootItems {
			roll := rng.Roll(100)
			if roll <= item.Chance {
				name := item.ItemID
				if ent, ok := defs.Entities[item.ItemID]; ok {
					if n, ok := ent.Props["name"].(string); ok {
						name = n
					}
				}
				effs = append(effs, types.Effect{
					Type:   "give_item",
					Params: map[string]any{"item": item.ItemID},
				})
				output = append(output, fmt.Sprintf("You found: %s!", name))
			}
		}
	}

	// Gold drop.
	if gold, ok := def.Props["loot_gold"]; ok {
		if g, ok := gold.(int); ok && g > 0 {
			effs = append(effs, types.Effect{
				Type:   "inc_counter",
				Params: map[string]any{"counter": "gold", "amount": g},
			})
			output = append(output, fmt.Sprintf("You found %d gold.", g))
		}
	}

	return effs, output
}

// combatantName returns the display name for a combatant.
func (e *Engine) combatantName(id string) string {
	if id == "player" {
		return "You"
	}
	return e.entityName(id)
}
