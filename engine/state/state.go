// Package state manages the mutable game state and property lookups
// with override layering (runtime state overrides base definitions).
package state

import "github.com/nathoo/questcore/types"

// Defs holds the immutable game definitions loaded from Lua.
type Defs struct {
	Game        types.GameDef
	Rooms       map[string]types.RoomDef
	Entities    map[string]types.EntityDef
	GlobalRules []types.RuleDef
	Handlers    []types.EventHandler
}

// NewState creates a fresh game state from definitions.
func NewState(defs *Defs) *types.State {
	stats := map[string]int{}
	for k, v := range defs.Game.PlayerStats {
		stats[k] = v
	}
	return &types.State{
		Player: types.Player{
			Location:  defs.Game.Start,
			Inventory: []string{},
			Stats:     stats,
		},
		Entities:   map[string]types.EntityState{},
		Flags:      map[string]bool{},
		Counters:   map[string]int{},
		TurnCount:  0,
		RNGSeed:    0,
		CommandLog: []string{},
	}
}

// GetFlag returns the value of a flag. Unset flags return false.
func GetFlag(s *types.State, name string) bool {
	return s.Flags[name]
}

// GetCounter returns the value of a counter. Unset counters return 0.
func GetCounter(s *types.State, name string) int {
	return s.Counters[name]
}

// HasItem returns true if the player has the given item in inventory.
func HasItem(s *types.State, itemID string) bool {
	for _, id := range s.Player.Inventory {
		if id == itemID {
			return true
		}
	}
	return false
}

// PlayerLocation returns the player's current room ID.
func PlayerLocation(s *types.State) string {
	return s.Player.Location
}

// GetEntityProp returns a property value for an entity, checking
// runtime state overrides first, then falling back to the base definition.
// Returns the value and whether it was found.
func GetEntityProp(s *types.State, defs *Defs, entityID string, prop string) (any, bool) {
	// Check runtime state override first.
	if es, ok := s.Entities[entityID]; ok {
		if v, ok := es.Props[prop]; ok {
			return v, true
		}
	}
	// Fall back to base definition.
	if def, ok := defs.Entities[entityID]; ok {
		if v, ok := def.Props[prop]; ok {
			return v, true
		}
	}
	return nil, false
}

// EntityLocation returns the effective location of an entity, checking
// the runtime state override first, then the base definition.
func EntityLocation(s *types.State, defs *Defs, entityID string) string {
	if es, ok := s.Entities[entityID]; ok && es.Location != "" {
		return es.Location
	}
	if def, ok := defs.Entities[entityID]; ok {
		if loc, ok := def.Props["location"]; ok {
			if s, ok := loc.(string); ok {
				return s
			}
		}
	}
	return ""
}

// EntitiesInRoom returns the IDs of all entities whose effective location
// matches the given room ID.
func EntitiesInRoom(s *types.State, defs *Defs, roomID string) []string {
	var result []string
	for id := range defs.Entities {
		if EntityLocation(s, defs, id) == roomID {
			result = append(result, id)
		}
	}
	return result
}

// InCombat returns true if the player is currently in combat.
func InCombat(s *types.State) bool {
	return s.Combat.Active
}

// GetStat reads a stat for the given target. Target is "player" or an entity ID.
// For player: reads from s.Player.Stats.
// For entity: reads from EntityState.Props (runtime override), then EntityDef.Props (base).
// Returns the value and whether it was found.
func GetStat(s *types.State, defs *Defs, target string, stat string) (int, bool) {
	if target == "player" {
		v, ok := s.Player.Stats[stat]
		return v, ok
	}
	val, ok := GetEntityProp(s, defs, target, stat)
	if !ok {
		return 0, false
	}
	return toInt(val)
}

// SetStat writes a stat for the given target. Target is "player" or an entity ID.
func SetStat(s *types.State, target string, stat string, value int) {
	if target == "player" {
		if s.Player.Stats == nil {
			s.Player.Stats = map[string]int{}
		}
		s.Player.Stats[stat] = value
		return
	}
	es := s.Entities[target]
	if es.Props == nil {
		es.Props = map[string]any{}
	}
	es.Props[stat] = value
	s.Entities[target] = es
}

// toInt coerces a value to int. Handles int, float64 (from JSON/Lua), and int64.
func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case float64:
		return int(n), true
	case int64:
		return int(n), true
	default:
		return 0, false
	}
}

// RoomExits returns the effective exits for a room. Runtime exit overrides
// (from open_exit/close_exit effects) are layered on top of base exits.
func RoomExits(s *types.State, defs *Defs, roomID string) map[string]string {
	room, ok := defs.Rooms[roomID]
	if !ok {
		return nil
	}
	// Start with a copy of base exits.
	exits := make(map[string]string, len(room.Exits))
	for dir, target := range room.Exits {
		exits[dir] = target
	}
	// Apply runtime overrides stored as entity state props.
	// Convention: exit overrides stored as "exit:<direction>" props on room entity.
	if es, ok := s.Entities["room:"+roomID]; ok {
		for key, val := range es.Props {
			if len(key) > 5 && key[:5] == "exit:" {
				dir := key[5:]
				if target, ok := val.(string); ok {
					if target == "" {
						delete(exits, dir) // close_exit
					} else {
						exits[dir] = target // open_exit
					}
				}
			}
		}
	}
	return exits
}
