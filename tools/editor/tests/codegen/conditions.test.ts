import { describe, it, expect } from 'vitest';
import { generateCondition } from '../../src/lib/codegen/conditions.js';
import type { Condition } from '../../src/lib/model/export-types.js';

describe('generateCondition', () => {
	it('generates HasItem', () => {
		const c: Condition = { type: 'has_item', entity: 'rusty_key' };
		expect(generateCondition(c)).toBe('HasItem("rusty_key")');
	});

	it('generates FlagSet', () => {
		const c: Condition = { type: 'flag_set', flag: 'met_captain' };
		expect(generateCondition(c)).toBe('FlagSet("met_captain")');
	});

	it('generates FlagNot', () => {
		const c: Condition = { type: 'flag_not', flag: 'case_unlocked' };
		expect(generateCondition(c)).toBe('FlagNot("case_unlocked")');
	});

	it('generates FlagIs', () => {
		const c: Condition = { type: 'flag_is', flag: 'debug', value: true };
		expect(generateCondition(c)).toBe('FlagIs("debug", true)');
	});

	it('generates InRoom', () => {
		const c: Condition = { type: 'in_room', room: 'library' };
		expect(generateCondition(c)).toBe('InRoom("library")');
	});

	it('generates PropIs', () => {
		const c: Condition = { type: 'prop_is', entity: 'cave_goblin', prop: 'alive', value: true };
		expect(generateCondition(c)).toBe('PropIs("cave_goblin", "alive", true)');
	});

	it('generates CounterGt', () => {
		const c: Condition = { type: 'counter_gt', counter: 'score', value: 50 };
		expect(generateCondition(c)).toBe('CounterGt("score", 50)');
	});

	it('generates CounterLt', () => {
		const c: Condition = { type: 'counter_lt', counter: 'turns', value: 10 };
		expect(generateCondition(c)).toBe('CounterLt("turns", 10)');
	});

	it('generates InCombat', () => {
		const c: Condition = { type: 'in_combat' };
		expect(generateCondition(c)).toBe('InCombat()');
	});

	it('generates Not with nested condition', () => {
		const c: Condition = { type: 'not', condition: { type: 'in_combat' } };
		expect(generateCondition(c)).toBe('Not(InCombat())');
	});

	it('generates deeply nested Not', () => {
		const c: Condition = {
			type: 'not',
			condition: { type: 'has_item', entity: 'key' }
		};
		expect(generateCondition(c)).toBe('Not(HasItem("key"))');
	});
});
