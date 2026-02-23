// Package types defines the shared data structures for the QuestCore engine.
// This package contains only type definitions — no logic, no methods.
package types

// Intent is the parsed representation of a player command.
type Intent struct {
	Verb   string
	Object string // optional
	Target string // optional
}

// Effect is a single atomic state mutation instruction.
type Effect struct {
	Type   string
	Params map[string]any
}

// Event is emitted after effects are applied.
type Event struct {
	Type string
	Data map[string]any
}

// Result is the output of a single game step.
type Result struct {
	Effects []Effect
	Events  []Event
	Output  []string
}

// MatchCriteria defines what intent a rule matches against.
type MatchCriteria struct {
	Verb       string
	Object     string         // specific entity ID
	Target     string         // specific entity ID
	ObjectKind string         // match by entity kind (e.g. "item")
	TargetProp map[string]any // target must have these props
	ObjectProp map[string]any // object must have these props
}

// Condition is a predicate that must be true for a rule to fire.
type Condition struct {
	Type   string         // "has_item", "flag_is", "flag_set", "flag_not", etc.
	Params map[string]any // condition-specific parameters
	Negate bool           // true if wrapped in Not()
	Inner  *Condition     // for Not(): the negated inner condition
}

// RuleDef is a single rule that maps an intent to effects.
type RuleDef struct {
	ID          string
	Scope       string // "room:<id>", "entity:<id>", "global"
	When        MatchCriteria
	Conditions  []Condition
	Effects     []Effect
	Priority    int
	SourceOrder int
}

// TopicDef defines a single dialogue topic for an NPC.
type TopicDef struct {
	Text     string
	Requires []Condition
	Effects  []Effect
}

// EntityDef is the base definition of a world entity (item, NPC, etc.).
type EntityDef struct {
	ID     string
	Kind   string              // "item", "npc", "entity", "room"
	Props  map[string]any      // base properties from Lua
	Rules  []RuleDef           // rules scoped to this entity
	Topics map[string]TopicDef // NPC topics (nil for non-NPCs)
}

// RoomDef is the base definition of a room.
type RoomDef struct {
	ID          string
	Description string
	Exits       map[string]string // direction → room_id
	Rules       []RuleDef
	Fallbacks   map[string]string // verb → custom failure text
}

// GameDef holds game metadata from Lua.
type GameDef struct {
	Title   string
	Author  string
	Version string
	Start   string // starting room ID
	Intro   string
}

// Player holds the player's runtime state.
type Player struct {
	Location  string
	Inventory []string
	Stats     map[string]int
}

// EntityState holds runtime overrides for an entity.
type EntityState struct {
	Location string         // overrides base location if non-empty
	Props    map[string]any // overrides base props
}

// State is the complete mutable game state.
type State struct {
	Player     Player
	Entities   map[string]EntityState // runtime property overrides
	Flags      map[string]bool
	Counters   map[string]int
	TurnCount  int
	RNGSeed    int64
	CommandLog []string
}

// EventHandler is a rule triggered by an event rather than a player command.
type EventHandler struct {
	EventType  string
	Conditions []Condition
	Effects    []Effect
}
