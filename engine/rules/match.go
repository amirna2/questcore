package rules

import (
	"github.com/nathoo/questcore/engine/state"
	"github.com/nathoo/questcore/types"
)

// MatchesIntent checks if a rule's When criteria match the resolved intent.
func MatchesIntent(when types.MatchCriteria, verb, objectID, targetID string,
	s *types.State, defs *state.Defs) bool {
	// Verb is required and must match.
	if when.Verb != verb {
		return false
	}

	// If When specifies an object, it must match the resolved object.
	if when.Object != "" && when.Object != objectID {
		return false
	}

	// If When specifies a target, it must match the resolved target.
	if when.Target != "" && when.Target != targetID {
		return false
	}

	// If When specifies an object kind, the resolved object must be that kind.
	if when.ObjectKind != "" && objectID != "" {
		if def, ok := defs.Entities[objectID]; ok {
			if def.Kind != when.ObjectKind {
				return false
			}
		} else {
			return false
		}
	}

	// If When specifies object properties, they must all match.
	if len(when.ObjectProp) > 0 && objectID != "" {
		for prop, expected := range when.ObjectProp {
			actual, ok := state.GetEntityProp(s, defs, objectID, prop)
			if !ok || actual != expected {
				return false
			}
		}
	}

	// If When specifies target properties, they must all match.
	if len(when.TargetProp) > 0 && targetID != "" {
		for prop, expected := range when.TargetProp {
			actual, ok := state.GetEntityProp(s, defs, targetID, prop)
			if !ok || actual != expected {
				return false
			}
		}
	}

	return true
}

// Specificity returns a numeric score for ranking rules.
// Higher is more specific.
func Specificity(rule types.RuleDef) int {
	score := 0
	if rule.When.Target != "" {
		score += 4
	}
	if rule.When.Object != "" {
		score += 2
	}
	if len(rule.When.ObjectProp) > 0 || len(rule.When.TargetProp) > 0 {
		score += 1
	}
	return score
}
