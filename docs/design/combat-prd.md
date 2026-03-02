# Product Requirements Document: Combat & Player Health

**Project:** QuestCore v2 — Combat System
**Status:** Draft
**Depends on:** QuestCore MVP (complete)

---

## 1. Overview

Add turn-based combat to QuestCore. Players encounter enemies during
exploration, enter a combat mode, and choose actions each round until one side
is defeated or the player flees.

This is the "dungeon mode" capability described in the original PRD — the
feature that bridges QuestCore from pure adventure game engine to adventure +
RPG engine.

---

## 2. Goals

1. **D&D-lite combat feel.** Player faces a threat, weighs options, makes a
   decision. Dice rolls add variance and tension.
2. **Declarative, data-driven.** Game authors define enemies, stats, and
   behaviors in Lua. The engine provides the combat mechanics.
3. **Respect existing architecture.** Combat uses the same rules engine,
   effects pipeline, and event system. No parallel engine.
4. **Deterministic.** Same state + same commands + same RNG seed = identical
   combat outcome. Replayable and testable.
5. **Author-overridable.** Game authors can write rules that override default
   combat behavior (e.g., silver weapons do bonus damage to undead).

---

## 3. Target Experience

### Entering Combat

The player enters a room. A goblin is there. Combat begins — either
automatically (via an event handler on room entry) or when the player types
`attack goblin`.

The TUI shifts to combat mode: health bars for both combatants, a restricted
command prompt.

```
Secret Passage
A narrow stone corridor. Something moves in the shadows.

A Cave Goblin blocks your path!

╭─ COMBAT ─────────────────────────────────────╮
│  Cave Goblin          ██████████░░  12/12 HP  │
│  You                  ████████████  20/20 HP  │
╰───────────────────────────────────────────────╯

What do you do? (attack, defend, use <item>, flee)
```

### Combat Round

Each round: the player chooses an action, then the enemy acts. Damage is
calculated with dice rolls. Both sides see the results.

```
> attack

You strike the Cave Goblin!
  Roll: 1d6+5 → [4]+5 = 9 vs defense 1 → 8 damage

The Cave Goblin slashes at you!
  Roll: 1d6+4 → [2]+4 = 6 vs defense 2 → 4 damage

╭─ COMBAT ─────────────────────────────────────╮
│  Cave Goblin          ████░░░░░░░░   4/12 HP  │
│  You                  ████████████  16/20 HP  │
╰───────────────────────────────────────────────╯
```

### Defending

The player sacrifices their attack to boost defense for the round.

```
> defend

You brace yourself. (+2 defense this round)
The Cave Goblin strikes — but you deflect the blow!
  Roll: 1d6+4 → [3]+4 = 7 vs defense 4 → 3 damage
```

### Using Items

The player uses a consumable item instead of attacking. The enemy still gets
a turn.

```
> use healing_herb

You eat the healing herb. (+8 HP)
The Cave Goblin strikes!
  Roll: 1d6+4 → [5]+4 = 9 vs defense 2 → 7 damage
```

### Fleeing

The player attempts to escape. Success depends on a dice roll. On failure, the
enemy gets a free attack.

```
> flee

You turn and run!
  Roll: 1d6 → [5] — you escape!

Library
A dusty room filled with ancient tomes...
```

```
> flee

You try to run but the goblin blocks the exit!
  The Cave Goblin gets a free attack!
  Roll: 1d6+4 → [6]+4 = 10 vs defense 2 → 8 damage
```

### Victory

When the enemy's HP reaches 0, combat ends. Loot is awarded.

```
> attack

You land a decisive blow!
  Roll: 1d6+5 → [6]+5 = 11 vs defense 1 → 10 damage
  The Cave Goblin is defeated!

The goblin crumples to the ground.
  Loot: Rusty Goblin Blade, 5 gold

Secret Passage
A narrow stone corridor. Water drips from above.
```

### Defeat

When the player's HP reaches 0, the game is over.

```
The Cave Goblin strikes you down!

╭─ GAME OVER ──────────────────────────────────╮
│  You were slain by the Cave Goblin.          │
│                                              │
│  /load to restore a save                     │
│  /quit to exit                               │
╰──────────────────────────────────────────────╯
```

---

## 4. Player Stats

The player has four core combat stats:

| Stat      | Description                        | Example |
|-----------|------------------------------------|---------|
| `hp`      | Current health points              | 16      |
| `max_hp`  | Maximum health points              | 20      |
| `attack`  | Added to attack roll               | 5       |
| `defense` | Subtracted from incoming damage    | 2       |

Stats are defined by the game author in `Game {}` metadata:

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

Stats are fixed for this version. No leveling, no XP, no stat growth.
Progression comes from items, flags, and game-authored rules (e.g., a blessing
that grants +2 attack via a custom combat rule).

---

## 5. Enemies

Enemies are entities defined in Lua with combat stats and behavior.

```lua
Enemy "cave_goblin" {
    name        = "Cave Goblin",
    description = "A snarling goblin clutching a rusty blade.",
    location    = "secret_passage",
    stats = {
        hp      = 12,
        max_hp  = 12,
        attack  = 4,
        defense = 1,
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

### Enemy Behavior

Each round, the enemy selects an action based on weighted random choice (using
the deterministic RNG). Available enemy actions:

| Action    | Behavior                                       |
|-----------|------------------------------------------------|
| `attack`  | Deal damage to the player                      |
| `defend`  | Boost defense for this round instead of attacking |
| `flee`    | Attempt to escape (removes enemy from combat)  |

Weights are relative, not percentages. `{ attack = 70, defend = 20, flee = 10 }`
means attack 70% of the time, defend 20%, flee 10%.

### Enemy Death

When an enemy's HP reaches 0:

1. Combat ends
2. Loot is awarded (items added to inventory, gold added to counter)
3. Enemy is removed from the world (or marked as dead via property override)
4. An `enemy_defeated` event is emitted

The game author can attach event handlers to `enemy_defeated` for custom
behavior (e.g., a door opens when the guardian is slain).

---

## 6. Combat Actions

### attack

Roll `1d6 + player.attack`. Subtract `enemy.defense`. Minimum 1 damage.

### defend

Skip attack. Add a defense bonus (+2) for this round only. Resets next round.

### flee

Roll `1d6`. On 4+ (configurable per enemy), escape combat. Player returns to
the room they came from (or stays in current room if combat was triggered
in-room). On failure, the enemy gets a free attack and the player deals no
damage.

### use \<item\>

Use a consumable item (e.g., healing herb). The item's effects are applied via
the normal rules engine. The enemy still gets their turn. This replaces the
player's attack for the round.

---

## 7. Damage Model

**Formula:** `1d6 + attacker_attack - defender_defense` (minimum 1)

- `1d6` = roll one six-sided die using the deterministic RNG
- Add attacker's `attack` stat
- Subtract defender's `defense` stat
- Result is clamped to a minimum of 1 (you always deal at least 1 damage)

This applies symmetrically: same formula for player attacking enemy and enemy
attacking player.

### Defend Bonus

When a combatant defends, their `defense` is temporarily increased by +2 for
that round only. The bonus is not permanent.

---

## 8. Healing

Healing happens through consumable items defined by the game author:

```lua
Item "healing_herb" {
    name        = "Healing Herb",
    description = "A fragrant herb that restores vitality.",
    location    = "garden",
    consumable  = true,
    heal_amount = 8,
}
```

Using a healing item during combat (or outside combat) restores HP up to
`max_hp`. The item is consumed (removed from inventory).

Healing is not a built-in combat action — it works through the existing rules
engine. The game author writes a rule for healing items, or the engine provides
a default rule for items with `consumable = true` and `heal_amount`.

---

## 9. Combat Triggers

Combat can start in two ways:

### Player-initiated

The player types `attack <enemy>`. A rule matches and produces a
`StartCombat()` effect.

### Auto-engage

An event handler on `room_entered` checks for hostile enemies in the room and
triggers `StartCombat()`.

Both are authored in Lua. The engine provides no automatic "enemies attack on
sight" behavior — the game author decides when and how combat starts.

---

## 10. Author Overrides

Game authors can override any default combat behavior using rules. The rules
engine's resolution order means author-defined rules take priority over engine
defaults.

**Examples:**

- A silver weapon does double damage to undead
- A specific enemy is immune to normal attacks (requires a magic item)
- An enemy begs for mercy at low HP, giving the player a choice
- A boss fight has special phases triggered by HP thresholds
- A room provides a combat advantage (high ground = +1 attack)

This is the same pattern as the rest of QuestCore: engine provides defaults,
game content overrides them.

---

## 11. Scope

### In Scope (this version)

- Player stats: hp, max_hp, attack, defense
- Enemy definitions in Lua with stats, behavior, loot
- 1v1 turn-based combat (one enemy at a time)
- Four combat actions: attack, defend, flee, use item
- D&D-style damage: `1d6 + attack - defense` (min 1)
- Weighted random enemy behavior (deterministic via RNG seed)
- Loot drops (items + gold)
- Player death = game over (load save to continue)
- Enemy death = removed from world + event emitted
- Healing via consumable items
- Combat triggers via rules and event handlers
- Author-overridable combat rules
- TUI combat display (health bars, combat prompt)
- Save/load preserves combat state (mid-combat saves)

### Out of Scope (deferred)

- **Multi-enemy encounters** — fight one enemy at a time
- **Equipment slots** — no weapon/armor equip system; items affect combat
  through custom rules only
- **XP and leveling** — stats are fixed; progression via items and flags
- **Magic / spells** — no spell system; authors can simulate magic through
  items and rules
- **Enemy respawning** — defeated enemies stay dead unless the author writes
  rules to respawn them
- **Retreat / safe zones** — no concept of areas where combat can't happen
- **Status effects** — no poison, stun, etc. (authors can simulate with
  flags and custom rules)
- **Difficulty settings** — single difficulty; authors control balance through
  stat values

---

## 12. Success Criteria

The combat system is complete when:

1. A game author can define an enemy in Lua and the player can fight it
2. Combat is turn-based, 1v1, with attack/defend/flee/use item
3. Damage uses dice rolls and is deterministic given the same RNG seed
4. Enemy behavior is weighted random and deterministic
5. Player death shows game over; enemy death drops loot and emits events
6. Authors can override default combat behavior with custom rules
7. Combat state survives save/load
8. The TUI displays health bars and combat status
9. All combat logic is testable via the existing test patterns
10. The Lost Crown example game includes at least one combat encounter

---

## 13. Open Questions

| # | Question | Leaning |
|---|----------|---------|
| 1 | Should `defend` give +2 flat or double defense? | +2 flat (simpler, predictable) |
| 2 | Should failed flee have a guaranteed enemy hit or a normal attack roll? | Normal attack roll (fairer) |
| 3 | Can the player `talk` to an enemy during combat? | No for MVP. Could enable "negotiate" later. |
| 4 | Should enemies be able to use items/abilities? | Not in this pass. Behavior is attack/defend/flee only. |
| 5 | What happens to combat on `/save` then `/load`? | Restore exact combat state. Resume the fight. |
