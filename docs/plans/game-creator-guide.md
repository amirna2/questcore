# Plan: Lua Authoring Guide for Game Creators

## Context

QuestCore's engine is complete and the Lost Crown example game is playable, but there's no documentation for someone who wants to *write a game*. Right now they'd need to reverse-engineer `loader/api.go` and the Lost Crown source. A proper authoring guide makes QuestCore usable by anyone — not just us.

## Approach

Create a single comprehensive document at `docs/lua-authoring-guide.md` that serves as the complete reference for game creators. It should be tutorial-flavored at the start (getting a minimal game running) and reference-flavored toward the end (complete API tables).

## Document Structure

### 1. Introduction
- What QuestCore is (one paragraph)
- How Lua fits in: declarative tables, compile-time only, no runtime scripting
- What you need: a directory of `.lua` files, the `questcore` binary

### 2. Quick Start — Your First Game
- Minimal 2-room game walkthrough (game.lua + rooms.lua)
- How to run it: `questcore games/my_game`
- What the player sees

### 3. File Structure & Loading
- Directory layout convention (game.lua, rooms.lua, items.lua, npcs.lua, rules.lua)
- Load order: game.lua first, then alphabetical
- All files share the same global namespace

### 4. Game Metadata — `Game {}`
- Required fields: `title`, `start`
- Optional fields: `author`, `version`, `intro`

### 5. Rooms — `Room "id" {}`
- `description`, `exits`, `fallbacks`, `rules`
- Exit directions (compass + up/down)
- Dynamic exits via `OpenExit`/`CloseExit`

### 6. Entities
- **Items** — `Item "id" {}`: name, description, location, takeable, custom props
- **NPCs** — `NPC "id" {}`: name, description, location, topics, custom props
- **Generic** — `Entity "id" {}`: for scenery objects with rules
- Property override system (runtime state overrides base definitions)

### 7. Rules — The Heart of the Engine
- `Rule("id", When{...}, conditions, Then{...})`
- 3-arg vs 4-arg form
- Resolution order: room → target entity → object entity → global
- Specificity ranking and first-match-wins
- Scoping rules to rooms and entities via `rules = { rule_marker }`
- Pattern: specific rule first, fallback second

### 8. When — Matching Player Intent
- `verb`, `object`, `target`
- `object_kind`, `object_prop`, `target_prop`
- `priority` for tiebreaking

### 9. Conditions Reference (table)
- `HasItem(id)`, `FlagSet(name)`, `FlagNot(name)`, `FlagIs(name, bool)`
- `InRoom(id)`, `PropIs(entity, prop, value)`
- `CounterGt(name, n)`, `CounterLt(name, n)`
- `Not(condition)`
- All conditions are AND'd

### 10. Effects Reference (table)
- Output: `Say(text)`
- Inventory: `GiveItem(id)`, `RemoveItem(id)`
- State: `SetFlag(name, bool)`, `IncCounter(name, n)`, `SetCounter(name, n)`, `SetProp(entity, prop, value)`
- Movement: `MoveEntity(entity, room)`, `MovePlayer(room)`
- World: `OpenExit(room, dir, target)`, `CloseExit(room, dir)`
- Events: `EmitEvent(type)`
- Dialogue: `StartDialogue(npc)`
- Control: `Stop()`

### 11. Template Variables in `Say()`
- `{verb}`, `{object}`, `{target}`
- `{player.location}`, `{player.inventory}`
- `{room.description}`
- `{object.name}`, `{object.description}`, `{target.name}`

### 12. Events & Handlers — `On()`
- Built-in events emitted by effects
- Custom events via `EmitEvent`
- Handler structure: conditions + effects
- Single-pass (no recursion)

### 13. NPC Dialogue — Topics
- Topic structure: `text`, `requires`, `effects`
- How `talk <npc>` and `talk <npc> about <topic>` work
- Condition-gated topics for progression

### 14. Built-in Verbs & Behavior
- Table of all recognized verbs with aliases
- Which have built-in behavior (go, look, examine, take, drop, inventory, wait, talk)
- Which are rule-only (attack, open, close, push, pull, etc.)
- How rules can override built-in behavior

### 15. Patterns & Recipes
- Conditional item acquisition (key + lock)
- Quest progression with flags
- Scenery (non-entity objects in descriptions)
- Multi-state interactions (examine before/after)
- Winning conditions with event handlers
- Room-scoped fallback messages

### 16. Validation Errors & Debugging
- Complete list of fatal errors and what causes them
- Warnings and what they mean
- Meta-commands: `/trace`, `/state` for debugging
- Tips for testing your game

## Files to Create
- `docs/lua-authoring-guide.md` — the guide itself (single file, ~800-1000 lines)

## Files to Reference (read-only, for accuracy)
- `loader/api.go` — all Lua constructor signatures
- `loader/compile.go` — field mappings
- `loader/validate.go` — validation error messages
- `engine/engine.go` — built-in verb behavior
- `engine/parser/parser.go` — verb aliases
- `engine/effects/effects.go` — effect types and templates
- `engine/rules/rules.go` — resolution order
- `engine/dialogue/dialogue.go` — topic mechanics
- `games/lost_crown/` — example code to reference/excerpt

## Verification
- All Lua function signatures match `loader/api.go` exactly
- All effect types match `engine/effects/effects.go`
- All condition types match `loader/api.go` registerConditionHelpers
- All template variables match `engine/effects/effects.go` interpolation
- All verb aliases match `engine/parser/parser.go`
- All validation errors match `loader/validate.go`
- Quick Start example should be a valid game that loads and runs
