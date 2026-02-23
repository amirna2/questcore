# Phase 1: Bubble Tea TUI for QuestCore

## Context

The game currently renders plain text to stdout — no colors, no layout, no
input history. We're adding a full-screen TUI using Bubble Tea with a
scrollable narrative pane, persistent status bar, and styled text. The plain
CLI is preserved as a `--plain` fallback.

Engine code is completely untouched. The TUI is a new rendering layer that
calls `Engine.Step()` and reads state, exactly like the existing CLI.

## Dependencies

Bump `go.mod` to Go 1.24. Add:
- `github.com/charmbracelet/bubbletea` (latest)
- `github.com/charmbracelet/lipgloss` (latest)
- `github.com/charmbracelet/bubbles` (latest — viewport + textinput components)

## Files to create

### `tui/history.go` — Input history ring buffer
- `History` struct with `entries []string`, `max int`, `cursor int`
- `NewHistory(max)`, `Push(cmd)`, `Prev() (string, bool)`, `Next() (string, bool)`, `ResetCursor()`
- Skip consecutive duplicates on Push
- Standalone, no engine dependency

### `tui/style.go` — Line classification + lipgloss styles
- `lineKind` enum: `kindRoomDesc`, `kindYouSee`, `kindExits`, `kindDialogue`, `kindSystem`, `kindError`, `kindTrace`
- `classifyLine(line string) lineKind` — heuristic pattern matching:
  - `"You see:"` prefix → kindYouSee
  - `"Exits:"` prefix → kindExits
  - `"["` prefix + `"]"` suffix → kindSystem
  - `"[trace]"` prefix → kindTrace
  - `"You don't see"` / `"You can't"` / `"You don't have"` → kindError
  - Lines containing substantial `'quoted speech'` → kindDialogue
  - Everything else → kindRoomDesc
- `classifyAndStyle(line) string` — applies lipgloss style per kind
- Style palette: white room desc, bold entity names, dim exits, yellow dialogue, gray system, red errors, green player input

### `tui/status.go` — Status bar rendering
- `roomDisplayName(id string) string` — `"great_hall"` → `"Great Hall"`
- `renderStatusBar() string` — reads `Engine.State` and `Defs`:
  - Room name (derived from ID)
  - Exits (sorted, from `state.RoomExits()`)
  - Inventory count + item names (from `state.GetEntityProp()`)
  - Turn count
  - Rendered as a full-width inverted-color bar

### `tui/tui.go` — Bubble Tea model, the main file
- **Model struct:**
  - `engine *engine.Engine`, `defs *state.Defs`
  - `viewport viewport.Model` (from bubbles — scrollable narrative)
  - `input textinput.Model` (from bubbles — prompt with cursor)
  - `history *History`
  - `lines []string` (accumulated styled narrative lines)
  - `width`, `height`, `ready`, `trace`, `quitting`, `saveDir`, `lastCmd`

- **Layout** (three vertical regions):
  ```
  ┌─────────────────────────────────────┐
  │ Narrative viewport (fills space)    │ ← scrollable, PgUp/PgDn
  ├─────────────────────────────────────┤
  │ Hall | Exits: n,s | Inv: 2 | T:5    │ ← 1-line status bar
  ├─────────────────────────────────────┤
  │ > player input_                     │ ← 1-line text input
  └─────────────────────────────────────┘
  ```

- **Init():** returns cmd that produces intro text + first `look` result
- **Update():**
  - `WindowSizeMsg` → create/resize viewport (height = terminal - 2)
  - `KeyMsg "enter"` → process input synchronously (engine is fast):
    - Handle "again"/"g" repeat
    - Handle meta-commands (`/save`, `/load`, `/quit`, `/help`, `/state`, `/trace`)
    - Call `engine.Step(input)`, style output, append to viewport
  - `KeyMsg "up"/"down"` → history navigation
  - `KeyMsg "pgup"/"pgdown"` → forward to viewport for scrolling
  - `KeyMsg "ctrl+c"` → quit
  - `gameOutputMsg` → append styled lines, auto-scroll viewport to bottom
- **View():** `viewport.View() + "\n" + statusBar + "\n" + input.View()`
- **Viewport keymap:** disable Up/Down (used for history), keep PgUp/PgDn/Ctrl+U/Ctrl+D
- **Meta-commands:** same behavior as `cli/cli.go`, but return `[]string` instead of printing. Reuse `engine/save` package for save/load.

- **`Run(eng, defs) error`** — creates Model, runs `tea.NewProgram(m, tea.WithAltScreen())`

### `tui/tui_test.go` — Tests
- `TestRoomDisplayName` — table-driven: ID → display name
- `TestClassifyLine` — table-driven: line → expected lineKind
- `TestHistory_*` — Push, Prev, Next, max size, no duplicates
- `TestHandleMeta_*` — save, load, quit, help dispatching

## Files to modify

### `cmd/questcore/main.go`
- Add `--plain` flag (simple arg parsing, no flag package)
- Add `isTerminal()` using `os.Stdout.Stat()` with `os.ModeCharDevice`
- Default: TUI mode via `tui.Run(eng, defs)`
- `--plain` or piped stdout: existing CLI via `cli.New(eng, defs).Run()`
- Move game title print to plain-mode only (TUI shows it in the narrative)

## Files NOT modified
- `engine/` — all files untouched
- `types/` — untouched
- `loader/` — untouched
- `cli/` — untouched, all existing tests pass

## Implementation order
1. `go get` dependencies, bump go.mod to 1.24
2. `tui/history.go` — standalone, testable immediately
3. `tui/style.go` — depends only on lipgloss
4. `tui/status.go` — depends on engine/state, lipgloss
5. `tui/tui.go` — main model, wires everything
6. `tui/tui_test.go` — tests for all above
7. `cmd/questcore/main.go` — flag + TUI dispatch

## Verification
```bash
go build ./...                    # compiles
go vet ./...                      # clean
go test ./...                     # all tests pass (existing + new)
go run ./cmd/questcore ./games/lost_crown          # TUI mode
go run ./cmd/questcore --plain ./games/lost_crown  # plain mode
echo "look" | go run ./cmd/questcore ./games/lost_crown  # pipe → plain fallback
```

Manual TUI checks: viewport scrolls, status bar updates on room change,
input history with up/down, /save and /load work, /quit exits cleanly,
PgUp/PgDn scroll narrative.
