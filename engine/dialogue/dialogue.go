// Package dialogue implements the NPC topic system.
package dialogue

import (
	"github.com/nathoo/questcore/engine/rules"
	"github.com/nathoo/questcore/engine/state"
	"github.com/nathoo/questcore/types"
)

// AvailableTopics returns topic keys whose conditions are met.
func AvailableTopics(npcID string, s *types.State, defs *state.Defs) []string {
	ent, ok := defs.Entities[npcID]
	if !ok || ent.Topics == nil {
		return nil
	}

	var result []string
	for key, topic := range ent.Topics {
		if rules.EvalAllConditions(topic.Requires, s, defs) {
			result = append(result, key)
		}
	}
	return result
}

// SelectTopic returns the text and effects for a chosen topic.
// Returns empty text and nil effects if topic doesn't exist or conditions not met.
func SelectTopic(npcID, topicKey string, s *types.State, defs *state.Defs) (string, []types.Effect) {
	ent, ok := defs.Entities[npcID]
	if !ok || ent.Topics == nil {
		return "", nil
	}

	topic, ok := ent.Topics[topicKey]
	if !ok {
		return "", nil
	}

	if !rules.EvalAllConditions(topic.Requires, s, defs) {
		return "", nil
	}

	return topic.Text, topic.Effects
}
