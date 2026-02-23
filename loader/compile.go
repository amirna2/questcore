// Package loader loads Lua game content into Go structs at compile time.
// The Lua VM is discarded after loading — zero Lua at runtime.
package loader

import (
	"fmt"
	"sort"

	"github.com/nathoo/questcore/engine/state"
	"github.com/nathoo/questcore/types"
	lua "github.com/yuin/gopher-lua"
)

// rawRoom holds a room table before compilation.
type rawRoom struct {
	id    string
	table *lua.LTable
}

// rawEntity holds an entity table before compilation.
type rawEntity struct {
	id    string
	kind  string
	table *lua.LTable
}

// rawRule holds a rule before compilation.
type rawRule struct {
	id         string
	when       *lua.LTable
	conditions *lua.LTable // may be nil
	then       *lua.LTable
	scope      string
	order      int
}

// rawHandler holds an event handler before compilation.
type rawHandler struct {
	eventType string
	table     *lua.LTable
}

// getString returns a string field from a Lua table, or "" if missing.
func getString(tbl *lua.LTable, key string) string {
	v := tbl.RawGetString(key)
	if s, ok := v.(lua.LString); ok {
		return string(s)
	}
	return ""
}

// getBool returns a bool field from a Lua table, or the default if missing.
func getBool(tbl *lua.LTable, key string, def bool) bool {
	v := tbl.RawGetString(key)
	if b, ok := v.(lua.LBool); ok {
		return bool(b)
	}
	return def
}

// getNumber returns a numeric field from a Lua table, or 0 if missing.
func getNumber(tbl *lua.LTable, key string) float64 {
	v := tbl.RawGetString(key)
	if n, ok := v.(lua.LNumber); ok {
		return float64(n)
	}
	return 0
}

// getInt returns an int field from a Lua table, or 0 if missing.
func getInt(tbl *lua.LTable, key string) int {
	return int(getNumber(tbl, key))
}

// getTable returns a table field from a Lua table, or nil if missing.
func getTable(tbl *lua.LTable, key string) *lua.LTable {
	v := tbl.RawGetString(key)
	if t, ok := v.(*lua.LTable); ok {
		return t
	}
	return nil
}

// toGoValue converts a Lua value to a Go value recursively.
func toGoValue(v lua.LValue) any {
	switch val := v.(type) {
	case lua.LBool:
		return bool(val)
	case lua.LNumber:
		f := float64(val)
		if f == float64(int(f)) {
			return int(f)
		}
		return f
	case *lua.LNilType:
		return nil
	case lua.LString:
		return string(val)
	case *lua.LTable:
		// Check if it's an array (sequential integer keys starting at 1).
		maxN := val.MaxN()
		if maxN > 0 {
			arr := make([]any, 0, maxN)
			for i := 1; i <= maxN; i++ {
				arr = append(arr, toGoValue(val.RawGetInt(i)))
			}
			return arr
		}
		// Otherwise treat as map.
		m := map[string]any{}
		val.ForEach(func(k, v lua.LValue) {
			if ks, ok := k.(lua.LString); ok {
				m[string(ks)] = toGoValue(v)
			}
		})
		return m
	default:
		return nil
	}
}

// tableToStringMap converts a Lua table to a map[string]string.
func tableToStringMap(tbl *lua.LTable) map[string]string {
	if tbl == nil {
		return nil
	}
	m := map[string]string{}
	tbl.ForEach(func(k, v lua.LValue) {
		if ks, ok := k.(lua.LString); ok {
			if vs, ok := v.(lua.LString); ok {
				m[string(ks)] = string(vs)
			}
		}
	})
	return m
}

// tableToAnyMap converts a Lua table to a map[string]any.
func tableToAnyMap(tbl *lua.LTable) map[string]any {
	if tbl == nil {
		return nil
	}
	m := map[string]any{}
	tbl.ForEach(func(k, v lua.LValue) {
		if ks, ok := k.(lua.LString); ok {
			m[string(ks)] = toGoValue(v)
		}
	})
	return m
}

// compile converts all collected Lua data into a Defs struct.
func compile(coll *collector) (*state.Defs, error) {
	defs := &state.Defs{
		Rooms:    map[string]types.RoomDef{},
		Entities: map[string]types.EntityDef{},
	}

	// Game.
	if coll.game == nil {
		return nil, fmt.Errorf("no Game{} definition found")
	}
	defs.Game = compileGame(coll.game)

	// Rooms — track which rules are scoped to each room.
	for _, raw := range coll.rooms {
		room, scopedIDs, err := compileRoom(raw)
		if err != nil {
			return nil, fmt.Errorf("compiling room %s: %w", raw.id, err)
		}
		defs.Rooms[room.ID] = room
		markScopedRules(coll, scopedIDs, "room:"+raw.id)
	}

	// Entities — track which rules are scoped to each entity.
	for _, raw := range coll.entities {
		entity, scopedIDs, err := compileEntity(raw)
		if err != nil {
			return nil, fmt.Errorf("compiling entity %s: %w", raw.id, err)
		}
		defs.Entities[entity.ID] = entity
		markScopedRules(coll, scopedIDs, "entity:"+raw.id)
	}

	// Rules.
	for i := range coll.rules {
		rule, err := compileRule(coll.rules[i])
		if err != nil {
			return nil, fmt.Errorf("compiling rule %s: %w", coll.rules[i].id, err)
		}
		switch {
		case rule.Scope == "global":
			defs.GlobalRules = append(defs.GlobalRules, rule)
		case len(rule.Scope) > 5 && rule.Scope[:5] == "room:":
			roomID := rule.Scope[5:]
			if r, ok := defs.Rooms[roomID]; ok {
				r.Rules = append(r.Rules, rule)
				defs.Rooms[roomID] = r
			}
		case len(rule.Scope) > 7 && rule.Scope[:7] == "entity:":
			entityID := rule.Scope[7:]
			if e, ok := defs.Entities[entityID]; ok {
				e.Rules = append(e.Rules, rule)
				defs.Entities[entityID] = e
			}
		}
	}

	// Handlers.
	for _, raw := range coll.handlers {
		handler, err := compileHandler(raw)
		if err != nil {
			return nil, fmt.Errorf("compiling handler: %w", err)
		}
		defs.Handlers = append(defs.Handlers, handler)
	}

	return defs, nil
}

func compileGame(tbl *lua.LTable) types.GameDef {
	return types.GameDef{
		Title:   getString(tbl, "title"),
		Author:  getString(tbl, "author"),
		Version: getString(tbl, "version"),
		Start:   getString(tbl, "start"),
		Intro:   getString(tbl, "intro"),
	}
}

// compileRoom compiles a raw room into a RoomDef and returns rule IDs scoped to it.
func compileRoom(raw rawRoom) (types.RoomDef, []string, error) {
	tbl := raw.table
	room := types.RoomDef{
		ID:          raw.id,
		Description: getString(tbl, "description"),
		Exits:       tableToStringMap(getTable(tbl, "exits")),
		Fallbacks:   tableToStringMap(getTable(tbl, "fallbacks")),
	}

	// Collect scoped rule IDs from the rules field.
	var scopedIDs []string
	if rulesTable := getTable(tbl, "rules"); rulesTable != nil {
		rulesTable.ForEach(func(_, v lua.LValue) {
			if marker, ok := v.(*lua.LTable); ok {
				id := getString(marker, "__rule_id")
				if id != "" {
					scopedIDs = append(scopedIDs, id)
				}
			}
		})
	}

	return room, scopedIDs, nil
}

// compileEntity compiles a raw entity into an EntityDef and returns rule IDs scoped to it.
func compileEntity(raw rawEntity) (types.EntityDef, []string, error) {
	tbl := raw.table
	entity := types.EntityDef{
		ID:    raw.id,
		Kind:  raw.kind,
		Props: map[string]any{},
	}

	// Special fields that don't go into Props.
	skip := map[string]bool{
		"rules": true, "topics": true,
	}

	// All non-special fields go into Props.
	tbl.ForEach(func(k, v lua.LValue) {
		if ks, ok := k.(lua.LString); ok {
			key := string(ks)
			if !skip[key] {
				entity.Props[key] = toGoValue(v)
			}
		}
	})

	// Default: items are takeable unless explicitly set.
	if raw.kind == "item" {
		if _, ok := entity.Props["takeable"]; !ok {
			entity.Props["takeable"] = true
		}
	}

	// Topics for NPCs.
	if topicsTbl := getTable(tbl, "topics"); topicsTbl != nil {
		entity.Topics = compileTopics(topicsTbl)
	}

	// Collect scoped rule IDs.
	var scopedIDs []string
	if rulesTable := getTable(tbl, "rules"); rulesTable != nil {
		rulesTable.ForEach(func(_, v lua.LValue) {
			if marker, ok := v.(*lua.LTable); ok {
				id := getString(marker, "__rule_id")
				if id != "" {
					scopedIDs = append(scopedIDs, id)
				}
			}
		})
	}

	return entity, scopedIDs, nil
}

func compileTopics(tbl *lua.LTable) map[string]types.TopicDef {
	topics := map[string]types.TopicDef{}
	tbl.ForEach(func(k, v lua.LValue) {
		key, ok := k.(lua.LString)
		if !ok {
			return
		}
		topicTbl, ok := v.(*lua.LTable)
		if !ok {
			return
		}
		topic := types.TopicDef{
			Text: getString(topicTbl, "text"),
		}
		if reqTbl := getTable(topicTbl, "requires"); reqTbl != nil {
			topic.Requires = compileConditions(reqTbl)
		}
		if effTbl := getTable(topicTbl, "effects"); effTbl != nil {
			topic.Effects = compileEffects(effTbl)
		}
		topics[string(key)] = topic
	})
	return topics
}

func compileRule(raw rawRule) (types.RuleDef, error) {
	rule := types.RuleDef{
		ID:          raw.id,
		Scope:       raw.scope,
		When:        compileMatchCriteria(raw.when),
		Effects:     compileEffects(raw.then),
		SourceOrder: raw.order,
	}
	if raw.conditions != nil {
		rule.Conditions = compileConditions(raw.conditions)
	}
	// Check for priority in the When table.
	rule.Priority = getInt(raw.when, "priority")
	return rule, nil
}

func compileMatchCriteria(tbl *lua.LTable) types.MatchCriteria {
	mc := types.MatchCriteria{
		Verb:       getString(tbl, "verb"),
		Object:     getString(tbl, "object"),
		Target:     getString(tbl, "target"),
		ObjectKind: getString(tbl, "object_kind"),
	}
	if tp := getTable(tbl, "target_prop"); tp != nil {
		mc.TargetProp = tableToAnyMap(tp)
	}
	if op := getTable(tbl, "object_prop"); op != nil {
		mc.ObjectProp = tableToAnyMap(op)
	}
	return mc
}

func compileConditions(tbl *lua.LTable) []types.Condition {
	var conditions []types.Condition
	tbl.ForEach(func(k, v lua.LValue) {
		// Only process integer-keyed entries (array elements).
		if _, ok := k.(lua.LNumber); !ok {
			return
		}
		if condTbl, ok := v.(*lua.LTable); ok {
			conditions = append(conditions, compileCondition(condTbl))
		}
	})
	return conditions
}

func compileCondition(tbl *lua.LTable) types.Condition {
	condType := getString(tbl, "type")

	if condType == "not" {
		innerTbl := getTable(tbl, "inner")
		if innerTbl != nil {
			inner := compileCondition(innerTbl)
			return types.Condition{
				Type:   "not",
				Negate: true,
				Inner:  &inner,
			}
		}
	}

	params := map[string]any{}
	tbl.ForEach(func(k, v lua.LValue) {
		if ks, ok := k.(lua.LString); ok {
			key := string(ks)
			if key != "type" {
				params[key] = toGoValue(v)
			}
		}
	})

	return types.Condition{
		Type:   condType,
		Params: params,
	}
}

func compileEffects(tbl *lua.LTable) []types.Effect {
	var effects []types.Effect
	tbl.ForEach(func(k, v lua.LValue) {
		if _, ok := k.(lua.LNumber); !ok {
			return
		}
		if effTbl, ok := v.(*lua.LTable); ok {
			effects = append(effects, compileEffect(effTbl))
		}
	})
	return effects
}

func compileEffect(tbl *lua.LTable) types.Effect {
	effType := getString(tbl, "type")
	params := map[string]any{}
	tbl.ForEach(func(k, v lua.LValue) {
		if ks, ok := k.(lua.LString); ok {
			key := string(ks)
			if key != "type" {
				params[key] = toGoValue(v)
			}
		}
	})
	return types.Effect{
		Type:   effType,
		Params: params,
	}
}

func compileHandler(raw rawHandler) (types.EventHandler, error) {
	handler := types.EventHandler{
		EventType: raw.eventType,
	}

	// The handler table has conditions and effects.
	if condTbl := getTable(raw.table, "conditions"); condTbl != nil {
		handler.Conditions = compileConditions(condTbl)
	}
	if effTbl := getTable(raw.table, "effects"); effTbl != nil {
		handler.Effects = compileEffects(effTbl)
	}

	return handler, nil
}

// markScopedRules updates raw rules in the collector to set their scope.
func markScopedRules(coll *collector, ruleIDs []string, scope string) {
	idSet := map[string]bool{}
	for _, id := range ruleIDs {
		idSet[id] = true
	}
	for i := range coll.rules {
		if idSet[coll.rules[i].id] {
			coll.rules[i].scope = scope
		}
	}
}

// sortedLuaFiles returns .lua files in a directory, with game.lua first
// and the rest sorted alphabetically.
func sortedLuaFiles(files []string) []string {
	var gameFile string
	var others []string
	for _, f := range files {
		if f == "game.lua" {
			gameFile = f
		} else {
			others = append(others, f)
		}
	}
	sort.Strings(others)
	if gameFile != "" {
		return append([]string{gameFile}, others...)
	}
	return others
}
