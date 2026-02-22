# QuestCore

A deterministic, data-driven game engine for text adventure and RPG games. Go engine, Lua content, JSON saves.

Think King's Quest meets a modern rules engine — all game behavior is defined in Lua data files, compiled once at startup into Go structs. Zero Lua execution during gameplay. Same state + same command = identical result, always.

## Quick Start

```bash
go build -o questcore ./cmd/questcore
./questcore games/lost_crown/
```

Requires Go 1.21+.

## How to Play

Type commands in natural English. The parser understands 90+ verb synonyms, multi-word names, and articles.

### Movement

```
go north          walk east         run south
n / s / e / w     ne / nw / se / sw
up / down         u / d
```

### Interaction

```
look              examine fireplace     take the rusty key
read old book     use key on door       open chest
drop sword        give coin to merchant
```

### NPCs

```
talk to captain           speak with scholar
ask elara about crown     ask captain about passage
```

### Utility

```
inventory (i)     wait (z)          again (g)
/help             /save [name]      /load [name]
/quit
```

## Creating Games

Games are directories of Lua files. QuestCore loads them at startup and compiles them into Go structs — Lua is a data language here, not a scripting runtime.

### Rooms

```lua
Room "great_hall" {
    description = "A grand hall with a massive fireplace.",
    exits = { north = "throne_room", east = "library" },
    fallbacks = { take = "Everything here belongs to the king." }
}
```

### Items

```lua
Item "rusty_key" {
    name = "rusty key",
    description = "A small iron key, rough with rust.",
    location = "castle_gates"
}
```

### NPCs

```lua
NPC "scholar" {
    name = "Scholar Elara",
    description = "An elderly woman in ink-stained robes.",
    location = "library",
    topics = {
        greet = {
            text = "'The answer lies in the books, as it always does.'",
            effects = { SetFlag("met_scholar", true) }
        },
        passage = {
            text = "'Push the third stone from the left.'",
            requires = { HasItem("old_book"), FlagSet("met_scholar") },
            effects = { SetFlag("knows_passage", true) }
        }
    }
}
```

### Rules

Rules map intents to effects. First match wins.

```lua
Rule("push_wall_library",
    When { verb = "push", object = "wall" },
    { InRoom("library"), FlagSet("knows_passage") },
    Then {
        Say("The wall slides away, revealing a dark passage!"),
        OpenExit("library", "north", "secret_passage"),
        SetFlag("passage_open", true)
    }
)
```

### Effects

`Say`, `GiveItem`, `RemoveItem`, `SetFlag`, `IncCounter`, `SetCounter`, `SetProp`, `MoveEntity`, `MovePlayer`, `OpenExit`, `CloseExit`, `EmitEvent`, `Stop`

### Conditions

`HasItem`, `FlagSet`, `FlagNot`, `FlagIs`, `InRoom`, `PropIs`, `CounterGt`, `CounterLt`, `Not`

## Project Structure

```
cmd/questcore/     Entry point
cli/               Terminal I/O, meta-commands (/save, /load, /help)
engine/
  parser/          Command string → Intent (verb/object/target)
  resolve/         Entity name → entity ID (room-scoped, partial matching)
  rules/           Rules pipeline: collect → filter → rank → select → effects
  effects/         ApplyEffects: the single point of state mutation
  events/          Event emission and handler dispatch
  dialogue/        NPC topic system
  state/           State struct, property lookups, entity helpers
  save/            JSON serialization
  engine.go        Step() orchestrator wiring it all together
types/             Shared data types (no logic)
loader/            Lua VM, sandbox, compile, validate
games/             Example game content
```

## Architecture

Seven invariants the engine never violates:

1. **Lua is compile-time only.** VM runs once at load, then discarded.
2. **Rules engine is a pure function.** `(state, intent) → effects`.
3. **Effects are the instruction set.** Small, fixed, atomic operations.
4. **First match wins.** Room → target entity → object entity → global → fallback.
5. **State is flat.** Flags are bools, counters are ints, no deep nesting.
6. **Engine knows nothing about game content.** All behavior comes from Lua.
7. **Determinism.** Same state + same command + same RNG seed = identical result.

## License

MIT — see [LICENSE](LICENSE).
