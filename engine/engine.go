// Package engine provides the Step() orchestrator that wires together
// parsing, resolution, rules, effects, and events into a single turn.
package engine

import (
	"fmt"
	"sort"
	"strings"

	"github.com/nathoo/questcore/engine/dialogue"
	"github.com/nathoo/questcore/engine/effects"
	"github.com/nathoo/questcore/engine/events"
	"github.com/nathoo/questcore/engine/parser"
	"github.com/nathoo/questcore/engine/resolve"
	"github.com/nathoo/questcore/engine/rules"
	"github.com/nathoo/questcore/engine/state"
	"github.com/nathoo/questcore/types"
)

// Engine holds the game definitions and mutable state.
type Engine struct {
	Defs  *state.Defs
	State *types.State
	RNG   *RNG
}

// New creates a new engine from definitions.
func New(defs *state.Defs) *Engine {
	s := state.NewState(defs)
	return &Engine{
		Defs:  defs,
		State: s,
		RNG:   NewRNG(s.RNGSeed),
	}
}

// RestoreRNG re-creates the RNG from seed and advances to the saved position.
func (e *Engine) RestoreRNG(seed int64, position int64) {
	e.RNG = RestoreRNG(seed, position)
}

// Step processes one player command and returns the result.
func (e *Engine) Step(input string) types.Result {
	var result types.Result

	// 0. Game over — block all gameplay commands.
	if state.GetFlag(e.State, "game_over") {
		result.Output = append(result.Output, "Game over. Use /load to restore a save or /quit to exit.")
		return result
	}

	// 1. Parse input.
	intent := parser.Parse(input)

	// 2. Log the command.
	e.State.CommandLog = append(e.State.CommandLog, input)

	// 3. Empty input.
	if intent.Verb == "" {
		result.Output = append(result.Output, "What do you want to do?")
		return result
	}

	// 3a. Combat mode: rewrite "go" → "flee" and restrict commands.
	if state.InCombat(e.State) {
		if intent.Verb == "go" {
			intent.Verb = "flee"
			intent.Object = ""
		}
		if !isCombatVerb(intent.Verb) {
			result.Output = append(result.Output, "You're in the middle of a fight! (attack, defend, use <item>, flee)")
			return result
		}
	}

	// 4. Determine resolution strategy based on verb.
	var objectID, targetID string
	var resolveErr error

	switch intent.Verb {
	case "go":
		// Direction is the object, no entity resolution needed.
		objectID = intent.Object

	case "inventory", "wait":
		// No resolution needed.

	case "attack":
		// During combat, target is implicit (the combat enemy).
		if state.InCombat(e.State) {
			objectID = e.State.Combat.EnemyID
		} else if intent.Object != "" {
			objectID, targetID, resolveErr = e.resolveEntities(intent)
		}

	case "defend", "flee":
		// No resolution needed.

	case "talk":
		// Resolve only the NPC (object), not the topic (target).
		if intent.Object != "" {
			res, err := resolve.Resolve(e.State, e.Defs, types.Intent{Verb: "talk", Object: intent.Object})
			if err != nil {
				resolveErr = err
			} else {
				objectID = res.ObjectID
			}
		}

	case "look":
		if intent.Object != "" {
			// "look <thing>" → resolve entity.
			objectID, targetID, resolveErr = e.resolveEntities(intent)
		}

	default:
		// Resolve entities for all other verbs.
		objectID, targetID, resolveErr = e.resolveEntities(intent)
	}

	// 5. If resolution failed, try rules with the raw name before giving up.
	// This allows rules for scenery nouns (e.g. "push wall", "examine throne")
	// that aren't defined as entities but have rules attached.
	if resolveErr != nil {
		if objectID == "" && intent.Object != "" {
			objectID = intent.Object
		}
		if targetID == "" && intent.Target != "" {
			targetID = intent.Target
		}
	}

	// 6. Run rules pipeline.
	effs, matched := rules.Evaluate(e.State, e.Defs, intent, objectID, targetID)

	// 7. If a rule matched, the resolution failure doesn't matter.
	if matched {
		resolveErr = nil
	}

	// 7a. No rule matched AND resolution failed → scenery fallback or error.
	if !matched && resolveErr != nil {
		if msg := e.sceneryFallback(intent); msg != "" {
			result.Output = append(result.Output, msg)
		} else {
			result.Output = append(result.Output, resolveErr.Error())
		}
		e.State.TurnCount++
		return result
	}

	// 7b. If matched → use rule effects. Otherwise → built-in or combat behavior.
	if !matched {
		if state.InCombat(e.State) {
			// Default combat behavior.
			combatEffs, combatOut := e.defaultCombatBehavior(intent, "player")
			effs = combatEffs
			result.Output = append(result.Output, combatOut...)
		} else {
			builtinEffs, builtinOut := e.builtinBehavior(intent, objectID)
			if builtinOut != nil || builtinEffs != nil {
				// Built-in handled this verb. Use its output instead of fallback.
				effs = builtinEffs
				result.Output = append(result.Output, builtinOut...)
			}
			// If built-in didn't handle it either, fall through with fallback effs.
		}
	}

	// 8. Apply effects.
	ctx := effects.Context{Verb: intent.Verb, ObjectID: objectID, TargetID: targetID, Actor: "player"}
	evts, output := effects.Apply(e.State, e.Defs, effs, ctx)
	result.Effects = append(result.Effects, effs...)
	result.Events = append(result.Events, evts...)
	result.Output = append(result.Output, output...)

	// 9. Dispatch events (single pass).
	eventEffs := events.Dispatch(evts, e.State, e.Defs)

	// 10. Apply event effects (events NOT re-dispatched).
	if len(eventEffs) > 0 {
		evts2, output2 := effects.Apply(e.State, e.Defs, eventEffs, ctx)
		result.Effects = append(result.Effects, eventEffs...)
		result.Events = append(result.Events, evts2...)
		result.Output = append(result.Output, output2...)
	}

	// 11. Enemy turn (if still in combat after player's action).
	if state.InCombat(e.State) {
		enemyResult := e.runEnemyTurn()
		result.Effects = append(result.Effects, enemyResult.Effects...)
		result.Events = append(result.Events, enemyResult.Events...)
		result.Output = append(result.Output, enemyResult.Output...)
	}

	// 12. End-of-round cleanup.
	if state.InCombat(e.State) {
		e.State.Combat.RoundCount++
		e.State.Combat.Defending = false
		// Clear enemy defending flag.
		enemyID := e.State.Combat.EnemyID
		if es, ok := e.State.Entities[enemyID]; ok {
			if _, hasDefending := es.Props["defending"]; hasDefending {
				es.Props["defending"] = false
				e.State.Entities[enemyID] = es
			}
		}
	}

	// 13. Track RNG position for save/load.
	e.State.RNGPosition = e.RNG.Position()

	// 14. Increment turn count.
	e.State.TurnCount++

	return result
}

// runEnemyTurn executes the enemy's turn through the same pipeline.
func (e *Engine) runEnemyTurn() types.Result {
	var result types.Result
	enemyID := e.State.Combat.EnemyID

	// Select enemy action.
	enemyIntent := EnemyTurn(e.State, e.Defs, e.RNG)

	// Try rules pipeline with enemy as actor.
	effs, matched := rules.Evaluate(e.State, e.Defs, enemyIntent, "", "")

	if !matched {
		// Use default combat behavior for enemy.
		combatEffs, combatOut := e.defaultCombatBehavior(enemyIntent, enemyID)
		effs = combatEffs
		result.Output = append(result.Output, combatOut...)
	}

	// Apply enemy effects.
	ctx := effects.Context{Verb: enemyIntent.Verb, Actor: enemyID}
	evts, output := effects.Apply(e.State, e.Defs, effs, ctx)
	result.Effects = append(result.Effects, effs...)
	result.Events = append(result.Events, evts...)
	result.Output = append(result.Output, output...)

	// Dispatch events from enemy turn.
	eventEffs := events.Dispatch(evts, e.State, e.Defs)
	if len(eventEffs) > 0 {
		evts2, output2 := effects.Apply(e.State, e.Defs, eventEffs, ctx)
		result.Effects = append(result.Effects, eventEffs...)
		result.Events = append(result.Events, evts2...)
		result.Output = append(result.Output, output2...)
	}

	return result
}

// defaultCombatBehavior routes combat verbs to their default implementations.
func (e *Engine) defaultCombatBehavior(intent types.Intent, actor string) ([]types.Effect, []string) {
	switch intent.Verb {
	case "attack":
		return e.defaultCombatAttack(actor)
	case "defend":
		return e.defaultCombatDefend(actor)
	case "flee":
		return e.defaultCombatFlee(actor)
	default:
		return nil, nil
	}
}

// resolveEntities resolves intent object/target names to entity IDs.
func (e *Engine) resolveEntities(intent types.Intent) (objectID, targetID string, err error) {
	res, err := resolve.Resolve(e.State, e.Defs, intent)
	if err != nil {
		return "", "", err
	}
	return res.ObjectID, res.TargetID, nil
}

// builtinBehavior provides default verb handling when no rule matched.
// Returns effects to apply and direct output text.
// Returns (nil, nil) if the verb is not a recognized built-in.
func (e *Engine) builtinBehavior(intent types.Intent, objectID string) ([]types.Effect, []string) {
	switch intent.Verb {
	case "go":
		return e.builtinGo(objectID)
	case "look":
		if objectID == "" {
			return e.builtinLook()
		}
		return nil, nil // look with object falls through to fallback
	case "inventory":
		return e.builtinInventory()
	case "examine", "read":
		return e.builtinExamine(objectID)
	case "take":
		return e.builtinTake(objectID)
	case "drop":
		return e.builtinDrop(objectID)
	case "talk":
		return e.builtinTalk(intent, objectID)
	case "wait":
		return nil, []string{"Time passes."}
	default:
		return nil, nil
	}
}

func (e *Engine) builtinGo(direction string) ([]types.Effect, []string) {
	if direction == "" {
		return nil, []string{"Go where?"}
	}

	exits := state.RoomExits(e.State, e.Defs, e.State.Player.Location)
	target, ok := exits[direction]
	if !ok {
		return nil, []string{"You can't go that way."}
	}

	effs := []types.Effect{
		{Type: "move_player", Params: map[string]any{"room": target}},
	}
	return effs, e.describeRoom(target)
}

func (e *Engine) builtinLook() ([]types.Effect, []string) {
	return nil, e.describeRoom(e.State.Player.Location)
}

func (e *Engine) builtinInventory() ([]types.Effect, []string) {
	inv := e.State.Player.Inventory
	if len(inv) == 0 {
		return nil, []string{"You are carrying nothing."}
	}
	var names []string
	for _, id := range inv {
		names = append(names, e.entityName(id))
	}
	return nil, []string{"You are carrying: " + strings.Join(names, ", ") + "."}
}

func (e *Engine) builtinExamine(objectID string) ([]types.Effect, []string) {
	if objectID == "" {
		return nil, nil
	}
	desc, ok := state.GetEntityProp(e.State, e.Defs, objectID, "description")
	if !ok {
		return nil, []string{"You see nothing special about it."}
	}
	if s, ok := desc.(string); ok {
		return nil, []string{s}
	}
	return nil, []string{"You see nothing special about it."}
}

func (e *Engine) builtinTake(objectID string) ([]types.Effect, []string) {
	if objectID == "" {
		return nil, nil
	}
	takeable, _ := state.GetEntityProp(e.State, e.Defs, objectID, "takeable")
	if takeable != true {
		return nil, []string{"You can't take that."}
	}
	if state.HasItem(e.State, objectID) {
		return nil, []string{"You already have that."}
	}
	effs := []types.Effect{
		{Type: "give_item", Params: map[string]any{"item": objectID}},
	}
	return effs, []string{fmt.Sprintf("You take the %s.", e.entityName(objectID))}
}

func (e *Engine) builtinDrop(objectID string) ([]types.Effect, []string) {
	if objectID == "" {
		return nil, nil
	}
	if !state.HasItem(e.State, objectID) {
		return nil, []string{"You don't have that."}
	}
	effs := []types.Effect{
		{Type: "remove_item", Params: map[string]any{"item": objectID}},
		{Type: "move_entity", Params: map[string]any{"entity": objectID, "room": e.State.Player.Location}},
	}
	return effs, []string{fmt.Sprintf("You drop the %s.", e.entityName(objectID))}
}

func (e *Engine) builtinTalk(intent types.Intent, npcID string) ([]types.Effect, []string) {
	if npcID == "" {
		return nil, []string{"Talk to whom?"}
	}

	// Check entity has topics.
	ent, ok := e.Defs.Entities[npcID]
	if !ok || ent.Topics == nil || len(ent.Topics) == 0 {
		return nil, []string{"You can't talk to that."}
	}

	npcName := e.entityName(npcID)
	topicKey := intent.Target

	if topicKey != "" {
		// Player specified a topic.
		text, effs := dialogue.SelectTopic(npcID, topicKey, e.State, e.Defs)
		if text == "" {
			// Topic not found — hint at what's available.
			available := dialogue.AvailableTopics(npcID, e.State, e.Defs)
			if len(available) > 0 {
				sort.Strings(available)
				return nil, []string{fmt.Sprintf("%s has nothing to say about that. You could ask about: %s.", npcName, strings.Join(available, ", "))}
			}
			return nil, []string{fmt.Sprintf("%s has nothing to say right now.", npcName)}
		}
		return effs, []string{text}
	}

	// No topic specified — auto-play first available topic.
	available := dialogue.AvailableTopics(npcID, e.State, e.Defs)
	if len(available) == 0 {
		return nil, []string{fmt.Sprintf("%s has nothing to say right now.", npcName)}
	}

	// Pick first available (stable: sort for determinism).
	sort.Strings(available)
	text, effs := dialogue.SelectTopic(npcID, available[0], e.State, e.Defs)
	if text == "" {
		return nil, []string{fmt.Sprintf("%s has nothing to say right now.", npcName)}
	}
	return effs, []string{text}
}

// sceneryFallback checks if the object noun appears in descriptions the player
// can see: room description, visible entity descriptions, and inventory item
// descriptions. If so, it returns a generic response instead of "you don't see
// that here" — the player clearly sees it in the description text.
func (e *Engine) sceneryFallback(intent types.Intent) string {
	if intent.Object == "" {
		return ""
	}
	objLower := strings.ToLower(intent.Object)

	// Collect all descriptions the player can currently "see".
	var descriptions []string

	// Room description.
	if room, ok := e.Defs.Rooms[e.State.Player.Location]; ok {
		descriptions = append(descriptions, room.Description)
	}

	// Visible entities in room.
	for _, id := range state.EntitiesInRoom(e.State, e.Defs, e.State.Player.Location) {
		if desc, ok := state.GetEntityProp(e.State, e.Defs, id, "description"); ok {
			if s, ok := desc.(string); ok {
				descriptions = append(descriptions, s)
			}
		}
	}

	// Inventory items.
	for _, id := range e.State.Player.Inventory {
		if desc, ok := state.GetEntityProp(e.State, e.Defs, id, "description"); ok {
			if s, ok := desc.(string); ok {
				descriptions = append(descriptions, s)
			}
		}
	}

	for _, desc := range descriptions {
		descLower := strings.ToLower(desc)
		// Check full phrase match.
		if strings.Contains(descLower, objLower) {
			return e.sceneryMessage(intent.Verb, intent.Object)
		}
		// Check significant word match (4+ chars).
		for _, word := range strings.Fields(objLower) {
			if len(word) >= 4 && strings.Contains(descLower, word) {
				return e.sceneryMessage(intent.Verb, intent.Object)
			}
		}
	}

	return ""
}

func (e *Engine) sceneryMessage(verb, object string) string {
	switch verb {
	case "examine", "look":
		return fmt.Sprintf("You see nothing special about the %s.", object)
	case "take", "get":
		return fmt.Sprintf("You can't take the %s.", object)
	default:
		return fmt.Sprintf("You can't do anything useful with the %s.", object)
	}
}

// describeRoom produces the standard room description output.
func (e *Engine) describeRoom(roomID string) []string {
	room, ok := e.Defs.Rooms[roomID]
	if !ok {
		return []string{"You are somewhere unknown."}
	}

	var output []string
	output = append(output, room.Description)

	// List visible entities.
	entities := state.EntitiesInRoom(e.State, e.Defs, roomID)
	if len(entities) > 0 {
		sort.Strings(entities) // deterministic order
		var names []string
		for _, id := range entities {
			names = append(names, e.entityName(id))
		}
		output = append(output, "You see: "+strings.Join(names, ", ")+".")
	}

	// List exits.
	exits := state.RoomExits(e.State, e.Defs, roomID)
	if len(exits) > 0 {
		dirs := make([]string, 0, len(exits))
		for dir := range exits {
			dirs = append(dirs, dir)
		}
		sort.Strings(dirs) // deterministic order
		output = append(output, "Exits: "+strings.Join(dirs, ", ")+".")
	}

	return output
}

// entityName returns the display name of an entity.
func (e *Engine) entityName(entityID string) string {
	if name, ok := state.GetEntityProp(e.State, e.Defs, entityID, "name"); ok {
		if s, ok := name.(string); ok {
			return s
		}
	}
	return entityID
}
