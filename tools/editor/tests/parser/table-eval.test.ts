import { describe, it, expect } from 'vitest';
import { evalExpression, evalTable } from '../../src/lib/parser/table-eval.js';
import type { LuaTableConstructor, LuaStringLiteral, LuaNumericLiteral, LuaBooleanLiteral, LuaCallExpression, LuaIdentifier } from '../../src/lib/parser/lua-types.js';

describe('evalExpression', () => {
	it('evaluates StringLiteral', () => {
		const node: LuaStringLiteral = { type: 'StringLiteral', value: 'hello', raw: '"hello"' };
		expect(evalExpression(node)).toBe('hello');
	});

	it('evaluates NumericLiteral', () => {
		const node: LuaNumericLiteral = { type: 'NumericLiteral', value: 42, raw: '42' };
		expect(evalExpression(node)).toBe(42);
	});

	it('evaluates BooleanLiteral', () => {
		const node: LuaBooleanLiteral = { type: 'BooleanLiteral', value: true, raw: 'true' };
		expect(evalExpression(node)).toBe(true);
	});

	it('evaluates negative numbers', () => {
		const node = {
			type: 'UnaryExpression' as const,
			operator: '-',
			argument: { type: 'NumericLiteral' as const, value: 5, raw: '5' }
		};
		expect(evalExpression(node)).toBe(-5);
	});

	it('evaluates a known helper call (HasItem)', () => {
		const node: LuaCallExpression = {
			type: 'CallExpression',
			base: { type: 'Identifier', name: 'HasItem' } as LuaIdentifier,
			arguments: [{ type: 'StringLiteral', value: 'key', raw: '"key"' } as LuaStringLiteral]
		};
		expect(evalExpression(node)).toEqual({ type: 'has_item', entity: 'key' });
	});
});

describe('evalTable', () => {
	it('evaluates a record table', () => {
		const node: LuaTableConstructor = {
			type: 'TableConstructorExpression',
			fields: [
				{
					type: 'TableKeyString',
					key: { type: 'Identifier', name: 'name' } as LuaIdentifier,
					value: { type: 'StringLiteral', value: 'test', raw: '"test"' } as LuaStringLiteral
				},
				{
					type: 'TableKeyString',
					key: { type: 'Identifier', name: 'count' } as LuaIdentifier,
					value: { type: 'NumericLiteral', value: 3, raw: '3' } as LuaNumericLiteral
				}
			]
		};

		expect(evalTable(node)).toEqual({ name: 'test', count: 3 });
	});

	it('evaluates an array table', () => {
		const node: LuaTableConstructor = {
			type: 'TableConstructorExpression',
			fields: [
				{
					type: 'TableValue',
					value: { type: 'StringLiteral', value: 'a', raw: '"a"' } as LuaStringLiteral
				},
				{
					type: 'TableValue',
					value: { type: 'StringLiteral', value: 'b', raw: '"b"' } as LuaStringLiteral
				}
			]
		};

		expect(evalTable(node)).toEqual(['a', 'b']);
	});

	it('evaluates an empty table as empty record', () => {
		const node: LuaTableConstructor = {
			type: 'TableConstructorExpression',
			fields: []
		};

		expect(evalTable(node)).toEqual({});
	});
});
