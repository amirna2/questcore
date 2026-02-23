package loader

import (
	"fmt"
	"os"
	"strings"

	"github.com/nathoo/questcore/engine/state"
	"github.com/nathoo/questcore/types"
)

// ValidationError collects all validation errors and warnings.
type ValidationError struct {
	Errors   []string
	Warnings []string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed with %d error(s):\n  %s",
		len(e.Errors), strings.Join(e.Errors, "\n  "))
}

// Known effect types.
var validEffectTypes = map[string]bool{
	"say":            true,
	"give_item":      true,
	"remove_item":    true,
	"set_flag":       true,
	"inc_counter":    true,
	"set_counter":    true,
	"set_prop":       true,
	"move_entity":    true,
	"move_player":    true,
	"open_exit":      true,
	"close_exit":     true,
	"emit_event":     true,
	"start_dialogue": true,
	"stop":           true,
}

// Known condition types.
var validConditionTypes = map[string]bool{
	"has_item":   true,
	"flag_set":   true,
	"flag_not":   true,
	"flag_is":    true,
	"in_room":    true,
	"prop_is":    true,
	"counter_gt": true,
	"counter_lt": true,
	"not":        true,
}

// validate checks the compiled defs for referential integrity and consistency.
func validate(defs *state.Defs) error {
	ve := &ValidationError{}

	// Game title required.
	if defs.Game.Title == "" {
		ve.Errors = append(ve.Errors, "Game.Title is required")
	}

	// Start room exists.
	if defs.Game.Start == "" {
		ve.Errors = append(ve.Errors, "Game.Start is required")
	} else if _, ok := defs.Rooms[defs.Game.Start]; !ok {
		ve.Errors = append(ve.Errors, fmt.Sprintf(
			"start room %q not found in defined rooms", defs.Game.Start))
	}

	// Exit targets valid.
	for roomID, room := range defs.Rooms {
		for dir, target := range room.Exits {
			if _, ok := defs.Rooms[target]; !ok {
				ve.Errors = append(ve.Errors, fmt.Sprintf(
					"room %q exit %q points to undefined room %q", roomID, dir, target))
			}
		}
		// Validate room rules.
		validateRules(room.Rules, defs, ve)
	}

	// Rule IDs unique across all scopes.
	ruleIDs := map[string]bool{}
	allRules := collectAllRules(defs)
	for _, rule := range allRules {
		if ruleIDs[rule.ID] {
			ve.Errors = append(ve.Errors, fmt.Sprintf(
				"duplicate rule ID %q", rule.ID))
		}
		ruleIDs[rule.ID] = true
	}

	// Validate global rules.
	validateRules(defs.GlobalRules, defs, ve)

	// Validate entity rules.
	for _, entity := range defs.Entities {
		validateRules(entity.Rules, defs, ve)

		// Validate topic conditions and effects.
		for _, topic := range entity.Topics {
			validateConditions(topic.Requires, defs, ve)
			validateEffects(topic.Effects, defs, ve)
		}
	}

	// Validate handlers.
	for _, handler := range defs.Handlers {
		validateConditions(handler.Conditions, defs, ve)
		validateEffects(handler.Effects, defs, ve)
	}

	// Warnings: dangling item locations.
	for entityID, entity := range defs.Entities {
		if loc, ok := entity.Props["location"].(string); ok && loc != "" {
			if _, ok := defs.Rooms[loc]; !ok {
				ve.Warnings = append(ve.Warnings, fmt.Sprintf(
					"entity %q location %q does not match any defined room", entityID, loc))
			}
		}
	}

	// Print warnings to stderr.
	for _, w := range ve.Warnings {
		fmt.Fprintf(os.Stderr, "warning: %s\n", w)
	}

	if len(ve.Errors) > 0 {
		return ve
	}
	return nil
}

func validateRules(rules []types.RuleDef, defs *state.Defs, ve *ValidationError) {
	for _, rule := range rules {
		validateConditions(rule.Conditions, defs, ve)
		validateEffects(rule.Effects, defs, ve)

		// Warn on unrecognized verbs in When.
		if rule.When.Verb != "" {
			verb := rule.When.Verb
			if !isKnownVerb(verb) {
				ve.Warnings = append(ve.Warnings, fmt.Sprintf(
					"rule %q uses unrecognized verb %q", rule.ID, verb))
			}
		}
	}
}

func validateConditions(conditions []types.Condition, defs *state.Defs, ve *ValidationError) {
	for _, cond := range conditions {
		if !validConditionTypes[cond.Type] {
			ve.Errors = append(ve.Errors, fmt.Sprintf(
				"unknown condition type %q", cond.Type))
		}

		// Check entity/room refs in conditions.
		switch cond.Type {
		case "has_item":
			if item, ok := cond.Params["item"].(string); ok && !isTemplate(item) {
				if _, ok := defs.Entities[item]; !ok {
					ve.Errors = append(ve.Errors, fmt.Sprintf(
						"condition has_item references undefined entity %q", item))
				}
			}
		case "in_room":
			if room, ok := cond.Params["room"].(string); ok && !isTemplate(room) {
				if _, ok := defs.Rooms[room]; !ok {
					ve.Errors = append(ve.Errors, fmt.Sprintf(
						"condition in_room references undefined room %q", room))
				}
			}
		case "prop_is":
			if entity, ok := cond.Params["entity"].(string); ok && !isTemplate(entity) {
				if _, ok := defs.Entities[entity]; !ok {
					ve.Errors = append(ve.Errors, fmt.Sprintf(
						"condition prop_is references undefined entity %q", entity))
				}
			}
		case "not":
			if cond.Inner != nil {
				validateConditions([]types.Condition{*cond.Inner}, defs, ve)
			}
		}
	}
}

func validateEffects(effects []types.Effect, defs *state.Defs, ve *ValidationError) {
	for _, eff := range effects {
		if !validEffectTypes[eff.Type] {
			ve.Errors = append(ve.Errors, fmt.Sprintf(
				"unknown effect type %q", eff.Type))
		}

		// Check entity/room refs in effects.
		switch eff.Type {
		case "give_item":
			if item, ok := eff.Params["item"].(string); ok && !isTemplate(item) {
				if _, ok := defs.Entities[item]; !ok {
					ve.Errors = append(ve.Errors, fmt.Sprintf(
						"effect give_item references undefined entity %q", item))
				}
			}
		case "remove_item":
			if item, ok := eff.Params["item"].(string); ok && !isTemplate(item) {
				if _, ok := defs.Entities[item]; !ok {
					ve.Errors = append(ve.Errors, fmt.Sprintf(
						"effect remove_item references undefined entity %q", item))
				}
			}
		case "set_prop":
			if entity, ok := eff.Params["entity"].(string); ok && !isTemplate(entity) {
				if _, ok := defs.Entities[entity]; !ok {
					ve.Errors = append(ve.Errors, fmt.Sprintf(
						"effect set_prop references undefined entity %q", entity))
				}
			}
		case "move_entity":
			if entity, ok := eff.Params["entity"].(string); ok && !isTemplate(entity) {
				if _, ok := defs.Entities[entity]; !ok {
					ve.Errors = append(ve.Errors, fmt.Sprintf(
						"effect move_entity references undefined entity %q", entity))
				}
			}
			if room, ok := eff.Params["room"].(string); ok && !isTemplate(room) {
				if _, ok := defs.Rooms[room]; !ok {
					ve.Errors = append(ve.Errors, fmt.Sprintf(
						"effect move_entity references undefined room %q", room))
				}
			}
		case "move_player":
			if room, ok := eff.Params["room"].(string); ok && !isTemplate(room) {
				if _, ok := defs.Rooms[room]; !ok {
					ve.Errors = append(ve.Errors, fmt.Sprintf(
						"effect move_player references undefined room %q", room))
				}
			}
		case "open_exit":
			if room, ok := eff.Params["room"].(string); ok && !isTemplate(room) {
				if _, ok := defs.Rooms[room]; !ok {
					ve.Errors = append(ve.Errors, fmt.Sprintf(
						"effect open_exit references undefined room %q", room))
				}
			}
			if target, ok := eff.Params["target"].(string); ok && !isTemplate(target) {
				if _, ok := defs.Rooms[target]; !ok {
					ve.Errors = append(ve.Errors, fmt.Sprintf(
						"effect open_exit target references undefined room %q", target))
				}
			}
		case "close_exit":
			if room, ok := eff.Params["room"].(string); ok && !isTemplate(room) {
				if _, ok := defs.Rooms[room]; !ok {
					ve.Errors = append(ve.Errors, fmt.Sprintf(
						"effect close_exit references undefined room %q", room))
				}
			}
		case "start_dialogue":
			if npc, ok := eff.Params["npc"].(string); ok && !isTemplate(npc) {
				if _, ok := defs.Entities[npc]; !ok {
					ve.Errors = append(ve.Errors, fmt.Sprintf(
						"effect start_dialogue references undefined entity %q", npc))
				}
			}
		}
	}
}

// collectAllRules gathers all rules from all scopes.
func collectAllRules(defs *state.Defs) []types.RuleDef {
	var all []types.RuleDef
	all = append(all, defs.GlobalRules...)
	for _, room := range defs.Rooms {
		all = append(all, room.Rules...)
	}
	for _, entity := range defs.Entities {
		all = append(all, entity.Rules...)
	}
	return all
}

// isTemplate returns true if the string contains a template variable.
func isTemplate(s string) bool {
	return strings.Contains(s, "{") && strings.Contains(s, "}")
}

// isKnownVerb returns true if the verb is recognized by the parser.
var knownVerbs = map[string]bool{
	"look": true, "examine": true, "take": true, "drop": true,
	"go": true, "use": true, "open": true, "close": true,
	"talk": true, "give": true, "push": true, "pull": true,
	"attack": true, "inventory": true, "wait": true,
	"read": true, "eat": true, "drink": true, "climb": true,
	"unlock": true, "lock": true, "search": true, "listen": true,
	"smell": true, "touch": true, "taste": true, "throw": true,
	"put": true, "ask": true, "tell": true, "show": true,
	"say": true, "move": true, "enter": true, "leave": true,
	"help": true, "save": true, "load": true, "quit": true,
	// Direction verbs.
	"north": true, "south": true, "east": true, "west": true,
	"northeast": true, "northwest": true, "southeast": true, "southwest": true,
	"up": true, "down": true,
}

func isKnownVerb(verb string) bool {
	return knownVerbs[verb]
}
