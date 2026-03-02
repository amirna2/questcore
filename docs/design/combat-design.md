# Technical Design: Combat & Player Health

**Depends on:** [Combat PRD](combat-prd.md)
**Status:** Draft

---

## 1. Architecture: How Combat Fits

Combat is **not** a separate engine. It is a game mode — a state flag that
modifies behavior within the existing pipeline.

```
┌─────────────────────────────────────────────────────┐
│                   Game Loop                         │
│                                                     │
│  ┌──────────┐    ┌──────────┐    ┌──────────────┐   │
│  │  Parse   │───▶│  Step()  │───▶│ Apply Effects│   │
│  │  Intent  │    │  Rules   │    │ + Events     │   │
│  └──────────┘    └──────────┘    └──────────────┘   │
│       │                │                │           │
│       │          ┌─────▼─────┐          │           │
│       │          │ COMBAT?   │          │           │
│       │          │           │          │           │
│       │          │ yes: enemy│          │           │
│       │          │ AI turn   │          │           │
│       │          │ (same     │          │           │
│       │          │  pipeline)│          │           │
│       │          └───────────┘          │           │
│       │                                 │           │
│  ┌────▼─────────────────────────────────▼────────┐  │
│  │              Render Output                    │  │
│  │  (normal mode OR combat mode display)         │  │
│  └───────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────┘
```

The critical insight: **the enemy's turn is just another `Step()` call.**
After the player's action resolves, the engine synthesizes an intent for the
enemy (based on behavior weights + RNG) and runs it through the same pipeline.

---

## 2. State Model Changes

### 2.1 Combat State (added to `types.State`)

```go
type CombatState struct {
    Active     bool   // true when in combat
    EnemyID    string // entity ID of the enemy
    RoundCount int    // current combat round
    Defending  bool   // true if player chose defend this round
}
```

Added to `State`:

```go
type State struct {
    Player     Player
    Entities   map[string]EntityState
    Flags      map[string]bool
    Counters   map[string]int
    TurnCount  int
    RNGSeed    int64
    RNG        *RNG              // deterministic RNG instance
    CommandLog []string
    Combat     CombatState       // NEW
}
```

### 2.2 Player Stats

`Player.Stats` already exists as `map[string]int`. It will be initialized
from `Game.PlayerStats` during `NewState()`.

```go
type Player struct {
    Location  string
    Inventory []string
    Stats     map[string]int  // hp, max_hp, attack, defense
}
```

No struct changes needed — just initialization logic in the loader.

### 2.3 Enemy Stats (runtime)

Enemy HP and stat overrides are stored in `EntityState`, the same override
mechanism used for all entities:

```go
// Enemy "cave_goblin" with 8/12 HP remaining:
state.Entities["cave_goblin"] = EntityState{
    Props: map[string]any{
        "hp": 8,       // runtime override of base HP
        "alive": true,
    },
}
```

When an enemy is defeated, `alive` is set to `false` and the entity stays in
the world (for examine, loot text, etc.) but no longer blocks or engages.

---

## 3. Definition Model Changes

### 3.1 New Lua Constructor: `Enemy`

```lua
Enemy "cave_goblin" {
    name        = "Cave Goblin",
    description = "A snarling goblin clutching a rusty blade.",
    location    = "secret_passage",
    stats = {
        hp = 12, max_hp = 12,
        attack = 4, defense = 1,
    },
    behavior = {
        { action = "attack", weight = 70 },
        { action = "defend", weight = 20 },
        { action = "flee",   weight = 10 },
    },
    loot = {
        items = { { id = "goblin_blade", chance = 50 } },
        gold  = 5,
    },
}
```

Compiles to `EntityDef` with `Kind = "enemy"` and combat data stored in
`Props`:

```go
EntityDef{
    ID:   "cave_goblin",
    Kind: "enemy",
    Props: map[string]any{
        "name":        "Cave Goblin",
        "description": "A snarling goblin clutching a rusty blade.",
        "location":    "secret_passage",
        "hp":          12,
        "max_hp":      12,
        "attack":      4,
        "defense":     1,
        "alive":       true,
        "behavior":    []BehaviorEntry{{Action: "attack", Weight: 70}, ...},
        "loot_items":  []LootEntry{{ID: "goblin_blade", Chance: 50}},
        "loot_gold":   5,
    },
}
```

Alternatively, we can add dedicated fields to a new `EnemyDef` type. But
storing in `Props` keeps the entity system unified — an enemy is just an entity
with combat properties. This means existing property lookup, override layering,
and save/load all work without changes.

**Decision: use Props.** Add typed helper structs for the loader to validate
and compile behavior/loot, but store the compiled data in Props for the runtime.

### 3.2 Game Metadata: Player Stats

```lua
Game {
    title = "The Lost Crown",
    start = "castle_gates",
    player_stats = {
        hp = 20, max_hp = 20,
        attack = 5, defense = 2,
    },
}
```

Added to `GameDef`:

```go
type GameDef struct {
    Title       string
    Author      string
    Version     string
    Start       string
    Intro       string
    PlayerStats map[string]int  // NEW
}
```

If `PlayerStats` is nil, the player has no combat stats and combat features
are effectively disabled (the game is adventure-only).

---

## 4. New Conditions

| Condition | Lua Helper | Description |
|-----------|------------|-------------|
| `in_combat` | `InCombat()` | Player is currently in combat |
| `in_combat_with` | `InCombatWith("entity_id")` | Player is fighting this specific enemy |
| `stat_gt` | `StatGt("entity_or_player", "stat", value)` | Stat is greater than value |
| `stat_lt` | `StatLt("entity_or_player", "stat", value)` | Stat is less than value |

### `InCombat()` and `InCombatWith()`

```go
// InCombat: check state.Combat.Active
Condition{Type: "in_combat"}

// InCombatWith: check state.Combat.Active && state.Combat.EnemyID == id
Condition{Type: "in_combat_with", Params: map[string]any{"entity": "cave_goblin"}}
```

### `StatGt()` / `StatLt()`

These check entity or player stats. The entity parameter can be `"player"` or
an entity ID.

```go
// StatLt("cave_goblin", "hp", 4) — goblin HP is below 4
Condition{Type: "stat_lt", Params: map[string]any{
    "entity": "cave_goblin", "stat": "hp", "value": 4,
}}

// StatGt("player", "hp", 10) — player HP is above 10
Condition{Type: "stat_gt", Params: map[string]any{
    "entity": "player", "stat": "hp", "value": 10,
}}
```

For player stats, the lookup reads from `state.Player.Stats`. For entity
stats, it reads from `state.Entities[id].Props` with fallback to
`defs.Entities[id].Props` (same layered override pattern).

---

## 5. New Effects

| Effect | Lua Helper | Description |
|--------|------------|-------------|
| `start_combat` | `StartCombat("entity_id")` | Enter combat mode with enemy |
| `end_combat` | `EndCombat()` | Exit combat mode |
| `damage` | `Damage("entity_or_player", amount)` | Deal damage, clamp to 0, check death |
| `heal` | `Heal("entity_or_player", amount)` | Restore HP, clamp to max_hp |
| `set_stat` | `SetStat("entity_or_player", "stat", value)` | Set a stat to a specific value |

### `StartCombat(entity_id)`

```go
Effect{Type: "start_combat", Params: map[string]any{"enemy": "cave_goblin"}}
```

Behavior:
1. Set `state.Combat.Active = true`
2. Set `state.Combat.EnemyID = entity_id`
3. Set `state.Combat.RoundCount = 0`
4. Initialize enemy runtime stats from base definition (copy base HP etc.
   into `EntityState.Props` if not already set)
5. Emit `combat_started` event

### `EndCombat()`

```go
Effect{Type: "end_combat"}
```

Behavior:
1. Set `state.Combat.Active = false`
2. Clear `state.Combat.EnemyID`
3. Clear `state.Combat.Defending`
4. Emit `combat_ended` event

### `Damage(target, amount)`

```go
Effect{Type: "damage", Params: map[string]any{"target": "cave_goblin", "amount": 8}}
```

Behavior:
1. If target is `"player"`: decrement `state.Player.Stats["hp"]`
2. If target is an entity: decrement entity's `hp` prop in `EntityState`
3. Clamp HP to minimum 0
4. Emit `entity_damaged` event with `{target, amount, remaining_hp}`
5. If HP reaches 0:
   - For enemy: emit `enemy_defeated` event, set `alive = false`, call
     `EndCombat()`, process loot
   - For player: emit `player_defeated` event, trigger game over

### `Heal(target, amount)`

```go
Effect{Type: "heal", Params: map[string]any{"target": "player", "amount": 8}}
```

Behavior:
1. Increment target's `hp`
2. Clamp to `max_hp`
3. Emit `entity_healed` event

### `SetStat(target, stat, value)`

```go
Effect{Type: "set_stat", Params: map[string]any{
    "target": "player", "stat": "attack", "value": 7,
}}
```

Direct stat override. Useful for author rules that grant temporary or
permanent stat changes.

---

## 6. Combat Mode Game Loop

### 6.1 Modified Turn Sequence

The core `Step()` function gains awareness of combat state:

```
1.  Read player input
2.  Parse into Intent
3.  IF combat active:
      a. Validate intent is a combat action (attack, defend, flee, use)
      b. If not: "You're in combat! attack, defend, use <item>, or flee."
4.  Step(state, intent):
      a-j. Normal pipeline (resolve, collect, filter, rank, select, produce,
           apply, events)
5.  IF combat active AND combat didn't just end:
      a. Enemy AI selects action (weighted random via RNG)
      b. Synthesize enemy Intent
      c. Step(state, enemy_intent) — same pipeline, enemy as actor
      d. Check for player death
6.  Advance turn counter
7.  Render output (combat display if in combat, normal otherwise)
```

### 6.2 Command Restriction

During combat, only these intents are valid:

| Command | Intent |
|---------|--------|
| `attack` | `{attack, _, _}` (target is implicit: combat enemy) |
| `defend` | `{defend, _, _}` |
| `flee` | `{flee, _, _}` |
| `use <item>` | `{use, item, _}` |
| `inventory` | `{inventory, _, _}` (allowed — check what you have) |
| `look` | `{look, _, _}` (allowed — see the battlefield) |

All other commands: "You're in the middle of a fight!"

### 6.3 Enemy AI

After the player's action resolves, the engine runs the enemy's turn:

```go
func EnemyTurn(s *types.State, defs *state.Defs) types.Intent {
    enemy := s.Combat.EnemyID
    behavior := getEnemyBehavior(defs, enemy) // []BehaviorEntry

    // Weighted random selection using deterministic RNG
    action := weightedSelect(s.RNG, behavior)

    return types.Intent{
        Verb:   action, // "attack", "defend", or "flee"
        Object: "",
        Target: "",
    }
}
```

The enemy's intent is then run through the **same** rules pipeline. This means
game authors can write rules that intercept enemy actions:

```lua
-- Goblin can't flee if player has the "blocking" flag
Rule("block_goblin_flee",
    When { verb = "flee" },
    { InCombatWith("cave_goblin"), FlagSet("blocking_exit") },
    Then {
        Say("The goblin tries to run, but you block the exit!"),
        Stop()
    }
)
```

### 6.4 Actor Context

The rules engine needs to know **who** is acting — the player or the enemy.
Add an `Actor` field to the step context:

```go
type StepContext struct {
    Actor string // "player" or entity ID of the acting enemy
}
```

This lets conditions like "if the actor is the goblin" work, and lets damage
effects know who is attacking whom.

---

## 7. Damage Calculation

### 7.1 Formula

```
damage = max(1, roll(1d6) + attacker.attack - defender.defense)
```

- `roll(1d6)` — deterministic random integer 1-6 using `state.RNG`
- `attacker.attack` — the acting combatant's attack stat
- `defender.defense` — the receiving combatant's defense stat
  (includes defend bonus if applicable)
- Result clamped to minimum 1

### 7.2 Defend Bonus

When a combatant chooses `defend`:
- Set a transient flag: `state.Combat.Defending = true` (for player) or
  `SetProp(enemy, "defending", true)` for enemy
- During damage calculation, add +2 to defender's defense
- The flag is cleared at the start of the next round

### 7.3 Default Combat Rules

The engine provides default rules for combat actions. These fire if no
author-defined rule matches first.

```
// Pseudo-rules (implemented in Go, not Lua):

"default_combat_attack":
    When: verb = "attack", in_combat = true
    Effect: calculate damage, apply Damage() to combat target, Say() result

"default_combat_defend":
    When: verb = "defend", in_combat = true
    Effect: set defending flag, Say("You brace yourself.")

"default_combat_flee":
    When: verb = "flee", in_combat = true
    Effect: roll 1d6, on 4+: EndCombat() + MovePlayer(previous_room),
            on fail: Say("You can't escape!") + enemy free attack
```

These are **default rules at the lowest priority.** Author rules with the
same `When` criteria always win (first match wins, author rules have higher
scope/specificity).

### 7.4 Implementation: Where Does Damage Calc Live?

The damage formula is **engine code**, not a rule effect. The `default_combat_attack`
rule calls a Go function that:

1. Reads attacker and defender stats
2. Rolls `1d6` via RNG
3. Computes damage
4. Produces `Damage()` and `Say()` effects

This keeps the formula deterministic and testable as a unit. Game authors
override it by writing a higher-priority rule that produces their own
`Damage()` effects with custom amounts.

---

## 8. RNG: Deterministic Randomness

### 8.1 RNG State

`State.RNGSeed` already exists. Add an RNG instance that is seeded from it:

```go
type RNG struct {
    src *rand.Rand
}

func NewRNG(seed int64) *RNG {
    return &RNG{src: rand.New(rand.NewSource(seed))}
}

func (r *RNG) Roll(sides int) int {
    return r.src.Intn(sides) + 1 // 1 to sides, inclusive
}

func (r *RNG) WeightedSelect(weights []int) int {
    total := sum(weights)
    roll := r.src.Intn(total)
    cumulative := 0
    for i, w := range weights {
        cumulative += w
        if roll < cumulative {
            return i
        }
    }
    return len(weights) - 1
}
```

The RNG is part of state. It advances deterministically. Same seed + same
sequence of calls = same results. This preserves the deterministic replay
invariant.

### 8.2 Save/Load

When saving, persist the RNG state (or the number of RNG calls made, so it
can be replayed from seed). The simplest approach: save the current RNG seed
value after each state mutation. `math/rand.Rand` can be serialized by saving
its source state.

---

## 9. Loot System

### 9.1 Loot Definition

```lua
loot = {
    items = {
        { id = "goblin_blade", chance = 50 },  -- 50% drop
        { id = "healing_herb", chance = 25 },   -- 25% drop
    },
    gold = 5,    -- always awarded
}
```

### 9.2 Loot Resolution

On enemy defeat:

1. For each loot item: roll `1d100` via RNG. If roll <= `chance`, add item
   to player inventory via `GiveItem` effect.
2. Add `gold` to the `gold` counter via `IncCounter` effect.
3. Produce `Say()` effects listing what was found.
4. Emit `loot_awarded` event with details.

### 9.3 Loot Compilation

The loader compiles loot entries into typed structs:

```go
type LootEntry struct {
    ItemID string
    Chance int // 1-100
}

type LootDef struct {
    Items []LootEntry
    Gold  int
}
```

Stored in entity Props as compiled data. Validated at load time: item IDs
must reference defined entities, chance must be 1-100.

---

## 10. Death & Game Over

### 10.1 Player Death

When `Damage()` reduces player HP to 0:

1. Emit `player_defeated` event with `{enemy: combat_enemy_id}`
2. Set `state.Combat.Active = false`
3. Set a `game_over` flag on state
4. Produce output: "You were slain by {enemy.name}!"

The game loop checks `game_over` and renders the game over screen. The player
can `/load` to restore a save or `/quit`.

Game authors can attach handlers to `player_defeated` for custom death
messages or behavior.

### 10.2 Enemy Death

When `Damage()` reduces enemy HP to 0:

1. Set enemy prop `alive = false`
2. Process loot (see section 9)
3. Call `EndCombat()`
4. Emit `enemy_defeated` event with `{enemy: enemy_id}`
5. Produce output: "The {enemy.name} is defeated!"

The enemy entity remains in the world with `alive = false`. The game author
can write examine rules that show different text for dead enemies.

### 10.3 Enemy Flee

If the enemy's behavior roll selects `flee`:

1. Roll `1d6`. On 4+, the enemy escapes.
2. On success: remove enemy from room (set location to `""` or a special
   "fled" value), call `EndCombat()`, emit `enemy_fled` event.
3. On failure: enemy loses their turn (no attack).

---

## 11. Events

### 11.1 New Events

| Event | Emitted When | Data |
|-------|-------------|------|
| `combat_started` | `StartCombat()` executes | `{enemy: id}` |
| `combat_ended` | `EndCombat()` executes | `{}` |
| `combat_round_end` | After both combatant turns complete | `{round: n}` |
| `entity_damaged` | `Damage()` executes | `{target: id, amount: n, remaining: n}` |
| `entity_healed` | `Heal()` executes | `{target: id, amount: n, current: n}` |
| `enemy_defeated` | Enemy HP reaches 0 | `{enemy: id}` |
| `enemy_fled` | Enemy successfully flees | `{enemy: id}` |
| `player_defeated` | Player HP reaches 0 | `{enemy: id}` |
| `loot_awarded` | Loot processed after victory | `{items: [...], gold: n}` |

These follow the existing single-pass event model. Handlers can fire on any
of these events to produce additional effects.

### 11.2 Example Handlers

```lua
-- Door opens when guardian is defeated
On("enemy_defeated", {
    conditions = { FlagIs("enemy", "guardian") },
    effects = {
        Say("With the guardian slain, the sealed door crumbles."),
        OpenExit("temple", "north", "inner_sanctum"),
    }
})

-- Custom death message
On("player_defeated", {
    conditions = { InRoom("secret_passage") },
    effects = {
        Say("The darkness of the passage swallows you..."),
    }
})
```

---

## 12. TUI Changes

### 12.1 Combat Display

When `state.Combat.Active` is true, the TUI renders:

```
╭─ COMBAT ─────────────────────────────────────╮
│  Cave Goblin          ████████░░░░   8/12 HP  │
│  You                  ██████████░░  18/20 HP  │
╰───────────────────────────────────────────────╯
```

- Health bars are proportional (`hp / max_hp`)
- Updated after each action

### 12.2 Combat Prompt

Replace the normal `> ` prompt with a combat-specific prompt that shows
available actions:

```
What do you do? (attack, defend, use <item>, flee)
```

### 12.3 Game Over Screen

```
╭─ GAME OVER ──────────────────────────────────╮
│  You were slain by the Cave Goblin.          │
│                                              │
│  /load to restore a save                     │
│  /quit to exit                               │
╰──────────────────────────────────────────────╯
```

The game loop blocks on this screen. Only `/load`, `/quit`, and `/save` meta-
commands are accepted.

---

## 13. Validation Changes

### 13.1 New Checks (Fatal)

| Check | Description |
|-------|-------------|
| Enemy has required stats | `hp`, `max_hp`, `attack`, `defense` must be present and > 0 |
| Behavior weights valid | Each entry has `action` (string) and `weight` (int > 0) |
| Behavior actions known | Action must be `"attack"`, `"defend"`, or `"flee"` |
| Loot item references | Every `id` in `loot.items` must reference a defined entity |
| Loot chance range | `chance` must be 1-100 |
| `StartCombat` target exists | Entity referenced in `StartCombat()` must exist and be kind `"enemy"` |

### 13.2 New Warnings

| Warning | Description |
|---------|-------------|
| Enemy with no behavior | Enemy has no behavior table (defaults to attack-only) |
| Player stats missing | `player_stats` not defined in `Game{}` but `Enemy` entities exist |

---

## 14. Save/Load Changes

### 14.1 Combat State Serialization

Add `combat` field to save JSON:

```json
{
    "combat": {
        "active": true,
        "enemy_id": "cave_goblin",
        "round_count": 3,
        "defending": false
    }
}
```

### 14.2 Enemy Runtime State

Enemy HP and alive status are already stored in `EntityState.Props`, which is
already serialized. No changes needed for enemy state persistence.

### 14.3 RNG State

The RNG must be restorable. Options:

**Option A: Save RNG position.** Track how many times the RNG has been called
since initialization. On load, re-seed and advance by that count.

**Option B: Save current seed state.** `math/rand` sources can export their
state. Save and restore directly.

**Recommendation: Option A.** It's simpler and aligns with the deterministic
replay model (replay from seed + command log).

Add to save format:

```json
{
    "rng_seed": 8674665223082153551,
    "rng_position": 47
}
```

---

## 15. Dependency Graph

No new packages. Combat logic lives in existing packages:

| Package | Changes |
|---------|---------|
| `types/` | Add `CombatState` struct, `LootEntry`, `BehaviorEntry`. Add `PlayerStats` to `GameDef`. |
| `engine/state/` | Add combat state helpers: `InCombat()`, `GetStat()`, `SetStat()`. |
| `engine/effects/` | Add `start_combat`, `end_combat`, `damage`, `heal`, `set_stat` effect handlers. |
| `engine/rules/` | Add `in_combat`, `in_combat_with`, `stat_gt`, `stat_lt` condition evaluators. Add default combat rules. |
| `engine/engine.go` | Modify `Step()` to handle combat mode: command restriction, enemy AI turn, round management. |
| `loader/` | Add `Enemy` Lua constructor. Parse `player_stats` from `Game{}`. Compile behavior/loot tables. |
| `loader/validate.go` | Add enemy stat validation, behavior validation, loot reference validation. |
| `engine/save/` | Serialize/deserialize `CombatState` and RNG position. |
| `cli/` (TUI) | Combat display: health bars, combat prompt, game over screen. |

### New internal code (not new packages)

| Location | Purpose |
|----------|---------|
| `engine/combat.go` | `EnemyTurn()`, `DamageCalc()`, `ProcessLoot()`, default combat rules. RNG helpers. |
| `engine/rng.go` | Deterministic RNG wrapper with position tracking. |

---

## 16. Implementation Order

Each step is independently testable before proceeding.

### Phase 1: Foundation

1. **RNG system** — Deterministic RNG wrapper with `Roll()`,
   `WeightedSelect()`, position tracking. Unit test: same seed = same
   sequence.

2. **Type additions** — `CombatState`, `BehaviorEntry`, `LootEntry`.
   `PlayerStats` on `GameDef`. No logic, just types.

3. **State helpers** — `InCombat()`, `GetStat()`, `SetStat()` in
   `engine/state/`. Unit test: stat read/write with override layering.

### Phase 2: Conditions & Effects

4. **New conditions** — `in_combat`, `in_combat_with`, `stat_gt`, `stat_lt`
   evaluators in `engine/rules/conditions.go`. Table-driven tests.

5. **New effects** — `start_combat`, `end_combat`, `damage`, `heal`,
   `set_stat` in `engine/effects/`. Test each: verify state mutations,
   event emission, death check.

### Phase 3: Combat Engine

6. **Damage calculation** — Pure function: `DamageCalc(attacker, defender,
   rng) -> (amount, roll)`. Table-driven tests with fixed RNG.

7. **Enemy AI** — `EnemyTurn()`: weighted random action selection. Test
   with known RNG seeds.

8. **Loot processing** — `ProcessLoot()`: roll for items, add gold. Test
   deterministic loot drops.

9. **Default combat rules** — Implement `default_combat_attack`,
   `default_combat_defend`, `default_combat_flee` as lowest-priority rules.

### Phase 4: Game Loop Integration

10. **Step() modifications** — Combat mode command restriction, enemy AI turn
    after player turn, round management. Integration tests: full combat
    sequence with known RNG.

11. **Game over handling** — Player death detection, game over state, command
    restriction to `/load`/`/quit`.

### Phase 5: Loader & Validation

12. **Enemy constructor** — `Enemy` Lua constructor in loader. Compile stats,
    behavior, loot.

13. **Player stats loading** — Parse `player_stats` from `Game{}`, initialize
    in `NewState()`.

14. **Validation** — Enemy stat validation, behavior validation, loot
    reference checks. Test with invalid fixtures.

### Phase 6: Save/Load & TUI

15. **Save/load** — Serialize `CombatState` and RNG position. Round-trip test:
    save mid-combat, load, resume, verify identical outcome.

16. **TUI combat display** — Health bars, combat prompt, game over screen.

### Phase 7: Content & Integration

17. **Lost Crown combat encounter** — Add a goblin to the secret passage.
    Full playthrough test.

18. **Deterministic replay test** — Play a combat encounter, record commands,
    replay, verify identical output.

---

## 17. Testing Strategy

### Unit Tests

| Test | What |
|------|------|
| `TestDamageCalc_*` | Fixed RNG seed, verify damage values for various stat combos |
| `TestDamageCalc_MinimumOne` | Damage never goes below 1 |
| `TestDefendBonus` | Defense +2 when defending |
| `TestEnemyAI_WeightedSelect` | Known seed produces expected action sequence |
| `TestLootDrop_Deterministic` | Known seed produces expected loot |
| `TestStartCombat` | Combat state initialized correctly |
| `TestEndCombat` | Combat state cleared correctly |
| `TestDamageEffect_EnemyDeath` | Enemy HP 0 triggers defeat, loot, end combat |
| `TestDamageEffect_PlayerDeath` | Player HP 0 triggers game over |
| `TestHealEffect_ClampToMax` | Healing doesn't exceed max_hp |
| `TestCombatConditions` | `InCombat`, `InCombatWith`, `StatGt`, `StatLt` |
| `TestCombatCommandRestriction` | Non-combat commands rejected during combat |
| `TestFleeSuccess` | Player escapes on high roll |
| `TestFleeFail` | Player takes damage on failed flee |

### Integration Tests

| Test | What |
|------|------|
| `TestCombatEncounter_FullSequence` | Load game, enter room, fight enemy, win, check loot and state |
| `TestCombatEncounter_PlayerDeath` | Fight enemy, lose, verify game over state |
| `TestCombatEncounter_Flee` | Fight enemy, flee, verify player is in previous room |
| `TestCombatSaveLoad_MidCombat` | Save during combat, load, resume, verify identical outcome |
| `TestCombatReplay` | Play combat with command log, replay, verify identical output |
| `TestAuthorOverride_CustomDamage` | Author rule overrides default combat attack |

---

## 18. Architectural Invariants (unchanged)

This design preserves all existing invariants:

| Invariant | How |
|-----------|-----|
| Lua is compile-time only | `Enemy` compiles to Go structs. No Lua at runtime. |
| Rules engine is pure | `(state, intent) → effects`. Combat actions go through same pipeline. |
| Effects are the instruction set | New effects (`damage`, `heal`, etc.) are atomic operations. |
| First match wins | Author combat rules beat default combat rules via scope/specificity. |
| State is flat | `CombatState` is a flat struct. Enemy stats are in `EntityState.Props`. |
| Engine knows nothing about content | Engine provides combat mechanics. Content defines enemies and encounters. |
| Determinism | RNG is seeded and tracked. Same seed + same commands = same combat. |

---

## 19. Open Design Questions

| # | Question | Notes |
|---|----------|-------|
| 1 | Should `Enemy` be a new Lua constructor or reuse `NPC` with combat props? | Leaning new constructor — clearer semantics, easier validation. |
| 2 | Can an enemy also have dialogue topics? (negotiate?) | Not in v1. Could add `topics` to Enemy later. |
| 3 | Should there be a `player_defeated` vs `game_over` distinction? | `player_defeated` is an event (overridable). Game over is a state (engine-handled). |
| 4 | Where does the "previous room" for flee come from? | Track `previous_location` on state when entering combat. |
| 5 | Should enemy defend bonus be same as player (+2)? | Yes, same formula. Symmetry is simpler. |
| 6 | Should we support `attack <enemy>` during combat (redundant target)? | Yes — parse it, but ignore the target (implicit in combat). |
