import { describe, it, expect } from 'vitest';
import { generateRules, generateEvents } from '../../src/lib/codegen/rules.js';
import type { Rule, EventHandler } from '../../src/lib/model/export-types.js';

describe('generateRules', () => {
	it('generates a rule with conditions and effects', () => {
		const rules: Rule[] = [
			{
				id: 'read_old_book',
				when: { verb: 'read', object: 'old_book' },
				conditions: [{ type: 'has_item', entity: 'old_book' }],
				effects: [
					{ type: 'say', text: 'You read the page carefully.' },
					{ type: 'set_flag', flag: 'found_book_clue', value: true }
				]
			}
		];

		const output = generateRules(rules);
		expect(output).toContain('Rule("read_old_book",');
		expect(output).toContain('When { verb = "read", object = "old_book" }');
		expect(output).toContain('{ HasItem("old_book") }');
		expect(output).toContain('Then {');
		expect(output).toContain('Say("You read the page carefully.")');
		expect(output).toContain('SetFlag("found_book_clue", true)');
	});

	it('generates a rule without conditions', () => {
		const rules: Rule[] = [
			{
				id: 'simple',
				when: { verb: 'look' },
				conditions: [],
				effects: [{ type: 'say', text: 'You look around.' }]
			}
		];

		const output = generateRules(rules);
		// Should not have an empty conditions table
		expect(output).not.toContain('{ },');
		expect(output).toContain('When { verb = "look" }');
		expect(output).toContain('Then {');
	});

	it('generates a rule with target in When clause', () => {
		const rules: Rule[] = [
			{
				id: 'use_key',
				when: { verb: 'use', object: 'rusty_key', target: 'silver_dagger' },
				conditions: [
					{ type: 'has_item', entity: 'rusty_key' },
					{ type: 'in_room', room: 'armory' }
				],
				effects: [{ type: 'say', text: 'You unlock the case.' }]
			}
		];

		const output = generateRules(rules);
		expect(output).toContain('object = "rusty_key", target = "silver_dagger"');
		expect(output).toContain('HasItem("rusty_key"), InRoom("armory")');
	});
});

describe('generateEvents', () => {
	it('generates an event handler with conditions and effects', () => {
		const events: EventHandler[] = [
			{
				event: 'room_entered',
				conditions: [
					{ type: 'in_room', room: 'secret_passage' },
					{ type: 'prop_is', entity: 'cave_goblin', prop: 'alive', value: true },
					{ type: 'not', condition: { type: 'in_combat' } }
				],
				effects: [
					{ type: 'say', text: 'A Cave Goblin blocks your path!' },
					{ type: 'start_combat', entity: 'cave_goblin' }
				]
			}
		];

		const output = generateEvents(events);
		expect(output).toContain('On("room_entered", {');
		expect(output).toContain('conditions = {');
		expect(output).toContain('InRoom("secret_passage")');
		expect(output).toContain('Not(InCombat())');
		expect(output).toContain('effects = {');
		expect(output).toContain('StartCombat("cave_goblin")');
		expect(output).toContain('})');
	});

	it('generates an event without conditions', () => {
		const events: EventHandler[] = [
			{
				event: 'crown_recovered',
				conditions: [],
				effects: [{ type: 'say', text: 'Congratulations!' }]
			}
		];

		const output = generateEvents(events);
		expect(output).not.toContain('conditions');
		expect(output).toContain('effects = {');
	});
});
