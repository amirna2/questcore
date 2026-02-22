# Plan: QuestCore Layer 8 — Lua Loader

## Context

Layers 1-7 and 9-11 are complete (184 tests passing). The loader is the last
complex piece — it bridges Lua game content and the Go engine. After this,
only CLI (Layer 12), entry point (13), and example game (14) remain.

The loader must: set up a sandboxed Lua VM, register constructors/helpers as
globals, load `.lua` files from a game directory, compile Lua tables into
`state.Defs`, validate references, and discard the VM. Zero Lua at runtime.

---

## 1. Add gopher-lua dependency

```
go get github.com/yuin/gopher-lua
```

---

## 2. File Structure

```
loader/
├── loader.go          # Load(dir), VM setup, sandbox, file discovery
├── api.go             # Register constructors + helpers into Lua globals
├── compile.go         # Lua LTable → Go struct conversion
├── validate.go        # Post-compile validation (fail fast)
├── loader_test.go     # Integration tests with testdata/
├── compile_test.go    # Unit tests for compilation
├── validate_test.go   # Unit tests for validation
└── testdata/
    ├── minimal/       # 1 room, game.lua only
    │   └── game.lua
    ├── full/          # Complete game: rooms, items, NPCs, rules, handlers
    │   ├── game.lua
    │   ├── rooms.lua
    │   ├── items.lua
    │   ├── npcs.lua
    │   └── rules.lua
    ├── invalid_refs/  # Broken room/entity references
    │   ├── game.lua
    │   └── rooms.lua
    ├── duplicate_rules/
    │   ├── game.lua
    │   └── rules.lua
    ├── bad_lua/       # Syntax error
    │   └── game.lua
    └── no_game/       # Missing Game{} call
        └── rooms.lua
```

---

## 3. `loader/loader.go` — Entry Point

### Public API

```go
func Load(dir string) (*state.Defs, error)
```

### Collector (unexported)

Lua constructors append to a shared collector. After all files execute,
`compile()` reads it:

```go
type collector struct {
    game        *lua.LTable
    rooms       []rawRoom       // {id, table}
    entities    []rawEntity     // {id, kind, table}
    rules       []rawRule       // {id, when, conditions, then, scope, order}
    handlers    []rawHandler    // {eventType, table}
    sourceOrder int
}
```

### Load() lifecycle

1. Create VM with `SkipOpenLibs: true`
2. Selectively open safe libs (base, table, string, math)
3. Sandbox: nil out dangerous globals (dofile, loadfile, load, rawset,
   rawget, rawequal, collectgarbage) and math.randomseed.
   os/io/debug/package never opened.
4. Create collector, register API via `registerAPI(L, coll)`
5. Discover `.lua` files — `game.lua` first, rest alphabetical
6. Execute each file via `L.DoFile()`
7. `compile(coll)` → `*state.Defs`
8. `validate(defs)` → error if any check fails
9. `L.Close()` — VM discarded
10. Return `(defs, nil)`

---

## 4. `loader/api.go` — Lua API Registration

### Constructors

All use the **collector pattern** — append data, don't return Go values.

| Function | Lua Syntax | Mechanism |
|----------|-----------|-----------|
| `Game` | `Game { title = "..." }` | `coll.game = tbl` |
| `Room` | `Room "id" { ... }` | Curried: returns function that takes table |
| `Item` | `Item "id" { ... }` | Curried, `kind = "item"` |
| `NPC` | `NPC "id" { ... }` | Curried, `kind = "npc"` |
| `Entity` | `Entity "id" { ... }` | Curried, `kind = "entity"` |
| `Rule` | `Rule("id", when, conds, then)` | 4 args, appends to `coll.rules`, returns marker table |
| `On` | `On("event", { ... })` | Appends to `coll.handlers` |
| `When` | `When { verb = "..." }` | Pass-through (returns table) |
| `Then` | `Then { Say("...") }` | Pass-through (returns table) |

### Rule scoping mechanism

`Rule(...)` always appends to `coll.rules` with `scope = "global"` and
returns a marker table with `__rule_id`. When `compile()` processes a
Room/Entity's `rules` field, it finds these markers and re-scopes the
matching collector entries to `"room:<id>"` or `"entity:<id>"`.

### Condition helpers (each returns a Lua table)

| Helper | Result |
|--------|--------|
| `HasItem("key")` | `{type="has_item", item="key"}` |
| `FlagSet("flag")` | `{type="flag_set", flag="flag"}` |
| `FlagNot("flag")` | `{type="flag_not", flag="flag"}` |
| `FlagIs("flag", true)` | `{type="flag_is", flag="flag", value=true}` |
| `InRoom("hall")` | `{type="in_room", room="hall"}` |
| `PropIs("e", "p", v)` | `{type="prop_is", entity="e", prop="p", value=v}` |
| `CounterGt("c", 5)` | `{type="counter_gt", counter="c", value=5}` |
| `CounterLt("c", 5)` | `{type="counter_lt", counter="c", value=5}` |
| `Not(cond)` | `{type="not", inner=cond}` |

### Effect helpers (each returns a Lua table)

| Helper | Result |
|--------|--------|
| `Say("text")` | `{type="say", text="text"}` |
| `GiveItem("id")` | `{type="give_item", item="id"}` |
| `RemoveItem("id")` | `{type="remove_item", item="id"}` |
| `SetFlag("f", true)` | `{type="set_flag", flag="f", value=true}` |
| `IncCounter("c", 1)` | `{type="inc_counter", counter="c", amount=1}` |
| `SetCounter("c", 0)` | `{type="set_counter", counter="c", value=0}` |
| `SetProp("e","p",v)` | `{type="set_prop", entity="e", prop="p", value=v}` |
| `MoveEntity("e","r")` | `{type="move_entity", entity="e", room="r"}` |
| `MovePlayer("room")` | `{type="move_player", room="room"}` |
| `OpenExit("r","d","t")` | `{type="open_exit", room="r", direction="d", target="t"}` |
| `CloseExit("r","d")` | `{type="close_exit", room="r", direction="d"}` |
| `EmitEvent("type")` | `{type="emit_event", event="type"}` |
| `StartDialogue("npc")` | `{type="start_dialogue", npc="npc"}` |
| `Stop()` | `{type="stop"}` |

---

## 5. `loader/compile.go` — Lua Tables → Go Structs

### Conversion helpers

```go
getString(tbl, key) string
getBool(tbl, key, default) bool
getNumber(tbl, key) float64
getTable(tbl, key) *lua.LTable
toGoValue(lua.LValue) any          // recursive: handles bool/number/string/table
tableToStringMap(tbl) map[string]string
```

### Compilation functions

```go
compile(coll) → (*state.Defs, error)
  compileGame(tbl) → GameDef
  compileRoom(raw) → (RoomDef, []scopedRuleIDs, error)
  compileEntity(raw) → (EntityDef, []scopedRuleIDs, error)
  compileTopics(tbl) → map[string]TopicDef
  compileRule(raw) → (RuleDef, error)
  compileMatchCriteria(tbl) → MatchCriteria
  compileConditions(tbl) → []Condition
  compileCondition(tbl) → Condition
  compileEffects(tbl) → []Effect
  compileEffect(tbl) → Effect
  compileHandler(raw) → (EventHandler, error)
  markScopedRules(coll, ruleIDs, scope)
```

### Key behaviors

- **Item default:** `takeable = true` if not explicitly set
- **Entity props:** All fields except `rules` and `topics` go into `Props`
- **Rule scope:** After rooms/entities are compiled, scoped rules are attached
  to their owning Room/Entity's `Rules` slice
- **SourceOrder:** Auto-incremented via `coll.nextSourceOrder()`

---

## 6. `loader/validate.go` — Post-Compile Validation

### Errors (fatal — refuse to start)

| Check | Description |
|-------|-------------|
| Start room exists | `Game.Start` points to a defined room |
| Game title required | `Game.Title` is non-empty |
| Exit targets valid | Every `room.Exits[dir]` points to a defined room |
| Rule IDs unique | No duplicate rule IDs across all scopes |
| Effect types valid | Every effect type is in the known set of 14 |
| Condition types valid | Every condition type is recognized |
| Entity refs in effects | `give_item`, `set_prop`, etc. reference defined entities |
| Room refs in effects | `move_player`, `open_exit` reference defined rooms |
| Entity refs in conditions | `prop_is`, `has_item` reference defined entities |
| Room refs in conditions | `in_room` references defined rooms |

Template vars (`{object}`, etc.) are skipped during ref checks.

### Warnings (print but continue)

- Dangling items (location doesn't match any room)
- Unrecognized verbs in `When` clauses

### Output format

```go
type ValidationError struct {
    Errors   []string
    Warnings []string
}
```

Warnings go to stderr. Errors collected and returned as a single error.

---

## 7. Test Strategy

### `validate_test.go` — Pure Go, no Lua needed

Construct `state.Defs` in Go, call `validate()`, assert errors/warnings.

- Valid defs → no error
- Missing start room → error
- Invalid exit target → error
- Duplicate rule ID → error
- Unknown effect type → error
- Unknown condition type → error
- Undefined entity in effect → error
- Template refs (`{object}`) → not flagged
- Dangling item location → warning only
- Unrecognized verb → warning only

### `compile_test.go` — Small Lua snippets executed in VM

Create a Lua VM, register API, execute snippets, pass resulting tables
to compile functions. Verify Go struct output.

- compileGame with all fields
- compileRoom with exits, fallbacks, rules
- compileEntity for item (default takeable), NPC (topics), generic entity
- compileCondition for all 9 condition types (table-driven)
- compileEffect for all 14 effect types (table-driven)
- compileMatchCriteria: verb-only, verb+object+target, object_kind, props
- Rule scope: room rule gets "room:id", global stays "global"

### `loader_test.go` — Integration with testdata/

- `TestLoad_MinimalGame` — smallest valid game loads
- `TestLoad_FullGame` — all features: rooms, items, NPCs, topics, rules,
  handlers, fallbacks. Verify every field.
- `TestLoad_InvalidRefs_Fails` — bad entity/room reference → error
- `TestLoad_DuplicateRuleIDs_Fails` — duplicate rule ID → error
- `TestLoad_BadLuaSyntax_Fails` — syntax error → error
- `TestLoad_NoGameDef_Fails` — missing Game{} → error
- `TestLoad_SandboxEnforced` — Lua file calling os.execute fails
- `TestLoad_ItemDefaultTakeable` — item without explicit takeable gets true
- `TestLoad_RuleScopeResolution` — room rules scoped correctly

---

## 8. Build Order

1. `go get github.com/yuin/gopher-lua`
2. `loader/compile.go` — conversion helpers + compile functions
3. `loader/api.go` — register all constructors/helpers
4. `loader/loader.go` — Load(), VM setup, sandbox, collector
5. `loader/validate.go` — validation checks
6. `loader/testdata/` — Lua fixture files
7. `loader/validate_test.go` — pure Go validation tests
8. `loader/compile_test.go` — compilation unit tests
9. `loader/loader_test.go` — integration tests

Steps 2-4 need to be built together (they're tightly coupled), but each
file has a clear single responsibility. Tests come after all source files
compile.

## 9. Verification

```
go build ./...
go vet ./...
go test ./... -v -count=1   # all 184 existing + new loader tests pass
```

## 10. Files

```
Modified:
  go.mod                           # add gopher-lua dependency

New:
  loader/loader.go
  loader/api.go
  loader/compile.go
  loader/validate.go
  loader/loader_test.go
  loader/compile_test.go
  loader/validate_test.go
  loader/testdata/minimal/game.lua
  loader/testdata/full/game.lua
  loader/testdata/full/rooms.lua
  loader/testdata/full/items.lua
  loader/testdata/full/npcs.lua
  loader/testdata/full/rules.lua
  loader/testdata/invalid_refs/game.lua
  loader/testdata/invalid_refs/rooms.lua
  loader/testdata/duplicate_rules/game.lua
  loader/testdata/duplicate_rules/rules.lua
  loader/testdata/bad_lua/game.lua
  loader/testdata/no_game/rooms.lua
```
