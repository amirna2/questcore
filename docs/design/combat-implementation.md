# Combat System Implementation Plan

## Context

QuestCore's MVP engine is feature-complete (parser, rules, effects, events, save/load, loader, TUI). The combat PRD and technical design (`docs/design/combat-prd.md`, `docs/design/combat-design.md`) define a turn-based 1v1 combat system that fits within the existing pipeline — combat is a game mode flag, not a separate engine. The enemy's turn is just another `Step()` call through the same rules pipeline.

This plan translates the 7-phase, 18-step design into 6 feature branches with concrete file changes and tests.

---

## Branch 1: `feat/combat-foundation` — RNG, Types, State Helpers

### 1.1 Deterministic RNG

**New file:** `engine/rng.go`

- `RNG` struct wrapping `math/rand.Rand` with position tracking
- `NewRNG(seed) *RNG`
- `Roll(sides) int` — 1 to sides inclusive, increments position
- `WeightedSelect(weights []int) int` — returns index, increments position
- `Position() int64` — for save/load
- `RestoreRNG(seed, position) *RNG` — re-seed and advance to position

**New file:** `engine/rng_test.go`

- `TestRNG_Deterministic` — same seed = same sequence
- `TestRNG_Roll_Range` — always 1..sides
- `TestRNG_WeightedSelect_Deterministic`
- `TestRNG_Position_Tracks`
- `TestRNG_Restore_MatchesPosition`

### 1.2 Type Additions

**Modified:** `types/types.go`

- Add `CombatState` struct: `Active bool`, `EnemyID string`, `RoundCount int`, `Defending bool`, `PreviousLocation string`
- Add `BehaviorEntry` struct: `Action string`, `Weight int`
- Add `LootEntry` struct: `ItemID string`, `Chance int`
- Add `PlayerStats map[string]int` to `GameDef`
- Add `Combat CombatState` and `RNGPosition int64` to `State`

**Modified:** `engine/effects/effects.go`

- Add `Actor string` to `Context` (needed later for enemy turns)

### 1.3 State Helpers

**Modified:** `engine/state/state.go`

- `InCombat(s) bool`
- `GetStat(s, defs, target, stat) (int, bool)` — "player" reads `s.Player.Stats`, entity reads via `GetEntityProp` with int coercion
- `SetStat(s, target, stat, value)` — "player" writes `s.Player.Stats`, entity writes via `EntityState.Props`
- Modify `NewState()` to copy `defs.Game.PlayerStats` into `s.Player.Stats`

**Modified:** `engine/state/state_test.go`

- `TestInCombat_Active` / `_Inactive`
- `TestGetStat_Player` / `_Entity_FromDef` / `_Entity_Override` / `_Missing`
- `TestSetStat_Player` / `_Entity`
- `TestNewState_CopiesPlayerStats` / `_NoPlayerStats`

---

## Branch 2: `feat/combat-conditions-effects` — New Conditions & Effects

### 2.1 New Conditions

**Modified:** `engine/rules/conditions.go` — add cases to `EvalCondition` switch:

| Condition | Logic |
|-----------|-------|
| `in_combat` | `s.Combat.Active` |
| `in_combat_with` | `s.Combat.Active && s.Combat.EnemyID == params["entity"]` |
| `stat_gt` | `GetStat(target, stat) > value` |
| `stat_lt` | `GetStat(target, stat) < value` |

**Modified:** `engine/rules/conditions_test.go` — table-driven cases for each condition, both passing and failing.

### 2.2 New Effects

**Modified:** `engine/effects/effects.go` — add cases to `Apply` switch:

| Effect | Behavior |
|--------|----------|
| `start_combat` | Set `s.Combat` fields, copy base enemy stats to `EntityState`, emit `combat_started` |
| `end_combat` | Zero out `s.Combat`, emit `combat_ended` |
| `damage` | Decrement target HP (player or entity), clamp to 0, emit `entity_damaged`. On HP=0: set `alive=false` + emit `enemy_defeated` for enemies, set `game_over` flag + emit `player_defeated` for player |
| `heal` | Increment target HP, clamp to `max_hp`, emit `entity_healed` |
| `set_stat` | Direct stat assignment via `state.SetStat()` |

Helper functions (private): `initEnemyStats()`, `applyDamage()`, `applyHeal()`

**Modified:** `engine/effects/effects_test.go`

- `TestApply_StartCombat` / `_EndCombat`
- `TestApply_Damage_Entity` / `_Entity_Death` / `_Player` / `_Player_Death` / `_ClampsToZero`
- `TestApply_Heal_Player` / `_ClampsToMax` / `_Entity`
- `TestApply_SetStat_Player` / `_Entity`

---

## Branch 3: `feat/combat-engine` — Combat Loop, Damage Calc, Enemy AI

### 3.1 Damage Calculation

**New file:** `engine/combat.go`

```
DamageCalc(attackerAttack, defenderDefense int, defending bool, rng *RNG) (damage, roll int)
  → damage = max(1, roll(1d6) + attack - defense - defendBonus)
```

### 3.2 Enemy AI

In `engine/combat.go`:

```
EnemyTurn(s, defs, rng) Intent
  → reads behavior weights from entity Props
  → weighted random selection via RNG
  → returns Intent{Verb: action}
```

### 3.3 Loot Processing

In `engine/combat.go`:

```
ProcessLoot(s, defs, enemyID, rng) ([]Effect, []string)
  → roll per loot item, give_item if roll <= chance
  → inc_counter for gold
```

### 3.4 Default Combat Behavior

Engine methods (parallel to existing `builtinBehavior` pattern):

- `defaultCombatAttack(actor)` — calc damage, produce `damage` + `say` effects
- `defaultCombatDefend(actor)` — set defending flag, produce `say` effect
- `defaultCombatFlee(actor)` — roll 1d6, on 4+: `end_combat` + `move_player`, on fail: say "can't escape"

These fire only when no author rule matched during combat — lowest priority, same as built-in verbs.

### 3.5 Parser Additions

**Modified:** `engine/parser/parser.go`

- Add aliases: `defend`/`block`/`guard` → `"defend"`, `flee`/`escape` → `"flee"`
- Note: `run` stays mapped to `go`; Step() rewrites `go` → `flee` during combat

### 3.6 Step() Modifications

**Modified:** `engine/engine.go`

- Add `RNG *RNG` field to `Engine`, initialize in `New()`
- At top of `Step()`: if `game_over` flag set, block all commands
- Before resolve: if combat active and verb not in {attack, defend, flee, use, inventory, look}, reject with message. Rewrite `go` → `flee` during combat.
- After rules pipeline: if combat active and no rule matched, call `defaultCombatBehavior()`
- After player effects applied: if combat still active, run `enemyTurn()` through same pipeline (synthesize Intent, evaluate rules, apply effects, dispatch events)
- After enemy turn: increment `RoundCount`, clear defending flags
- Update `s.RNGPosition = e.RNG.Position()` before return
- Add `RestoreRNG()` method for save/load

**New file:** `engine/combat_test.go`

- `TestDamageCalc_*` — formula, minimum 1, defend bonus, deterministic
- `TestEnemyTurn_*` — weighted selection, no behavior defaults to attack, deterministic
- `TestProcessLoot_*` — deterministic drops, gold, no loot, edge cases

**Modified:** `engine/engine_test.go`

- `TestStep_CombatCommandRestriction`
- `TestStep_CombatAttack_PlayerDamagesEnemy`
- `TestStep_CombatAttack_EnemyCounterattacks`
- `TestStep_CombatDefend_ReducesDamage`
- `TestStep_CombatFlee_Success` / `_Failure`
- `TestStep_CombatRoundCount_Increments`
- `TestStep_CombatEnds_OnEnemyDefeat`
- `TestStep_LookDuringCombat_Allowed`
- `TestStep_GameOver_BlocksCommands`

---

## Branch 4: `feat/combat-loader` — Enemy Constructor, Validation

### 4.1 Enemy Lua Constructor

**Modified:** `loader/api.go`

- `Enemy` constructor (curried pattern like `Item`/`NPC`): stores as `rawEntity{kind: "enemy"}`
- Condition helpers: `InCombat()`, `InCombatWith(entity)`, `StatGt(entity, stat, value)`, `StatLt(entity, stat, value)`
- Effect helpers: `StartCombat(enemy)`, `EndCombat()`, `Damage(target, amount)`, `Heal(target, amount)`, `SetStat(target, stat, value)`

### 4.2 Compilation

**Modified:** `loader/compile.go`

- `compileGame()`: extract `player_stats` table → `GameDef.PlayerStats`
- `compileEntity()` for `kind == "enemy"`: extract `stats` table → flat Props (hp, max_hp, attack, defense), compile `behavior` → `[]BehaviorEntry` in Props, compile `loot` → `[]LootEntry` + gold in Props, set `alive = true`

### 4.3 Validation

**Modified:** `loader/validate.go`

- Add `start_combat`, `end_combat`, `damage`, `heal`, `set_stat` to `validEffectTypes`
- Add `in_combat`, `in_combat_with`, `stat_gt`, `stat_lt` to `validConditionTypes`
- Add `defend`, `flee` to `knownVerbs`
- New `validateEnemy()`: required stats present and positive, behavior weights valid, actions known, loot item refs exist, chance 1-100
- Warn: enemy with no behavior, enemies exist but no player_stats
- Validate `start_combat` target is kind "enemy"

**New testdata:** `loader/testdata/combat/` and `loader/testdata/invalid_enemy/`

**Modified:** `loader/compile_test.go`, `loader/validate_test.go`, `loader/loader_test.go`

---

## Branch 5: `feat/combat-save-tui` — Save/Load & TUI Display

### 5.1 Save/Load

**Modified:** `engine/save/save.go`

- Add `Combat types.CombatState` and `RNGPosition int64` to `SaveData`
- Include in `Save()` and `ApplySave()`
- Backward compatible: missing `combat` in old saves defaults to zero value (inactive)

**Modified:** `engine/save/save_test.go`

- `TestRoundTrip_WithCombatState`
- `TestRoundTrip_WithRNGPosition`
- `TestLoad_MissingCombat_DefaultsToInactive`

### 5.2 TUI Combat Display

**New file:** `tui/combat.go` — `renderCombatDisplay()` (health bars), `healthBar()`, `renderGameOver()`

**Modified:** `tui/tui.go` — inject combat display after combat turns, change prompt to `combat> ` during combat, block non-meta commands during game over

**Modified:** `tui/status.go` — show HP in status bar when player has stats

**Modified:** `tui/style.go` — combat border, HP bar, game over styles

---

## Branch 6: `feat/combat-content` — Lost Crown Encounter & Integration Tests

### 6.1 Game Content

**Modified:** `games/lost_crown/game.lua` — add `player_stats`

**New file:** `games/lost_crown/enemies.lua` — cave goblin with stats, behavior, loot

**Modified:** `games/lost_crown/rules.lua` — auto-engage on room entry via `On("room_entered")` handler

### 6.2 Integration Tests

**New file:** `engine/combat_integration_test.go`

- `TestCombatEncounter_FullSequence` — enter room, fight, win, verify loot and state
- `TestCombatEncounter_PlayerDeath` — fight, lose, verify game over
- `TestCombatEncounter_Flee` — fight, flee, verify previous room
- `TestCombatSaveLoad_MidCombat` — save mid-combat, load, resume, verify identical outcome
- `TestCombatReplay_Deterministic` — play encounter, replay same commands, verify identical output
- `TestAuthorOverride_CustomDamage` — author rule overrides default attack

---

## Key Design Decisions

1. **Default combat behavior as Engine methods** (not synthetic RuleDefs) — parallels existing `builtinBehavior` pattern, avoids polluting rule system
2. **RNG lives on Engine, state stores seed + position** — types have no methods by convention; `RestoreRNG()` replays from seed to position for save/load
3. **Behavior/loot stored in entity Props** — keeps entity system unified; existing override layering and save/load work without changes
4. **`PreviousLocation` in CombatState** — simplest way to support flee; set when `start_combat` fires
5. **`run` stays mapped to `go`** — Step() rewrites `go` → `flee` during combat rather than breaking existing alias

## Risks

- **`toGoValue` and typed structs**: behavior/loot must be compiled into typed Go structs (`[]BehaviorEntry`, `[]LootEntry`) before `toGoValue` processes remaining Props. Handle via skip map in `compileEntity`.
- **Enemy turn event cascading**: defeating an enemy via event handler must not trigger another combat in the same step. Mitigated by checking combat state after each effect batch.
- **Save/load backward compat**: zero values for new fields mean "not in combat" / "position 0" — no migration needed.

## Verification

After each branch:
1. `go test ./...` — all tests pass
2. `go vet ./...` — no issues
3. `golangci-lint run` — clean

After branch 6:
4. Load Lost Crown, walk into secret passage, fight goblin through to victory/defeat/flee
5. Save mid-combat, load, verify combat resumes identically
6. Replay same command sequence with same seed, verify identical output

---

## Step-by-Step Task Tracker

Built vertically — each phase ends with something you can run and test.

---

### Phase 1: `feat/combat-basic` — Minimal playable combat (attack + win)
_Goal: type `attack goblin`, see damage, enemy hits back, kill it. Playable in TUI._

- [ ] 1. Types: add `CombatState`, `BehaviorEntry`, `LootEntry` to `types/types.go`; add `PlayerStats` to `GameDef`; add `Combat` + `RNGPosition` to `State`
- [ ] 2. RNG: create `engine/rng.go` + `engine/rng_test.go` — Roll, WeightedSelect, Position, RestoreRNG
- [ ] 3. State helpers: add `InCombat()`, `GetStat()`, `SetStat()` to `engine/state/state.go`; modify `NewState()` to copy PlayerStats; add tests
- [ ] 4. Add `Actor` to `effects.Context`
- [ ] 5. Effects: add `start_combat`, `end_combat`, `damage` to `engine/effects/effects.go` + tests
- [ ] 6. Combat engine: create `engine/combat.go` — `DamageCalc()`, `defaultCombatAttack()` only
- [ ] 7. Engine: add `RNG` to Engine, modify `Step()` — combat command restriction, default attack fallback, enemy turn (attack-only AI for now)
- [ ] 8. Loader: add `Enemy` constructor + `StartCombat`/`EndCombat`/`Damage` effect helpers + `compileGame` player_stats + `compileEntity` enemy stats to `loader/`
- [ ] 9. Validation: add new types to valid maps, basic enemy stat validation
- [ ] 10. Content: add `player_stats` to Lost Crown `game.lua`, create `enemies.lua` with cave goblin (stats only, no behavior/loot yet), add attack trigger rule
- [ ] 11. `go test ./...` — all green
- [ ] 12. **PLAYTEST**: run game, go to goblin room, `attack goblin`, trade hits, win

### Phase 2: `feat/combat-actions` — Defend, flee, player death
_Goal: full combat action set. Defend reduces damage, flee escapes, dying = game over._

- [ ] 13. Parser: add `defend`/`block`/`guard` and `flee`/`escape` aliases
- [ ] 14. Effects: add `heal`, `set_stat` effects + tests
- [ ] 15. Default combat: add `defaultCombatDefend()`, `defaultCombatFlee()` to `engine/combat.go`
- [ ] 16. Step: add `go` → `flee` rewrite in combat, game over check at top of Step
- [ ] 17. Conditions: add `in_combat`, `in_combat_with`, `stat_gt`, `stat_lt` to conditions.go + tests
- [ ] 18. Loader: add condition helpers (`InCombat`, `StatGt`, etc.) + `Heal`/`SetStat` effect helpers
- [ ] 19. `go test ./...` — all green
- [ ] 20. **PLAYTEST**: defend (take less damage), flee (escape to previous room), die (see game over)

### Phase 3: `feat/combat-ai-loot` — Enemy AI + loot drops
_Goal: enemy uses weighted behavior (attack/defend/flee), drops loot on death._

- [ ] 21. Enemy AI: add `EnemyTurn()` with weighted selection to `engine/combat.go`
- [ ] 22. Replace attack-only AI in Step with full `EnemyTurn()`
- [ ] 23. Loot: add `ProcessLoot()` to `engine/combat.go`, wire into enemy death in effects
- [ ] 24. Loader: compile `behavior` → `[]BehaviorEntry`, compile `loot` → `[]LootEntry` + gold
- [ ] 25. Validation: behavior weights, actions, loot refs, loot chance range
- [ ] 26. Content: add behavior + loot to cave goblin, add `goblin_blade` item
- [ ] 27. `go test ./...` — all green
- [ ] 28. **PLAYTEST**: goblin defends/flees sometimes, drops blade + gold on death

### Phase 4: `feat/combat-save-tui` — Save/load mid-combat + TUI display
_Goal: health bars, combat prompt, save mid-fight and resume._

- [ ] 29. Save: add `Combat` + `RNGPosition` to SaveData, update Save/ApplySave + tests
- [ ] 30. Engine: add `RestoreRNG()`, wire into load path
- [ ] 31. TUI: create `tui/combat.go` — health bars, game over screen
- [ ] 32. TUI: modify `tui/tui.go` — inject combat display, combat prompt, game over blocking
- [ ] 33. TUI: modify `tui/status.go` — show HP; modify `tui/style.go` — combat styles
- [ ] 34. `go test ./...` — all green
- [ ] 35. **PLAYTEST**: see health bars, save mid-combat, load and resume, game over screen

### Phase 5: `feat/combat-integration` — Integration tests + polish
_Goal: comprehensive test coverage, deterministic replay verified._

- [ ] 36. Create `loader/testdata/combat/` and `loader/testdata/invalid_enemy/` fixtures
- [ ] 37. Add loader integration tests — `TestLoad_CombatGame`, compile tests, validation tests
- [ ] 38. Create `engine/combat_integration_test.go` — full sequence, death, flee, save/load mid-combat
- [ ] 39. Add `TestCombatReplay_Deterministic` — replay commands, verify identical output
- [ ] 40. Add `TestAuthorOverride_CustomDamage` — author rule beats default
- [ ] 41. `go test ./...` — all green
- [ ] 42. **FINAL PLAYTEST**: full Lost Crown combat encounter end-to-end
