// Package effects implements centralized state mutation via the Apply function.
// Every effect type is one atomic operation. No logic in effects.
package effects

import (
	"fmt"
	"strings"

	"github.com/nathoo/questcore/engine/state"
	"github.com/nathoo/questcore/types"
)

// Context carries the resolved intent context needed for template interpolation.
type Context struct {
	Verb     string
	ObjectID string
	TargetID string
	Actor    string // "player" or entity ID of the acting combatant
}

// Apply applies a list of effects to the game state, mutating it.
// Returns events emitted and output text collected.
func Apply(s *types.State, defs *state.Defs, effects []types.Effect, ctx Context) ([]types.Event, []string) {
	var events []types.Event
	var output []string

	for _, eff := range effects {
		switch eff.Type {
		case "say":
			text, _ := eff.Params["text"].(string)
			text = interpolate(text, s, defs, ctx)
			output = append(output, text)

		case "give_item":
			item, _ := eff.Params["item"].(string)
			item = resolveTemplate(item, ctx)
			s.Player.Inventory = append(s.Player.Inventory, item)
			// Remove from world by setting location to empty.
			ensureEntityState(s, item)
			es := s.Entities[item]
			es.Location = " " // sentinel: "nowhere" (non-empty to override base)
			s.Entities[item] = es
			events = append(events, types.Event{
				Type: "item_taken",
				Data: map[string]any{"item": item},
			})

		case "remove_item":
			item, _ := eff.Params["item"].(string)
			item = resolveTemplate(item, ctx)
			s.Player.Inventory = removeFromSlice(s.Player.Inventory, item)
			events = append(events, types.Event{
				Type: "item_dropped",
				Data: map[string]any{"item": item},
			})

		case "set_flag":
			flag, _ := eff.Params["flag"].(string)
			value, _ := eff.Params["value"].(bool)
			s.Flags[flag] = value
			events = append(events, types.Event{
				Type: "flag_changed",
				Data: map[string]any{"flag": flag, "value": value},
			})

		case "inc_counter":
			counter, _ := eff.Params["counter"].(string)
			amount := toInt(eff.Params["amount"])
			s.Counters[counter] += amount

		case "set_counter":
			counter, _ := eff.Params["counter"].(string)
			value := toInt(eff.Params["value"])
			s.Counters[counter] = value

		case "set_prop":
			entity, _ := eff.Params["entity"].(string)
			prop, _ := eff.Params["prop"].(string)
			value := eff.Params["value"]
			ensureEntityState(s, entity)
			es := s.Entities[entity]
			if es.Props == nil {
				es.Props = map[string]any{}
			}
			es.Props[prop] = value
			s.Entities[entity] = es

		case "move_entity":
			entity, _ := eff.Params["entity"].(string)
			room, _ := eff.Params["room"].(string)
			ensureEntityState(s, entity)
			es := s.Entities[entity]
			es.Location = room
			s.Entities[entity] = es
			events = append(events, types.Event{
				Type: "entity_moved",
				Data: map[string]any{"entity": entity, "room": room},
			})

		case "move_player":
			room, _ := eff.Params["room"].(string)
			s.Player.Location = room
			events = append(events, types.Event{
				Type: "room_entered",
				Data: map[string]any{"room": room},
			})

		case "open_exit":
			room, _ := eff.Params["room"].(string)
			direction, _ := eff.Params["direction"].(string)
			target, _ := eff.Params["target"].(string)
			key := "room:" + room
			ensureEntityState(s, key)
			es := s.Entities[key]
			if es.Props == nil {
				es.Props = map[string]any{}
			}
			es.Props["exit:"+direction] = target
			s.Entities[key] = es

		case "close_exit":
			room, _ := eff.Params["room"].(string)
			direction, _ := eff.Params["direction"].(string)
			key := "room:" + room
			ensureEntityState(s, key)
			es := s.Entities[key]
			if es.Props == nil {
				es.Props = map[string]any{}
			}
			es.Props["exit:"+direction] = ""
			s.Entities[key] = es

		case "emit_event":
			event, _ := eff.Params["event"].(string)
			events = append(events, types.Event{
				Type: event,
				Data: map[string]any{},
			})

		case "start_dialogue":
			// Stub — dialogue system is layer 9.
			npc, _ := eff.Params["npc"].(string)
			events = append(events, types.Event{
				Type: "dialogue_started",
				Data: map[string]any{"npc": npc},
			})

		case "start_combat":
			enemyID, _ := eff.Params["enemy"].(string)
			s.Combat.Active = true
			s.Combat.EnemyID = enemyID
			s.Combat.RoundCount = 0
			s.Combat.Defending = false
			s.Combat.PreviousLocation = s.Player.Location
			// Initialize enemy runtime stats from base def if not already set.
			initEnemyStats(s, defs, enemyID)
			events = append(events, types.Event{
				Type: "combat_started",
				Data: map[string]any{"enemy": enemyID},
			})

		case "end_combat":
			s.Combat = types.CombatState{}
			events = append(events, types.Event{
				Type: "combat_ended",
				Data: map[string]any{},
			})

		case "damage":
			target, _ := eff.Params["target"].(string)
			amount := toInt(eff.Params["amount"])
			remaining := applyDamage(s, defs, target, amount)
			events = append(events, types.Event{
				Type: "entity_damaged",
				Data: map[string]any{"target": target, "amount": amount, "remaining": remaining},
			})
			// Check for death.
			if remaining <= 0 {
				if target == "player" {
					enemyID := s.Combat.EnemyID // capture before clearing
					s.Flags["game_over"] = true
					s.Combat = types.CombatState{}
					events = append(events, types.Event{
						Type: "player_defeated",
						Data: map[string]any{"enemy": enemyID},
					})
				} else {
					// Enemy defeated.
					ensureEntityState(s, target)
					es := s.Entities[target]
					if es.Props == nil {
						es.Props = map[string]any{}
					}
					es.Props["alive"] = false
					s.Entities[target] = es
					// End combat when enemy is defeated.
					s.Combat = types.CombatState{}
					events = append(events, types.Event{
						Type: "enemy_defeated",
						Data: map[string]any{"enemy": target},
					})
					events = append(events, types.Event{
						Type: "combat_ended",
						Data: map[string]any{},
					})
				}
			}

		case "heal":
			target, _ := eff.Params["target"].(string)
			amount := toInt(eff.Params["amount"])
			current := applyHeal(s, defs, target, amount)
			events = append(events, types.Event{
				Type: "entity_healed",
				Data: map[string]any{"target": target, "amount": amount, "current": current},
			})

		case "set_stat":
			target, _ := eff.Params["target"].(string)
			stat, _ := eff.Params["stat"].(string)
			value := toInt(eff.Params["value"])
			state.SetStat(s, target, stat, value)

		case "stop":
			return events, output

		default:
			// Unknown effect type — ignore silently.
		}
	}

	return events, output
}

// interpolate replaces template variables in text.
func interpolate(text string, s *types.State, defs *state.Defs, ctx Context) string {
	r := strings.NewReplacer(
		"{verb}", ctx.Verb,
		"{object}", ctx.ObjectID,
		"{target}", ctx.TargetID,
		"{player.location}", s.Player.Location,
	)
	text = r.Replace(text)

	// {player.inventory} — formatted list.
	if strings.Contains(text, "{player.inventory}") {
		inv := formatInventory(s.Player.Inventory, defs)
		text = strings.ReplaceAll(text, "{player.inventory}", inv)
	}

	// {room.description}
	if strings.Contains(text, "{room.description}") {
		desc := ""
		if room, ok := defs.Rooms[s.Player.Location]; ok {
			desc = room.Description
		}
		text = strings.ReplaceAll(text, "{room.description}", desc)
	}

	// {object.name}, {object.description}
	text = replaceEntityProp(text, "{object.name}", ctx.ObjectID, "name", s, defs)
	text = replaceEntityProp(text, "{object.description}", ctx.ObjectID, "description", s, defs)

	// {target.name}
	text = replaceEntityProp(text, "{target.name}", ctx.TargetID, "name", s, defs)

	return text
}

// replaceEntityProp replaces a template variable with an entity property value.
func replaceEntityProp(text, placeholder, entityID, prop string, s *types.State, defs *state.Defs) string {
	if !strings.Contains(text, placeholder) {
		return text
	}
	val := ""
	if entityID != "" {
		if v, ok := state.GetEntityProp(s, defs, entityID, prop); ok {
			val = fmt.Sprintf("%v", v)
		}
	}
	return strings.ReplaceAll(text, placeholder, val)
}

// resolveTemplate handles {object} and {target} in effect params like GiveItem("{object}").
func resolveTemplate(s string, ctx Context) string {
	s = strings.ReplaceAll(s, "{object}", ctx.ObjectID)
	s = strings.ReplaceAll(s, "{target}", ctx.TargetID)
	return s
}

// formatInventory creates a human-readable inventory list.
func formatInventory(items []string, defs *state.Defs) string {
	if len(items) == 0 {
		return "You are carrying nothing."
	}
	var names []string
	for _, id := range items {
		name := id
		if def, ok := defs.Entities[id]; ok {
			if n, ok := def.Props["name"].(string); ok {
				name = n
			}
		}
		names = append(names, name)
	}
	return strings.Join(names, ", ")
}

func ensureEntityState(s *types.State, entityID string) {
	if _, ok := s.Entities[entityID]; !ok {
		s.Entities[entityID] = types.EntityState{}
	}
}

func removeFromSlice(slice []string, item string) []string {
	for i, v := range slice {
		if v == item {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

func toInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	case int64:
		return int(n)
	default:
		return 0
	}
}

// initEnemyStats copies base stats (hp, max_hp, attack, defense) into EntityState
// if they're not already set as runtime overrides.
func initEnemyStats(s *types.State, defs *state.Defs, enemyID string) {
	def, ok := defs.Entities[enemyID]
	if !ok {
		return
	}
	ensureEntityState(s, enemyID)
	es := s.Entities[enemyID]
	if es.Props == nil {
		es.Props = map[string]any{}
	}
	for _, stat := range []string{"hp", "max_hp", "attack", "defense"} {
		if _, exists := es.Props[stat]; !exists {
			if v, ok := def.Props[stat]; ok {
				es.Props[stat] = v
			}
		}
	}
	if _, exists := es.Props["alive"]; !exists {
		es.Props["alive"] = true
	}
	s.Entities[enemyID] = es
}

// applyDamage decrements the target's HP, clamping to 0. Returns remaining HP.
func applyDamage(s *types.State, defs *state.Defs, target string, amount int) int {
	hp, _ := state.GetStat(s, defs, target, "hp")
	hp -= amount
	if hp < 0 {
		hp = 0
	}
	state.SetStat(s, target, "hp", hp)
	return hp
}

// applyHeal increments the target's HP, clamping to max_hp. Returns current HP.
func applyHeal(s *types.State, defs *state.Defs, target string, amount int) int {
	hp, _ := state.GetStat(s, defs, target, "hp")
	maxHP, _ := state.GetStat(s, defs, target, "max_hp")
	hp += amount
	if maxHP > 0 && hp > maxHP {
		hp = maxHP
	}
	state.SetStat(s, target, "hp", hp)
	return hp
}
