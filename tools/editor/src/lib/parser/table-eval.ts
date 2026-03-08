/**
 * Evaluate Lua AST nodes into plain TypeScript values.
 * Handles literals, table constructors, and known DSL helper calls
 * (conditions and effects).
 */

import type {
	LuaExpression,
	LuaTableConstructor,
	LuaTableField
} from './lua-types.js';
import type { Condition, Effect } from '../model/export-types.js';

/** Evaluate any expression node to a JS value. */
export function evalExpression(node: LuaExpression): unknown {
	switch (node.type) {
		case 'StringLiteral':
			return node.value ?? extractStringFromRaw(node.raw);
		case 'NumericLiteral':
			return node.value;
		case 'BooleanLiteral':
			return node.value;
		case 'NilLiteral':
			return null;
		case 'UnaryExpression':
			if (node.operator === '-' && node.argument.type === 'NumericLiteral') {
				return -node.argument.value;
			}
			return null;
		case 'TableConstructorExpression':
			return evalTable(node);
		case 'CallExpression':
		case 'StringCallExpression':
		case 'TableCallExpression':
			return evalHelperCall(node);
		default:
			return null;
	}
}

/** Evaluate a table constructor to either a Record or an Array. */
export function evalTable(node: LuaTableConstructor): Record<string, unknown> | unknown[] {
	if (node.fields.length === 0) return {};

	// If all fields are TableValue (no keys), it's an array
	if (node.fields.every((f) => f.type === 'TableValue')) {
		return node.fields.map((f) => {
			if (f.type === 'TableValue') return evalExpression(f.value);
			return null;
		});
	}

	// Otherwise, it's a record
	const result: Record<string, unknown> = {};
	for (const field of node.fields) {
		if (field.type === 'TableKeyString') {
			result[field.key.name] = evalExpression(field.value);
		} else if (field.type === 'TableKey' && field.key.type === 'StringLiteral') {
			const key = field.key.value ?? extractStringFromRaw(field.key.raw);
			result[key] = evalExpression(field.value);
		} else if (field.type === 'TableValue') {
			// Mixed table — skip positional values in record context
		}
	}
	return result;
}

/**
 * Evaluate a known DSL helper call (condition or effect constructors).
 * Returns a typed Condition or Effect, or a plain object for unknown calls.
 */
function evalHelperCall(node: LuaExpression): Condition | Effect | Record<string, unknown> | null {
	const { name, args } = extractCallInfo(node);
	if (!name) return null;

	// Conditions
	switch (name) {
		case 'HasItem':
			return { type: 'has_item', entity: strArg(args, 0) };
		case 'FlagSet':
			return { type: 'flag_set', flag: strArg(args, 0) };
		case 'FlagNot':
			return { type: 'flag_not', flag: strArg(args, 0) };
		case 'FlagIs':
			return { type: 'flag_is', flag: strArg(args, 0), value: boolArg(args, 1) };
		case 'InRoom':
			return { type: 'in_room', room: strArg(args, 0) };
		case 'PropIs':
			return {
				type: 'prop_is',
				entity: strArg(args, 0),
				prop: strArg(args, 1),
				value: evalExpression(args[2]) as string | number | boolean
			};
		case 'CounterGt':
			return { type: 'counter_gt', counter: strArg(args, 0), value: numArg(args, 1) };
		case 'CounterLt':
			return { type: 'counter_lt', counter: strArg(args, 0), value: numArg(args, 1) };
		case 'InCombat':
			return { type: 'in_combat' };
		case 'Not': {
			const inner = evalHelperCall(args[0]);
			return { type: 'not', condition: inner as Condition };
		}

		// Effects
		case 'Say':
			return { type: 'say', text: strArg(args, 0) };
		case 'GiveItem':
			return { type: 'give_item', entity: strArg(args, 0) };
		case 'RemoveItem':
			return { type: 'remove_item', entity: strArg(args, 0) };
		case 'SetFlag':
			return { type: 'set_flag', flag: strArg(args, 0), value: boolArg(args, 1) };
		case 'IncCounter':
			return { type: 'inc_counter', counter: strArg(args, 0), amount: numArg(args, 1) };
		case 'SetCounter':
			return { type: 'set_counter', counter: strArg(args, 0), value: numArg(args, 1) };
		case 'SetProp':
			return {
				type: 'set_prop',
				entity: strArg(args, 0),
				prop: strArg(args, 1),
				value: evalExpression(args[2]) as string | number | boolean
			};
		case 'MoveEntity':
			return { type: 'move_entity', entity: strArg(args, 0), room: strArg(args, 1) };
		case 'MovePlayer':
			return { type: 'move_player', room: strArg(args, 0) };
		case 'OpenExit':
			return {
				type: 'open_exit',
				room: strArg(args, 0),
				direction: strArg(args, 1) as import('../model/export-types.js').Direction,
				target: strArg(args, 2)
			};
		case 'CloseExit':
			return { type: 'close_exit', room: strArg(args, 0), direction: strArg(args, 1) };
		case 'EmitEvent':
			return { type: 'emit_event', event: strArg(args, 0) };
		case 'StartDialogue':
			return { type: 'start_dialogue', entity: strArg(args, 0) };
		case 'StartCombat':
			return { type: 'start_combat', entity: strArg(args, 0) };
		case 'Stop':
			return { type: 'stop' };

		default:
			return null;
	}
}

/** Extract function name and arguments from any call expression shape. */
function extractCallInfo(node: LuaExpression): { name: string | null; args: LuaExpression[] } {
	switch (node.type) {
		case 'CallExpression':
			return {
				name: node.base.type === 'Identifier' ? node.base.name : null,
				args: node.arguments
			};
		case 'StringCallExpression':
			return {
				name: node.base.type === 'Identifier' ? node.base.name : null,
				args: [node.argument]
			};
		case 'TableCallExpression':
			return {
				name: node.base.type === 'Identifier' ? node.base.name : null,
				args: [node.arguments]
			};
		default:
			return { name: null, args: [] };
	}
}

function strArg(args: LuaExpression[], i: number): string {
	const val = evalExpression(args[i]);
	return typeof val === 'string' ? val : '';
}

function numArg(args: LuaExpression[], i: number): number {
	const val = evalExpression(args[i]);
	return typeof val === 'number' ? val : 0;
}

function boolArg(args: LuaExpression[], i: number): boolean {
	const val = evalExpression(args[i]);
	return typeof val === 'boolean' ? val : false;
}

/**
 * Extract a string value from a luaparse raw string literal.
 * When encodingMode is 'none' (default), StringLiteral.value is null
 * but .raw contains the quoted source text. This handles double-quoted,
 * single-quoted, and long bracket strings.
 */
function extractStringFromRaw(raw: string): string {
	// Long bracket strings: [[...]], [=[...]=], etc.
	const longMatch = raw.match(/^\[=*\[([\s\S]*)\]=*\]$/);
	if (longMatch) return longMatch[1];

	// Quoted strings: strip outer quotes and unescape
	if ((raw.startsWith('"') && raw.endsWith('"')) || (raw.startsWith("'") && raw.endsWith("'"))) {
		const inner = raw.slice(1, -1);
		return inner
			.replace(/\\n/g, '\n')
			.replace(/\\r/g, '\r')
			.replace(/\\t/g, '\t')
			.replace(/\\"/g, '"')
			.replace(/\\'/g, "'")
			.replace(/\\\\/g, '\\');
	}

	return raw;
}
