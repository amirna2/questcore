# TUI Plan — QuestCore Terminal UI Upgrade

## Overview

Replace the plain-text CLI with a structured terminal UI using Bubble Tea.
The goal is a better gameplay experience — persistent status information,
styled text, input history — without touching any engine code.

### Current state

The CLI (`cli/cli.go`) is a simple loop: print prompt, read line, call
`Engine.Step()`, print `result.Output`. No colors, no layout, no input
history. All output is `fmt.Fprintln` to a flat `io.Writer`.

### Target state

A full-screen TUI with three regions:

```
┌─────────────────────────────────────────────┐
│                                             │
│  The great hall stretches before you...     │
│  You see: quest scroll.                     │
│  Exits: east, south.                        │
│                                             │
│  > south                                    │
│                                             │
│  The throne room is grand but somber...     │
│  You see: old book, Scholar Elara.          │
│  Exits: east, north, south, west.           │
│                                             │
│                                             │
│                                             │
│                                             │
│                                             │
│                                             │
├─────────────────────────────────────────────┤
│ Hall  │ Exits: n,s,e │ Inv: key, book │ T:5 │
├─────────────────────────────────────────────┤
│ > talk to elara                             │
└─────────────────────────────────────────────┘
```

- **Narrative pane** (top, scrollable): game output with styled text
- **Status bar** (fixed, 1-2 lines): room name, exits, inventory summary,
  turn count — always visible
- **Input line** (bottom): prompt with line editing and history

---

## Design Principles

1. **Engine is untouched.** The TUI is a new rendering layer. It calls
   `Engine.Step()` and reads `types.Result` exactly as the plain CLI does.
   No changes to engine, rules, effects, types, or loader.

2. **Plain CLI preserved.** The existing plain-text CLI stays as a `--plain`
   flag (or pipe detection). Useful for accessibility, testing, scripting,
   and CI. All existing CLI tests continue to pass against the plain mode.

3. **Incremental delivery.** Each phase is independently useful and shippable.
   We don't need to build everything before it's an improvement.

4. **Dependency budget is small.** Bubble Tea + Lip Gloss from charmbracelet.
   These are the standard Go TUI libraries — well-maintained, widely used,
   minimal transitive dependencies.

---

## Phase 1: Core TUI layout + styled text

The minimum viable TUI. Replaces the default game experience.

### 1.1 New dependencies

```
github.com/charmbracelet/bubbletea   — Elm-architecture TUI framework
github.com/charmbracelet/lipgloss    — Terminal styling (colors, borders, layout)
github.com/charmbracelet/bubbles     — Reusable components (text input, viewport)
```

### 1.2 Package structure

```
tui/
├── tui.go        # Bubble Tea model, Update, View — main game TUI
├── styles.go     # Lip Gloss style definitions
├── status.go     # Status bar component
├── narrative.go  # Scrollable narrative viewport
└── input.go      # Input line with history
```

New package `tui/` alongside `cli/`. The `cli/` package is unchanged.

### 1.3 Layout

Three regions, top to bottom:

| Region | Height | Behavior |
|--------|--------|----------|
| Narrative pane | Fill remaining | Scrollable viewport. New output appends at bottom, auto-scrolls. Player can scroll up to re-read. |
| Status bar | 1 line, fixed | Current room name, available exits (abbreviated), inventory count or item list, turn count. |
| Input line | 1 line, fixed | `> ` prompt. Line editing (cursor movement, backspace, delete). Input history (up/down arrows). |

### 1.4 Text styling

Apply distinct styles to different output types. This requires classifying
output lines, which can be done by convention since the engine produces
structured output:

| Content | Style |
|---------|-------|
| Room descriptions | Default/white |
| "You see: ..." lines | Item names highlighted (bold or accent color) |
| "Exits: ..." lines | Dim / muted |
| NPC dialogue (quotes) | Italic or distinct color (e.g. yellow) |
| System messages `[...]` | Dim gray |
| Error messages | Red |
| Player input (echoed) | Green, prefixed with `> ` |

For Phase 1, classification is heuristic (prefix matching on "You see:",
"Exits:", lines starting with `'` or `"`, `[` prefix). This is good enough
— the engine output format is stable and predictable.

### 1.5 Meta-command handling

Meta-commands (`/save`, `/load`, `/quit`, etc.) are handled by the TUI model
directly, reusing the same logic as the plain CLI. `/quit` sends a
`tea.Quit` message. `/help` renders into the narrative pane.

### 1.6 Entry point changes

`cmd/questcore/main.go` gets a `--plain` flag:

```go
flag.BoolVar(&plain, "plain", false, "use plain text output (no TUI)")
```

- Default (no flag): TUI mode via `tui.Run(eng, defs)`
- `--plain` flag: current CLI mode via `cli.New(eng, defs).Run()`
- Pipe detection: if stdout is not a terminal, auto-fall back to plain mode

### 1.7 Deliverables

- [ ] `tui/` package with working Bubble Tea game loop
- [ ] Styled narrative output (colors for different line types)
- [ ] Persistent status bar with room, exits, inventory, turn count
- [ ] Input line with history (up/down arrow)
- [ ] `--plain` flag to preserve existing CLI behavior
- [ ] Pipe/redirect detection for automatic plain fallback
- [ ] Existing CLI tests unmodified and passing

---

## Phase 2: Side panel for inventory + stats

Adds a right-side panel for persistent game state display. This becomes
essential when stats/combat are introduced.

### 2.1 Layout change

```
┌────────────────────────────────┬────────────┐
│                                │ INVENTORY  │
│  Narrative pane                │ ────────── │
│  (scrollable)                  │ rusty key  │
│                                │ old book   │
│                                │            │
│                                │ STATS      │
│                                │ ────────── │
│                                │ HP: 20/20  │
│                                │ Gold: 15   │
│                                │            │
├────────────────────────────────┴────────────┤
│ Hall │ Exits: n,s,e │ T:5                   │
├─────────────────────────────────────────────┤
│ > _                                         │
└─────────────────────────────────────────────┘
```

### 2.2 Side panel content

The side panel reads directly from `Engine.State`:

- **Inventory section:** `State.Player.Inventory` — resolved to display names
- **Stats section:** `State.Player.Stats` — when stats exist (HP, gold, etc.)
- **Flags/counters:** Optional debug view when trace mode is active

The panel updates after every `Step()` call. No new engine API needed — all
data is already on `types.State` and `state.Defs`.

### 2.3 Responsive width

- Terminal < 80 cols: no side panel, inventory stays in status bar only
- Terminal 80-120 cols: narrow side panel (inventory only)
- Terminal > 120 cols: full side panel (inventory + stats)

### 2.4 Deliverables

- [ ] Right-side panel component
- [ ] Inventory display with entity name resolution
- [ ] Stats display (reads `Player.Stats` map)
- [ ] Responsive layout based on terminal width
- [ ] Panel toggle keybinding (e.g. `Tab` or `F2`)

---

## Phase 3: Input enhancements

### 3.1 Tab completion

Context-aware completion for:
- **Verbs:** `look`, `take`, `examine`, `go`, `talk`, `ask`, `use`, etc.
- **Visible entities:** items and NPCs in the current room
- **Inventory items:** for `drop`, `use`, `give`
- **Directions:** for `go` — only exits available from current room
- **Meta-commands:** `/save`, `/load`, `/help`, etc.

Completion source is built from `Engine.State` + `Engine.Defs` after each
turn. No engine changes needed.

### 3.2 Keybindings

| Key | Action |
|-----|--------|
| `Enter` | Submit command |
| `Up/Down` | Input history |
| `Tab` | Auto-complete |
| `Ctrl+L` | Clear narrative / scroll to bottom |
| `PgUp/PgDn` | Scroll narrative |
| `Esc` | Cancel current input |

### 3.3 Deliverables

- [ ] Tab completion engine
- [ ] Completion popup / inline suggestion display
- [ ] Keyboard shortcut help in status bar or on `F1`

---

## Phase 4: Visual polish

### 4.1 Room title headers

When entering a new room, display a styled header:

```
━━━ The Great Hall ━━━━━━━━━━━━━━━━━━━━━━━━━━
The great hall stretches before you...
```

This requires rooms to have a `name` property (separate from `id`). If not
present, fall back to the room ID formatted nicely.

### 4.2 Dividers between turns

Visual separator between turns in the narrative so scrollback is easier
to read:

```
You take the rusty key.
───
> go north

━━━ The Garden ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
A peaceful garden with...
```

### 4.3 Color theme

Define a small color palette that works on both light and dark terminals.
Use Lip Gloss adaptive colors:

```go
var (
    RoomDesc    = lipgloss.AdaptiveColor{Light: "#333", Dark: "#ccc"}
    ItemHighlight = lipgloss.AdaptiveColor{Light: "#0066cc", Dark: "#66aaff"}
    NPCDialogue = lipgloss.AdaptiveColor{Light: "#996600", Dark: "#ffcc00"}
    SystemMsg   = lipgloss.AdaptiveColor{Light: "#888", Dark: "#666"}
    ErrorMsg    = lipgloss.AdaptiveColor{Light: "#cc0000", Dark: "#ff4444"}
    StatusBg    = lipgloss.AdaptiveColor{Light: "#e0e0e0", Dark: "#333333"}
)
```

### 4.4 Deliverables

- [ ] Room name headers on room entry
- [ ] Turn dividers in narrative
- [ ] Adaptive color theme (light/dark terminal support)
- [ ] Optional: configurable theme file

---

## Architecture notes

### Dependency graph

```
cmd/questcore
    ↓
    ├→ tui    → engine (read-only: Step, State, Defs)
    ├→ cli    → engine (unchanged)
    └→ loader → types

tui and cli are siblings — neither depends on the other.
engine, types, loader are completely untouched.
```

### What changes per package

| Package | Changes |
|---------|---------|
| `tui/` | **New.** All TUI code lives here. |
| `cmd/questcore/` | Add `--plain` flag, wire TUI as default. |
| `cli/` | None. Preserved as-is for `--plain` mode. |
| `engine/` | None. |
| `types/` | None. |
| `loader/` | None. |

### State access pattern

The TUI reads game state for the status bar and side panel:

```go
// After each Step():
room := eng.Defs.Rooms[eng.State.Player.Location]
inv  := eng.State.Player.Inventory
exits := state.RoomExits(eng.State, eng.Defs, eng.State.Player.Location)
turn := eng.State.TurnCount
```

This is all read-only. The TUI never mutates state — only `Engine.Step()`
does that (via `ApplyEffects`). This preserves the engine's determinism
invariant.

### Testing strategy

- **Plain CLI tests:** unchanged, continue to pass as-is.
- **TUI unit tests:** test the Bubble Tea model's `Update` function with
  synthetic messages (key presses, window resize). Verify model state
  transitions without rendering.
- **Style tests:** snapshot tests for styled output if needed, but
  mostly visual verification.
- **Integration:** the TUI calls the same `Engine.Step()` as the CLI.
  Engine integration tests already cover correctness.

---

## Implementation order

Start with Phase 1. Each subsequent phase is an independent follow-up that
builds on the previous layout but doesn't require rework.

**Phase 1 estimated scope:** ~400-600 lines of new Go code in `tui/`,
~10 lines changed in `cmd/questcore/main.go`. Zero lines changed in engine.

---

## Decisions

1. **Room names:** derive from room ID — replace underscores with spaces
   and title-case. `"great_hall"` → `"Great Hall"`. No schema changes
   needed. If a `Name` field is added to `RoomDef` later, prefer it over
   the derived name.

## Open questions

1. **Combat UI:** when combat is eventually added, it will likely need a
   dedicated combat mode in the TUI (enemy HP, action menu). This plan
   doesn't design that yet — it should be planned when combat is scoped.

2. **Mouse support:** Bubble Tea supports mouse events. Could enable
   click-on-exit for navigation, click-on-item for examine. Worth
   considering but not in initial phases.
