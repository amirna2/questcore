/**
 * Reactive game project store using Svelte 5 runes.
 * Single source of truth for all editor state.
 */

import type { Entity, EventHandler, GameMeta, Room, Rule } from '../model/export-types.js';
import type { EditorItem, EditorProject, ProjectIndexes } from '../model/editor-types.js';
import {
	createGameMeta,
	createRoom,
	createItemEntity,
	createNPCEntity,
	createEnemyEntity,
	createRule,
	createEventHandler,
	wrapEditorItem
} from '../model/defaults.js';

/** What kind of thing is selected in the explorer tree */
export type SelectionKind = 'game' | 'room' | 'entity' | 'rule' | 'event';

export interface Selection {
	kind: SelectionKind;
	id: string | null; // null for 'game' (singleton)
}

function createProjectStore() {
	let project = $state<EditorProject>({
		meta: wrapEditorItem(createGameMeta(), 0),
		rooms: [],
		entities: [],
		rules: [],
		events: []
	});

	let selection = $state<Selection | null>(null);

	// Derived indexes for O(1) lookups
	const indexes = $derived<ProjectIndexes>({
		roomById: new Map(project.rooms.map((r) => [r.data.id, r.data])),
		entityById: new Map(project.entities.map((e) => [e.data.id, e.data])),
		ruleById: new Map(project.rules.map((r) => [r.data.id, r.data])),
		entitiesByRoom: project.entities.reduce((acc, e) => {
			const loc = e.data.location;
			if (!acc.has(loc)) acc.set(loc, []);
			acc.get(loc)!.push(e.data);
			return acc;
		}, new Map<string, Entity[]>())
	});

	// ── Selection ──────────────────────────────────────────────

	function select(sel: Selection | null) {
		selection = sel;
	}

	function getSelected(): Selection | null {
		return selection;
	}

	// ── Getters ────────────────────────────────────────────────

	function getProject(): EditorProject {
		return project;
	}

	function getIndexes(): ProjectIndexes {
		return indexes;
	}

	// ── CRUD: Rooms ────────────────────────────────────────────

	function addRoom(id: string): EditorItem<Room> {
		const room = createRoom(id);
		const item = wrapEditorItem(room, project.rooms.length);
		project.rooms.push(item);
		return item;
	}

	function updateRoom(id: string, updater: (room: Room) => void) {
		const item = project.rooms.find((r) => r.data.id === id);
		if (item) {
			updater(item.data);
			item.meta.dirty = true;
		}
	}

	function removeRoom(id: string) {
		project.rooms = project.rooms.filter((r) => r.data.id !== id);
		if (selection?.kind === 'room' && selection.id === id) {
			selection = null;
		}
	}

	// ── CRUD: Entities ─────────────────────────────────────────

	function addEntity(id: string, kind: 'item' | 'npc' | 'enemy'): EditorItem<Entity> {
		let entity: Entity;
		switch (kind) {
			case 'item':
				entity = createItemEntity(id);
				break;
			case 'npc':
				entity = createNPCEntity(id);
				break;
			case 'enemy':
				entity = createEnemyEntity(id);
				break;
		}
		const item = wrapEditorItem(entity, project.entities.length);
		project.entities.push(item);
		return item;
	}

	function updateEntity(id: string, updater: (entity: Entity) => void) {
		const item = project.entities.find((e) => e.data.id === id);
		if (item) {
			updater(item.data);
			item.meta.dirty = true;
		}
	}

	function removeEntity(id: string) {
		project.entities = project.entities.filter((e) => e.data.id !== id);
		if (selection?.kind === 'entity' && selection.id === id) {
			selection = null;
		}
	}

	// ── CRUD: Rules ────────────────────────────────────────────

	function addRule(id: string): EditorItem<Rule> {
		const rule = createRule(id);
		const item = wrapEditorItem(rule, project.rules.length);
		project.rules.push(item);
		return item;
	}

	function updateRule(id: string, updater: (rule: Rule) => void) {
		const item = project.rules.find((r) => r.data.id === id);
		if (item) {
			updater(item.data);
			item.meta.dirty = true;
		}
	}

	function removeRule(id: string) {
		project.rules = project.rules.filter((r) => r.data.id !== id);
		if (selection?.kind === 'rule' && selection.id === id) {
			selection = null;
		}
	}

	// ── CRUD: Events ───────────────────────────────────────────

	function addEvent(eventName: string): EditorItem<EventHandler> {
		const handler = createEventHandler(eventName);
		const item = wrapEditorItem(handler, project.events.length);
		project.events.push(item);
		return item;
	}

	function updateEvent(index: number, updater: (handler: EventHandler) => void) {
		const item = project.events[index];
		if (item) {
			updater(item.data);
			item.meta.dirty = true;
		}
	}

	function removeEvent(index: number) {
		project.events.splice(index, 1);
	}

	// ── Game Meta ──────────────────────────────────────────────

	function updateGameMeta(updater: (meta: GameMeta) => void) {
		updater(project.meta.data);
		project.meta.meta.dirty = true;
	}

	// ── Bulk Operations ────────────────────────────────────────

	function loadProject(newProject: EditorProject) {
		project = newProject;
		selection = null;
	}

	function newProject() {
		project = {
			meta: wrapEditorItem(createGameMeta(), 0),
			rooms: [],
			entities: [],
			rules: [],
			events: []
		};
		selection = null;
	}

	return {
		get project() { return project; },
		get selection() { return selection; },
		get indexes() { return indexes; },
		select,
		getSelected,
		getProject,
		getIndexes,
		addRoom,
		updateRoom,
		removeRoom,
		addEntity,
		updateEntity,
		removeEntity,
		addRule,
		updateRule,
		removeRule,
		addEvent,
		updateEvent,
		removeEvent,
		updateGameMeta,
		loadProject,
		newProject
	};
}

export const projectStore = createProjectStore();
