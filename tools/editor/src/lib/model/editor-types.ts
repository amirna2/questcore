/**
 * Editor model types — wraps export model with editor-specific metadata.
 * The store and UI work with these types. Codegen strips the meta layer.
 */

import type { Entity, EventHandler, GameMeta, Room, Rule } from './export-types.js';

export interface EditorMeta {
	dirty: boolean;
	sortOrder: number;
	collapsed: boolean;
	importWarnings: string[];
	importSource?: string;
}

export interface EditorItem<T> {
	data: T;
	meta: EditorMeta;
}

export interface EditorProject {
	meta: EditorItem<GameMeta>;
	rooms: EditorItem<Room>[];
	entities: EditorItem<Entity>[];
	rules: EditorItem<Rule>[];
	events: EditorItem<EventHandler>[];
}

/** Derived indexes — rebuilt on change, not persisted */
export interface ProjectIndexes {
	roomById: Map<string, Room>;
	entityById: Map<string, Entity>;
	ruleById: Map<string, Rule>;
	entitiesByRoom: Map<string, Entity[]>;
}
