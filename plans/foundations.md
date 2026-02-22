# Plan: QuestCore Foundation (Layers 1-3)

## Context

QuestCore is a greenfield Go game engine for text adventures. No code exists yet.
We're building the first three layers — the foundational types, state management,
and command parser — following the architecture in DESIGN.md.

## What We're Building

### 1. Project Setup
- Initialize Go module (`go mod init`)
- Create directory structure for all packages (empty dirs are fine for now)

### 2. `types/types.go` — Shared Data Types
All shared data structures with no logic. This is the engine's vocabulary.

Types to define (from DESIGN.md §2, §4, §6):
- `Intent` — verb, object, target
- `Effect` — type + params
- `Event` — type + data
- `Result` — effects, events, output lines
- `MatchCriteria` — verb, object, target, object_kind, target/object props
- `Condition` — type + params (with `Not` support via nested condition)
- `RuleDef` — ID, scope, when, conditions, effects, priority, source order
- `EntityDef` — ID, kind, props, rules
- `RoomDef` — ID, description, exits, rules, fallbacks
- `GameDef` — title, author, version, start room, intro
- `State` — player, entities, flags, counters, turn count, RNG seed, command log
- `Player` — location, inventory, stats
- `EntityState` — location override, property overrides
- `EventHandler` — event type, conditions, effects

### 3. `engine/state/state.go` — State Management
State struct with constructor and property lookups.

Functions:
- `NewState(gameDef, rooms, entities)` — initialize fresh state from definitions
- `GetEntityProp(entityID, prop)` — check state override, fall back to base def
- `GetFlag(name) bool`
- `GetCounter(name) int`
- `HasItem(itemID) bool`
- `PlayerLocation() string`
- `EntitiesInRoom(roomID) []string` — entities whose effective location matches

The key behavior here: **override layering**. When reading an entity property,
check `State.Entities[id].Props[key]` first, fall back to `EntityDef.Props[key]`.

Test file: `engine/state/state_test.go`
- Test property override layering
- Test flag/counter access with defaults
- Test inventory checks
- Test entity room filtering

### 4. `engine/parser/parser.go` — Command Parser
Intentionally dumb parser: string → Intent.

Behavior:
- Split input on whitespace
- Expand direction shortcuts (`n` → `go north`, etc.)
- Expand verb aliases (`l` → `look`, `x` → `examine`, `get` → `take`, `i` → `inventory`)
- Strip prepositions (`on`, `at`, `to`, `with`, `in`, `from`)
- Produce Intent: `{verb, object?, target?}`
- Handle `pick up` as two-word verb → `take`
- Handle `look at` → `examine`

Test file: `engine/parser/parser_test.go`
- Table-driven tests covering:
  - Basic verbs: `look`, `inventory`
  - Direction shortcuts: `n`, `se`, `up`
  - Verb aliases: `l`, `x`, `get`, `i`
  - Object commands: `take key`, `drop sword`
  - Preposition stripping: `use key on door`, `talk to guard`
  - Multi-word: `pick up key`, `look at painting`
  - Edge cases: empty input, single word, unknown verb (pass through)

## File List

```
questcore/
├── go.mod
├── types/
│   └── types.go
├── engine/
│   ├── state/
│   │   ├── state.go
│   │   └── state_test.go
│   └── parser/
│       ├── parser.go
│       └── parser_test.go
```

## Verification

1. `go build ./...` — compiles cleanly
2. `go vet ./...` — no issues
3. `go test ./engine/state/ -v` — all state tests pass
4. `go test ./engine/parser/ -v` — all parser tests pass
5. `go test ./... -v` — everything green
