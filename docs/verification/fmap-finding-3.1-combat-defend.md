# FMap Analysis: Finding 3.1 — Axiom 3+4 Violation in Combat Defend Path

**Date:** 2026-03-29
**Finding:** 3.1 from `hamilton-axioms-report-03-29.md`
**Location:** `engine/combat.go:130`
**Axioms violated:** 3 (Output Access Rights), 4 (Input Access Rights), 6 (Ordering — side-channel dependency)
**Reference:** `hamilton-six-axioms-reference.md`, Figure 2 of Hamilton & Hackler CSER 2007

---

## Variables (Ordered Sets)

| Variable | Role | Description |
|----------|------|-------------|
| `s0` | input | Game state at turn start |
| `cmd` | input | Player command string |
| `intent` | local | Parsed verb + object |
| `objID` | local | Resolved entity ID |
| `effs` | local | Effect list |
| `out0` | local | Text output from combat |
| `s1` | output | Game state at turn end |
| `evts` | output | Emitted events |
| `out1` | output | Text output from Apply |

---

## Intended Structure — Join (Dependent Composition)

```
Step(s0, cmd) = (s1, evts, out0, out1)
 |-- parseCombat(s0, cmd) = (intent, objID)              [left]
 |-- defaultCombatBehavior(s0, intent) = (effs, out0)    [middle, depends on left]
 '-- Apply(s0, effs) = (s1, evts, out1)                  [right, depends on middle]
```

**Axiom 3 output assignments by parent (Step):**

| Child | Assigned outputs |
|-------|-----------------|
| `parseCombat` | `intent`, `objID` |
| `defaultCombatBehavior` | `effs`, `out0` |
| `Apply` | `s1`, `evts`, `out1` |

`s1` is assigned to `Apply` only. No other child may write it.

---

## Actual Structure — Axiom 3+4 Violation

```
defaultCombatBehavior(s0, intent) = (effs, out0)          [parent]
 '-- Or[intent.verb]                                       [decision]
      |-- [verb=="attack"] -> combatAttack(s0, intent) = (effs, out0)     PASS
      |-- [verb=="defend"] -> combatDefend(s0, intent) = (effs, out0)     FAIL
      '-- [verb=="flee"]   -> combatFlee(s0, intent) = (effs, out0)       PASS
```

Decomposing the failing branch:

```
combatDefend(s0, intent) = (effs, out0)                   [parent]
 '-- Or[actor]                                             [decision]
      |-- [actor=="player"]  -> (nil, out0) + WRITES s0    VIOLATION
      '-- [actor!="player"] -> (effs, out0)                PASS
```

### Violation Detail

`combatDefend` was assigned outputs: `(effs, out0)`
`combatDefend` actually writes to: `s0` (`s0.Combat.Defending = true`)

The `[actor=="player"]` branch at `combat.go:130`:
- Returns `effs = nil` (produces no effects)
- Returns `out0 = ["You brace yourself..."]`
- Mutates `s0.Combat.Defending = true` — **uncontrolled write**

The `[actor!="player"]` branch at `combat.go:135-137` (correct):
- Returns `effs = [{set_prop, entity, "defending", true}]`
- Returns `out0 = ["The goblin braces..."]`
- Does not write to `s0` — **controlled, via effs passed to Apply**

### Axiom Violations

**Axiom 4 (Input Access Rights):** `s0` is an input. Children receive inputs for reference only. `combatDefend` mutates `s0.Combat.Defending`, altering the parent's input. The paper states: *"the parent does not have the ability to alter its domain elements"* — neither do its children.

**Axiom 3 (Output Access Rights):** The mutation of state was assigned to `Apply`, not to `combatDefend`. The child writes to a range variable belonging to a sibling. The parent's output variable `s1` must appear as an output of `Apply` only — `combatDefend` was assigned `(effs, out0)`, not `s1`.

**Axiom 6 (Ordering — consequence):** Because `combatDefend` mutates `s0` before `Apply` runs, `Apply` receives an already-modified state. A dependency exists between `combatDefend` and `Apply` that is not expressed in the data flow (effs). The parent's control of ordering is undermined through a side-channel.

### Or Structure Rule Violation

Figure 2 (CSER 2007) requires: *"Outputs of both children are identical to parent outputs (including order)."*

The parent `combatDefend` declares outputs `(effs, out0)`. The `[actor=="player"]` branch returns `(nil, out0)` but smuggles its real work through a side-channel mutation of `s0`. Its effective output is `(s0', out0)` — which does not match the parent's declared output set.

---

## Corrected Structure

```
combatDefend(s0, intent) = (effs, out0)                   [parent]
 '-- Or[actor]                                             [decision]
      |-- [actor=="player"]  -> (effs, out0)               PASS
      |     effs = [{set_prop, "player", "defending", true}]
      |     out0 = ["You brace yourself..."]
      '-- [actor!="player"] -> (effs, out0)                PASS
            effs = [{set_prop, enemyID, "defending", true}]
            out0 = ["The goblin braces..."]
```

Both branches produce `effs` for `Apply` to consume. `s0` is never written by `combatDefend`. `s1` is produced solely by `Apply`. Axioms 3, 4, and 6 are satisfied. Output access rights are restored to parent control.
