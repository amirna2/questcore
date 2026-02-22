# Product Requirements Document

**Project:** QuestCore: A Terminal Adventure & RPG Engine
**Goal:** Build a deterministic, developer-friendly engine to create classic 80s/90s-style adventure games (King’s Quest / Leisure Suit Larry) and D&D-style dungeon crawlers.

---

# 1. Product Vision

A **text-based game engine** that:

* Uses a **constrained command interface** (not full natural language)
* Supports **state-driven puzzles and exploration**
* Supports **optional combat and RPG mechanics**
* Is **deterministic, testable, and replayable**
* Separates **engine logic from game content**

---

# 2. Non-Goals (important)

We explicitly do NOT build:

* Full natural language parser (Inform-style)
* GUI engine / graphics rendering
* Full D&D ruleset fidelity
* Real-time simulation (turn-based only)
* LLM-driven game logic (optional later)

---

# 3. Target Experience

### Adventure mode (KQ / LSL)

* Explore rooms
* Solve puzzles via item interactions
* Talk to NPCs
* Discover hidden dependencies
* Rich text feedback for actions

### Dungeon mode (D&D-lite)

* Explore dungeon
* Fight enemies
* Manage HP, items, equipment
* Gain loot and progress

---

# 4. Core Principles

1. **Deterministic**

   * Same inputs → same outputs
   * Enables replay and testing

2. **Data-driven**

   * Game content defined in YAML/JSON
   * Engine remains generic

3. **Simple grammar**

   * Verb + object [+ target]
   * No complex NLP

4. **Explicit over implicit**

   * No magic rules (avoid Inform-style hidden behavior)

5. **Composable systems**

   * Adventure and RPG features share same core

---

# 5. Core Game Loop

```text
Render state → Player input → Parse → Action → Apply → Events → Output → Repeat
```

Formal:

```text
(state, command) → (new_state, events, output_lines)
```

---

# 6. System Architecture

## 6.1 State Model

Single source of truth:

```python
state = {
  player: {...},
  entities: {...},
  flags: {...},
  counters: {...},
  world: {...}
}
```

---

## 6.2 Entities

Everything is an entity:

* player
* NPCs
* items
* environment objects (door, chest, etc.)

Entities are defined by **properties** (component-like):

```python
door = {
  location: "hall",
  locked: True,
  openable: True
}
```

---

## 6.3 Locations

Graph-based world:

```python
room = {
  id: "hall",
  description: "...",
  exits: {
    north: "throne_room"
  }
}
```

---

## 6.4 Player

```python
player = {
  location: "hall",
  inventory: [],
  stats: {
    hp: 10,
    strength: 5
  }
}
```

---

# 7. Command System

## 7.1 Grammar (MVP)

```text
look
look at <object>
go <direction>
take <object>
drop <object>
use <object> [on <target>]
talk <npc>
attack <target>
inventory
examine <object>
```

Aliases allowed:

* `l`, `i`, `n`, `s`, etc.

---

## 7.2 Parsing Output

```python
Intent {
  verb: "use",
  object: "key",
  target: "door"
}
```

---

# 8. Action System

## 8.1 Action Pipeline

1. Parse intent
2. Resolve entities
3. Validate action
4. Apply effects
5. Emit events
6. Generate output

---

## 8.2 Action Result

```python
ActionResult {
  new_state,
  events: [],
  output: []
}
```

---

# 9. Event System

## 9.1 Event Emission

```python
emit("door_unlocked")
```

## 9.2 Event Handlers

```python
on_event("door_unlocked"):
  set_flag("castle_access", True)
```

## 9.3 Built-in triggers

* `on_enter(room)`
* `on_take(item)`
* `on_use(item, target)`
* `on_talk(npc)`
* `on_turn`
* `on_death(entity)`

---

# 10. Rules System (Conditions + Effects)

This is the heart of the engine.

## 10.1 Conditions

* has_item
* flag_is
* counter_gt / lt
* entity_property
* location
* random (optional)

## 10.2 Effects

* set_flag
* inc_counter
* give_item
* remove_item
* move_entity
* modify_stat
* damage / heal
* spawn / despawn
* print text
* start_dialogue
* start_combat

---

# 11. Action Resolution Order

Critical for flexibility:

1. Specific rule match (room + verb + object + target)
2. Entity-specific rule
3. Global rule
4. Default fallback

---

# 12. Dialogue System

## 12.1 Topics

NPCs expose topics:

```yaml
topics:
  greeting:
    text: "Hello traveler"
  quest:
    requires: flag_met_npc
    text: "I need help..."
```

## 12.2 Unlocking

Topics appear based on conditions.

---

# 13. Combat System (Dungeon Mode)

Minimal, not full D&D.

## 13.1 Stats

* hp
* attack
* defense

## 13.2 Actions

* attack
* defend
* use item
* flee

## 13.3 Turn system

* player acts
* enemies act
* `on_turn` triggers

---

# 14. Inventory System

* add/remove items
* capacity optional
* equipment slots optional

---

# 15. Save / Load

* serialize full state
* reload state
* deterministic replay support

---

# 16. Replay / Debugging

Log commands:

```text
take key
go north
use key door
```

Re-run for debugging.

---

# 17. Content Format (MVP)

YAML or JSON.

## Includes:

* rooms
* entities
* rules
* dialogues

---

# 18. CLI Interface

* Input line
* Output text
* Optional:

  * colored keywords
  * command suggestions

---

# 19. MVP Scope (strict)

To avoid overbuilding:

## Required

* rooms + navigation
* items + inventory
* use item on target
* flags + conditions
* rules engine
* dialogue topics
* save/load

## Optional (v2)

* combat
* counters (money, score)
* procedural generation

---

# 20. Example Game (Validation)

Build a small demo:

### Scenario

* 6–10 rooms
* 5–10 items
* 1 NPC
* 2 puzzles
* optional combat encounter

If that works → engine is viable.

---

# 21. Success Criteria

You can:

* define a game entirely in data
* run it in terminal
* replay a session deterministically
* unit-test game logic
* support both puzzle and combat scenarios

---

# 22. Future Extensions (don’t build yet)

* scripting language (Lua / Python)
* LLM-assisted dialogue
* multiplayer / networked play
* GUI / web client
* modding tools

---

# 23. Key Risks

1. Overbuilding parser → keep grammar simple
2. Overengineering ECS → keep state simple
3. Complex rules → keep conditions/effects minimal
4. Mixing adventure + RPG too early → stage features

---

# Bottom line

Your product is:

> **A deterministic, data-driven, command-based game engine for adventure and RPG text games**

Not:

* a language (Inform)
* a UI tool (Twine)
* a roguelike engine

---

next design step is:

→ define the **actual data schema (YAML)** and
→ define the **core modules (code architecture)**

