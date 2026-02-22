# Plan: QuestCore Layers 4-6 (Resolve, Rules, Effects)

## Context

Layers 1-3 are complete: `types/`, `engine/state/`, `engine/parser/` with 62 passing tests.
Now we build the engine's heart — entity resolution, the rules pipeline, and
effect application. These three packages form the core game loop:
`parse → resolve → rules → effects → new state`.

## Pre-requisite: Add GlobalRules to Defs

The `state.Defs` struct currently holds `Game`, `Rooms`, `Entities`, `Handlers`.
We need to add `GlobalRules []types.RuleDef` for standalone rules (from `rules.lua`).

**File:** `engine/state/state.go` — add `GlobalRules` field to `Defs`.

---

## Layer 4: `engine/resolve/resolve.go`

Entity name → ID resolution (DESIGN.md §5.5).

### API

```go
// ResolveResult holds the resolved entity IDs for an intent.
type ResolveResult struct {
    ObjectID string // resolved object entity ID (empty if none)
    TargetID string // resolved target entity ID (empty if none)
}

// Resolve maps object/target name strings to entity IDs.
// Returns an error for ambiguity or not-found.
func Resolve(s *types.State, defs *state.Defs, intent types.Intent) (ResolveResult, error)
```

### Resolution logic (per name)

1. If name is empty → skip (no resolution needed)
2. Exact ID match → use it
3. Check entities in current room: match against `name` prop (case-insensitive)
4. Check player inventory: match against `name` prop (case-insensitive)
5. If multiple matches → `AmbiguityError` with candidates
6. If zero matches → `NotFoundError`

### Error types

```go
type AmbiguityError struct {
    Name       string
    Candidates []string // entity IDs
}

type NotFoundError struct {
    Name string
}
```

### Tests (`engine/resolve/resolve_test.go`)

- Resolve by exact ID
- Resolve by name (case-insensitive)
- Room-scoped: only finds entities in current room
- Inventory: finds items in inventory
- Ambiguity: two items with same name → error with candidates
- Not found: → error
- No object/target: → empty result, no error
- Entity with runtime location override

---

## Layer 5: `engine/rules/` — The Pipeline

The hardest part. Three files: `rules.go` (pipeline), `match.go` (When matching),
`conditions.go` (condition evaluation).

### 5a: `engine/rules/conditions.go` — Condition evaluation

```go
// EvalCondition evaluates a single condition against the current state.
func EvalCondition(c types.Condition, s *types.State, defs *state.Defs) bool
```

Condition types to evaluate:
| Type | Params | Logic |
|------|--------|-------|
| `has_item` | `item` | `state.HasItem(s, item)` |
| `flag_set` | `flag` | `state.GetFlag(s, flag) == true` |
| `flag_not` | `flag` | `state.GetFlag(s, flag) == false` |
| `flag_is` | `flag`, `value` | `state.GetFlag(s, flag) == value` |
| `counter_gt` | `counter`, `value` | `state.GetCounter(s, counter) > value` |
| `counter_lt` | `counter`, `value` | `state.GetCounter(s, counter) < value` |
| `in_room` | `room` | `state.PlayerLocation(s) == room` |
| `prop_is` | `entity`, `prop`, `value` | `state.GetEntityProp(s, defs, entity, prop) == value` |
| `not` | (uses Inner) | `!EvalCondition(*c.Inner, s, defs)` |

All conditions in a rule are AND'd: `EvalAllConditions([]Condition, state, defs) bool`.

### 5b: `engine/rules/match.go` — When matching

```go
// MatchesIntent checks if a rule's When criteria match the resolved intent.
func MatchesIntent(when types.MatchCriteria, verb, objectID, targetID string,
    defs *state.Defs) bool
```

Matching logic:
- `when.Verb` must equal intent verb (required)
- `when.Object` if set, must equal objectID
- `when.Target` if set, must equal targetID
- `when.ObjectKind` if set, must match entity's Kind
- `when.ObjectProp` if set, all props must match on object entity
- `when.TargetProp` if set, all props must match on target entity

Also: `Specificity(rule) int` — numeric score for ranking:
- +4 if When has Target
- +2 if When has Object
- +1 if When has property conditions (ObjectProp or TargetProp)

### 5c: `engine/rules/rules.go` — The 6-step pipeline

```go
// Evaluate runs the full rules pipeline. Returns the matched effects,
// or fallback effects if no rule matches.
func Evaluate(s *types.State, defs *state.Defs,
    intent types.Intent, objectID, targetID string) []types.Effect
```

**Step 2 — Collect:** Gather candidate rules in resolution order:
1. Current room's rules (`defs.Rooms[location].Rules`)
2. Target entity's rules (`defs.Entities[targetID].Rules`) — if target exists
3. Object entity's rules (`defs.Entities[objectID].Rules`) — if object exists
4. Global rules (`defs.GlobalRules`)

Each bucket is kept separate for resolution-order priority.

**Step 3 — Filter:** For each candidate:
- `MatchesIntent(rule.When, verb, objectID, targetID, defs)` must be true
- `EvalAllConditions(rule.Conditions, s, defs)` must be true

**Step 4 — Rank:** Within each scope bucket, sort filtered rules by:
1. Specificity (descending)
2. Priority (descending)
3. SourceOrder (ascending — earlier wins)

**Step 5 — Select:** Walk buckets in order (room → target → object → global).
First bucket with a passing rule → take its top-ranked rule. **First match wins.**

**Step 6 — Produce:** Return `rule.Effects`. No mutation.

**Fallback:** If no rule matches after all buckets:
1. Check entity fallback (`defs.Entities[objectID].Props["fallbacks"]`)
2. Check room fallback (`defs.Rooms[location].Fallbacks[verb]`)
3. Check room default fallback (`defs.Rooms[location].Fallbacks["default"]`)
4. Global default: `Say("You can't do that.")`

### Tests (`engine/rules/rules_test.go`, `conditions_test.go`, `match_test.go`)

**conditions_test.go** — table-driven:
- Each condition type: has_item, flag_set, flag_not, flag_is, counter_gt, counter_lt, in_room, prop_is
- Not() negation
- EvalAllConditions with all passing, one failing

**match_test.go** — table-driven:
- Verb match / mismatch
- Object match (specific ID)
- Target match
- ObjectKind match
- Property matching
- Specificity scoring

**rules_test.go** — integration-style:
- Room rule beats global rule (resolution order)
- Target entity rule beats object entity rule
- More specific rule beats less specific (within same scope)
- Priority breaks tie at same specificity
- SourceOrder breaks tie at same priority
- Fallback: entity fallback, room fallback, room default, global default
- No rules match → global fallback
- Conditions gate rule: condition fails → skip to next

---

## Layer 6: `engine/effects/effects.go`

Centralized state mutation. Every effect type is one atomic operation.

### API

```go
// Context carries the resolved intent context needed for template interpolation.
type Context struct {
    Verb     string
    ObjectID string
    TargetID string
}

// Apply applies effects to state. Returns events emitted and output text.
func Apply(s *types.State, defs *state.Defs, effects []types.Effect,
    ctx Context) ([]types.Event, []string)
```

### Effect handlers

| Effect | Params | Mutation |
|--------|--------|----------|
| `say` | `text` | Append interpolated text to output |
| `give_item` | `item` | Add to inventory, set entity location to `""` (nowhere) |
| `remove_item` | `item` | Remove from inventory |
| `set_flag` | `flag`, `value` | `state.Flags[flag] = value`; emit `flag_changed` |
| `inc_counter` | `counter`, `amount` | `state.Counters[counter] += amount` |
| `set_counter` | `counter`, `value` | `state.Counters[counter] = value` |
| `set_prop` | `entity`, `prop`, `value` | Set entity property override |
| `move_entity` | `entity`, `room` | Set entity location override; emit `entity_moved` |
| `move_player` | `room` | `state.Player.Location = room`; emit `room_entered` |
| `open_exit` | `room`, `direction`, `target` | Set `room:<id>` exit prop |
| `close_exit` | `room`, `direction` | Set `room:<id>` exit prop to `""` |
| `emit_event` | `event` | Emit custom event |
| `start_dialogue` | `npc` | (stub for now — dialogue is layer 9) |
| `stop` | (none) | Stop processing remaining effects |

### Template interpolation

For `say` text: replace `{object}`, `{target}`, `{verb}`, `{object.name}`,
`{object.description}`, `{target.name}`, `{player.location}`, `{player.inventory}`,
`{room.description}` with resolved values.

Simple string replacement — no expressions. Use `state.GetEntityProp` for lookups.

### Tests (`engine/effects/effects_test.go`)

Table-driven per effect type:
- say: outputs text, template interpolation
- give_item: adds to inventory, clears entity location
- remove_item: removes from inventory
- set_flag: sets flag, emits flag_changed event
- move_player: updates location, emits room_entered event
- set_prop: sets entity property override
- move_entity: sets entity location, emits entity_moved event
- open_exit / close_exit: modifies room exit props
- inc_counter / set_counter: modifies counter
- stop: halts further effects
- unknown effect type: ignored (no crash)

---

## File List

```
Modified:
  engine/state/state.go          # Add GlobalRules to Defs

New:
  engine/resolve/resolve.go
  engine/resolve/resolve_test.go
  engine/rules/conditions.go
  engine/rules/conditions_test.go
  engine/rules/match.go
  engine/rules/match_test.go
  engine/rules/rules.go
  engine/rules/rules_test.go
  engine/effects/effects.go
  engine/effects/effects_test.go
```

## Verification

1. `go build ./...` — compiles cleanly
2. `go vet ./...` — no issues
3. `go test ./engine/resolve/ -v` — resolve tests pass
4. `go test ./engine/rules/ -v` — rules tests pass
5. `go test ./engine/effects/ -v` — effects tests pass
6. `go test ./... -v` — all 62 existing + new tests green
