/**
 * Generate game.lua from GameMeta.
 */

import type { GameMeta } from '../model/export-types.js';
import { luaString } from './lua-utils.js';

export function generateGame(meta: GameMeta): string {
	const lines: string[] = [];
	lines.push('Game {');
	lines.push(`    title   = ${luaString(meta.title)},`);
	lines.push(`    author  = ${luaString(meta.author)},`);
	lines.push(`    version = ${luaString(meta.version)},`);
	lines.push(`    start   = ${luaString(meta.start)},`);
	lines.push(`    intro   = ${luaString(meta.intro)},`);

	if (meta.playerStats) {
		const s = meta.playerStats;
		lines.push('    player_stats = {');
		lines.push(`        hp = ${s.hp}, max_hp = ${s.maxHp},`);
		lines.push(`        attack = ${s.attack}, defense = ${s.defense},`);
		lines.push('    },');
	}

	lines.push('}');
	return lines.join('\n') + '\n';
}
