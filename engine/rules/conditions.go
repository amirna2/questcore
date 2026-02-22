// Package rules implements the 6-step rules engine pipeline.
package rules

import (
	"github.com/nathoo/questcore/engine/state"
	"github.com/nathoo/questcore/types"
)

// EvalCondition evaluates a single condition against the current state.
func EvalCondition(c types.Condition, s *types.State, defs *state.Defs) bool {
	switch c.Type {
	case "has_item":
		item, _ := c.Params["item"].(string)
		return state.HasItem(s, item)

	case "flag_set":
		flag, _ := c.Params["flag"].(string)
		return state.GetFlag(s, flag)

	case "flag_not":
		flag, _ := c.Params["flag"].(string)
		return !state.GetFlag(s, flag)

	case "flag_is":
		flag, _ := c.Params["flag"].(string)
		value, _ := c.Params["value"].(bool)
		return state.GetFlag(s, flag) == value

	case "counter_gt":
		counter, _ := c.Params["counter"].(string)
		value := toInt(c.Params["value"])
		return state.GetCounter(s, counter) > value

	case "counter_lt":
		counter, _ := c.Params["counter"].(string)
		value := toInt(c.Params["value"])
		return state.GetCounter(s, counter) < value

	case "in_room":
		room, _ := c.Params["room"].(string)
		return state.PlayerLocation(s) == room

	case "prop_is":
		entity, _ := c.Params["entity"].(string)
		prop, _ := c.Params["prop"].(string)
		expected := c.Params["value"]
		actual, ok := state.GetEntityProp(s, defs, entity, prop)
		if !ok {
			return expected == nil
		}
		return actual == expected

	case "not":
		if c.Inner == nil {
			return true
		}
		return !EvalCondition(*c.Inner, s, defs)

	default:
		return false
	}
}

// EvalAllConditions returns true if all conditions pass (AND logic).
// An empty condition list is vacuously true.
func EvalAllConditions(conditions []types.Condition, s *types.State, defs *state.Defs) bool {
	for _, c := range conditions {
		if !EvalCondition(c, s, defs) {
			return false
		}
	}
	return true
}

// toInt converts an any value to int, handling float64 from JSON/Lua.
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
