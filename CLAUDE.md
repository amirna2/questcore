# CLAUDE.md — QuestCore Project Guide

## What This Is

QuestCore is a deterministic, data-driven, command-based game engine for text
adventure and RPG games (King's Quest / Leisure Suit Larry / D&D-lite). Go
engine, Lua content, JSON saves. This is a real project, not a toy — treat it
as production-quality software engineering.

Read `docs/DESIGN.md` before making any architectural decisions. It is the source of
truth for architecture, data model, rules engine, Lua API, module structure, and
scope. If something contradicts the design doc, stop and ask before proceeding.

## Expertise Required

You are operating as:

- **Game engine architect.** You understand state machines, entity systems,
  rule engines, turn-based game loops, and command parsing. You know the
  difference between data and logic, and you keep them separated.
- **Go engineer.** Idiomatic Go: small interfaces, explicit error handling,
  table-driven tests, no unnecessary abstractions. You write Go that reads
  like Go, not Java-in-Go or Python-in-Go.
- **Lua integration specialist.** You understand embedding Lua in a host
  application, sandboxing, the Lua C API model (as exposed by gopher-lua),
  and the difference between using Lua as a config language vs. a scripting
  runtime.
- **Software engineer.** You write testable, maintainable, well-structured
  code. You respect module boundaries, avoid circular dependencies, and keep
  the dependency graph clean.

## Architecture Invariants (do not violate)

These are non-negotiable. If a change would break one of these, it's wrong.

1. **Lua is compile-time only.** Lua runs once at load, compiles to Go structs,
   VM is discarded. Zero Lua execution during gameplay. No exceptions.
2. **Rules engine is a pure function.** `(state, intent) -> effects`. Rules
   produce effects. They do not mutate state. `ApplyEffects` is the single
   point of mutation.
3. **Effects are the instruction set.** Small, fixed, dumb. Each effect is one
   atomic operation. No logic in effects. If you think you need a new effect
   type, think twice, then think again.
4. **First match wins.** Rule resolution stops at the first matching rule.
   Resolution order: room -> target entity -> object entity -> global -> fallback.
5. **State is flat.** No deep nesting, no inheritance, no class hierarchies.
   Flags are bools, counters are ints, entity state is property overrides.
6. **Engine knows nothing about game content.** The engine is generic. All
   game-specific behavior comes from Lua-defined rules, entities, and rooms.
7. **Determinism.** Same state + same command + same RNG seed = identical
   result. Always. If something breaks this, it's a bug.

## Go Conventions

### Style
- Follow standard Go conventions: `gofmt`, `go vet`, `golint` clean.
- Use `context.Context` where appropriate for cancellation, not everywhere.
- Errors are values. Handle them explicitly. No `panic` in library code.
- No `init()` functions. Explicit initialization only.
- Prefer returning errors over logging and continuing.

### Naming
- Package names are short, lowercase, single-word where possible.
- Exported types and functions have doc comments.
- Avoid stutter: `rules.Rule` not `rules.RulesRule`.
- Use `New` prefix for constructors: `state.NewState()`, `rules.NewEngine()`.

### Structure
- Follow the module structure in `docs/DESIGN.md` section 14. Packages under
  `engine/` are sub-packages with clear, single responsibilities.
- `types/` holds shared data types with no logic. No business logic in types.
- `loader/` handles all Lua interaction. No Lua imports anywhere else.
- Keep the dependency graph acyclic: `cmd -> cli -> engine <- loader`, with
  `types` at the bottom.

### Interfaces
- Keep interfaces small (1-3 methods). Define them where they're consumed,
  not where they're implemented.
- Don't create interfaces preemptively. Start with concrete types. Extract
  interfaces when you need them for testing or polymorphism.

## Testing Strategy

This is a game engine. Correctness matters more than coverage metrics. Test
the things that would be painful to debug if they broke.

### What to Test

- **Rules engine pipeline.** This is the heart of the engine. Test each
  pipeline step independently: matching, condition evaluation, specificity
  ranking, resolution order. Use table-driven tests heavily.
- **Effect application.** Every effect type gets a test. Verify state
  mutations are correct and isolated.
- **Parser.** Table-driven: input string -> expected Intent. Cover aliases,
  prepositions, direction shortcuts, edge cases (empty input, unknown verbs).
- **Entity resolution.** Ambiguity detection, room-scoped lookups, inventory
  lookups.
- **Lua loader.** Load a known Lua content directory, verify compiled Go
  structs match expectations. Test sandbox enforcement (ensure dangerous
  globals are removed).
- **Validation.** Test that invalid content produces the right errors. Missing
  room references, duplicate rule IDs, invalid effect types.
- **Save/load round-trip.** Serialize state to JSON, deserialize, verify
  equality. Test that saves don't include definitions.
- **Deterministic replay.** Load game, play a sequence of commands, record
  output. Replay same commands, verify identical output.
- **Integration: small game scenarios.** Load a minimal Lua game (2-3 rooms,
  a few items, a couple of rules), play through a sequence, assert on state
  and output at each step.

### How to Test

- Use Go's built-in `testing` package. No test frameworks.
- Table-driven tests for anything with multiple cases.
- Test files live next to the code they test: `rules_test.go` next to `rules.go`.
- Use `testdata/` directories for Lua fixtures and expected outputs.
- Keep tests fast. No network, no disk I/O except reading test fixtures.
- Name test cases descriptively: `TestRuleResolution_RoomRuleBeatsGlobal`,
  not `TestCase1`.
- Test edge cases: empty inventory, unknown entity, rule with no conditions,
  room with no exits.

### What NOT to Test

- Don't test Go standard library behavior (JSON marshaling works).
- Don't test trivial getters/setters.
- Don't chase 100% coverage. Test the logic that matters.

## Implementation Approach

### Build Order

Follow this order. Each layer builds on the previous and is independently
testable before moving on:

1. **`types/`** — Define all shared data types. This is the vocabulary of the
   engine. Get it right first.
2. **`engine/state/`** — State struct, property lookups with override layering.
3. **`engine/parser/`** — Command string -> Intent. Independently testable.
4. **`engine/resolve/`** — Entity name -> ID resolution.
5. **`engine/rules/`** — The pipeline: collect, filter, rank, select, produce.
   This is the hardest part. Take time here.
6. **`engine/effects/`** — `ApplyEffects`. Centralized mutation.
7. **`engine/events/`** — Event emission and single-pass handler dispatch.
8. **`loader/`** — Lua VM, sandbox, constructors, compile, validate.
9. **`engine/dialogue/`** — Topic system.
10. **`engine/save/`** — JSON serialization.
11. **`engine/engine.go`** — `Step()` function wiring everything together.
12. **`cli/`** — Terminal I/O, formatting, meta-commands.
13. **`cmd/questcore/`** — Entry point.
14. **`games/lost_crown/`** — Example game content in Lua.

### General Principles

- **Make it work, then make it right.** Get the pipeline flowing end-to-end
  with a minimal game before polishing anything.
- **No premature abstraction.** Three similar lines of code is better than a
  premature helper. Extract when you see a real pattern, not a hypothetical one.
- **Read the code before changing it.** Understand existing patterns and
  conventions before adding new code. Consistency matters.
- **Small, focused commits.** Each commit should do one thing. "Add parser
  with tests" not "add parser, rules, and half of effects".
- **No dead code.** Don't leave commented-out code, unused imports, or
  placeholder functions. If it's not used, delete it.
- **No TODO comments without a plan.** If something needs to be done later,
  it should be in the design doc under v2, not scattered in code comments.

### Error Handling

- Loader errors (bad Lua, missing references) are **fatal** — refuse to start.
- Runtime errors during gameplay should **never crash**. Produce a player-visible
  error message and continue. The game loop must be resilient.
- Use typed errors or sentinel errors where callers need to distinguish cases.
- Wrap errors with context: `fmt.Errorf("loading room %s: %w", id, err)`.

### Performance

- This is a turn-based text game. Performance is not a concern.
- Clarity beats optimization. Always.
- Do not optimize unless you've measured a problem.

## Lua Content Conventions

- Lua helpers (`Room`, `Item`, `Rule`, `When`, `Then`, etc.) are pure
  constructors. They build tables. They execute no logic.
- Validate all Lua output during the compile step. Once compiled to Go structs,
  the data is trusted.
- Test Lua content loading with representative fixtures in `testdata/`.

## What Not to Build

Respect the MVP scope in `docs/DESIGN.md` section 15. These do not exist yet:

- Combat system
- Scripting escape hatch (`function(ctx)`)
- Container entities
- Light/dark rooms
- Timed events
- Anything not in the "Must Have" list

If you think the MVP needs something not on the list, raise it for discussion
before implementing.

## File Reference

| File | Purpose |
|------|---------|
| `docs/DESIGN.md` | Architecture and technical design (source of truth) |
| `docs/questcore.md` | Original PRD (product context and vision) |
| `CLAUDE.md` | This file (engineering guide and conventions) |
