/**
 * Factory functions for creating new editor items with sensible defaults.
 */

import type { Entity, EventHandler, GameMeta, Room, Rule } from './export-types.js';
import type { EditorItem, EditorMeta } from './editor-types.js';

function defaultMeta(sortOrder: number): EditorMeta {
	return {
		dirty: true,
		sortOrder,
		collapsed: false,
		importWarnings: []
	};
}

export function wrapEditorItem<T>(data: T, sortOrder: number): EditorItem<T> {
	return { data, meta: defaultMeta(sortOrder) };
}

export function createGameMeta(): GameMeta {
	return {
		title: '',
		author: '',
		version: '1.0',
		start: '',
		intro: ''
	};
}

export function createRoom(id: string): Room {
	return {
		id,
		description: '',
		exits: {},
		fallbacks: {},
		rules: []
	};
}

export function createItemEntity(id: string): Entity {
	return {
		kind: 'item',
		id,
		name: '',
		description: '',
		location: '',
		customProperties: {},
		rules: [],
		takeable: true
	};
}

export function createNPCEntity(id: string): Entity {
	return {
		kind: 'npc',
		id,
		name: '',
		description: '',
		location: '',
		customProperties: {},
		rules: [],
		topics: {}
	};
}

export function createEnemyEntity(id: string): Entity {
	return {
		kind: 'enemy',
		id,
		name: '',
		description: '',
		location: '',
		customProperties: {},
		rules: [],
		stats: { hp: 10, maxHp: 10, attack: 3, defense: 1 },
		behavior: [
			{ action: 'attack', weight: 70 },
			{ action: 'defend', weight: 30 }
		],
		loot: { items: [], gold: 0 }
	};
}

export function createRule(id: string): Rule {
	return {
		id,
		when: { verb: '' },
		conditions: [],
		effects: []
	};
}

export function createEventHandler(event: string): EventHandler {
	return {
		event,
		conditions: [],
		effects: []
	};
}
