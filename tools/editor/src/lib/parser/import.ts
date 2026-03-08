/**
 * Import orchestrator: .lua files → EditorProject.
 *
 * Pipeline: source → luaparse AST → walker → declaration mapper → EditorProject
 */

// @ts-expect-error luaparse has no TypeScript declarations
import * as luaparse from 'luaparse';
import type { LuaChunk } from './lua-types.js';
import { walkChunk } from './walker.js';
import { mapGameMeta, mapRoom, mapEntity, mapRule, mapEventHandler } from './declaration-map.js';
import type { EditorProject } from '../model/editor-types.js';
import { createGameMeta, wrapEditorItem } from '../model/defaults.js';

export interface ImportResult {
	project: EditorProject;
	warnings: string[];
	summary: {
		rooms: number;
		entities: number;
		rules: number;
		events: number;
	};
}

/**
 * Import a set of .lua source strings into an EditorProject.
 * Each entry in the map is filename → source content.
 */
export function importLuaFiles(files: Record<string, string>): ImportResult {
	const allWarnings: string[] = [];
	const project: EditorProject = {
		meta: wrapEditorItem(createGameMeta(), 0),
		rooms: [],
		entities: [],
		rules: [],
		events: []
	};

	let sortCounter = 0;

	for (const [filename, source] of Object.entries(files)) {
		let chunk: LuaChunk;
		try {
			chunk = luaparse.parse(source) as LuaChunk;
		} catch (err) {
			allWarnings.push(`Failed to parse ${filename}: ${(err as Error).message}`);
			continue;
		}

		const { declarations, warnings } = walkChunk(chunk);
		for (const w of warnings) {
			allWarnings.push(`${filename}: ${w}`);
		}

		for (const decl of declarations) {
			const order = sortCounter++;

			switch (decl.kind) {
				case 'Game': {
					const meta = mapGameMeta(decl);
					project.meta = wrapEditorItem(meta, 0);
					project.meta.meta.importSource = filename;
					break;
				}
				case 'Room': {
					const room = mapRoom(decl);
					const item = wrapEditorItem(room, order);
					item.meta.importSource = filename;
					item.meta.dirty = false;
					project.rooms.push(item);
					break;
				}
				case 'Item':
				case 'NPC':
				case 'Enemy':
				case 'Entity': {
					const entity = mapEntity(decl);
					const item = wrapEditorItem(entity, order);
					item.meta.importSource = filename;
					item.meta.dirty = false;
					project.entities.push(item);
					break;
				}
				case 'Rule': {
					const rule = mapRule(decl);
					const item = wrapEditorItem(rule, order);
					item.meta.importSource = filename;
					item.meta.dirty = false;
					project.rules.push(item);
					break;
				}
				case 'On': {
					const handler = mapEventHandler(decl);
					const item = wrapEditorItem(handler, order);
					item.meta.importSource = filename;
					item.meta.dirty = false;
					project.events.push(item);
					break;
				}
			}
		}
	}

	return {
		project,
		warnings: allWarnings,
		summary: {
			rooms: project.rooms.length,
			entities: project.entities.length,
			rules: project.rules.length,
			events: project.events.length
		}
	};
}
