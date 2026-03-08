import { describe, it, expect } from 'vitest';
import { generateItems, generateNPCs, generateEnemies } from '../../src/lib/codegen/entities.js';
import type { Entity } from '../../src/lib/model/export-types.js';

describe('generateItems', () => {
	it('generates a basic takeable item', () => {
		const entities: Entity[] = [
			{
				kind: 'item',
				id: 'rusty_key',
				name: 'rusty key',
				description: 'A small iron key.',
				location: 'castle_gates',
				customProperties: {},
				rules: [],
				takeable: true
			}
		];

		const output = generateItems(entities);
		expect(output).toContain('Item "rusty_key"');
		expect(output).toContain('name = "rusty key"');
		expect(output).toContain('location = "castle_gates"');
		// takeable=true is the default, should not appear
		expect(output).not.toContain('takeable');
	});

	it('generates a non-takeable item with takeable = false', () => {
		const entities: Entity[] = [
			{
				kind: 'item',
				id: 'silver_dagger',
				name: 'silver dagger',
				description: 'A fine dagger.',
				location: 'armory',
				customProperties: {},
				rules: [],
				takeable: false
			}
		];

		const output = generateItems(entities);
		expect(output).toContain('takeable = false');
	});

	it('filters out non-item entities', () => {
		const entities: Entity[] = [
			{
				kind: 'npc',
				id: 'captain',
				name: 'Captain',
				description: 'A guard.',
				location: 'gates',
				customProperties: {},
				rules: [],
				topics: {}
			}
		];

		const output = generateItems(entities);
		expect(output).toBe('');
	});
});

describe('generateNPCs', () => {
	it('generates an NPC with topics', () => {
		const entities: Entity[] = [
			{
				kind: 'npc',
				id: 'captain',
				name: 'Captain Aldric',
				description: 'The captain of the guard.',
				location: 'castle_gates',
				customProperties: {},
				rules: [],
				topics: {
					greet: {
						text: 'Welcome, adventurer.',
						requires: [],
						effects: [{ type: 'set_flag', flag: 'met_captain', value: true }]
					},
					crown: {
						text: 'The crown vanished.',
						requires: [{ type: 'flag_set', flag: 'met_captain' }],
						effects: []
					}
				}
			}
		];

		const output = generateNPCs(entities);
		expect(output).toContain('NPC "captain"');
		expect(output).toContain('topics = {');
		expect(output).toContain('greet = {');
		expect(output).toContain('SetFlag("met_captain", true)');
		expect(output).toContain('FlagSet("met_captain")');
	});
});

describe('generateEnemies', () => {
	it('generates an enemy with stats, behavior, and loot', () => {
		const entities: Entity[] = [
			{
				kind: 'enemy',
				id: 'cave_goblin',
				name: 'Cave Goblin',
				description: 'A snarling goblin.',
				location: 'secret_passage',
				customProperties: {},
				rules: [],
				stats: { hp: 12, maxHp: 12, attack: 4, defense: 1 },
				behavior: [
					{ action: 'attack', weight: 70 },
					{ action: 'defend', weight: 20 },
					{ action: 'flee', weight: 10 }
				],
				loot: {
					items: [{ id: 'goblin_blade', chance: 50 }],
					gold: 5
				}
			}
		];

		const output = generateEnemies(entities);
		expect(output).toContain('Enemy "cave_goblin"');
		expect(output).toContain('max_hp  = 12');
		expect(output).toContain('attack  = 4');
		expect(output).toContain('{ action = "attack", weight = 70 }');
		expect(output).toContain('{ id = "goblin_blade", chance = 50 }');
		expect(output).toContain('gold  = 5');
	});
});
