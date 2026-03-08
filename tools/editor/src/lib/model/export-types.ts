/**
 * Export model types — map 1:1 to the QuestCore Lua DSL.
 * Codegen operates exclusively on these types.
 */

// ── Game Metadata ──────────────────────────────────────────────

export interface GameMeta {
	title: string;
	author: string;
	version: string;
	start: string; // room ID reference
	intro: string;
	playerStats?: CombatStats;
}

// ── Rooms ──────────────────────────────────────────────────────

export type Direction =
	| 'north'
	| 'south'
	| 'east'
	| 'west'
	| 'northeast'
	| 'northwest'
	| 'southeast'
	| 'southwest'
	| 'up'
	| 'down';

export interface Room {
	id: string;
	description: string;
	exits: Partial<Record<Direction, string>>;
	fallbacks: Record<string, string>;
	rules: string[];
}

// ── Entities ───────────────────────────────────────────────────

export type Entity = ItemEntity | NPCEntity | EnemyEntity | GenericEntity;

interface BaseEntity {
	id: string;
	name: string;
	description: string;
	location: string; // room ID reference
	customProperties: Record<string, string | number | boolean>;
	rules: string[];
}

export interface ItemEntity extends BaseEntity {
	kind: 'item';
	takeable: boolean;
}

export interface NPCEntity extends BaseEntity {
	kind: 'npc';
	topics: Record<string, Topic>;
}

export interface EnemyEntity extends BaseEntity {
	kind: 'enemy';
	stats: CombatStats;
	behavior: BehaviorWeight[];
	loot: LootTable;
}

export interface GenericEntity extends BaseEntity {
	kind: 'entity';
}

export interface Topic {
	text: string;
	requires: Condition[];
	effects: Effect[];
}

export interface CombatStats {
	hp: number;
	maxHp: number;
	attack: number;
	defense: number;
}

export interface BehaviorWeight {
	action: string;
	weight: number;
}

export interface LootTable {
	items: { id: string; chance: number }[];
	gold: number;
}

// ── Rules ──────────────────────────────────────────────────────

export interface Rule {
	id: string;
	when: WhenClause;
	conditions: Condition[];
	effects: Effect[];
}

export interface WhenClause {
	verb: string;
	object?: string;
	target?: string;
	objectKind?: string;
	objectProp?: Record<string, string | number | boolean>;
	targetProp?: Record<string, string | number | boolean>;
	priority?: number;
}

// ── Conditions ─────────────────────────────────────────────────

export type Condition =
	| { type: 'has_item'; entity: string }
	| { type: 'flag_set'; flag: string }
	| { type: 'flag_not'; flag: string }
	| { type: 'flag_is'; flag: string; value: boolean }
	| { type: 'in_room'; room: string }
	| { type: 'prop_is'; entity: string; prop: string; value: string | number | boolean }
	| { type: 'counter_gt'; counter: string; value: number }
	| { type: 'counter_lt'; counter: string; value: number }
	| { type: 'in_combat' }
	| { type: 'not'; condition: Condition };

// ── Effects ────────────────────────────────────────────────────

export type Effect =
	| { type: 'say'; text: string }
	| { type: 'give_item'; entity: string }
	| { type: 'remove_item'; entity: string }
	| { type: 'set_flag'; flag: string; value: boolean }
	| { type: 'inc_counter'; counter: string; amount: number }
	| { type: 'set_counter'; counter: string; value: number }
	| { type: 'set_prop'; entity: string; prop: string; value: string | number | boolean }
	| { type: 'move_entity'; entity: string; room: string }
	| { type: 'move_player'; room: string }
	| { type: 'open_exit'; room: string; direction: Direction; target: string }
	| { type: 'close_exit'; room: string; direction: string }
	| { type: 'emit_event'; event: string }
	| { type: 'start_dialogue'; entity: string }
	| { type: 'start_combat'; entity: string }
	| { type: 'stop' };

// ── Event Handlers ─────────────────────────────────────────────

export interface EventHandler {
	event: string;
	conditions: Condition[];
	effects: Effect[];
}
