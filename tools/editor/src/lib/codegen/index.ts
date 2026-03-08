/**
 * Orchestrator: generates all .lua files from an EditorProject.
 */

import type { EditorProject } from '../model/editor-types.js';
import { generateGame } from './game.js';
import { generateRooms } from './rooms.js';
import { generateItems, generateNPCs, generateEnemies } from './entities.js';
import { generateRules, generateEvents } from './rules.js';

export interface GeneratedFiles {
	'game.lua': string;
	'rooms.lua': string;
	'items.lua': string;
	'npcs.lua': string;
	'enemies.lua': string;
	'rules.lua': string;
}

export function generateAll(project: EditorProject): GeneratedFiles {
	const entities = project.entities.map((e) => e.data);
	const rules = project.rules.map((r) => r.data);
	const events = project.events.map((e) => e.data);

	const rulesLua = generateRules(rules);
	const eventsLua = generateEvents(events);

	return {
		'game.lua': generateGame(project.meta.data),
		'rooms.lua': generateRooms(project.rooms.map((r) => r.data)),
		'items.lua': generateItems(entities),
		'npcs.lua': generateNPCs(entities),
		'enemies.lua': generateEnemies(entities),
		'rules.lua': rulesLua + (eventsLua ? '\n' + eventsLua : '')
	};
}
