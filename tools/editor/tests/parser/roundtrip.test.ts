import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { importLuaFiles } from '../../src/lib/parser/import.js';
import { generateGame } from '../../src/lib/codegen/game.js';
import { generateRooms } from '../../src/lib/codegen/rooms.js';
import { generateItems, generateNPCs, generateEnemies } from '../../src/lib/codegen/entities.js';
import { generateRules, generateEvents } from '../../src/lib/codegen/rules.js';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const GAMES_DIR = join(__dirname, '..', '..', '..', '..', 'games', 'lost_crown');

function readLuaFile(name: string): string {
	return readFileSync(join(GAMES_DIR, name), 'utf-8');
}

describe('Lost Crown import', () => {
	const files: Record<string, string> = {};

	// Read all Lost Crown files
	for (const name of ['game.lua', 'rooms.lua', 'items.lua', 'npcs.lua', 'enemies.lua', 'rules.lua']) {
		files[name] = readLuaFile(name);
	}

	const { project, warnings, summary } = importLuaFiles(files);

	it('imports without errors', () => {
		// Some warnings are expected for comments etc, but no parse failures
		const parseFailures = warnings.filter((w) => w.includes('Failed to parse'));
		expect(parseFailures).toHaveLength(0);
	});

	it('finds the correct number of rooms', () => {
		expect(summary.rooms).toBe(8);
	});

	it('finds the correct number of entities', () => {
		// 7 items (6 in items.lua + goblin_blade in enemies.lua) + 2 NPCs + 1 enemy = 10
		expect(summary.entities).toBe(10);
	});

	it('finds rules', () => {
		expect(summary.rules).toBeGreaterThan(0);
	});

	it('finds event handlers', () => {
		expect(summary.events).toBeGreaterThan(0);
	});

	it('parses game metadata correctly', () => {
		const meta = project.meta.data;
		expect(meta.title).toBe('The Lost Crown');
		expect(meta.author).toBe('QuestCore Team');
		expect(meta.version).toBe('0.1.0');
		expect(meta.start).toBe('castle_gates');
		expect(meta.playerStats).toBeDefined();
		expect(meta.playerStats!.hp).toBe(20);
		expect(meta.playerStats!.maxHp).toBe(20);
		expect(meta.playerStats!.attack).toBe(5);
		expect(meta.playerStats!.defense).toBe(2);
	});

	it('parses rooms with exits correctly', () => {
		const greatHall = project.rooms.find((r) => r.data.id === 'great_hall');
		expect(greatHall).toBeDefined();
		expect(greatHall!.data.exits).toEqual({
			south: 'castle_gates',
			east: 'library',
			north: 'throne_room',
			west: 'armory'
		});
	});

	it('parses room fallbacks correctly', () => {
		const gates = project.rooms.find((r) => r.data.id === 'castle_gates');
		expect(gates).toBeDefined();
		expect(gates!.data.fallbacks).toHaveProperty('open');
		expect(gates!.data.fallbacks).toHaveProperty('close');
	});

	it('parses item entities correctly', () => {
		const dagger = project.entities.find(
			(e) => e.data.id === 'silver_dagger' && e.data.kind === 'item'
		);
		expect(dagger).toBeDefined();
		expect(dagger!.data.kind).toBe('item');
		if (dagger!.data.kind === 'item') {
			expect(dagger!.data.takeable).toBe(false);
		}
	});

	it('parses NPC topics with conditions and effects', () => {
		const scholar = project.entities.find(
			(e) => e.data.id === 'scholar' && e.data.kind === 'npc'
		);
		expect(scholar).toBeDefined();
		if (scholar!.data.kind === 'npc') {
			const passageTopic = scholar!.data.topics.passage;
			expect(passageTopic).toBeDefined();
			expect(passageTopic.requires).toHaveLength(2); // HasItem + FlagSet
			expect(passageTopic.effects).toHaveLength(2); // SetFlag + Say
		}
	});

	it('parses enemy stats correctly', () => {
		const goblin = project.entities.find(
			(e) => e.data.id === 'cave_goblin' && e.data.kind === 'enemy'
		);
		expect(goblin).toBeDefined();
		if (goblin!.data.kind === 'enemy') {
			expect(goblin!.data.stats.hp).toBe(12);
			expect(goblin!.data.stats.maxHp).toBe(12);
			expect(goblin!.data.stats.attack).toBe(4);
			expect(goblin!.data.stats.defense).toBe(1);
			expect(goblin!.data.behavior).toHaveLength(3);
			expect(goblin!.data.loot.items).toHaveLength(1);
			expect(goblin!.data.loot.gold).toBe(5);
		}
	});

	it('parses rules with conditions and effects', () => {
		const readBook = project.rules.find((r) => r.data.id === 'read_old_book');
		expect(readBook).toBeDefined();
		expect(readBook!.data.when.verb).toBe('read');
		expect(readBook!.data.when.object).toBe('old_book');
		expect(readBook!.data.conditions).toHaveLength(1);
		expect(readBook!.data.conditions[0].type).toBe('has_item');
		expect(readBook!.data.effects.length).toBeGreaterThan(0);
	});

	it('parses rules with target in When clause', () => {
		const useKey = project.rules.find((r) => r.data.id === 'take_dagger_with_key');
		expect(useKey).toBeDefined();
		expect(useKey!.data.when.target).toBe('silver_dagger');
	});

	it('parses event handlers', () => {
		const crownEvent = project.events.find((e) => e.data.event === 'crown_recovered');
		expect(crownEvent).toBeDefined();
		expect(crownEvent!.data.effects.length).toBeGreaterThan(0);
	});
});

describe('Round-trip: import → export', () => {
	const files: Record<string, string> = {};
	for (const name of ['game.lua', 'rooms.lua', 'items.lua', 'npcs.lua', 'enemies.lua', 'rules.lua']) {
		files[name] = readLuaFile(name);
	}

	const { project } = importLuaFiles(files);

	it('re-exports game.lua with correct structure', () => {
		const output = generateGame(project.meta.data);
		expect(output).toContain('Game {');
		expect(output).toContain('title   = "The Lost Crown"');
		expect(output).toContain('start   = "castle_gates"');
		expect(output).toContain('player_stats = {');
		expect(output).toContain('hp = 20, max_hp = 20');
	});

	it('re-exports rooms.lua preserving all rooms', () => {
		const rooms = project.rooms.map((r) => r.data);
		const output = generateRooms(rooms);

		expect(output).toContain('Room "castle_gates"');
		expect(output).toContain('Room "great_hall"');
		expect(output).toContain('Room "throne_room"');
		expect(output).toContain('Room "library"');
		expect(output).toContain('Room "armory"');
		expect(output).toContain('Room "tower_stairs"');
		expect(output).toContain('Room "tower_top"');
		expect(output).toContain('Room "secret_passage"');
	});

	it('re-exports items preserving takeable=false', () => {
		const entities = project.entities.map((e) => e.data);
		const output = generateItems(entities);

		expect(output).toContain('Item "rusty_key"');
		expect(output).toContain('Item "silver_dagger"');
		expect(output).toContain('takeable = false');
	});

	it('re-exports NPCs with topics', () => {
		const entities = project.entities.map((e) => e.data);
		const output = generateNPCs(entities);

		expect(output).toContain('NPC "captain"');
		expect(output).toContain('NPC "scholar"');
		expect(output).toContain('topics = {');
		expect(output).toContain('SetFlag("met_captain", true)');
		expect(output).toContain('FlagSet("met_captain")');
	});

	it('re-exports enemies with stats and loot', () => {
		const entities = project.entities.map((e) => e.data);
		const output = generateEnemies(entities);

		expect(output).toContain('Enemy "cave_goblin"');
		expect(output).toContain('max_hp  = 12');
		expect(output).toContain('{ action = "attack", weight = 70 }');
		expect(output).toContain('{ id = "goblin_blade", chance = 50 }');
	});

	it('re-exports rules with conditions', () => {
		const rules = project.rules.map((r) => r.data);
		const output = generateRules(rules);

		expect(output).toContain('Rule("read_old_book"');
		expect(output).toContain('When { verb = "read", object = "old_book" }');
		expect(output).toContain('HasItem("old_book")');
	});

	it('re-exports event handlers', () => {
		const events = project.events.map((e) => e.data);
		const output = generateEvents(events);

		expect(output).toContain('On("crown_recovered"');
		expect(output).toContain('On("room_entered"');
	});
});
