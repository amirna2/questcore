// Package resolve maps entity names from parsed intents to entity IDs.
package resolve

import (
	"fmt"
	"strings"

	"github.com/nathoo/questcore/engine/state"
	"github.com/nathoo/questcore/types"
)

// Result holds the resolved entity IDs for an intent.
type Result struct {
	ObjectID string
	TargetID string
}

// AmbiguityError indicates multiple entities matched a name.
type AmbiguityError struct {
	Name       string
	Candidates []string
}

func (e *AmbiguityError) Error() string {
	names := strings.Join(e.Candidates, ", ")
	return fmt.Sprintf("which %s? (%s)", e.Name, names)
}

// NotFoundError indicates no entity matched a name.
type NotFoundError struct {
	Name string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("you don't see %q here", e.Name)
}

// Resolve maps object/target name strings from an intent to entity IDs.
func Resolve(s *types.State, defs *state.Defs, intent types.Intent) (Result, error) {
	var res Result
	var err error

	if intent.Object != "" {
		res.ObjectID, err = resolveName(s, defs, intent.Object)
		if err != nil {
			return res, err
		}
	}

	if intent.Target != "" {
		res.TargetID, err = resolveName(s, defs, intent.Target)
		if err != nil {
			return res, err
		}
	}

	return res, nil
}

// resolveName resolves a single name string to an entity ID.
func resolveName(s *types.State, defs *state.Defs, name string) (string, error) {
	// 1. Exact entity ID match.
	if _, ok := defs.Entities[name]; ok {
		return name, nil
	}

	// 2. Search by name property among visible entities.
	var matches []string
	nameLower := strings.ToLower(name)

	// Check entities in current room.
	for id, def := range defs.Entities {
		if !isVisible(s, defs, id) {
			continue
		}
		if matchesName(s, defs, id, def, nameLower) {
			matches = append(matches, id)
		}
	}

	// Check player inventory (items not in a room but carried).
	for _, itemID := range s.Player.Inventory {
		// Skip if already matched (could be visible in room and inventory).
		if containsStr(matches, itemID) {
			continue
		}
		if def, ok := defs.Entities[itemID]; ok {
			if matchesName(s, defs, itemID, def, nameLower) {
				matches = append(matches, itemID)
			}
		}
	}

	switch len(matches) {
	case 0:
		return "", &NotFoundError{Name: name}
	case 1:
		return matches[0], nil
	default:
		return "", &AmbiguityError{Name: name, Candidates: matches}
	}
}

// isVisible returns true if the entity is in the player's current room.
func isVisible(s *types.State, defs *state.Defs, entityID string) bool {
	loc := state.EntityLocation(s, defs, entityID)
	return loc == s.Player.Location
}

// matchesName checks if an entity's name property matches the query (case-insensitive).
// Supports exact match, word-based partial match, and entity ID match.
func matchesName(s *types.State, defs *state.Defs, id string, def types.EntityDef, nameLower string) bool {
	// Check runtime override for name first, then base prop.
	if nameVal, ok := state.GetEntityProp(s, defs, id, "name"); ok {
		if nameStr, ok := nameVal.(string); ok {
			entityNameLower := strings.ToLower(nameStr)
			// Exact match.
			if entityNameLower == nameLower {
				return true
			}
			// Word-based partial match: query matches any word in the name.
			// e.g. "key" matches "rusty key", "guard" matches "castle guard".
			for _, word := range strings.Fields(entityNameLower) {
				if word == nameLower {
					return true
				}
			}
		}
	}
	// Check entity ID (e.g. "rusty_key" matches "rusty_key").
	idLower := strings.ToLower(id)
	if idLower == nameLower {
		return true
	}
	// Underscore normalization: "rusty key" matches entity ID "rusty_key".
	if strings.ReplaceAll(nameLower, " ", "_") == idLower {
		return true
	}
	return false
}

func containsStr(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
