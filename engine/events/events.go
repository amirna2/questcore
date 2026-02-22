// Package events implements single-pass event handler dispatch.
// Event handlers produce additional effects but do not recurse.
package events

import (
	"github.com/nathoo/questcore/engine/rules"
	"github.com/nathoo/questcore/engine/state"
	"github.com/nathoo/questcore/types"
)

// Dispatch runs event handlers against the emitted events. Single pass —
// no recursion. Returns additional effects produced by matching handlers.
func Dispatch(events []types.Event, s *types.State, defs *state.Defs) []types.Effect {
	var result []types.Effect

	for _, event := range events {
		for _, handler := range defs.Handlers {
			if handler.EventType != event.Type {
				continue
			}
			if !rules.EvalAllConditions(handler.Conditions, s, defs) {
				continue
			}
			result = append(result, handler.Effects...)
		}
	}

	return result
}
