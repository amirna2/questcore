/**
 * Generate rooms.lua from Room[].
 */

import type { Room } from '../model/export-types.js';
import { luaString } from './lua-utils.js';

export function generateRooms(rooms: Room[]): string {
	return rooms.map(generateRoom).join('\n');
}

function generateRoom(room: Room): string {
	const lines: string[] = [];
	lines.push(`Room ${luaString(room.id)} {`);
	lines.push(`    description = ${luaString(room.description)},`);

	const exitEntries = Object.entries(room.exits);
	if (exitEntries.length > 0) {
		lines.push('    exits = {');
		for (const [dir, target] of exitEntries) {
			lines.push(`        ${dir} = ${luaString(target!)}`);
		}
		lines.push('    },');
	}

	const fallbackEntries = Object.entries(room.fallbacks);
	if (fallbackEntries.length > 0) {
		lines.push('    fallbacks = {');
		for (const [verb, msg] of fallbackEntries) {
			lines.push(`        ${verb} = ${luaString(msg)}`);
		}
		lines.push('    }');
	}

	lines.push('}');
	return lines.join('\n') + '\n';
}
