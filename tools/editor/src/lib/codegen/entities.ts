/**
 * Generate items.lua, npcs.lua, enemies.lua from Entity[].
 */

import type {
	Entity,
	EnemyEntity,
	ItemEntity,
	NPCEntity,
	Topic
} from '../model/export-types.js';
import { luaString, luaValue } from './lua-utils.js';
import { generateCondition } from './conditions.js';
import { generateEffect } from './effects.js';

export function generateItems(entities: Entity[]): string {
	const items = entities.filter((e): e is ItemEntity => e.kind === 'item');
	return items.map(generateItem).join('\n');
}

export function generateNPCs(entities: Entity[]): string {
	const npcs = entities.filter((e): e is NPCEntity => e.kind === 'npc');
	return npcs.map(generateNPC).join('\n');
}

export function generateEnemies(entities: Entity[]): string {
	const enemies = entities.filter((e): e is EnemyEntity => e.kind === 'enemy');
	return enemies.map(generateEnemy).join('\n');
}

function generateItem(item: ItemEntity): string {
	const lines: string[] = [];
	lines.push(`Item ${luaString(item.id)} {`);
	lines.push(`    name = ${luaString(item.name)},`);
	lines.push(`    description = ${luaString(item.description)},`);
	lines.push(`    location = ${luaString(item.location)}`);

	if (!item.takeable) {
		// Replace last line to add trailing comma, then add takeable
		lines[lines.length - 1] += ',';
		lines.push('    takeable = false');
	}

	const propEntries = Object.entries(item.customProperties);
	if (propEntries.length > 0) {
		lines[lines.length - 1] += ',';
		for (const [key, value] of propEntries) {
			lines.push(`    ${key} = ${luaValue(value)}`);
		}
	}

	lines.push('}');
	return lines.join('\n') + '\n';
}

function generateNPC(npc: NPCEntity): string {
	const lines: string[] = [];
	lines.push(`NPC ${luaString(npc.id)} {`);
	lines.push(`    name = ${luaString(npc.name)},`);
	lines.push(`    description = ${luaString(npc.description)},`);
	lines.push(`    location = ${luaString(npc.location)},`);

	const topicEntries = Object.entries(npc.topics);
	if (topicEntries.length > 0) {
		lines.push('    topics = {');
		for (const [key, topic] of topicEntries) {
			lines.push(...generateTopic(key, topic).map((l) => '        ' + l));
		}
		lines.push('    }');
	}

	lines.push('}');
	return lines.join('\n') + '\n';
}

function generateTopic(key: string, topic: Topic): string[] {
	const lines: string[] = [];
	lines.push(`${key} = {`);
	lines.push(`    text = ${luaString(topic.text)},`);

	if (topic.requires.length > 0) {
		const conds = topic.requires.map(generateCondition).join(', ');
		lines.push(`    requires = { ${conds} }`);

		if (topic.effects.length > 0) {
			lines[lines.length - 1] += ',';
		}
	}

	if (topic.effects.length > 0) {
		if (topic.effects.length === 1) {
			lines.push(`    effects = { ${generateEffect(topic.effects[0])} }`);
		} else {
			lines.push('    effects = {');
			for (const effect of topic.effects) {
				lines.push(`        ${generateEffect(effect)},`);
			}
			lines.push('    }');
		}
	}

	lines.push('},');
	return lines;
}

function generateEnemy(enemy: EnemyEntity): string {
	const lines: string[] = [];
	lines.push(`Enemy ${luaString(enemy.id)} {`);
	lines.push(`    name        = ${luaString(enemy.name)},`);
	lines.push(`    description = ${luaString(enemy.description)},`);
	lines.push(`    location    = ${luaString(enemy.location)},`);

	// Stats
	const s = enemy.stats;
	lines.push('    stats = {');
	lines.push(`        hp      = ${s.hp},`);
	lines.push(`        max_hp  = ${s.maxHp},`);
	lines.push(`        attack  = ${s.attack},`);
	lines.push(`        defense = ${s.defense},`);
	lines.push('    },');

	// Behavior
	if (enemy.behavior.length > 0) {
		lines.push('    behavior = {');
		for (const b of enemy.behavior) {
			lines.push(`        { action = ${luaString(b.action)}, weight = ${b.weight} },`);
		}
		lines.push('    },');
	}

	// Loot
	lines.push('    loot = {');
	if (enemy.loot.items.length > 0) {
		lines.push(
			'        items = { ' +
				enemy.loot.items
					.map((i) => `{ id = ${luaString(i.id)}, chance = ${i.chance} }`)
					.join(', ') +
				' },'
		);
	}
	lines.push(`        gold  = ${enemy.loot.gold},`);
	lines.push('    },');

	lines.push('}');
	return lines.join('\n') + '\n';
}
