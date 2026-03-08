# Technical Design: QuestCore Game Editor

**Status:** Draft
**Depends on:** [Lua Authoring Guide](../lua-authoring-guide.md), [DESIGN.md](DESIGN.md)

---

## 1. Overview

A standalone web application for visually authoring QuestCore games. The editor
provides form-based editing for all content types (rooms, entities, rules,
events) and exports valid `.lua` files that the QuestCore engine loads directly.

**Goals:**
- Lower the barrier to entry for game creation — no Lua knowledge required
- Provide instant feedback via client-side validation
- Generate clean, human-readable Lua that can be hand-edited afterward
- Support round-trip: import existing `.lua` game files for editing

**Non-goals (v1):**
- Real-time playtesting in-browser
- Multiplayer collaboration
- Asset management (images, audio)
- Visual room map editor (deferred to v2)

**Key architectural constraint:** The editor is **fully standalone**. It has no
dependency on the QuestCore Go engine. The `.lua` files are the integration
contract — the editor produces them, the engine consumes them. An author can
create a game in the editor, export `.lua` files, and hand them to anyone with
the `questcore` binary. The two tools never need to communicate.

---

## 2. Tech Stack

| Layer        | Choice                | Rationale                                        |
|--------------|-----------------------|--------------------------------------------------|
| Framework    | SvelteKit             | Minimal boilerplate, excellent reactivity for forms |
| Language     | TypeScript            | Type safety for the content model                |
| Styling      | Tailwind CSS          | Rapid UI development, consistent design system   |
| State        | Svelte 5 runes        | `$state`, `$derived` — native reactivity         |
| Lua parsing  | `luaparse`            | Full Lua 5.1 AST parser for import (round-trip) |
| Persistence  | LocalStorage + file export | No server needed                          |
| Testing      | Vitest + Playwright   | Unit tests for model/codegen, E2E for editor flows |
| Build        | Vite (via SvelteKit)  | Fast dev server, optimized production builds     |

### Why SvelteKit over Vue 3

- Less ceremony for form-heavy UIs (two-way binding is native)
- Compiles away the framework — snappy load for a tool app
- File-based routing built in
- TypeScript support is first-class in Svelte 5

---

## 3. Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                       SvelteKit App                           │
│                                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐        │
│  │   UI Layer   │  │   UI Layer   │  │   UI Layer   │        │
│  │  Room Editor │  │ Entity Editor│  │ Rule Builder  │        │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘        │
│         │                 │                  │                │
│  ┌──────▼─────────────────▼──────────────────▼────────────┐   │
│  │           Game Store (Svelte 5 $state)                 │   │
│  │           EditorProject { EditorItem<T>[] }            │   │
│  │           + derived indexes (Map lookups)              │   │
│  └──────┬──────────────────────┬──────────────────────────┘   │
│         │                      │                              │
│         │  ┌───────────────────▼──────────────────────┐       │
│         │  │  Mapping: EditorItem<T> → T (strip meta) │       │
│         │  └───────┬───────────────────────┬──────────┘       │
│         │          │                       │                  │
│  ┌──────▼──────────▼───┐  ┌────────────────▼──────────────┐   │
│  │  Lua Codegen        │  │  DSL-Aware Parser (import)    │   │
│  │  Export Model → .lua│  │  .lua → luaparse AST          │   │
│  └─────────────────────┘  │  AST → Export Model → Editor  │   │
│         │                 └───────────────────────────────┘   │
│         │                       │                             │
│  ┌──────▼───────────────────────▼─────────────────────────┐   │
│  │              Validator                                 │   │
│  │  Reference checks, required fields, ID dups            │   │
│  └────────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────┘
```

### Key Modules

| Module              | Responsibility                                          |
|---------------------|---------------------------------------------------------|
| `lib/model/`        | TypeScript types — editor model (`EditorItem<T>`) and export model (`Room`, `Entity`, etc.) |
| `lib/store/`        | Reactive state (`EditorProject`), CRUD operations, derived indexes |
| `lib/codegen/`      | Serializes export model types to `.lua` files           |
| `lib/parser/`       | DSL-aware import: `.lua` → luaparse AST → export model → editor model |
| `lib/validator/`    | Client-side validation (references, required fields)    |
| `routes/`           | SvelteKit pages — one per editor section                |
| `lib/components/`   | Reusable UI components (forms, lists, pickers)          |

---

## 4. Data Model (TypeScript)

The data model has two layers: the **editor model** (what the store holds,
what the UI binds to) and the **export model** (what maps 1:1 to Lua DSL
constructs). This separation keeps the UI free from DSL quirks and leaves
room for editor-only concerns like draft state, ordering, and undo history.

### 4.1 Editor Model (store layer)

The editor model wraps each content object with editor-specific metadata.
The `data` field holds the exportable content; the `meta` field holds
editor-only state that never appears in the `.lua` output.

```typescript
// Editor wrappers — what the store and UI work with
interface EditorItem<T> {
  data: T;                    // the exportable content
  meta: EditorMeta;           // editor-only state
}

interface EditorMeta {
  dirty: boolean;             // unsaved changes
  sortOrder: number;          // display ordering in lists
  collapsed: boolean;         // UI collapse state
  importWarnings: string[];   // issues from import
  importSource?: string;      // original file path (if imported)
}
```

The root project uses **ordered arrays** as the source of truth (for
serialization, snapshots, and persistence) with derived index maps
for O(1) lookups:

```typescript
interface EditorProject {
  meta: EditorItem<GameMeta>;
  rooms: EditorItem<Room>[];
  entities: EditorItem<Entity>[];
  rules: EditorItem<Rule>[];
  events: EditorItem<EventHandler>[];
}

// Derived indexes — rebuilt on change, not persisted
interface ProjectIndexes {
  roomById: Map<string, Room>;
  entityById: Map<string, Entity>;
  ruleById: Map<string, Rule>;
  entitiesByRoom: Map<string, Entity[]>;
}
```

### 4.2 Export Model (codegen layer)

The export model types map 1:1 to the Lua DSL. The codegen module operates
only on these types — it never sees `EditorMeta`. The mapping is trivial:
`editorItems.map(item => item.data)`.

#### Game Metadata

```typescript
interface GameMeta {
  title: string;           // required
  author: string;
  version: string;
  start: string;           // room ID reference
  intro: string;
  playerStats?: {          // combat system
    hp: number;
    maxHp: number;
    attack: number;
    defense: number;
  };
}
```

#### Rooms

```typescript
interface Room {
  id: string;              // unique identifier, used in Lua as Room "id" {}
  description: string;
  exits: Record<Direction, string>;   // direction → room ID
  fallbacks: Record<string, string>;  // verb → message
  rules: string[];         // rule IDs scoped to this room
}

type Direction =
  | "north" | "south" | "east" | "west"
  | "northeast" | "northwest" | "southeast" | "southwest"
  | "up" | "down";
```

#### Entities

```typescript
// Discriminated union — the `kind` field determines the type
type Entity = ItemEntity | NPCEntity | EnemyEntity | GenericEntity;

interface BaseEntity {
  id: string;
  name: string;
  description: string;
  location: string;           // room ID reference
  customProperties: Record<string, string | number | boolean>;
  rules: string[];            // rule IDs scoped to this entity
}

interface ItemEntity extends BaseEntity {
  kind: "item";
  takeable: boolean;          // default: true
}

interface NPCEntity extends BaseEntity {
  kind: "npc";
  topics: Record<string, Topic>;
}

interface EnemyEntity extends BaseEntity {
  kind: "enemy";
  stats: CombatStats;
  behavior: BehaviorWeight[];
  loot: LootTable;
}

interface GenericEntity extends BaseEntity {
  kind: "entity";
}

interface Topic {
  text: string;
  requires: Condition[];      // optional
  effects: Effect[];          // optional
}

interface CombatStats {
  hp: number;
  maxHp: number;
  attack: number;
  defense: number;
}

interface BehaviorWeight {
  action: string;             // "attack" | "defend" | "flee"
  weight: number;             // 0-100
}

interface LootTable {
  items: { id: string; chance: number }[];
  gold: number;
}
```

#### Rules

```typescript
interface Rule {
  id: string;                 // globally unique
  when: WhenClause;
  conditions: Condition[];    // optional (empty = no conditions)
  effects: Effect[];          // the Then {} block
}

interface WhenClause {
  verb: string;
  object?: string;            // entity ID or raw noun
  target?: string;            // entity ID or raw noun
  objectKind?: string;        // "item" | "npc"
  objectProp?: Record<string, string | number | boolean>;
  targetProp?: Record<string, string | number | boolean>;
  priority?: number;
}
```

#### Conditions

```typescript
// Discriminated union on `type`
type Condition =
  | { type: "has_item";    entity: string }
  | { type: "flag_set";    flag: string }
  | { type: "flag_not";    flag: string }
  | { type: "flag_is";     flag: string; value: boolean }
  | { type: "in_room";     room: string }
  | { type: "prop_is";     entity: string; prop: string; value: string | number | boolean }
  | { type: "counter_gt";  counter: string; value: number }
  | { type: "counter_lt";  counter: string; value: number }
  | { type: "in_combat" }
  | { type: "not";         condition: Condition };
```

#### Effects

```typescript
// Discriminated union on `type`
type Effect =
  | { type: "say";            text: string }
  | { type: "give_item";      entity: string }
  | { type: "remove_item";    entity: string }
  | { type: "set_flag";       flag: string; value: boolean }
  | { type: "inc_counter";    counter: string; amount: number }
  | { type: "set_counter";    counter: string; value: number }
  | { type: "set_prop";       entity: string; prop: string; value: string | number | boolean }
  | { type: "move_entity";    entity: string; room: string }
  | { type: "move_player";    room: string }
  | { type: "open_exit";      room: string; direction: Direction; target: string }
  | { type: "close_exit";     room: string; direction: string }
  | { type: "emit_event";     event: string }
  | { type: "start_dialogue"; entity: string }
  | { type: "start_combat";   entity: string }
  | { type: "stop" };
```

#### Event Handlers

```typescript
interface EventHandler {
  event: string;              // event type string
  conditions: Condition[];
  effects: Effect[];
}
```

---

## 5. Lua Code Generation

The codegen module converts the TypeScript model into `.lua` files. Output
should be clean, readable, and match the style in the authoring guide.

### 5.1 File Organization

The editor generates one file per content type, matching convention:

```
output/
├── game.lua        ← GameMeta
├── rooms.lua       ← All rooms
├── items.lua       ← Items + generic entities
├── npcs.lua        ← NPCs
├── enemies.lua     ← Enemies (if any)
├── rules.lua       ← Rules + event handlers
```

### 5.2 Codegen Strategy

Each model type has a dedicated serializer function. Codegen operates on the
**export model** types only — it receives plain arrays extracted from the
editor wrappers via `project.rooms.map(r => r.data)`:

```typescript
// lib/codegen/rooms.ts
function generateRooms(rooms: Room[]): string

// lib/codegen/entities.ts
function generateItems(entities: Entity[]): string
function generateNPCs(entities: Entity[]): string
function generateEnemies(entities: Entity[]): string

// lib/codegen/rules.ts
function generateRules(rules: Rule[]): string
function generateEvents(events: EventHandler[]): string
```

### 5.3 Example Output

Given a room in the editor model:

```typescript
{
  id: "library",
  description: "Floor-to-ceiling shelves overflow with books.",
  exits: { west: "great_hall" },
  fallbacks: {},
  rules: []
}
```

Generated Lua:

```lua
Room "library" {
    description = "Floor-to-ceiling shelves overflow with books.",
    exits = {
        west = "great_hall"
    }
}
```

Empty tables (`fallbacks = {}`, `rules = {}`) are omitted for cleanliness.

### 5.4 Condition & Effect Serialization

Conditions and effects map 1:1 to Lua constructor calls:

| Model `type`    | Lua output                              |
|-----------------|-----------------------------------------|
| `has_item`      | `HasItem("entity_id")`                  |
| `flag_set`      | `FlagSet("flag_name")`                  |
| `flag_not`      | `FlagNot("flag_name")`                  |
| `flag_is`       | `FlagIs("flag_name", true)`             |
| `in_room`       | `InRoom("room_id")`                     |
| `prop_is`       | `PropIs("entity_id", "prop", value)`    |
| `counter_gt`    | `CounterGt("counter", value)`           |
| `counter_lt`    | `CounterLt("counter", value)`           |
| `in_combat`     | `InCombat()`                            |
| `not`           | `Not(inner_condition)`                  |
| `say`           | `Say("text")`                           |
| `give_item`     | `GiveItem("entity_id")`                 |
| `remove_item`   | `RemoveItem("entity_id")`              |
| `set_flag`      | `SetFlag("flag_name", true)`            |
| `inc_counter`   | `IncCounter("counter", amount)`         |
| `set_counter`   | `SetCounter("counter", value)`          |
| `set_prop`      | `SetProp("entity_id", "prop", value)`   |
| `move_entity`   | `MoveEntity("entity_id", "room_id")`    |
| `move_player`   | `MovePlayer("room_id")`                 |
| `open_exit`     | `OpenExit("room_id", "dir", "target")`  |
| `close_exit`    | `CloseExit("room_id", "dir")`           |
| `emit_event`    | `EmitEvent("event_type")`               |
| `start_dialogue`| `StartDialogue("entity_id")`            |
| `start_combat`  | `StartCombat("entity_id")`              |
| `stop`          | `Stop()`                                |

---

## 6. Lua Import (Round-Trip)

### 6.1 Strategy

The editor uses **`luaparse`**, a full Lua 5.1 parser, to import existing
`.lua` game files. Since gopher-lua (QuestCore's Lua runtime) implements
Lua 5.1, the grammar match is exact. No pattern matching or regex — a proper
AST handles formatting variations, comments, nesting, and future DSL growth.

**Import pipeline:**

```
.lua source files
    │
    ▼
luaparse              Parse each file into a Lua AST
    │                  (statements, expressions, table constructors)
    ▼
AST walker            Identify top-level calls: Game(), Room(), Item(),
    │                  NPC(), Enemy(), Entity(), Rule(), On()
    ▼
Table evaluator       Convert TableConstructorExpression nodes into
    │                  plain TypeScript objects (recursive for nesting)
    ▼
Declaration mapper    Map each declaration to the editor's typed model
    │                  (e.g., Room AST node → Room interface)
    ▼
GameProject           Fully reconstructed editor model, ready to edit
```

### 6.2 AST Walker Detail

The walker is **explicitly DSL-aware**. It does not attempt to interpret
arbitrary Lua. It traverses top-level statements and matches on
`CallExpression` nodes by function name against a known allowlist of
QuestCore constructors:

**Known declarations:** `Game`, `Room`, `Item`, `NPC`, `Enemy`, `Entity`,
`Rule`, `On`

**Known helpers (inside tables):** `When`, `Then`, `HasItem`, `FlagSet`,
`FlagNot`, `FlagIs`, `InRoom`, `PropIs`, `CounterGt`, `CounterLt`,
`InCombat`, `Not`, `Say`, `GiveItem`, `RemoveItem`, `SetFlag`, `IncCounter`,
`SetCounter`, `SetProp`, `MoveEntity`, `MovePlayer`, `OpenExit`, `CloseExit`,
`EmitEvent`, `StartDialogue`, `StartCombat`, `Stop`

Any top-level statement that is **not** a recognized declaration produces a
warning: `"Unrecognized statement at line N — skipped"`. This ensures the
import is honest about what it understood vs. what it ignored.

| Lua Call Pattern                        | AST Shape                                    |
|-----------------------------------------|----------------------------------------------|
| `Game { ... }`                          | `CallExpression(Game, TableConstructor)`      |
| `Room "id" { ... }`                     | `CallExpression(CallExpression(Room, "id"), TableConstructor)` |
| `Item "id" { ... }`                     | Same curried pattern as Room                  |
| `Rule("id", When{...}, Then{...})`      | `CallExpression(Rule, "id", ...args)`         |
| `On("event", { ... })`                  | `CallExpression(On, "event", TableConstructor)` |

The curried syntax `Room "id" { ... }` is Lua syntactic sugar for
`Room("id")({...})` — the parser represents this as nested call expressions.
The walker handles both forms.

### 6.3 Table Evaluator

Converts Lua table constructor AST nodes to TypeScript objects:

- `StringLiteral` → `string`
- `NumericLiteral` → `number`
- `BooleanLiteral` → `boolean`
- `TableConstructorExpression` with `TableKey` entries → `Record<string, ...>`
- `TableConstructorExpression` with `TableValue` entries → `Array<...>`
- Nested `CallExpression` inside tables (e.g., `HasItem("key")`) → mapped
  to the appropriate `Condition` or `Effect` union type by function name

### 6.4 What Imports Cleanly

- All standard DSL declarations (Room, Item, NPC, Enemy, Entity, Rule, On)
- Nested tables of any depth (topics, behavior, loot, exits)
- All string formats (`"double"`, `'single'`, `[[multiline]]`)
- Comments (ignored by parser — not preserved on round-trip)
- Any whitespace/formatting style

### 6.5 Limitations

- **Lua logic is not imported.** Loops, variables, conditionals, and helper
  functions that programmatically generate content cannot be reconstructed
  into editor model objects. The editor imports the declarations it finds;
  dynamically generated declarations are invisible to it.
- **Comments are not preserved** on round-trip. Exporting from the editor
  produces clean Lua without the original comments.
- **Formatting is normalized** to the editor's output style.

These limitations are clearly communicated in the import UI. The import page
shows a summary: "Found X rooms, Y entities, Z rules" with warnings for any
unrecognized top-level statements.

---

## 7. Client-Side Validation

The editor validates the game model in real-time as the author edits. Errors
are shown inline next to the relevant field.

### 7.1 Validation Rules

| Category       | Check                                                   |
|----------------|---------------------------------------------------------|
| **Required**   | Game title, Game start room, Room description            |
| **References** | All room ID refs (exits, entity locations, effects) point to existing rooms |
| **References** | All entity ID refs (conditions, effects) point to existing entities |
| **Uniqueness** | Room IDs are unique, entity IDs are unique, rule IDs are unique |
| **Exits**      | Exit targets exist as defined rooms                      |
| **Start room** | `meta.start` references a defined room                   |
| **Rules**      | Scoped rule IDs exist in the rules list                  |
| **Verbs**      | When clause verbs match the known verb list (warning, not error) |
| **Entities**   | Entity locations reference defined rooms (warning)       |

### 7.2 Validation Levels

- **Error** — Prevents export. The game would not load in QuestCore.
- **Warning** — Allows export. Potential issue that may cause runtime surprises.

---

## 8. UI Structure

### 8.1 Page Layout

The editor uses a sidebar navigation + main content area layout:

```
┌──────────┬──────────────────────────────────────┐
│          │                                      │
│  Game    │           Main Content               │
│  Rooms   │         (active editor)              │
│  Items   │                                      │
│  NPCs    │                                      │
│  Enemies │                                      │
│  Rules   │                                      │
│  Events  │                                      │
│          │                                      │
│──────────│                                      │
│  Import  │                                      │
│  Export  │                                      │
│  Errors  │                                      │
└──────────┴──────────────────────────────────────┘
```

### 8.2 Routes

| Route             | Content                                          |
|-------------------|--------------------------------------------------|
| `/`               | Dashboard — project overview, validation summary |
| `/game`           | Game metadata editor                             |
| `/rooms`          | Room list + room editor                          |
| `/rooms/[id]`     | Single room editor (description, exits, fallbacks) |
| `/entities`       | Entity list (all types), filterable              |
| `/entities/[id]`  | Entity editor (form adapts to kind)              |
| `/rules`          | Rule list + rule builder                         |
| `/rules/[id]`     | Single rule editor                               |
| `/events`         | Event handler list + editor                      |
| `/export`         | Export preview + download                        |
| `/import`         | Import `.lua` files                              |

### 8.3 Key UI Components

| Component            | Description                                       |
|----------------------|---------------------------------------------------|
| `RoomExitEditor`     | Direction picker + room ID dropdown for each exit |
| `EntityPicker`       | Searchable dropdown of entity IDs, filtered by kind |
| `RoomPicker`         | Searchable dropdown of room IDs                   |
| `VerbPicker`         | Dropdown of known verbs with alias hints          |
| `ConditionBuilder`   | Add/remove conditions, each with type-specific fields |
| `EffectBuilder`      | Ordered list of effects, each with type-specific fields |
| `TopicEditor`        | Topic key + text + optional conditions/effects    |
| `ValidationBadge`    | Error/warning count shown on sidebar items        |
| `LuaPreview`         | Read-only code view of generated Lua for any item |

### 8.4 Rule Builder Detail

The rule builder is the most complex UI component. It uses a structured form
rather than free-text editing:

```
┌─────────────────────────────────────────────┐
│  Rule: take_gem_with_key                    │
│                                             │
│  When ─────────────────────────────────     │
│    Verb:   [take        ▼]                  │
│    Object: [gem         ▼] (entity picker)  │
│    Target: [            ▼] (optional)       │
│                                             │
│  Conditions ───────────────────────────     │
│    [+] HasItem  → [key ▼]                   │
│    [+] InRoom   → [armory ▼]                │
│    [×] remove                               │
│                                             │
│  Then (effects, drag to reorder) ──────     │
│    1. Say    → "You pry the gem loose."     │
│    2. GiveItem → [gem ▼]                    │
│    [+ Add Effect]                           │
│                                             │
│  Scope ────────────────────────────────     │
│    ○ Global  ● Room: [armory ▼]             │
│              ○ Entity: [       ▼]           │
│                                             │
│  [Preview Lua]  [Save]  [Delete]            │
└─────────────────────────────────────────────┘
```

---

## 9. Project Structure

```
questcore-editor/
├── package.json
├── svelte.config.js
├── vite.config.ts
├── tsconfig.json
├── tailwind.config.js
│
├── src/
│   ├── app.html
│   ├── app.css                    ← Tailwind imports
│   │
│   ├── lib/
│   │   ├── model/
│   │   │   ├── export-types.ts    ← Export model: Room, Entity, Rule, etc.
│   │   │   ├── editor-types.ts    ← Editor model: EditorItem<T>, EditorProject
│   │   │   ├── defaults.ts        ← Factory functions for new rooms/entities/etc
│   │   │   └── verbs.ts           ← Known verb list with aliases
│   │   │
│   │   ├── store/
│   │   │   ├── project.svelte.ts  ← GameProject reactive state
│   │   │   └── persistence.ts     ← LocalStorage save/load
│   │   │
│   │   ├── codegen/
│   │   │   ├── game.ts            ← GameMeta → game.lua
│   │   │   ├── rooms.ts           ← Room[] → rooms.lua
│   │   │   ├── entities.ts        ← Entity[] → items.lua, npcs.lua, enemies.lua
│   │   │   ├── rules.ts           ← Rule[] → rules.lua (includes events)
│   │   │   ├── conditions.ts      ← Condition → Lua string
│   │   │   ├── effects.ts         ← Effect → Lua string
│   │   │   └── index.ts           ← Orchestrator: project → zip of .lua files
│   │   │
│   │   ├── parser/
│   │   │   ├── import.ts          ← Orchestrator: .lua files → GameProject
│   │   │   ├── walker.ts          ← AST walker: extract top-level declarations
│   │   │   ├── table-eval.ts      ← Convert table constructor nodes → TS objects
│   │   │   └── declaration-map.ts ← Map evaluated tables → typed model interfaces
│   │   │
│   │   ├── validator/
│   │   │   ├── validate.ts        ← Full project validation
│   │   │   └── types.ts           ← ValidationError, ValidationWarning
│   │   │
│   │   └── components/
│   │       ├── EntityPicker.svelte
│   │       ├── RoomPicker.svelte
│   │       ├── VerbPicker.svelte
│   │       ├── ConditionBuilder.svelte
│   │       ├── EffectBuilder.svelte
│   │       ├── TopicEditor.svelte
│   │       ├── LuaPreview.svelte
│   │       ├── ValidationBadge.svelte
│   │       └── Sidebar.svelte
│   │
│   └── routes/
│       ├── +layout.svelte         ← Sidebar + main content shell
│       ├── +page.svelte           ← Dashboard
│       ├── game/
│       │   └── +page.svelte       ← Game metadata editor
│       ├── rooms/
│       │   ├── +page.svelte       ← Room list
│       │   └── [id]/
│       │       └── +page.svelte   ← Room editor
│       ├── entities/
│       │   ├── +page.svelte       ← Entity list
│       │   └── [id]/
│       │       └── +page.svelte   ← Entity editor
│       ├── rules/
│       │   ├── +page.svelte       ← Rule list
│       │   └── [id]/
│       │       └── +page.svelte   ← Rule editor
│       ├── events/
│       │   └── +page.svelte       ← Event handler list + editor
│       ├── export/
│       │   └── +page.svelte       ← Export preview + download
│       └── import/
│           └── +page.svelte       ← Import .lua files
│
├── tests/
│   ├── codegen/                   ← Codegen unit tests
│   │   ├── rooms.test.ts
│   │   ├── entities.test.ts
│   │   ├── rules.test.ts
│   │   └── conditions.test.ts
│   ├── parser/                    ← Parser unit tests
│   │   ├── walker.test.ts
│   │   ├── table-eval.test.ts
│   │   └── roundtrip.test.ts
│   ├── validator/                 ← Validator unit tests
│   │   └── validate.test.ts
│   └── e2e/                       ← Playwright E2E tests
│       ├── room-editor.test.ts
│       └── export-flow.test.ts
│
└── static/
    └── favicon.png
```

---

## 10. Build Order

Phase-by-phase implementation. Each phase is independently useful and testable.

Import is co-developed with codegen in Phase 1 because parser and codegen
together define the real contract. If the round-trip doesn't work for rooms,
it won't work for anything — better to find out immediately.

### Phase 1: Model + Parser + Codegen spike (rooms only)

1. Define all TypeScript types — both editor model and export model
2. Integrate `luaparse` as a dependency
3. Implement parser pipeline for rooms: AST walker → table evaluator → Room
4. Implement Lua codegen for rooms
5. Round-trip test: parse Lost Crown `rooms.lua` → model → codegen → diff
6. Extend parser + codegen to remaining types (entities, rules, events)
7. Full round-trip test: import all Lost Crown `.lua` files, re-export, diff

**Deliverable:** A library that can parse `.lua` ↔ TypeScript model ↔ `.lua`
for all content types. The contract is proven before any UI is built.

### Phase 2: Core Editor UI

1. SvelteKit project scaffolding with Tailwind
2. Game store (reactive state with `EditorProject`, derived indexes)
3. Game metadata editor page
4. Room list + room editor (description, exits, fallbacks)
5. Entity list + entity editor (items, NPCs, enemies)
6. Export page (download `.lua` files as zip)
7. Import page (upload `.lua` files, summary, warnings)
8. LocalStorage persistence

**Deliverable:** A functional editor that can create games from scratch,
import existing games, and export valid `.lua` files.

### Phase 3: Rule Builder

1. Rule list page
2. Rule builder form (When + Conditions + Effects)
3. Condition builder component (type-specific fields)
4. Effect builder component (type-specific fields, drag-to-reorder)
5. Rule scoping (assign to room or entity)
6. Event handler editor

**Deliverable:** Full rule editing capability.

### Phase 4: Validation + Polish

1. Client-side validator
2. Inline error display on all editor pages
3. Validation summary on dashboard
4. Cross-reference checking (entity picker shows only valid options)
5. Lua preview panel (see generated code for any item)

**Deliverable:** Production-quality editing experience with guardrails.

### Phase 5: Richer Import Coverage

1. Handle `local` variable assignments used as rule markers
2. Surface detailed warnings for Lua logic that can't be imported
3. Import provenance tracking (show which file each item came from)
4. Conflict resolution when importing into an existing project

**Deliverable:** Robust import for real-world games with edge cases.

---

## 11. Future Enhancements (v2)

| Feature                     | Description                                       |
|-----------------------------|---------------------------------------------------|
| **Visual room map**         | Interactive node graph for room layout            |
| **In-browser playtesting**  | Embed engine logic for live preview in browser    |
| **AI assist**               | Natural language → model changes via LLM           |
| **Project file format**     | `.questcore` JSON project file for richer saves    |
| **Undo/redo**               | Full operation history                             |
| **Templates**               | Starter game templates (mystery, dungeon crawl)   |
| **Theme system**            | Dark/light mode, customizable editor appearance    |

---

## 12. Open Questions

1. **Separate repo or monorepo?** The editor could live in `tools/editor/`
   within the QuestCore repo, or as a standalone `questcore-editor` repo.
   Monorepo is simpler for development; separate repo makes deployment cleaner.

2. **Deployment target?** Static site (GitHub Pages, Netlify) is the simplest.
   No server needed for v1 since everything is client-side.

3. **NPC topic editor complexity?** Topics have nested conditions and effects.
   Should the topic editor reuse the same ConditionBuilder/EffectBuilder
   components as the rule builder? (Recommended: yes, for consistency.)

4. **File-per-entity option?** Some authors prefer one file per room. Should
   the export offer this as an option, or stick with one-file-per-type?
