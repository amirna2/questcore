# Hamilton's Six Axioms of Control — QuestCore Analysis Report

**Date:** 2026-03-29
**Scope:** Full Go engine (`cmd/`, `engine/`, `loader/`, `cli/`, `tui/`, `types/`)
**Reference:** `hamilton-six-axioms-reference.md`
**Methodology:** Each axiom evaluated against every package boundary, function call, and data flow path. Severity ratings per the reference guide (CRITICAL / HIGH / MEDIUM / LOW).

---

## Executive Summary

| Axiom | Verdict | Findings |
|-------|---------|----------|
| 1 — Control of Invocation | **PASS** | 1 MEDIUM |
| 2 — Control of Output Responsibility | **WARN** | 2 MEDIUM |
| 3 — Control of Output Access Rights | **WARN** | 1 HIGH, 1 MEDIUM |
| 4 — Control of Input Access Rights | **PASS** | 1 MEDIUM |
| 5 — Control of Error Detection/Rejection | **WARN** | 1 HIGH, 3 MEDIUM |
| 6 — Control of Ordering and Priority | **PASS** | 1 LOW |

**Overall Assessment:** The architecture is remarkably well-structured for axiom compliance. The "Lua compile-time only" invariant, the pure rules pipeline, and the single-point-of-mutation design (`effects.Apply`) are textbook Hamilton-safe patterns. The findings below are mostly edge cases in the combat subsystem and error handling paths — the core engine pipeline is clean.

---

## Axiom 1 — Control of Invocation

> *A parent controls the invocation of only its immediate children.*

### Verdict: **PASS** (1 MEDIUM finding)

The invocation hierarchy is clean. `main` → `loader`/`engine`/`cli`/`tui`. `Engine.Step()` invokes its immediate children (`parser`, `resolve`, `rules`, `effects`, `events`, `dialogue`) without skipping levels. No circular invocations exist. The dependency graph is acyclic by design.

#### Finding 1.1 — `events` package invokes `rules.EvalAllConditions` (cross-sibling call)

| | |
|---|---|
| **Severity** | MEDIUM |
| **Location** | `engine/events/events.go:21` |
| **Pattern** | Sibling invocation |

`events.Dispatch()` directly calls `rules.EvalAllConditions()` to evaluate handler conditions. Both `events` and `rules` are children of `engine`. This is a sibling calling a sibling — technically the parent (`Engine.Step`) should mediate.

**Mitigant:** This is a deliberate reuse of condition evaluation logic. The alternative (duplicating `EvalAllConditions` into `events` or hoisting it into a shared utility) would be worse. The call is read-only and well-bounded. Acceptable pragmatic trade-off.

---

## Axiom 2 — Control of Output Responsibility

> *A parent must ensure delivery of its output. It cannot delegate this responsibility.*

### Verdict: **WARN** (2 MEDIUM findings)

#### Finding 2.1 — `effects.Apply` silently ignores unknown effect types

| | |
|---|---|
| **Severity** | MEDIUM (revised from HIGH — see rationale below) |
| **Location** | `engine/effects/effects.go:228` |
| **Pattern** | Missing domain rejection (Axiom 5), with Axiom 2 consequence |

```go
default:
    // Unknown effect type — ignore silently.
```

**Revised analysis (2026-03-29):** This is primarily an **Axiom 5** violation, not Axiom 2. `Apply`'s domain is `[]types.Effect` where each `Effect.Type` must be one of the known effect types. An `Effect` with an unknown type is an invalid input — Axiom 5 requires the parent to detect and reject it. The Axiom 2 consequence (the parent's output silently vanishes) is downstream of the Axiom 5 failure: if `Apply` had rejected the invalid input, `Engine.Step` would receive the rejection signal and could handle it.

**Two input paths with different risk profiles:**

1. **Lua-defined rules (~95% of effects):** Validated by `loader/validate.go:214` against `validEffectTypes`. The loader's Axiom 5 enforcement guarantees these are valid before the engine ever sees them. Per the Axiom 5 clarification in the reference — *"domain validation at internal boundaries is necessary only when the domain of the receiving function is narrower than or different from what the producing function guarantees"* — no runtime re-validation is needed for this path.

2. **Engine-constructed effects:** Hardcoded string literals in `engine.go:336-337`, `393-395`, `406-409` and `combat.go:120-121`, `135-136`, `151-152`, `171-172`, `202-203`, `214-215`. These bypass the loader entirely. A typo (e.g., `"move_playr"`) would compile, pass existing tests, and silently do nothing at runtime. The risk is narrow (developer typo in Go source) but real.

**Severity downgrade rationale:** The dominant input path (Lua rules) is fully validated upstream. The vulnerable path (engine-constructed effects) uses hardcoded string literals that are unlikely to drift, and the failure mode is visible during development (the effect simply doesn't fire). This is a compile-time-preventable problem, not a runtime data integrity issue.

**Recommendation:** Define effect type constants (`const EffectSay = "say"`, etc.) in `types/` and use them in both `Apply`'s switch and all effect constructors. This eliminates the typo risk at compile time — the Go compiler becomes the Axiom 5 enforcer for path 2. The silent `default` in `Apply` should still log a diagnostic as defense-in-depth.

#### Finding 2.2 — Type assertion failures silently produce zero-values

| | |
|---|---|
| **Severity** | MEDIUM |
| **Location** | `engine/effects/effects.go:30`, `35`, `49`, `58-59`, `77-79`, `89-90`, `100-101`, etc. |
| **Pattern** | Silent degradation / incomplete output |

Throughout `effects.Apply`, parameters are extracted via type assertions with the blank identifier:

```go
text, _ := eff.Params["text"].(string)
item, _ := eff.Params["item"].(string)
flag, _ := eff.Params["flag"].(string)
```

If a param is missing or wrong type, the value silently becomes `""` or `0`. For `say`, this produces empty output. For `set_flag`, this sets the flag named `""`. For `give_item`, this adds `""` to inventory. The parent never learns that its delegated work was corrupted.

**Mitigant:** The Lua loader validates effect parameters at compile time, so malformed effects are unlikely in practice. However, built-in behaviors in `engine.go` construct effects directly (e.g., lines 336-338, 393-395, 407-409), bypassing the loader. A typo in a param key would silently degrade.

**Recommendation:** Consider a lightweight param extraction helper that logs or returns an error for missing required params.

#### Finding 2.3 — `defaultCombatBehavior` returns `(nil, nil)` for unknown combat verbs

| | |
|---|---|
| **Severity** | MEDIUM |
| **Location** | `engine/engine.go:283` |
| **Pattern** | Silent no-op / lost output |

```go
default:
    return nil, nil
```

If `EnemyTurn` returns an unknown verb (e.g., from a malformed behavior table), both the rules pipeline and `defaultCombatBehavior` will miss, and the enemy turn produces no output and no effects. The player sees nothing — the enemy's turn silently vanishes.

**Mitigant:** `loader/validate.go:371` validates behavior actions at load time, so this path is unreachable in practice. But the runtime code has no defense-in-depth.

---

## Axiom 3 — Control of Output Access Rights

> *A parent controls which output variables each immediate child may alter.*

### Verdict: **WARN** (1 HIGH, 1 MEDIUM finding)

#### Finding 3.1 — `defaultCombatDefend` mutates state directly, bypassing the effect system

| | |
|---|---|
| **Severity** | HIGH |
| **Location** | `engine/combat.go:131` |
| **Pattern** | Uncontrolled output mutation / bypassing designated mutation point |

```go
func (e *Engine) defaultCombatDefend(actor string) ([]types.Effect, []string) {
    if actor == "player" {
        e.State.Combat.Defending = true  // <-- DIRECT STATE MUTATION
        return nil, []string{"You brace yourself. (+2 defense this round)"}
    }
```

This is the only place in the entire codebase where a function other than `effects.Apply` directly mutates game state during a turn. The architecture invariant states: *"ApplyEffects is the single point of mutation."* This function violates Axiom 3 (the child writes to state the parent didn't assign it) **and** the project's own architectural invariant #2.

For enemy defending (line 135-137), the code correctly produces a `set_prop` effect and returns it. The player path should do the same — perhaps a `set_stat` or `set_prop` effect on a player combat property.

**Additionally:** `engine/engine.go:215-224` (end-of-round cleanup) directly mutates `Combat.RoundCount`, `Combat.Defending`, and entity props. This is another direct mutation outside `effects.Apply`. While cleanup is arguably a parent-level concern (not delegated), it would be cleaner and more traceable as effects.

#### Finding 3.2 — `Engine.Step` builds `result.Output` from multiple sources without clear ownership

| | |
|---|---|
| **Severity** | MEDIUM |
| **Location** | `engine/engine.go:158-165`, `172-175`, `182-186`, `193-199`, `207-211` |
| **Pattern** | Multiple children writing to same output |

`result.Output` is appended to by: built-in behaviors, `effects.Apply` (twice — once for main effects, once for event effects), loot processing, and enemy turn processing. While the parent (`Step`) mediates all of these sequentially, the output is an append-only accumulator passed through multiple code paths. This is controlled but complex — the ordering of output is implicitly determined by code order, not explicitly by the parent.

**Mitigant:** The sequential nature of the game loop makes this safe. Each append path runs to completion before the next starts. No concurrent access.

---

## Axiom 4 — Control of Input Access Rights

> *A parent controls which input variables each immediate child may read. Children cannot alter the parent's inputs.*

### Verdict: **PASS** (1 MEDIUM finding)

The architecture scores well here. `state.Defs` is immutable after loading. The `types.State` pointer is passed explicitly. No global state or environment variables are accessed during gameplay. Children receive inputs through function parameters.

#### Finding 4.1 — `Engine` struct fields are publicly accessible

| | |
|---|---|
| **Severity** | MEDIUM |
| **Location** | `engine/engine.go:21-25` |
| **Pattern** | Uncontrolled input access |

```go
type Engine struct {
    Defs  *state.Defs
    State *types.State
    RNG   *RNG
}
```

All fields are exported. `cli.CLI` and `tui` access `Engine.State` and `Engine.Defs` directly (e.g., `cli/cli.go:150` — `c.Engine.State`, `cli/cli.go:188` — `save.ApplySave(c.Engine.State, sd)`). This means children (`cli`, `tui`) can read **and write** the parent's state without mediation.

**Mitigant:** In Go, struct field visibility is the primary access control mechanism. The `cli` and `tui` packages are thin presentation layers that legitimately need state access for save/load and status display. The risk is low in a single-binary, single-team project.

---

## Axiom 5 — Control of Error Detection and Rejection

> *A parent must detect and reject any input not in its valid domain.*

### Verdict: **WARN** (1 HIGH, 3 MEDIUM findings)

#### Finding 5.1 — `save.ApplySave` performs no validation of loaded data against current definitions

| | |
|---|---|
| **Severity** | HIGH |
| **Location** | `engine/save/save.go:73-83` |
| **Pattern** | Missing domain validation at system boundary |

```go
func ApplySave(s *types.State, sd *SaveData) {
    s.Player = sd.Player
    s.Flags = sd.Flags
    // ... direct assignment, no validation
}
```

`ApplySave` blindly overwrites state with deserialized data. It does not validate:
- Whether `sd.Player.Location` is a valid room ID
- Whether inventory items reference existing entities
- Whether `sd.Combat.EnemyID` is a valid enemy
- Whether the save's game title/version matches the loaded definitions
- Whether entity state references valid entities

A corrupted or hand-edited save file could put the engine into an invalid state that causes downstream panics (e.g., nil map access in `state.GetEntityProp` if the room doesn't exist).

**Recommendation:** Add a `ValidateSave(sd *SaveData, defs *state.Defs) error` function that checks referential integrity before applying. This is a system boundary (external JSON input) where Axiom 5 demands validation.

#### Finding 5.2 — `loader.sandbox` does not remove `require`

| | |
|---|---|
| **Severity** | MEDIUM |
| **Location** | `loader/loader.go:100-117` |
| **Pattern** | Incomplete domain rejection |

The sandbox removes `dofile`, `loadfile`, `load`, `loadstring`, `rawset`, `rawget`, `rawequal`, and `collectgarbage`. But `require` is not explicitly removed. While `SkipOpenLibs: true` means the package library isn't loaded (so `require` likely doesn't exist), this is defense-in-depth — explicitly removing it would close the gap.

**Mitigant:** `lua.Options{SkipOpenLibs: true}` means the `package` lib (which provides `require`) is never opened. The risk is theoretical.

#### Finding 5.3 — `conditions.go` returns `false` for unknown condition types instead of signaling error

| | |
|---|---|
| **Severity** | MEDIUM |
| **Location** | `engine/rules/conditions.go:80-82` |
| **Pattern** | Silent degradation instead of rejection |

```go
default:
    return false
```

An unknown condition type silently evaluates to `false`, causing its rule to never match. Like Finding 2.1, the loader validates condition types at compile time, but the runtime has no defense-in-depth. An unknown condition should at minimum log a diagnostic, or ideally the type system should make this unreachable.

#### Finding 5.4 — `effects.Apply` does not reject unknown effect types (reclassified from Finding 2.1)

| | |
|---|---|
| **Severity** | MEDIUM |
| **Location** | `engine/effects/effects.go:228` |
| **Pattern** | Missing domain rejection |

See revised Finding 2.1 above. Originally classified as Axiom 2 (Output Responsibility) HIGH. Reclassified as Axiom 5 (Domain Validation) MEDIUM after applying the per-parent domain validation clarification from the updated Hamilton reference. The Axiom 2 consequence (lost output) is downstream of the Axiom 5 root cause (failure to reject invalid input).

---

## Axiom 6 — Control of Ordering and Priority

> *A parent controls the execution order and priority of its immediate children.*

### Verdict: **PASS** (1 LOW finding)

The engine is single-threaded and turn-based. `Step()` executes its children in a strict, deterministic sequence: parse → resolve → rules → effects → events → event-effects → loot → enemy-turn → cleanup. No goroutines, no channels, no concurrent access. The determinism invariant is well-maintained.

#### Finding 6.1 — Event handler ordering depends on slice order from loader

| | |
|---|---|
| **Severity** | LOW |
| **Location** | `engine/events/events.go:17` |
| **Pattern** | Implicit ordering dependency |

```go
for _, handler := range defs.Handlers {
```

Event handlers are iterated in the order they were compiled from Lua. If multiple handlers match the same event, their effects are concatenated in this order. The parent (`Dispatch`) does not explicitly control priority — it inherits order from the loader. This is deterministic (Lua files are sorted, declarations are ordered), but the priority is implicit rather than explicit.

**Mitigant:** The single-pass, no-recursion design bounds the impact. And unlike the rules pipeline (which has explicit specificity/priority ranking), event handlers are intentionally simpler — all matching handlers fire. The current design is adequate for the game's complexity.

---

## Cross-Axiom Analysis

### Output/Input Set Separation (Axioms 3 + 4)

**PASS.** The `types.State` is the only mutable structure. `state.Defs` is immutable. Functions generally read from Defs and write to State, maintaining separation. The one exception is `defaultCombatDefend` (Finding 3.1).

### Completeness of Return Paths (Axioms 1 + 2)

**PASS.** Every function in the engine returns to its caller. No goroutines are spawned during gameplay. No fire-and-forget calls. The `effects.Apply` function returns collected events and output for every code path. The `stop` effect (line 225) is the only early return, and it's explicitly designed to short-circuit.

### Single Reference / Single Assignment (Axioms 3 + 4 + 6)

**PASS.** No concurrent access exists. State is mutated sequentially through a single code path. The `EntityState` value type (Go struct copied from map) prevents aliasing issues.

### Nodal Family Independence (Axioms 1 + 4)

**PASS.** Functions do not vary behavior based on caller identity. `effects.Apply` processes the same effect types regardless of whether they came from rules, built-ins, or event handlers. The `Context.Actor` field is data, not a behavioral switch on caller identity.

---

## Architectural Commendations

These patterns deserve explicit recognition as Hamilton-safe-by-design:

1. **Lua compile-time only (Axiom 5).** By rejecting invalid content at load time and discarding the VM, the entire class of "invalid Lua at runtime" errors is eliminated by construction.

2. **Rules engine as pure function (Axioms 2 + 3).** `rules.Evaluate()` reads state but mutates nothing. It returns data (effects) for the parent to apply. This is textbook Axiom 3 compliance — the child produces output within its assigned scope only.

3. **Effects as instruction set (Axiom 3).** By funneling all mutation through `effects.Apply`, the codebase achieves single-point-of-mutation. The parent (`Step`) knows exactly which child can alter state and controls when it runs.

4. **Single-pass event dispatch (Axiom 6).** By not recursing, `events.Dispatch` guarantees bounded execution. The parent always regains control after one pass. No child can starve or block the parent.

5. **Deterministic RNG with position tracking (Axiom 6).** Ordering is deterministic regardless of save/load cycles. `RestoreRNG(seed, position)` perfectly reconstructs the RNG state.

6. **Loader validation pipeline (Axiom 5).** The `validate()` function performs comprehensive domain validation: room references, entity references, duplicate rule IDs, effect/condition types, enemy stats. Invalid content is rejected before the engine ever sees it.

---

## Summary of Recommendations

| # | Finding | Severity | Recommendation |
|---|---------|----------|----------------|
| 2.1 | Silent discard of unknown effects (reclassified: Axiom 5) | MEDIUM | Define effect type constants in `types/`; add diagnostic log in `default` branch |
| 3.1 | Direct state mutation in `defaultCombatDefend` | HIGH | Route player defending through effect system |
| 5.1 | No validation of loaded save data | HIGH | Add `ValidateSave()` at the system boundary |
| 2.2 | Silent zero-values on type assertion failure | MEDIUM | Add param extraction with error reporting |
| 2.3 | Silent no-op on unknown combat verb | MEDIUM | Add fallback output for unrecognized enemy actions |
| 3.2 | Multiple output accumulators in `Step` | MEDIUM | Document output ordering contract (low priority) |
| 4.1 | Exported Engine struct fields | MEDIUM | Consider accessor methods (low priority) |
| 5.2 | `require` not explicitly sandboxed | MEDIUM | Add `require` to sandbox removal list |
| 5.3 | Unknown conditions silently return false | MEDIUM | Log diagnostic for unknown condition types |
| 1.1 | events→rules sibling call | MEDIUM | Acceptable pragmatic trade-off (no action needed) |
| 6.1 | Implicit event handler ordering | LOW | Document ordering contract |
