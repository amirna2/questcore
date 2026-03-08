/**
 * Map walker Declarations to typed export model interfaces.
 * Converts evaluated table data into Room, Entity, Rule, EventHandler, GameMeta.
 */

import type {
	Condition,
	Direction,
	Effect,
	Entity,
	EventHandler,
	GameMeta,
	Rule,
	WhenClause,
	BehaviorWeight,
	CombatStats,
	LootTable,
	Topic
} from '../model/export-types.js';
import type { Declaration } from './walker.js';
import { evalExpression, evalTable } from './table-eval.js';

export function mapGameMeta(decl: Declaration): GameMeta {
	const t = evalTable(decl.table) as Record<string, unknown>;
	const playerStats = t.player_stats as Record<string, unknown> | undefined;

	return {
		title: str(t.title),
		author: str(t.author),
		version: str(t.version),
		start: str(t.start),
		intro: str(t.intro),
		...(playerStats
			? {
					playerStats: {
						hp: num(playerStats.hp),
						maxHp: num(playerStats.max_hp),
						attack: num(playerStats.attack),
						defense: num(playerStats.defense)
					}
				}
			: {})
	};
}

export function mapRoom(decl: Declaration): import('../model/export-types.js').Room {
	const t = evalTable(decl.table) as Record<string, unknown>;

	return {
		id: decl.id ?? '',
		description: str(t.description),
		exits: (t.exits as Partial<Record<Direction, string>>) ?? {},
		fallbacks: (t.fallbacks as Record<string, string>) ?? {},
		rules: asStringArray(t.rules)
	};
}

export function mapEntity(decl: Declaration): Entity {
	const t = evalTable(decl.table) as Record<string, unknown>;
	const id = decl.id ?? '';

	const base = {
		id,
		name: str(t.name),
		description: str(t.description),
		location: str(t.location),
		customProperties: extractCustomProperties(t, decl.kind),
		rules: asStringArray(t.rules)
	};

	switch (decl.kind) {
		case 'Item':
			return {
				...base,
				kind: 'item',
				takeable: t.takeable !== false // default true
			};

		case 'NPC':
			return {
				...base,
				kind: 'npc',
				topics: mapTopics(t.topics as Record<string, unknown> | undefined)
			};

		case 'Enemy':
			return {
				...base,
				kind: 'enemy',
				stats: mapCombatStats(t.stats as Record<string, unknown> | undefined),
				behavior: mapBehavior(t.behavior as unknown[] | undefined),
				loot: mapLoot(t.loot as Record<string, unknown> | undefined)
			};

		default:
			return { ...base, kind: 'entity' };
	}
}

export function mapRule(decl: Declaration): Rule {
	if (!decl.ruleArgs) {
		return { id: decl.id ?? '', when: { verb: '' }, conditions: [], effects: [] };
	}

	const { when, conditions, then: thenTable } = decl.ruleArgs;

	// Parse When table
	const whenData = evalTable(when) as Record<string, unknown>;
	const whenClause: WhenClause = {
		verb: str(whenData.verb),
		...(whenData.object ? { object: str(whenData.object) } : {}),
		...(whenData.target ? { target: str(whenData.target) } : {}),
		...(whenData.object_kind ? { objectKind: str(whenData.object_kind) } : {}),
		...(whenData.priority !== undefined ? { priority: num(whenData.priority) } : {})
	};

	// Parse conditions (array of helper call expressions)
	const parsedConditions: Condition[] = conditions
		.map((node) => evalExpression(node))
		.filter((c): c is Condition => c !== null && typeof c === 'object' && 'type' in c);

	// Parse Then effects
	const thenFields = thenTable.fields;
	const parsedEffects: Effect[] = thenFields
		.filter((f) => f.type === 'TableValue')
		.map((f) => evalExpression(f.value))
		.filter((e): e is Effect => e !== null && typeof e === 'object' && 'type' in e);

	return {
		id: decl.id ?? '',
		when: whenClause,
		conditions: parsedConditions,
		effects: parsedEffects
	};
}

export function mapEventHandler(decl: Declaration): EventHandler {
	const t = evalTable(decl.table) as Record<string, unknown>;

	// Conditions
	const rawConditions = t.conditions;
	const parsedConditions: Condition[] = Array.isArray(rawConditions)
		? (rawConditions.filter(
				(c): c is Condition => c !== null && typeof c === 'object' && 'type' in c
			) as Condition[])
		: [];

	// Effects
	const rawEffects = t.effects;
	const parsedEffects: Effect[] = Array.isArray(rawEffects)
		? (rawEffects.filter(
				(e): e is Effect => e !== null && typeof e === 'object' && 'type' in e
			) as Effect[])
		: [];

	return {
		event: decl.id ?? '',
		conditions: parsedConditions,
		effects: parsedEffects
	};
}

// ── Helpers ────────────────────────────────────────────────────

function str(v: unknown): string {
	return typeof v === 'string' ? v : '';
}

function num(v: unknown): number {
	return typeof v === 'number' ? v : 0;
}

function asStringArray(v: unknown): string[] {
	if (!Array.isArray(v)) return [];
	return v.filter((x): x is string => typeof x === 'string');
}

/** Known fields for each entity kind — anything else is a custom property. */
const KNOWN_FIELDS: Record<string, Set<string>> = {
	Item: new Set(['name', 'description', 'location', 'takeable', 'rules']),
	NPC: new Set(['name', 'description', 'location', 'topics', 'rules']),
	Enemy: new Set(['name', 'description', 'location', 'stats', 'behavior', 'loot', 'rules']),
	Entity: new Set(['name', 'description', 'location', 'rules'])
};

function extractCustomProperties(
	t: Record<string, unknown>,
	kind: string
): Record<string, string | number | boolean> {
	const known = KNOWN_FIELDS[kind] ?? KNOWN_FIELDS['Entity'];
	const result: Record<string, string | number | boolean> = {};

	for (const [key, value] of Object.entries(t)) {
		if (known.has(key)) continue;
		if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') {
			result[key] = value;
		}
	}

	return result;
}

function mapTopics(raw: Record<string, unknown> | undefined): Record<string, Topic> {
	if (!raw) return {};
	const result: Record<string, Topic> = {};

	for (const [key, value] of Object.entries(raw)) {
		if (typeof value !== 'object' || value === null) continue;
		const topicData = value as Record<string, unknown>;

		const requires = Array.isArray(topicData.requires)
			? (topicData.requires.filter(
					(c): c is Condition => c !== null && typeof c === 'object' && 'type' in c
				) as Condition[])
			: [];

		const effects = Array.isArray(topicData.effects)
			? (topicData.effects.filter(
					(e): e is Effect => e !== null && typeof e === 'object' && 'type' in e
				) as Effect[])
			: [];

		result[key] = {
			text: str(topicData.text),
			requires,
			effects
		};
	}

	return result;
}

function mapCombatStats(raw: Record<string, unknown> | undefined): CombatStats {
	if (!raw) return { hp: 0, maxHp: 0, attack: 0, defense: 0 };
	return {
		hp: num(raw.hp),
		maxHp: num(raw.max_hp),
		attack: num(raw.attack),
		defense: num(raw.defense)
	};
}

function mapBehavior(raw: unknown[] | undefined): BehaviorWeight[] {
	if (!raw) return [];
	return raw
		.filter((item): item is Record<string, unknown> => typeof item === 'object' && item !== null)
		.map((item) => ({
			action: str(item.action),
			weight: num(item.weight)
		}));
}

function mapLoot(raw: Record<string, unknown> | undefined): LootTable {
	if (!raw) return { items: [], gold: 0 };

	const items = Array.isArray(raw.items)
		? raw.items
				.filter(
					(item): item is Record<string, unknown> => typeof item === 'object' && item !== null
				)
				.map((item) => ({
					id: str(item.id),
					chance: num(item.chance)
				}))
		: [];

	return { items, gold: num(raw.gold) };
}
