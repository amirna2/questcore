# QuestCore Lua Authoring Guide

A complete reference for building text adventure games with QuestCore.

---

## Table of Contents

1. [Introduction](#1-introduction)
2. [Quick Start](#2-quick-start--your-first-game)
3. [File Structure & Loading](#3-file-structure--loading)
4. [Game Metadata](#4-game-metadata--game-)
5. [Rooms](#5-rooms--room-id-)
6. [Entities](#6-entities--items-npcs-and-objects)
7. [Rules](#7-rules--the-heart-of-the-engine)
8. [When — Matching Player Intent](#8-when--matching-player-intent)
9. [Conditions Reference](#9-conditions-reference)
10. [Effects Reference](#10-effects-reference)
11. [Template Variables](#11-template-variables-in-say)
12. [Events & Handlers](#12-events--handlers--on)
13. [NPC Dialogue — Topics](#13-npc-dialogue--topics)
14. [Built-in Verbs & Behavior](#14-built-in-verbs--behavior)
15. [Patterns & Recipes](#15-patterns--recipes)
16. [Validation Errors & Debugging](#16-validation-errors--debugging)

---

## 1. Introduction

QuestCore is a deterministic, data-driven game engine for text adventures and
RPGs. You write game content in Lua — rooms, items, NPCs, and rules — and
QuestCore compiles it into an interactive game.

**Key concept:** Lua runs once at load time. Your Lua files declare tables using
helper constructors (`Room`, `Item`, `Rule`, etc.). QuestCore compiles those
tables into Go data structures, then discards the Lua VM. There is zero Lua
execution during gameplay. Everything is declarative.

**What you need:**

- The `questcore` binary
- A directory containing one or more `.lua` files

---

## 2. Quick Start — Your First Game

Create a directory with two files:

**`games/my_game/game.lua`**

```lua
Game {
    title = "My First Game",
    author = "You",
    start = "clearing",
    intro = "You wake up in a forest clearing."
}

Room "clearing" {
    description = "A sunlit clearing surrounded by tall oaks. A path leads north.",
    exits = {
        north = "cave"
    }
}

Room "cave" {
    description = "A dark cave. Water drips from the ceiling. The clearing is to the south.",
    exits = {
        south = "clearing"
    }
}
```

Run it:

```
questcore games/my_game
```

That's it — a two-room game. The player starts in the clearing, can type
`north` to enter the cave and `south` to return. They can `look` to see the
room description and `inventory` to check what they're carrying.

From here, you add items, NPCs, rules, and dialogue to build a full game.

---

## 3. File Structure & Loading

A game is a directory of `.lua` files:

```
games/my_game/
├── game.lua          -- Game metadata (loaded first)
├── rooms.lua         -- Room definitions
├── items.lua         -- Item definitions
├── npcs.lua          -- NPC definitions with dialogue
└── rules.lua         -- Rules and event handlers
```

**Loading order:**

1. `game.lua` is loaded first (if it exists)
2. All other `.lua` files load in alphabetical order

All files share the same Lua global namespace. A variable defined in one file
is visible in all files loaded after it. You can split content across as many
files as you like — one file per room, one giant file, whatever works for you.

The filenames above are conventions, not requirements. QuestCore loads every
`.lua` file in the directory.

---

## 4. Game Metadata — `Game {}`

Every game needs exactly one `Game {}` call.

```lua
Game {
    title   = "The Lost Crown",   -- required
    author  = "QuestCore Team",   -- optional
    version = "0.1.0",            -- optional
    start   = "castle_gates",     -- required (must match a room ID)
    intro   = "The kingdom..."    -- optional (shown when game starts)
}
```

| Field     | Required | Description                        |
|-----------|----------|------------------------------------|
| `title`   | Yes      | Display name of the game           |
| `start`   | Yes      | ID of the room where the player starts |
| `author`  | No       | Author name                        |
| `version` | No       | Version string                     |
| `intro`   | No       | Text shown when the game begins    |

---

## 5. Rooms — `Room "id" {}`

Rooms are the locations the player moves between.

```lua
Room "great_hall" {
    description = "The great hall stretches before you. Faded tapestries line the walls.",
    exits = {
        south = "castle_gates",
        east  = "library",
        north = "throne_room"
    },
    fallbacks = {
        take = "Everything here belongs to the king.",
        open = "There's nothing to open."
    },
    rules = { examine_tapestries }
}
```

| Field         | Type   | Description                                         |
|---------------|--------|-----------------------------------------------------|
| `description` | string | Text shown when the player enters or types `look`   |
| `exits`       | table  | `{ direction = "room_id", ... }`                    |
| `fallbacks`   | table  | `{ verb = "custom error", ... }` for unhandled verbs |
| `rules`       | array  | Rule markers to scope rules to this room             |

### Exit Directions

Supported directions: `north`, `south`, `east`, `west`, `northeast`,
`northwest`, `southeast`, `southwest`, `up`, `down`.

All exits are validated at load time — the target room must exist.

### Dynamic Exits

Exits can be opened and closed at runtime by rules using `OpenExit()` and
`CloseExit()` effects. See [Effects Reference](#10-effects-reference).

### Fallback Messages

When a player uses a verb in a room and no rule matches, the engine checks the
room's `fallbacks` table. If the verb has an entry, that message is shown
instead of the generic default.

---

## 6. Entities — Items, NPCs, and Objects

Entities are the things in the world: items to pick up, NPCs to talk to, and
objects to interact with.

### Items

```lua
Item "rusty_key" {
    name        = "rusty key",
    description = "A small iron key, rough with rust.",
    location    = "castle_gates",
    takeable    = true               -- default for items; can omit
}
```

| Property      | Type   | Default | Description                                 |
|---------------|--------|---------|---------------------------------------------|
| `name`        | string | —       | Display name (shown in inventory, room lists) |
| `description` | string | —       | Text shown when player examines the item    |
| `location`    | string | —       | Room ID where the item starts               |
| `takeable`    | bool   | `true`  | Whether the player can pick it up           |

Items default to `takeable = true`. Set `takeable = false` for items that
require a rule to obtain (like an item locked in a case).

### NPCs

```lua
NPC "captain" {
    name        = "Captain Aldric",
    description = "The captain of the guard, hand on his sword.",
    location    = "castle_gates",
    topics = {
        greet = {
            text = "'Adventurer. The king awaits you.'",
            effects = { SetFlag("met_captain", true) }
        },
        crown = {
            text = "'The crown vanished three nights ago.'",
            requires = { FlagSet("met_captain") }
        }
    }
}
```

NPCs have the same properties as items plus a `topics` table for dialogue. See
[NPC Dialogue](#13-npc-dialogue--topics).

### Generic Entities

```lua
Entity "painting" {
    name        = "old painting",
    description = "A faded landscape.",
    location    = "entrance"
}
```

Use `Entity` for objects that are neither items nor NPCs — scenery, furniture,
or anything the player can see but not pick up.

### Custom Properties

You can add any property you want to an entity:

```lua
Item "locked_box" {
    name     = "locked box",
    location = "room1",
    locked   = true,      -- custom
    contents = "treasure"  -- custom
}
```

Custom properties are accessible in conditions with `PropIs()` and can be
changed at runtime with `SetProp()`.

### Property Overrides

At runtime, effects like `SetProp()` override base properties without changing
the original definition. The engine checks runtime overrides first, then falls
back to the base definition.

---

## 7. Rules — The Heart of the Engine

Rules define what happens when a player does something. A rule matches a player
intent and produces effects.

```lua
Rule("take_gem_with_key",
    When { verb = "take", object = "gem" },
    { HasItem("key") },
    Then {
        Say("You pry the gem loose with the key."),
        GiveItem("gem")
    }
)
```

### Anatomy of a Rule

```lua
Rule(id, when, [conditions], then)
```

| Argument     | Type   | Required | Description                                |
|--------------|--------|----------|--------------------------------------------|
| `id`         | string | Yes      | Globally unique rule identifier            |
| `when`       | table  | Yes      | Match criteria (see [When](#8-when--matching-player-intent)) |
| `conditions` | array  | No       | Conditions that must be true (see [Conditions](#9-conditions-reference)) |
| `then`       | table  | Yes      | Effects to produce (see [Effects](#10-effects-reference)) |

Two call forms are supported:

```lua
-- With conditions (4-argument form):
Rule("id", When{...}, { conditions... }, Then{...})

-- Without conditions (3-argument form):
Rule("id", When{...}, Then{...})
```

### Rule Scoping

Rules can be scoped to rooms or entities. `Rule()` returns a marker that you
include in a room or entity's `rules` array:

```lua
-- Define the rule (global by default).
local examine_painting = Rule("examine_painting",
    When { verb = "examine", object = "painting" },
    Then { Say("A beautiful landscape.") }
)

-- Scope it to a room.
Room "gallery" {
    description = "An art gallery.",
    rules = { examine_painting }
}
```

When scoped, the rule only matches when the player is in that room (or
interacting with that entity).

### Resolution Order

When multiple rules could match, the engine evaluates them in this order:

1. **Room-scoped rules** (rules in the current room's `rules` array)
2. **Target entity rules** (if the command has a target)
3. **Object entity rules** (if the command has an object)
4. **Global rules** (rules not scoped to any room or entity)

Within each scope, rules are ranked by:

1. **Specificity** — more specific matchers rank higher (target + object beats
   object alone)
2. **Priority** — explicit `priority` field in When (higher wins)
3. **Source order** — rules defined earlier win ties

**First match wins.** The engine stops at the first rule whose When matches and
whose conditions are all true.

### Common Pattern: Specific Before Fallback

Define the more specific rule first, then a fallback:

```lua
-- Specific: player has the key
Rule("take_gem_with_key",
    When { verb = "take", object = "gem" },
    { HasItem("key") },
    Then { Say("You pry the gem loose."), GiveItem("gem") }
)

-- Fallback: player doesn't have the key
Rule("take_gem_fail",
    When { verb = "take", object = "gem" },
    Then { Say("The gem is stuck. You need a tool."), Stop() }
)
```

The first rule has more conditions, so it's more specific and evaluates first.
If its conditions fail, the fallback fires.

---

## 8. When — Matching Player Intent

The `When` block defines what player command triggers a rule.

```lua
When {
    verb        = "take",                       -- match this verb
    object      = "gem",                        -- match this entity as object
    target      = "pedestal",                   -- match this entity as target
    object_kind = "item",                       -- match any entity of this kind
    object_prop = { takeable = true },          -- object must have this property
    target_prop = { locked = false },           -- target must have this property
    priority    = 10                            -- tiebreaker (higher wins)
}
```

| Field         | Type   | Description                                           |
|---------------|--------|-------------------------------------------------------|
| `verb`        | string | The verb to match (e.g., "take", "use", "push")       |
| `object`      | string | Specific entity ID to match as the command's object   |
| `target`      | string | Specific entity ID to match as the command's target   |
| `object_kind` | string | Match any entity of this kind ("item", "npc")         |
| `object_prop` | table  | Object must have all these property values             |
| `target_prop` | table  | Target must have all these property values              |
| `priority`    | int    | Tiebreaker when specificity is equal (default: 0)     |

All fields are optional, but you typically specify at least `verb`.

### How Commands Map to Objects and Targets

When a player types a command, the parser splits it into verb, object, and
target using prepositions:

```
use key on door    → verb: "use",  object: "key",  target: "door"
examine painting   → verb: "examine", object: "painting"
push wall          → verb: "push", object: "wall"
give coin to guard → verb: "give", object: "coin", target: "guard"
```

Prepositions used as delimiters: `on`, `at`, `to`, `with`, `in`, `from`,
`about`.

---

## 9. Conditions Reference

Conditions go in the conditions array (the third argument to `Rule()`). All
conditions must be true for the rule to fire (AND logic).

| Condition                            | Description                              |
|--------------------------------------|------------------------------------------|
| `HasItem("entity_id")`              | Player has item in inventory             |
| `FlagSet("flag_name")`              | Boolean flag is true                     |
| `FlagNot("flag_name")`              | Boolean flag is false (or unset)         |
| `FlagIs("flag_name", bool)`         | Flag equals specific value               |
| `InRoom("room_id")`                 | Player is in this room                   |
| `PropIs("entity_id", "prop", val)`  | Entity property equals value             |
| `CounterGt("counter", number)`      | Counter is greater than value            |
| `CounterLt("counter", number)`      | Counter is less than value               |
| `Not(condition)`                     | Negate any condition                     |

### Examples

```lua
-- Player has the key AND is in the armory
{ HasItem("rusty_key"), InRoom("armory") }

-- Flag is set AND player doesn't have the sword
{ FlagSet("quest_given"), Not(HasItem("sword")) }

-- Counter is above a threshold
{ CounterGt("score", 50) }

-- Entity property check
{ PropIs("silver_dagger", "takeable", true) }
```

There is no OR logic. To handle OR cases, write multiple rules.

---

## 10. Effects Reference

Effects go in the `Then {}` block. They execute in order and are atomic — each
one does exactly one thing.

### Output

| Effect            | Description                          |
|-------------------|--------------------------------------|
| `Say("text")`     | Display text to the player           |

Say supports [template variables](#11-template-variables-in-say).

### Inventory

| Effect                    | Description                          |
|---------------------------|--------------------------------------|
| `GiveItem("entity_id")`  | Add item to player inventory         |
| `RemoveItem("entity_id")`| Remove item from player inventory    |

### State

| Effect                                    | Description                                |
|-------------------------------------------|--------------------------------------------|
| `SetFlag("name", bool)`                  | Set a boolean flag                         |
| `IncCounter("name", amount)`             | Increment counter by amount (can be negative) |
| `SetCounter("name", value)`              | Set counter to exact value                 |
| `SetProp("entity_id", "prop", value)`    | Override an entity property at runtime     |

### Movement

| Effect                          | Description                          |
|---------------------------------|--------------------------------------|
| `MoveEntity("entity_id", "room_id")` | Move an entity to a room       |
| `MovePlayer("room_id")`              | Teleport the player to a room  |

### World

| Effect                                       | Description                            |
|----------------------------------------------|----------------------------------------|
| `OpenExit("room_id", "direction", "target")` | Make an exit available in a room      |
| `CloseExit("room_id", "direction")`          | Remove an exit from a room            |

### Events

| Effect                       | Description                              |
|------------------------------|------------------------------------------|
| `EmitEvent("event_type")`   | Trigger event handlers (see [Events](#12-events--handlers--on)) |

### Dialogue

| Effect                        | Description                    |
|-------------------------------|--------------------------------|
| `StartDialogue("npc_id")`    | Begin dialogue with an NPC     |

### Control Flow

| Effect   | Description                                              |
|----------|----------------------------------------------------------|
| `Stop()` | Stop processing effects and suppress default output      |

Use `Stop()` when a rule partially handles something and you want to prevent
the engine from showing a default message.

---

## 11. Template Variables in `Say()`

`Say()` text can include template variables that are replaced at runtime:

| Template               | Resolves To                              |
|------------------------|------------------------------------------|
| `{verb}`               | The parsed verb (e.g., "take")           |
| `{object}`             | The resolved object entity ID            |
| `{target}`             | The resolved target entity ID            |
| `{player.location}`    | Current room ID                          |
| `{player.inventory}`   | Formatted list of carried item names     |
| `{room.description}`   | Current room's description text          |
| `{object.name}`        | Object entity's `name` property          |
| `{object.description}` | Object entity's `description` property   |
| `{target.name}`        | Target entity's `name` property          |

### Example

```lua
Say("You examine {object.name}. {object.description}")
Say("You are in {player.location} carrying: {player.inventory}.")
```

Template variables also work in effect parameters like `GiveItem("{object}")`
to dynamically reference the matched entity.

---

## 12. Events & Handlers — `On()`

Events let you trigger side effects in response to things that happen during
a turn.

### Defining a Handler

```lua
On("event_type", {
    conditions = { InRoom("library") },   -- optional
    effects = {
        Say("A cold draft rushes out from the darkness.")
    }
})
```

| Field        | Type  | Required | Description                               |
|--------------|-------|----------|-------------------------------------------|
| `conditions` | array | No       | Conditions that must be true for handler to fire |
| `effects`    | array | No       | Effects to apply when handler fires        |

### Built-in Events

These events are emitted automatically by the engine:

| Event           | Emitted When                    |
|-----------------|---------------------------------|
| `item_taken`    | `GiveItem()` effect executes    |
| `item_dropped`  | `RemoveItem()` effect executes  |
| `flag_changed`  | `SetFlag()` effect executes     |
| `entity_moved`  | `MoveEntity()` effect executes  |
| `room_entered`  | `MovePlayer()` effect executes  |

### Custom Events

Use `EmitEvent("my_event")` in a rule's effects to trigger your own events:

```lua
Rule("take_crown",
    When { verb = "take", object = "lost_crown" },
    Then {
        Say("You lift the Lost Crown!"),
        GiveItem("lost_crown"),
        EmitEvent("crown_recovered")
    }
)

On("crown_recovered", {
    effects = {
        Say("=== CONGRATULATIONS ==="),
        Say("You have recovered the Lost Crown!")
    }
})
```

### Single-Pass Execution

Event handlers run once after all rule effects are applied. Handler effects do
not trigger additional events — there is no recursion.

---

## 13. NPC Dialogue — Topics

NPCs can have dialogue topics that the player accesses with the `talk` command.

### Defining Topics

```lua
NPC "scholar" {
    name     = "Scholar Elara",
    location = "library",
    topics = {
        greet = {
            text = "'Ah, the adventurer. The answer lies in the books.'",
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

| Field      | Type   | Required | Description                                  |
|------------|--------|----------|----------------------------------------------|
| `text`     | string | Yes      | What the NPC says                            |
| `requires` | array  | No       | Conditions for this topic to be available    |
| `effects`  | array  | No       | Effects when player selects this topic       |

### How Players Use Dialogue

- **`talk scholar`** — auto-plays the first available topic (alphabetically,
  for determinism)
- **`talk scholar about passage`** — plays a specific topic if its conditions
  are met
- If the player asks about an unavailable topic, available topics are listed
  as hints

Topics with unmet `requires` conditions are hidden from the player.

---

## 14. Built-in Verbs & Behavior

The parser recognizes many verbs. Some have built-in engine behavior that fires
when no rule matches. Others are purely rule-driven.

### Verbs with Built-in Behavior

These verbs do something even without rules:

| Verb        | Built-in Behavior                                        |
|-------------|----------------------------------------------------------|
| `go`        | Move player through exits. Shows room description.       |
| `look`      | Describe current room (entities, exits).                 |
| `examine`   | Show entity's `description` property.                    |
| `read`      | Same as `examine`.                                       |
| `take`      | Pick up item if `takeable = true`.                       |
| `drop`      | Remove item from inventory, place in current room.       |
| `inventory`  | List carried items.                                     |
| `talk`      | Activate NPC dialogue system.                            |
| `wait`      | "Time passes." (advances turn counter)                   |

**Rules can override any built-in behavior.** If a rule matches, it fires
instead of the built-in.

### Rule-Only Verbs

These verbs have no built-in behavior — they require rules to do anything:

`attack`, `open`, `close`, `push`, `pull`, `give`, `throw`, `use`, `eat`,
`drink`, `smell`, `listen`, `touch`, `climb`, `jump`, `unlock`, `tie`, `untie`,
`wear`, `wave`, `sing`, `pray`, `sleep`, `knock`, `yell`, `swim`, `buy`

### Verb Aliases

Players can type natural variations. The parser normalizes them:

| Player Types                                         | Parsed As   |
|------------------------------------------------------|-------------|
| `l`                                                  | `look`      |
| `x`, `inspect`, `check`, `study`, `observe`, `describe`, `search` | `examine` |
| `walk`, `run`, `move`, `head`, `proceed`, `enter`, `travel` | `go`        |
| `get`, `grab`, `hold`, `carry`, `catch`              | `take`      |
| `discard`                                            | `drop`      |
| `hit`, `fight`, `strike`, `kill`, `punch`, `kick`, `smash`, `destroy`, `break` | `attack` |
| `ask`, `speak`, `chat`, `converse`, `say`, `tell`    | `talk`     |
| `shut`                                               | `close`     |
| `press`, `shove`, `shift`                            | `push`      |
| `drag`, `tug`, `yank`                                | `pull`      |
| `offer`, `hand`, `feed`                              | `give`      |
| `toss`, `hurl`, `lob`                                | `throw`     |
| `consume`, `taste`, `bite`, `devour`                 | `eat`       |
| `sip`, `swallow`, `quaff`                            | `drink`     |
| `sniff`                                              | `smell`     |
| `hear`                                               | `listen`    |
| `feel`, `rub`                                        | `touch`     |
| `scale`                                              | `climb`     |
| `leap`, `hop`                                        | `jump`      |
| `don`                                                | `wear`      |
| `nap`, `rest`                                        | `sleep`     |
| `rap`                                                | `knock`     |
| `scream`, `shout`                                    | `yell`      |
| `dive`                                               | `swim`      |
| `purchase`                                           | `buy`       |
| `i`, `inv`                                           | `inventory` |
| `z`                                                  | `wait`      |

### Multi-Word Phrases

The parser also expands natural multi-word phrases:

| Player Types                     | Parsed As                    |
|----------------------------------|------------------------------|
| `look at X`, `look in X`, `look under X` | `examine X`         |
| `pick up X`                      | `take X`                     |
| `talk to X`, `speak with X`     | `talk X`                     |
| `put on X`                       | `wear X`                     |
| `put down X`                     | `drop X`                     |
| `take off X`                     | `remove X`                   |
| `turn on X`, `switch on X`      | `activate X`                 |
| `turn off X`, `switch off X`    | `deactivate X`               |

### Direction Shortcuts

Players can type directions directly without `go`:

`n`, `s`, `e`, `w`, `ne`, `nw`, `se`, `sw`, `u`, `d`, or the full names
(`north`, `south`, etc.) — all equivalent to `go <direction>`.

---

## 15. Patterns & Recipes

Common patterns you'll use when building games.

### Locked Item Requiring a Key

```lua
Item "silver_dagger" {
    name     = "silver dagger",
    location = "armory",
    takeable = false                 -- can't just pick it up
}

Rule("use_key_on_dagger",
    When { verb = "use", object = "rusty_key", target = "silver_dagger" },
    { HasItem("rusty_key"), InRoom("armory") },
    Then {
        Say("You unlock the display case and take the silver dagger."),
        RemoveItem("rusty_key"),
        SetProp("silver_dagger", "takeable", true),
        GiveItem("silver_dagger"),
        SetFlag("case_unlocked", true)
    }
)
```

### Quest Progression with Flags

```lua
-- Step 1: Meet the NPC
NPC "captain" {
    topics = {
        greet = {
            text = "'The crown is missing!'",
            effects = { SetFlag("met_captain", true) }
        },
        clue = {
            text = "'Check the library.'",
            requires = { FlagSet("met_captain") }   -- only after greeting
        }
    }
}

-- Step 2: Find a clue
Rule("read_book",
    When { verb = "read", object = "old_book" },
    { HasItem("old_book") },
    Then {
        Say("The book describes a hidden passage."),
        SetFlag("found_clue", true)
    }
)

-- Step 3: Use the clue
Rule("push_wall",
    When { verb = "push", object = "wall" },
    { InRoom("library"), FlagSet("found_clue") },
    Then {
        Say("The wall slides open!"),
        OpenExit("library", "north", "secret_passage")
    }
)
```

### Scenery — Non-Entity Objects

For objects mentioned in room descriptions but not defined as entities, write
rules that match on the raw noun:

```lua
Room "great_hall" {
    description = "A massive fireplace dominates the north wall."
}

Rule("examine_fireplace",
    When { verb = "examine", object = "fireplace" },
    { InRoom("great_hall") },
    Then { Say("The fireplace is cold and dark. Ashes sit in the grate.") }
)
```

The engine will try to match rules using the raw noun even when entity
resolution fails. Always pair scenery rules with an `InRoom()` condition.

If the player examines something mentioned in a visible description and no rule
matches, the engine automatically responds with "You see nothing special about
the X." — this prevents confusing "you don't see that here" messages for things
clearly described in the room text.

### Multi-State Interactions

Use flags to show different responses before and after a state change:

```lua
Rule("examine_pedestal_with_crown",
    When { verb = "examine", object = "pedestal" },
    { InRoom("secret_passage"), FlagNot("crown_found") },
    Then { Say("A stone pedestal. The Lost Crown gleams atop it.") }
)

Rule("examine_pedestal_empty",
    When { verb = "examine", object = "pedestal" },
    { InRoom("secret_passage"), FlagSet("crown_found") },
    Then { Say("The stone pedestal stands empty.") }
)
```

The first rule is more specific (it has `FlagNot("crown_found")` as an
additional condition). Once the flag is set, that condition fails and the second
rule matches instead.

### Winning Condition with Event Handler

```lua
Rule("take_crown",
    When { verb = "take", object = "lost_crown" },
    { InRoom("secret_passage") },
    Then {
        Say("You lift the Lost Crown!"),
        GiveItem("lost_crown"),
        SetFlag("crown_found", true),
        IncCounter("score", 100),
        EmitEvent("crown_recovered")
    }
)

On("crown_recovered", {
    effects = {
        Say(""),
        Say("=== CONGRATULATIONS ==="),
        Say("You have recovered the Lost Crown!"),
        Say("Final score: {score} points.")
    }
})
```

### Room Fallback Messages

Customize error messages for specific verbs in a room:

```lua
Room "throne_room" {
    description = "The throne sits empty on a raised dais.",
    fallbacks = {
        take = "Everything in the throne room belongs to the king."
    }
}
```

When the player tries to `take` something in the throne room and no rule
handles it, they see the custom message instead of the generic default.

---

## 16. Validation Errors & Debugging

### Fatal Errors (Prevent Game from Loading)

| Error | Cause |
|-------|-------|
| `no .lua files found in [dir]` | Empty game directory |
| `no Game{} definition found` | Missing `Game {}` call |
| `Game.Title is required` | `title` field missing from `Game {}` |
| `Game.Start is required` | `start` field missing from `Game {}` |
| `start room "X" not found in defined rooms` | `start` points to nonexistent room |
| `room "X" exit "Y" points to undefined room "Z"` | Exit target doesn't exist |
| `duplicate rule ID "X"` | Two rules have the same ID |
| `unknown condition type "X"` | Typo in condition helper name |
| `unknown effect type "X"` | Typo in effect helper name |
| `condition has_item references undefined entity "X"` | Entity doesn't exist |
| `condition in_room references undefined room "X"` | Room doesn't exist |
| `condition prop_is references undefined entity "X"` | Entity doesn't exist |
| `effect give_item references undefined entity "X"` | Entity doesn't exist |
| `effect remove_item references undefined entity "X"` | Entity doesn't exist |
| `effect set_prop references undefined entity "X"` | Entity doesn't exist |
| `effect move_entity references undefined entity "X"` | Entity doesn't exist |
| `effect move_entity references undefined room "X"` | Room doesn't exist |
| `effect move_player references undefined room "X"` | Room doesn't exist |
| `effect open_exit references undefined room "X"` | Source room doesn't exist |
| `effect open_exit target references undefined room "X"` | Target room doesn't exist |
| `effect close_exit references undefined room "X"` | Room doesn't exist |
| `effect start_dialogue references undefined entity "X"` | Entity doesn't exist |

### Warnings (Non-Fatal)

| Warning | Cause |
|---------|-------|
| `rule "X" uses unrecognized verb "Y"` | Verb not in the parser's known list |
| `entity "X" location "Y" does not match any defined room` | Item placed in nonexistent room |

### Debugging Tools

When running QuestCore, these meta-commands help you debug:

| Command   | Description                                   |
|-----------|-----------------------------------------------|
| `/trace`  | Toggle trace mode (shows rule matching info)  |
| `/state`  | Show current game state (flags, counters, etc) |
| `/save`   | Save the current game                         |
| `/load`   | Load a saved game                             |
| `/help`   | Show available commands                       |
| `/quit`   | Exit the game                                 |

### Tips

- Start small. Get two rooms working before adding 20.
- Test each rule as you write it. Add an item, write a rule, run the game, try
  the command.
- Use `Say()` liberally during development to confirm rules are firing.
- If a rule isn't firing, check: Is the verb correct? Is the object ID an exact
  match? Are all conditions true?
- Remember that `takeable` defaults to `true` for items. If you don't want an
  item to be grabbable, set `takeable = false` explicitly.

---

## Lua Environment

QuestCore runs Lua in a sandbox. You have access to:

- **Standard functions:** `print`, `type`, `tostring`, `tonumber`, `pairs`,
  `ipairs`, `assert`, `error`
- **Table library:** `table.insert`, `table.remove`, `table.concat`,
  `table.sort`
- **String library:** `string.format`, `string.sub`, `string.len`,
  `string.upper`, `string.lower`, etc.
- **Math library:** `math.floor`, `math.ceil`, `math.max`, `math.min`, etc.

You can use loops, conditionals, and helper functions to generate content:

```lua
-- Generate rooms programmatically
local corridor_rooms = { "corridor_1", "corridor_2", "corridor_3" }
for i, id in ipairs(corridor_rooms) do
    Room(id) {
        description = "Corridor section " .. i .. "."
    }
end
```

**Not available:** `dofile`, `loadfile`, `load`, `rawset`, `rawget`, `os`,
`io`, `debug`, `coroutine`, `math.randomseed`. These are removed to enforce
sandboxing and preserve determinism.
