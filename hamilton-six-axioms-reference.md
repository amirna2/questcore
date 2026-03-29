# Hamilton's Six Axioms of Control — Code Analysis Reference

> Derived from Margaret H. Hamilton's "Universal Systems Language for Preventative Systems Engineering" (CSER 2007) and the empirical study of the Apollo on-board flight software. These axioms define the formal relationships of control between a parent and its children in any system hierarchy.

---

## Axiom 1 — Control of Invocation

A parent controls the invocation of **only its immediate children**. The children collectively perform **no more and no less** than the parent's requirements. A parent cannot invoke itself, its own parent, its grandchildren, or siblings. No child should be extraneous — if removing a lower-level function leaves the parent's behavior unchanged, that function violates this axiom.

**What to look for:**

- Functions calling grandchildren directly (skipping levels of abstraction)
- Dead code or unreachable functions
- Functions whose removal wouldn't change parent behavior (extraneous functions proliferate test cases and complicate interfaces)
- Circular invocation (a child invoking its parent)
- A function invoking siblings — children of its own parent that it has no business calling

---

## Axiom 2 — Control of Output Responsibility (Codomain)

A parent is responsible for producing the correct output for every valid input. It may delegate work to children, but **it cannot delegate this responsibility**. The parent must ensure delivery of its output. A parent loses control when any of its children stop before completion, enter an endless loop, or fail to return required information.

**What to look for:**

- Children that stop before completion without the parent handling it (uncaught exceptions, early returns with no value)
- Children that can enter endless loops or deadlocks (blocking the parent from ever producing output)
- Children that fail to return required information to the parent (**severed return paths**, fire-and-forget calls where results are needed)
- Error swallowing — catching exceptions and continuing without producing the correct output
- Missing or incomplete return values from child functions

---

## Axiom 3 — Control of Output Access Rights

A parent controls **which output variables each immediate child may alter**. Each output variable of the parent must appear as an output of at least one child. Outputs are traceable instance by instance. The parent assigns to its children the right to alter specific output variables — no more, no less.

**What to look for:**

- Children writing to variables/state outside their assigned scope (global state mutation, writing to shared mutable state without parent mediation)
- Parent outputs that no child produces (dead output paths)
- Untraceable output modifications (side effects hidden in nested calls)
- Multiple children writing to the same output without parent-controlled coordination
- Children modifying state that belongs to other children or other parents

---

## Axiom 4 — Control of Input Access Rights

A parent controls **which input variables each immediate child may read**. Children receive inputs **for reference only** — they cannot alter the parent's inputs. Each input of the parent must be consumed by at least one child. The parent does not have the ability to alter its own domain elements either.

**What to look for:**

- Children mutating input parameters or input state (pass-by-reference mutation, modifying shared input objects)
- Parent inputs that no child ever reads (unused parameters)
- Children accessing inputs not granted by the parent (reaching into global state, environment variables, or closures for data that should flow through the interface)
- A parent modifying its own inputs before delegating to children

---

## Axiom 5 — Control of Error Detection and Rejection (Domain Validation)

A parent **must detect and reject** any input that is not in its valid domain. If an invalid input is received, the parent must ensure its rejection — not silently pass it to children, not log a warning and continue, not downgrade to a default.

**What to look for:**

- Missing input validation at function/module boundaries
- Errors being swallowed or downgraded (caught exceptions with no meaningful handling)
- Invalid inputs propagating silently to children
- Warnings logged where errors should be raised (the warning-without-recovery anti-pattern)
- Missing reject/error paths for edge cases
- Defensive programming that masks invalid state instead of rejecting it

---

## Axiom 6 — Control of Ordering and Priority

A parent controls the **execution order and priority** of its immediate children and their descendants. Priority is totally ordered: a parent's priority is always higher than its children's. Among siblings, ordering is deterministic and controlled by the parent. A process cannot interrupt itself or its parent. Dependent functions must exist at the same level.

**What to look for:**

- Race conditions from uncontrolled concurrent access
- Missing priority/ordering in task scheduling
- Children that can block or starve their parent
- Non-deterministic execution order where determinism is required
- Missing synchronization between dependent siblings
- Priority inversion (a child effectively taking priority over its parent)
- Dependent functions scattered across different levels of the hierarchy

---

## Cross-Axiom Derived Rules

These violations emerge from the interaction of multiple axioms:

### Output/Input Set Separation (Axioms 3 + 4)
A function's output variables **cannot** also be its input variables. If `f(y, x) = y` exists, access to `y` is uncontrolled by the parent at the next higher level. Outputs of one function can be inputs of another function only if both are at the same level.

### Completeness of Return Paths (Axioms 1 + 2)
Every invocation path must return control and data to the parent. **Severed return paths are the most critical violation pattern.** If a child is invoked and cannot return its result to the parent, the parent has lost control of its own output responsibility.

### Single Reference / Single Assignment (Axioms 3 + 4 + 6)
Each variable's relationships are predetermined instance by instance — no aliasing conflicts, no concurrent uncontrolled writes. SOOs can be defined independent of execution order because of this property.

### Nodal Family Independence (Axioms 1 + 4)
A parent and its children do not know about (are independent of) their invokers or users. This means a function should not behave differently based on who called it — it should depend only on its declared inputs.

---

## Usage with Claude Code

### Single File Scan
```
Scan this file against each of Hamilton's six axioms.
For each axiom, report: PASS, WARN, or FAIL with specific line numbers and the violation pattern.
Reference: /path/to/hamilton-six-axioms-reference.md
```

### Cross-Repository Trace
```
Trace the complete call path from [entry point] to [terminal operation].
At each boundary crossing (function call, service call, message publish),
evaluate whether Axiom 2 (output responsibility) is maintained —
does the parent ensure delivery of its output, or is the return path severed?
```

### Severity Rating Guide

| Severity | Meaning | Example |
|----------|---------|---------|
| **CRITICAL** | Axiom violation that can cause silent data loss or unrecoverable state | Severed return path (Ax 1+2), error swallowing (Ax 5) |
| **HIGH** | Axiom violation that degrades system reliability under stress | Race condition (Ax 6), uncontrolled global mutation (Ax 3) |
| **MEDIUM** | Axiom violation that complicates maintenance and traceability | Extraneous functions (Ax 1), unused parameters (Ax 4) |
| **LOW** | Structural smell suggesting potential axiom drift | Functions with too many responsibilities, deep nesting |

---

## Origin

These axioms were derived from the empirical study of the Apollo on-board flight software, where interface errors (data flow, priority, and timing errors) accounted for approximately 75% of all errors found. The axioms define a formal foundation such that **the entire class of interface errors is eliminated by construction** — the "Development Before the Fact" paradigm. Hamilton's key insight: the root problem with traditional approaches is that they support "fixing wrong things up" rather than "doing things in the right way in the first place."

> *"Correctness is accomplished by how a system is defined, by 'built-in' language properties."*
> — Margaret H. Hamilton
