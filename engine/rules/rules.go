package rules

import (
	"sort"

	"github.com/nathoo/questcore/engine/state"
	"github.com/nathoo/questcore/types"
)

// Evaluate runs the full rules pipeline (steps 2-6) and returns the matched
// effects. The bool indicates whether a rule actually matched (true) vs.
// fallback was used (false). Step() uses this to decide built-in behavior.
// Step 1 (resolve) is handled by the resolve package before calling this.
func Evaluate(s *types.State, defs *state.Defs,
	intent types.Intent, objectID, targetID string) ([]types.Effect, bool) {

	// Step 2: Collect candidate rules in resolution order buckets.
	buckets := collect(s, defs, objectID, targetID)

	// Steps 3-5: Filter, rank, and select.
	for _, bucket := range buckets {
		if winner := filterRankSelect(bucket, s, defs, intent.Verb, objectID, targetID); winner != nil {
			// Step 6: Produce effects.
			return winner.Effects, true
		}
	}

	// No rule matched — produce fallback.
	return fallback(s, defs, intent.Verb, objectID), false
}

// collect gathers candidate rules in resolution order (DESIGN.md §6.6):
// 1. Room-local rules
// 2. Target entity rules
// 3. Object entity rules
// 4. Global rules
func collect(s *types.State, defs *state.Defs, objectID, targetID string) [][]types.RuleDef {
	var buckets [][]types.RuleDef

	// 1. Current room's rules.
	if room, ok := defs.Rooms[s.Player.Location]; ok && len(room.Rules) > 0 {
		buckets = append(buckets, room.Rules)
	}

	// 2. Target entity's rules.
	if targetID != "" {
		if ent, ok := defs.Entities[targetID]; ok && len(ent.Rules) > 0 {
			buckets = append(buckets, ent.Rules)
		}
	}

	// 3. Object entity's rules.
	if objectID != "" && objectID != targetID {
		if ent, ok := defs.Entities[objectID]; ok && len(ent.Rules) > 0 {
			buckets = append(buckets, ent.Rules)
		}
	}

	// 4. Global rules.
	if len(defs.GlobalRules) > 0 {
		buckets = append(buckets, defs.GlobalRules)
	}

	return buckets
}

// filterRankSelect filters a bucket of rules, ranks them, and returns the
// top-ranked matching rule, or nil if none match.
func filterRankSelect(rules []types.RuleDef, s *types.State, defs *state.Defs,
	verb, objectID, targetID string) *types.RuleDef {

	// Step 3: Filter — When match + conditions.
	var candidates []types.RuleDef
	for _, rule := range rules {
		if !MatchesIntent(rule.When, verb, objectID, targetID, s, defs) {
			continue
		}
		if !EvalAllConditions(rule.Conditions, s, defs) {
			continue
		}
		candidates = append(candidates, rule)
	}

	if len(candidates) == 0 {
		return nil
	}

	// Step 4: Rank — specificity (desc) → priority (desc) → source order (asc).
	sort.SliceStable(candidates, func(i, j int) bool {
		si, sj := Specificity(candidates[i]), Specificity(candidates[j])
		if si != sj {
			return si > sj
		}
		if candidates[i].Priority != candidates[j].Priority {
			return candidates[i].Priority > candidates[j].Priority
		}
		return candidates[i].SourceOrder < candidates[j].SourceOrder
	})

	// Step 5: Select first.
	return &candidates[0]
}

// fallback produces effects when no rule matched.
// Resolution: entity fallback → room fallback (verb) → room fallback (default) → global default.
func fallback(s *types.State, defs *state.Defs, verb, objectID string) []types.Effect {
	// 1. Entity fallback.
	if objectID != "" {
		if def, ok := defs.Entities[objectID]; ok {
			if fb, ok := def.Props["fallbacks"]; ok {
				if fbMap, ok := fb.(map[string]any); ok {
					if text, ok := fbMap[verb].(string); ok {
						return []types.Effect{sayEffect(text)}
					}
					if text, ok := fbMap["default"].(string); ok {
						return []types.Effect{sayEffect(text)}
					}
				}
			}
		}
	}

	// 2. Room fallback (verb-specific).
	if room, ok := defs.Rooms[s.Player.Location]; ok {
		if text, ok := room.Fallbacks[verb]; ok {
			return []types.Effect{sayEffect(text)}
		}
		// 3. Room fallback (default).
		if text, ok := room.Fallbacks["default"]; ok {
			return []types.Effect{sayEffect(text)}
		}
	}

	// 4. Global default.
	return []types.Effect{sayEffect("You can't do that.")}
}

func sayEffect(text string) types.Effect {
	return types.Effect{
		Type:   "say",
		Params: map[string]any{"text": text},
	}
}
