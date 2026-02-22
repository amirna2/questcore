# QuestCore: Technical Design Document

**Engine:** Go
**Content:** Lua (sandboxed, tables-as-data, compile-time only)
**Saves:** JSON
**Architecture:** ECS-inspired (entity data + action systems + event triggers)

---

## 1. Architecture Overview

```
┌──────────────────────────────────────────────┐
│         Game Content (.lua files)            │
│   Rooms, Items, NPCs, Entities, Rules        │
│   Declarative tables + engine helper funcs   │
└──────────────────┬───────────────────────────┘
                   │ load → validate → compile
                   │ (one-time, no Lua at runtime)
┌──────────────────▼───────────────────────────┐
│              Go Engine                       │
│                                              │
│  ┌──────────┐  ┌──────────┐  ┌───────────┐ │
│  │  Parser   │  │  State   │  │ Validator │ │
│  │ verb+obj  │  │  (flat,  │  │ (on load) │ │
│  │ +target   │  │  pure)   │  │           │ │
│  └─────┬─────┘  └────▲─────┘  └───────────┘ │
│        │             │                       │
│  ┌─────▼─────────────┴───────────────────┐  │
│  │           Rules Engine (pure)          │  │
│  │  resolve → collect → filter → rank    │  │
│  │  → select → produce effects           │  │
│  └─────┬─────────────────────────────────┘  │
│        │                                     │
│  ┌─────▼─────────┐   ┌──────────────────┐  │
│  │ Effect Apply   │   │     Events       │  │
│  │ (centralized   │──▶│  (single pass)   │  │
│  │  mutation)     │   │                  │  │
│  └────────────────┘   └──────────────────┘  │
│                                              │
│  ┌───────────────────────────────────────┐  │
│  │              CLI / IO                  │  │
│  └───────────────────────────────────────┘  │
└──────────────────┬───────────────────────────┘
                   │ save / load
            ┌──────▼──────┐
            │  JSON saves  │
            └─────────────┘
```

Three layers:

| Layer | Language | Responsibility |
|-------|----------|----------------|
| **Engine** | Go | Game loop, parsing, state, rules resolution, effects, events, CLI, save/load |
| **Content** | Lua | World definition: rooms, items, NPCs, rules, dialogue. Compile-time only. |
| **Loader** | Go | Sandboxed Lua VM, compiles Lua tables → Go structs, validates, then discards VM |

The engine knows nothing about any specific game. All game-specific content
lives in Lua files. The engine provides the mechanics; Lua provides the world.

**Key invariant:** Lua runs once at load time. After compilation to Go structs,
no Lua executes during gameplay. The game loop is pure Go operating on Go data.

---

## 2. Core Contract

Everything in the engine revolves around this:

```go
type Intent struct {
    Verb   string
    Object string // optional
    Target string // optional
}

type Effect struct {
    Type   string
    Params map[string]any
}

type Event struct {
    Type string
    Data map[string]any
}

type Result struct {
    Effects []Effect
    Events  []Event
    Output  []string
}

// The fundamental operation. Pure function.
func Step(state *State, intent Intent) (*State, Result)
```

If this contract is clean, everything else stays clean.

The rules engine **produces** effects. It does not apply them.
A separate, centralized `ApplyEffects` function handles all state mutation:

```
(state, intent) → rules engine → effects → apply_effects(state, effects) → new_state
```

This gives you:
- Deterministic replay (effects are data, inspectable before application)
- Testability (assert on effects without running mutations)
- Debugging (log the effect list before/after)

---

## 3. Core Game Loop

```
Render output → Prompt input → Parse → Step(state, intent) → Output → Repeat
```

### Turn Sequence (detailed)

```
1.  Display current state output (room description on entry, etc.)
2.  Read player input
3.  Parse into Intent { verb, object?, target? }
4.  Step(state, intent):
    a. Resolve entity references (name → entity ID)
    b. Collect candidate rules (room, target, object, global)
    c. Filter by conditions (When match + Conditions)
    d. Rank by specificity + priority + source order
    e. Select first matching rule
    f. Produce effects (no mutation yet)
    g. Apply effects → new state
    h. Emit events from applied effects
    i. Run single pass of event handlers → more effects → apply
    j. Collect output lines
5.  Advance turn counter
6.  Repeat from 1
```

---

## 4. Data Model

### 4.1 Definitions vs. State

| | Definitions | State |
|---|---|---|
| **Source** | Lua content files (compiled to Go structs) | Runtime mutations |
| **Mutability** | Immutable after load | Mutable during play |
| **Contains** | Room descriptions, base entity properties, rules, dialogue trees | Player location, inventory, flags, counters, property overrides |
| **Persistence** | Reloaded from Lua on game start | Serialized to/from JSON |

Saves are small: they store only the delta from base definitions.
Loading a save = reload Lua definitions + apply saved state on top.

### 4.2 Game State

Flat. Explicit. No deep nesting. No inheritance.

```go
type State struct {
    Player      Player
    Entities    map[string]EntityState  // runtime property overrides
    Flags       map[string]bool         // binary switches
    Counters    map[string]int          // score, gold, turns, etc.
    TurnCount   int
    RNGSeed     int64                   // for deterministic replay
    CommandLog  []string                // for replay
}

type Player struct {
    Location  string
    Inventory []string
    Stats     map[string]int  // hp, attack, defense (v2)
}

type EntityState struct {
    Location string              // overrides base location if set
    Props    map[string]any      // overrides base props
}
```

**Flags**: binary on/off switches. `door_unlocked`, `met_guard`, `quest_started`.
**Counters**: numeric values. `score`, `gold`, `times_entered_hall`.
**EntityState**: property overrides layered on top of base definitions.
When reading entity property `X`: check state override first, fall back to
base definition.

### 4.3 Entities

Everything in the world is an entity: items, NPCs, doors, levers, furniture.

Entities have a **kind** determined by their Lua constructor:

| Constructor | Kind | Required Fields | Defaults |
|-------------|------|-----------------|----------|
| `Item` | `"item"` | `location` | `takeable = true` |
| `NPC` | `"npc"` | `location` | `topics = {}` |
| `Entity` | `"entity"` | `location` | (none) |
| `Room` | `"room"` | `description`, `exits` | `rules = {}` |

Internally, all entities compile to the same Go struct:

```go
type EntityDef struct {
    ID         string
    Kind       string            // "item", "npc", "entity", "room"
    Props      map[string]any    // base properties from Lua
    Rules      []RuleDef         // rules scoped to this entity
}
```

No inheritance. No class hierarchy. The `Kind` field lets the rules engine
match on entity type and lets the loader validate required fields per kind.

### 4.4 Rooms

```go
type RoomDef struct {
    ID          string
    Description string
    Exits       map[string]string   // direction → room_id
    Rules       []RuleDef           // rules scoped to this room
    Fallbacks   map[string]string   // verb → custom failure text (optional)
}
```

Exits can be opened/closed at runtime via effects (`open_exit`, `close_exit`).
Runtime exit changes stored in `State.Entities["room:<id>"]`.

---

## 5. Command Parsing

### 5.1 Design Principle

**Intentionally dumb.** Don't chase NLP.

### 5.2 Grammar

```
<verb>
<verb> <object>
<verb> <object> [preposition] <target>
```

Prepositions are stripped: `use key on door` → `{use, key, door}`.
Recognized prepositions: `on`, `at`, `to`, `with`, `in`, `from`.

### 5.3 Intent

```go
type Intent struct {
    Verb   string
    Object string   // optional
    Target string   // optional
}
```

### 5.4 Built-in Commands (MVP)

| Command | Aliases | Intent |
|---------|---------|--------|
| `look` | `l` | `{look, _, _}` |
| `look at <obj>` | `examine <obj>`, `x <obj>` | `{examine, obj, _}` |
| `go <dir>` | `north`, `n`, `south`, `s`, etc. | `{go, dir, _}` |
| `take <obj>` | `get <obj>`, `pick up <obj>` | `{take, obj, _}` |
| `drop <obj>` | | `{drop, obj, _}` |
| `use <obj>` | | `{use, obj, _}` |
| `use <obj> on <target>` | | `{use, obj, target}` |
| `talk <npc>` | `talk to <npc>` | `{talk, npc, _}` |
| `inventory` | `i` | `{inventory, _, _}` |
| `open <obj>` | | `{open, obj, _}` |
| `attack <target>` | `hit <target>` | `{attack, target, _}` |

Direction shortcuts (`n`, `s`, `e`, `w`, `ne`, `nw`, `se`, `sw`, `up`, `down`)
expand to `go <direction>` before parsing.

### 5.5 Entity Resolution

After parsing, the engine resolves names to entity IDs:

1. Check entities in current room
2. Check player inventory
3. Check NPCs in current room
4. If ambiguous: "Which key? The rusty key or the golden key?"
5. If not found: "You don't see that here."

---

## 6. Rules Engine

**The heart of the engine.** Implemented as a pure function pipeline.

### 6.1 Rule Definition

One type. Same structure everywhere, regardless of scope.

```go
type RuleDef struct {
    ID          string
    Scope       string            // "room:<id>", "entity:<id>", "global"
    When        MatchCriteria     // what intent triggers this rule
    Conditions  []Condition       // all must be true (AND)
    Effects     []Effect          // produced if rule matches
    Priority    int               // tie-break within same scope (higher wins)
    SourceOrder int               // tie-break at same priority (file order)
}
```

### 6.2 Pipeline (6 steps)

The rules engine is NOT a blob. It's an explicit pipeline:

```
Step 1: Resolve     →  map object/target names to entity IDs, resolve aliases
Step 2: Collect     →  gather candidate rules from: room, target entity,
                       object entity, global
Step 3: Filter      →  check When (verb, object, target match) +
                       Conditions (flags, inventory, props)
Step 4: Rank        →  sort by specificity → priority → source order
Step 5: Select      →  pick first match (first match wins)
Step 6: Produce     →  return effects list (NO mutation yet)
```

Effects are data. They're produced, inspected, logged, then applied centrally.

### 6.3 Match Criteria (When)

```go
type MatchCriteria struct {
    Verb       string             // required
    Object     string             // optional: specific entity ID
    Target     string             // optional: specific entity ID
    ObjectKind string             // optional: match by entity kind
    TargetProp map[string]any     // optional: target must have these props
    ObjectProp map[string]any     // optional: object must have these props
}
```

### 6.4 Conditions

All evaluated as **AND** (every condition must pass).
No OR for MVP — write two rules instead. `Not()` for negation.

| Condition | Lua Helper | Description |
|-----------|------------|-------------|
| has_item | `HasItem("key")` | Player has item in inventory |
| flag_is | `FlagIs("door_seen", true)` | Flag equals value |
| flag_set | `FlagSet("quest_started")` | Flag is true |
| flag_not | `FlagNot("quest_started")` | Flag is false or unset |
| counter_gt | `CounterGt("score", 10)` | Counter > value |
| counter_lt | `CounterLt("hp", 5)` | Counter < value |
| in_room | `InRoom("hall")` | Player is in room |
| prop_is | `PropIs("door", "locked", true)` | Entity property equals value |
| not | `Not(HasItem("key"))` | Negates inner condition |

### 6.5 Effects (the instruction set)

Small, fixed, dumb. No logic in effects. Each effect is one atomic operation.

| Effect | Lua Helper | Description |
|--------|------------|-------------|
| say | `Say("text")` | Output text to player |
| give_item | `GiveItem("key")` | Add item to inventory, remove from world |
| remove_item | `RemoveItem("key")` | Remove item from inventory |
| set_flag | `SetFlag("unlocked", true)` | Set flag value |
| inc_counter | `IncCounter("score", 10)` | Increment counter |
| set_counter | `SetCounter("score", 0)` | Set counter to value |
| set_prop | `SetProp("door", "locked", false)` | Set entity property |
| move_entity | `MoveEntity("guard", "throne")` | Move entity to room |
| move_player | `MovePlayer("dungeon")` | Move player to room |
| open_exit | `OpenExit("hall", "north", "secret")` | Add exit to room |
| close_exit | `CloseExit("hall", "north")` | Remove exit from room |
| emit_event | `EmitEvent("custom_event")` | Emit a named event |
| start_dialogue | `StartDialogue("guard")` | Enter dialogue mode |
| stop | `Stop()` | Halt further effect processing |

**This is the instruction set.** If an operation isn't in this list, it
doesn't exist. No `unlock_door_if_player_has_key` — that's a rule with
conditions and `SetProp("door", "locked", false)`.

### 6.6 Resolution Order

For a given intent at `player.location`:

```
1. Room-local rules        (current room's rules)
2. Target entity rules     (if intent has a target)
3. Object entity rules     (if intent has an object)
4. Player rules            (v2: class/perk rules)
5. Global rules
6. Fallback
```

**First match wins.** Resolution stops at the first rule where
When matches AND all Conditions pass.

This must be implemented in code exactly as written, not just documented.

### 6.7 Specificity (within a scope bucket)

When multiple rules in the same scope could match, rank by:

1. Match with `target` beats match without `target`
2. Match with `object` beats match without `object`
3. Match with property conditions beats match without
4. Higher `priority` wins
5. Earlier `source_order` wins (stable, deterministic)

Authors can predict which rule fires: more specific always wins.

### 6.8 Fallback System

If no rule matches at all, the engine produces a fallback response.

**Default:** `"You can't do that."`

**Overridable per room:**
```lua
Room "library" {
    description = "Dusty shelves line every wall...",
    exits = { south = "hall" },
    fallbacks = {
        take = "The librarian glares at you. Everything here is for reading only.",
        use  = "You're not sure how that would help in a library.",
        default = "The silence of the library swallows your attempt.",
    },
}
```

**Overridable per entity:**
```lua
Entity "statue" {
    location = "garden",
    props = { heavy = true },
    fallbacks = {
        take = "The statue is far too heavy to lift.",
        push = "It won't budge.",
    },
}
```

Resolution: entity fallback → room fallback → global default.
This is what gives Sierra games their personality.

---

## 7. Event System

### 7.1 Design

Simple. Not a full event bus.

```
effects applied → events emitted → single pass of event handlers → more effects → apply
```

That's it. No recursion. No cascading.

### 7.2 Events

```go
type Event struct {
    Type string           // "item_taken", "room_entered", etc.
    Data map[string]any   // event-specific payload
}
```

### 7.3 Built-in Events

| Event | Emitted When |
|-------|-------------|
| `room_entered` | Player moves to a new room |
| `item_taken` | Player takes an item |
| `item_dropped` | Player drops an item |
| `item_used` | Player uses an item |
| `flag_changed` | A flag is set or cleared |
| `entity_moved` | An entity changes location |
| `turn_end` | End of each turn |

### 7.4 Event Handlers

Defined in Lua as `On(...)` declarations:

```lua
On("room_entered", {
    conditions = { InRoom("throne_room"), FlagNot("seen_throne") },
    effects    = { Say("The throne glitters with gold."), SetFlag("seen_throne", true) },
})
```

### 7.5 Event Depth

**Single depth.** Event handlers produce effects. Those effects are applied.
But they do NOT trigger further event handlers. One pass only.

This prevents:
- Infinite loops (event A triggers event B triggers event A)
- Non-deterministic cascading
- Behavior that's impossible to debug

Revisit in v2 if a use case demands it.

---

## 8. Dialogue System

### 8.1 Topics

NPCs expose topics. Topics are gated by conditions.

```lua
NPC "guard" {
    name     = "Old Guard",
    location = "entrance",
    topics = {
        greeting = {
            text = "Halt! Who goes there?",
        },
        quest = {
            requires = { FlagSet("met_guard") },
            text     = "The king's crown has been stolen. Will you help?",
            effects  = { SetFlag("quest_started", true), GiveItem("quest_scroll") },
        },
        rumor = {
            requires = { FlagSet("quest_started") },
            text     = "I heard goblins in the eastern cave...",
        },
    },
}
```

### 8.2 Dialogue Flow

1. Player: `talk guard`
2. Engine: shows available topics (conditions met)
3. Player: `ask about quest` or selects by number
4. Engine: displays topic text, applies topic effects
5. Returns to exploration

No nested dialogue trees for MVP. Flat topic list only.

---

## 9. Lua Content Layer

### 9.1 Core Principle

> Lua is a config compiler, not a runtime.

Flow:
```
Lua files → load into sandboxed VM → validate → compile to Go structs → discard VM
```

After loading, **no Lua executes during gameplay**. The game loop operates
entirely on compiled Go structs. This keeps everything deterministic and
debuggable.

### 9.2 Sandboxing

On Lua VM initialization:

**Remove:** `os`, `io`, `debug`, `package`, `require`, `loadfile`, `dofile`,
`load`, `rawset`, `rawget`, `rawequal`, `collectgarbage`, `math.randomseed`

**Provide:**
- Basic Lua: `table`, `string`, `math` (minus `randomseed`), `pairs`,
  `ipairs`, `type`, `tostring`, `tonumber`
- Constructors: `Room`, `Item`, `NPC`, `Entity`, `Rule`, `On`, `Game`
- Condition helpers: `HasItem`, `FlagIs`, `FlagSet`, `FlagNot`, `InRoom`,
  `PropIs`, `CounterGt`, `CounterLt`, `Not`
- Effect helpers: `Say`, `GiveItem`, `RemoveItem`, `SetFlag`, `IncCounter`,
  `SetCounter`, `SetProp`, `MoveEntity`, `MovePlayer`, `OpenExit`,
  `CloseExit`, `EmitEvent`, `StartDialogue`, `Stop`
- Match helpers: `When`, `Then`

All helpers are **pure constructors** — they build data structures (Lua tables),
they don't execute logic.

### 9.3 Content File Structure

A game is a directory of Lua files:

```
mygame/
├── game.lua           -- Game metadata (title, author, start room)
├── rooms.lua          -- Room definitions
├── items.lua          -- Item definitions
├── npcs.lua           -- NPC definitions and dialogue
└── rules.lua          -- Global rules
```

### 9.4 Example Content

```lua
-- game.lua
Game {
    title   = "The Lost Crown",
    author  = "Example Author",
    version = "0.1.0",
    start   = "entrance",
    intro   = "You stand before the castle gates...",
}
```

```lua
-- rooms.lua
Room "entrance" {
    description = "A weathered stone archway marks the castle entrance. "
              .. "Iron torches flicker on either side.",
    exits = {
        north = "hall",
    },
}

Room "hall" {
    description = "A grand hall with towering marble columns. Dust motes "
              .. "drift through shafts of light from high windows.",
    exits = {
        south = "entrance",
        north = "throne_room",
        east  = "garden",
    },
    rules = {
        Rule("hall_unlock",
            When { verb = "use", object = "rusty_key", target = "iron_door" },
            { HasItem("rusty_key") },
            Then {
                RemoveItem("rusty_key"),
                SetProp("iron_door", "locked", false),
                OpenExit("hall", "west", "treasury"),
                Say("The key turns with a grinding screech. The door swings open."),
            }
        ),
    },
    fallbacks = {
        default = "Your footsteps echo through the grand hall, but nothing happens.",
    },
}
```

```lua
-- items.lua
Item "rusty_key" {
    name        = "Rusty Key",
    description = "An old iron key, covered in rust.",
    location    = "hall",
}

Item "quest_scroll" {
    name        = "Quest Scroll",
    description = "A royal decree requesting aid.",
    -- no location: given via effect, not placed in world
}
```

```lua
-- npcs.lua
NPC "guard" {
    name     = "Old Guard",
    location = "entrance",
    topics = {
        greeting = {
            text = "Halt! Who goes there?",
            effects = { SetFlag("met_guard", true) },
        },
        quest = {
            requires = { FlagSet("met_guard") },
            text     = "The king's crown has been stolen. Will you help?",
            effects  = { SetFlag("quest_started", true), GiveItem("quest_scroll") },
        },
    },
}
```

```lua
-- rules.lua (global defaults)
Rule("default_take",
    When { verb = "take", object_kind = "item" },
    {},
    Then { GiveItem("{object}"), Say("Taken.") }
)

Rule("default_drop",
    When { verb = "drop" },
    { HasItem("{object}") },
    Then { RemoveItem("{object}"), Say("Dropped.") }
)

Rule("default_look",
    When { verb = "look" },
    {},
    Then { Say("{room.description}") }
)

Rule("default_examine",
    When { verb = "examine" },
    {},
    Then { Say("{object.description}") }
)

Rule("default_inventory",
    When { verb = "inventory" },
    {},
    Then { Say("{player.inventory}") }
)
```

### 9.5 Template Variables

Effects that take strings support variable interpolation:

| Variable | Resolves To |
|----------|-------------|
| `{object}` | Resolved object entity ID |
| `{target}` | Resolved target entity ID |
| `{verb}` | Parsed verb |
| `{object.name}` | Object entity's `name` property |
| `{object.description}` | Object entity's `description` property |
| `{target.name}` | Target entity's `name` property |
| `{player.location}` | Current room ID |
| `{player.inventory}` | Formatted inventory list |
| `{room.description}` | Current room's description |

No expressions. Just property lookups.

---

## 10. Validation Layer

Runs immediately after Lua compilation, before gameplay starts. **Fail fast.**

### 10.1 Checks

| Check | Description |
|-------|-------------|
| Entity references | Every entity ID referenced in rules/effects exists |
| Room references | Every room ID in exits, `MovePlayer`, `InRoom` exists |
| Exit targets | Every exit points to a valid room ID |
| Rule IDs unique | No duplicate rule IDs across all scopes |
| Required fields | `Item` has `location` (or is given via effect), `Room` has `description` + `exits`, etc. |
| Effect types valid | Every effect `Type` is in the known instruction set |
| Condition types valid | Every condition type is recognized |
| Start room exists | `Game.start` points to a defined room |
| Verb recognition | Warn on verbs used in `When` that don't match known verbs |
| Dangling items | Warn on items with locations that don't match any room |

### 10.2 Output

On validation failure: print all errors, refuse to start.
On validation warnings: print warnings, start anyway.

```
ERROR: Rule "hall_unlock" references entity "iron_door" which is not defined
ERROR: Room "treasury" referenced in exit from "hall" but not defined
WARN:  Item "old_map" has location "cellar" but room "cellar" has no path from start
```

---

## 11. Logging & Replay

Built from day one, not bolted on later.

### 11.1 Command Log

Every command is logged:

```
> look
> take rusty_key
> go north
> use rusty_key on iron_door
> go west
```

Saved alongside game saves. Enables deterministic replay.

### 11.2 Debug Trace

Optional verbose mode showing engine internals:

```
[turn 4] Input: "use rusty_key on iron_door"
[turn 4] Intent: {verb: "use", object: "rusty_key", target: "iron_door"}
[turn 4] Candidates: 3 rules (1 room, 0 entity, 2 global)
[turn 4] Matched: hall_unlock (scope: room:hall, priority: 0)
[turn 4] Effects:
  - remove_item(rusty_key)
  - set_prop(iron_door, locked, false)
  - open_exit(hall, west, treasury)
  - say("The key turns with a grinding screech. The door swings open.")
[turn 4] Events: [exit_opened]
[turn 4] Event handlers matched: 0
```

This is essential for debugging puzzles and rule conflicts.

### 11.3 Replay

```
questcore replay --game ./games/lost_crown --log ./saves/session.log
```

Loads game definitions, replays commands from log with same RNG seed.
Output must be identical. If it's not, something broke determinism.

---

## 12. Save / Load

### 12.1 Save Format

JSON. Contains only mutable state, not definitions.

```json
{
    "version": "0.1.0",
    "game": "lost_crown",
    "turn": 42,
    "player": {
        "location": "throne_room",
        "inventory": ["quest_scroll", "golden_key"],
        "stats": { "hp": 8, "attack": 5 }
    },
    "flags": {
        "met_guard": true,
        "door_unlocked": true,
        "quest_started": true
    },
    "counters": {
        "score": 150,
        "gold": 30
    },
    "entity_state": {
        "iron_door": { "locked": false },
        "guard": { "location": "throne_room" }
    },
    "rng_seed": 8674665223082153551,
    "command_log": ["take rusty_key", "go north", "..."]
}
```

### 12.2 Loading a Save

1. Load game Lua files → compile to Go structs
2. Initialize fresh `State`
3. Deserialize JSON save
4. Apply saved state over fresh state
5. Resume play

### 12.3 Deterministic Replay

Replay = load game + play commands from log with same RNG seed.
Same inputs + same seed = identical output. Deviation = bug.

---

## 13. CLI Interface

### 13.1 Meta-commands

Prefixed with `/` to distinguish from game commands:

| Command | Description |
|---------|-------------|
| `/save [name]` | Save current state |
| `/load [name]` | Load saved state |
| `/quit` | Exit game |
| `/help` | Show available commands |
| `/replay <file>` | Replay command log |
| `/state` | Debug: dump current state |
| `/rules` | Debug: show matched rules for last command |
| `/trace` | Toggle debug trace output |

### 13.2 Output Formatting

- Room descriptions: default color
- Entity/item names: bold or highlighted
- NPC dialogue: distinct style
- System messages (save/load): dim/gray
- Errors: clearly marked
- Optional: suggestions on bad input ("Did you mean...?")

---

## 14. Go Module Structure

```
questcore/
├── cmd/
│   └── questcore/
│       └── main.go                # Entry point, wiring
│
├── engine/
│   ├── engine.go                  # Step(), game loop orchestrator
│   │
│   ├── parser/
│   │   └── parser.go             # Command string → Intent
│   │
│   ├── state/
│   │   └── state.go              # State struct, property lookups
│   │
│   ├── rules/
│   │   ├── rules.go              # Pipeline: collect, filter, rank, select
│   │   ├── match.go              # When matching logic
│   │   └── conditions.go         # Condition evaluation
│   │
│   ├── effects/
│   │   └── effects.go            # ApplyEffects: centralized state mutation
│   │
│   ├── events/
│   │   └── events.go             # Event emission + handler dispatch
│   │
│   ├── dialogue/
│   │   └── dialogue.go           # Topic selection, display
│   │
│   ├── resolve/
│   │   └── resolve.go            # Entity name → ID resolution
│   │
│   └── save/
│       └── save.go               # JSON serialize/deserialize
│
├── loader/
│   ├── loader.go                 # Lua VM setup, sandbox, load files
│   ├── api.go                    # Register constructors + helpers into Lua
│   ├── compile.go                # Lua tables → Go structs
│   └── validate.go               # Post-load validation, fail fast
│
├── cli/
│   ├── cli.go                    # Terminal I/O, prompt, formatting
│   └── commands.go               # Meta-commands: /save, /load, /quit, etc.
│
├── types/
│   └── types.go                  # Shared types: Intent, Effect, Event,
│                                 # Condition, RuleDef, EntityDef, RoomDef, etc.
│
├── games/
│   └── lost_crown/               # Example game
│       ├── game.lua
│       ├── rooms.lua
│       ├── items.lua
│       ├── npcs.lua
│       └── rules.lua
│
├── go.mod
├── go.sum
├── DESIGN.md
└── questcore.md
```

### Module Responsibilities

| Package | Responsibility | Depends On |
|---------|---------------|------------|
| `cmd/questcore` | Entry point, wiring | `engine`, `loader`, `cli`, `types` |
| `engine` | `Step()` orchestrator | `engine/*` sub-packages, `types` |
| `engine/parser` | Command string → Intent | `types` |
| `engine/state` | State struct, property reads | `types` |
| `engine/rules` | Rule pipeline (collect, filter, rank, select) | `types` |
| `engine/effects` | Apply effects → mutate state | `engine/state`, `types` |
| `engine/events` | Emit events, dispatch handlers | `engine/rules`, `engine/effects`, `types` |
| `engine/resolve` | Entity name → ID | `engine/state`, `types` |
| `engine/dialogue` | Topic system | `engine/state`, `types` |
| `engine/save` | JSON serialization | `engine/state`, `types` |
| `loader` | Lua → Go structs, validation | `types`, `gopher-lua` |
| `cli` | Terminal I/O, meta-commands | `engine`, `types` |
| `types` | Shared data types, no logic | (none) |

Dependency flow: `cmd → cli → engine ← loader`, with `types` at the bottom.
No circular dependencies.

---

## 15. MVP Scope

### Must Have (v1)

- [ ] Rooms + navigation (go, look)
- [ ] Items + inventory (take, drop, inventory, examine)
- [ ] Use item on target
- [ ] Flags + conditions
- [ ] Rules engine (full 6-step pipeline, resolution order)
- [ ] Fallback system (overridable per room/entity)
- [ ] Dialogue topics (talk, ask)
- [ ] Event system (single depth)
- [ ] Save/load (JSON)
- [ ] Command logging
- [ ] Debug trace mode
- [ ] Deterministic replay
- [ ] Lua content loading + sandboxing + validation
- [ ] Example game ("The Lost Crown")
- [ ] CLI with formatted output

### v2 (after MVP validates)

- [ ] Combat system (attack, defend, flee, HP, stats)
- [ ] Counters (score, gold, custom)
- [ ] Equipment slots
- [ ] `script = function(ctx)` escape hatch (sandboxed)
- [ ] Container entities (chest contains items)
- [ ] Light/dark rooms
- [ ] Timed events (trigger after N turns)

### Not Building

- Full NLP parser
- GUI / graphics
- Full D&D rules
- Real-time anything
- LLM integration
- Multiplayer
- Procedural generation

---

## 16. Key Design Decisions (log)

| # | Decision | Rationale |
|---|----------|-----------|
| 1 | Go engine + Lua content | Go: performance, single binary. Lua: expressive, sandboxed content. Proven game dev pattern. |
| 2 | Lua is compile-time only | Lua runs once at load → Go structs. No Lua during gameplay. Deterministic, debuggable. |
| 3 | Rules engine is a pure function | `(state, intent) → effects`. No mutation in rules. Centralized `ApplyEffects`. Testable, replayable. |
| 4 | Effects are a fixed instruction set | Small set of dumb, explicit operations. No logic in effects. The "assembly language" of state mutation. |
| 5 | 6-step rule pipeline | Resolve → collect → filter → rank → select → produce. Explicit, no blob. |
| 6 | ECS-inspired, not full ECS | Entity data + action systems + rule-based event triggers. No frames, no real-time. |
| 7 | Light entity types via constructors | `Item()`, `NPC()`, `Entity()` set a `kind` field. Validate per kind. No schema system. |
| 8 | Rules are one type, scoped by location | Room, entity, global rules share `RuleDef`. Scope determines resolution priority. |
| 9 | First match wins | Resolution stops at first match. Predictable. No fallthrough. |
| 10 | Conditions are AND-only (MVP) | All must pass. No OR — write two rules. `Not()` for negation. |
| 11 | Single-depth events (MVP) | Event handlers don't re-trigger events. No cascading. Revisit in v2. |
| 12 | Fallback system with personality | Overridable per room and entity. Sierra-style charm. |
| 13 | Validation on load, fail fast | Check all references, IDs, types before gameplay. No runtime surprises. |
| 14 | Logging and replay from day one | Command log + debug trace. Essential for development and testing. |
| 15 | Definitions immutable, state mutable | Lua = template. State = runtime delta. Saves are small. |
| 16 | Template variables, not expressions | `{object.name}` — property lookups only. No computed expressions. |

---

## 17. External Dependencies

| Dependency | Purpose |
|------------|---------|
| [`gopher-lua`](https://github.com/yuin/gopher-lua) | Pure-Go Lua 5.1 VM |
| Standard library | `encoding/json`, `fmt`, `os`, `bufio`, `strings`, `sort` |

Optional (CLI polish):

| Dependency | Purpose |
|------------|---------|
| [`fatih/color`](https://github.com/fatih/color) | Terminal colors |
| [`chzyer/readline`](https://github.com/chzyer/readline) | Line editing, history |

Minimal dependencies. No frameworks.

---

## 18. Risks & Mitigations

| Risk | Mitigation |
|------|-----------|
| Overengineering rules | Keep the instruction set small. If you need a new effect type, think twice. |
| Lua becoming runtime scripting | Enforced at architecture level: VM discarded after compile. No escape. |
| Unclear rule resolution | Resolution order is code, not just docs. Test it explicitly. |
| Complex state bugs | State is flat. Effects are dumb. Debug trace shows everything. |
| Scope creep | MVP list is locked. v2 features don't exist until v1 ships. |

---

## 19. Resolved Questions

- **Inventory capacity**: Fixed limit. Configurable per game in `Game{}` metadata.
- **Room revisit text**: Full description every time (MVP). Brief-on-revisit can be added in v2.
- **Ambiguity resolution**: Prompt the player. "Which key? The rusty key or the golden key?"

---

*This is a living document. Update as decisions are made during implementation.*
