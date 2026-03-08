/**
 * AST walker: traverse top-level statements in a parsed Lua file and
 * extract QuestCore DSL declarations (Game, Room, Item, NPC, Enemy,
 * Entity, Rule, On).
 */

import type {
	LuaChunk,
	LuaExpression,
	LuaStatement,
	LuaStringLiteral,
	LuaTableConstructor
} from './lua-types.js';

/** A recognized top-level declaration extracted from the AST. */
export interface Declaration {
	kind: 'Game' | 'Room' | 'Item' | 'NPC' | 'Enemy' | 'Entity' | 'Rule' | 'On';
	id: string | null;
	table: LuaTableConstructor;
	/** For Rule: the When, conditions, and Then arguments */
	ruleArgs?: {
		when: LuaTableConstructor;
		conditions: LuaExpression[];
		then: LuaTableConstructor;
	};
}

export interface WalkResult {
	declarations: Declaration[];
	warnings: string[];
}

const KNOWN_DECLARATIONS = new Set(['Game', 'Room', 'Item', 'NPC', 'Enemy', 'Entity', 'Rule', 'On']);

/**
 * Walk a parsed Lua chunk and extract all recognized declarations.
 */
export function walkChunk(chunk: LuaChunk): WalkResult {
	const declarations: Declaration[] = [];
	const warnings: string[] = [];

	for (const stmt of chunk.body) {
		if (stmt.type !== 'CallStatement') {
			warnings.push(`Unrecognized statement type "${stmt.type}" — skipped`);
			continue;
		}

		const decl = extractDeclaration(stmt.expression);
		if (decl) {
			declarations.push(decl);
		} else {
			warnings.push(`Unrecognized call expression — skipped`);
		}
	}

	return { declarations, warnings };
}

/**
 * Try to extract a Declaration from a call expression.
 * Handles these patterns:
 *
 * 1. Game { ... }           → TableCallExpression(Identifier("Game"), table)
 * 2. Room "id" { ... }      → TableCallExpression(StringCallExpression(Identifier("Room"), "id"), table)
 * 3. Rule("id", When{}, {conds}, Then{}) → CallExpression(Identifier("Rule"), args...)
 * 4. On("event", { ... })   → CallExpression(Identifier("On"), args...)
 */
function extractDeclaration(expr: LuaExpression): Declaration | null {
	// Pattern 1: Game { ... } — TableCallExpression with Identifier base
	if (expr.type === 'TableCallExpression' && expr.base.type === 'Identifier') {
		const name = expr.base.name;
		if (name === 'Game' && KNOWN_DECLARATIONS.has(name)) {
			return { kind: 'Game', id: null, table: expr.arguments };
		}
	}

	// Pattern 2: Room "id" { ... } — TableCallExpression with StringCallExpression base
	if (expr.type === 'TableCallExpression' && expr.base.type === 'StringCallExpression') {
		const innerBase = expr.base.base;
		if (innerBase.type === 'Identifier' && KNOWN_DECLARATIONS.has(innerBase.name)) {
			const kind = innerBase.name as Declaration['kind'];
			const id = extractStringValue(expr.base.argument);
			return { kind, id, table: expr.arguments };
		}
	}

	// Pattern 3: Rule("id", When{...}, {conds...}, Then{...})
	if (expr.type === 'CallExpression' && expr.base.type === 'Identifier') {
		const name = expr.base.name;

		if (name === 'Rule') {
			return extractRuleDeclaration(expr.arguments);
		}

		// Pattern 4: On("event", { ... })
		if (name === 'On') {
			return extractOnDeclaration(expr.arguments);
		}
	}

	return null;
}

function extractRuleDeclaration(args: LuaExpression[]): Declaration | null {
	if (args.length < 3) return null;

	// arg[0] = rule ID (string)
	const idNode = args[0];
	const id = idNode.type === 'StringLiteral' ? extractStringValue(idNode) : null;

	// Find the When and Then arguments
	let whenTable: LuaTableConstructor | null = null;
	let thenTable: LuaTableConstructor | null = null;
	const conditionNodes: LuaExpression[] = [];

	for (let i = 1; i < args.length; i++) {
		const arg = args[i];

		// When { ... } — TableCallExpression with Identifier "When"
		if (
			arg.type === 'TableCallExpression' &&
			arg.base.type === 'Identifier' &&
			arg.base.name === 'When'
		) {
			whenTable = arg.arguments;
			continue;
		}

		// Then { ... } — TableCallExpression with Identifier "Then"
		if (
			arg.type === 'TableCallExpression' &&
			arg.base.type === 'Identifier' &&
			arg.base.name === 'Then'
		) {
			thenTable = arg.arguments;
			continue;
		}

		// Conditions table — a bare TableConstructorExpression
		if (arg.type === 'TableConstructorExpression') {
			for (const field of arg.fields) {
				if (field.type === 'TableValue') {
					conditionNodes.push(field.value);
				}
			}
		}
	}

	if (!whenTable || !thenTable) return null;

	// Build a dummy table for the Declaration interface (Rule doesn't use it directly)
	const dummyTable: LuaTableConstructor = { type: 'TableConstructorExpression', fields: [] };

	return {
		kind: 'Rule',
		id,
		table: dummyTable,
		ruleArgs: {
			when: whenTable,
			conditions: conditionNodes,
			then: thenTable
		}
	};
}

function extractOnDeclaration(args: LuaExpression[]): Declaration | null {
	if (args.length < 2) return null;

	// arg[0] = event name (string)
	const eventNode = args[0];
	const event = eventNode.type === 'StringLiteral' ? extractStringValue(eventNode) : null;

	// arg[1] = table with conditions and effects
	const tableArg = args[1];
	if (tableArg.type !== 'TableConstructorExpression') return null;

	return {
		kind: 'On',
		id: event,
		table: tableArg
	};
}

/**
 * Extract a string value from a StringLiteral node.
 * In default encoding mode, `value` is null — fall back to `raw`.
 */
function extractStringValue(node: LuaStringLiteral): string {
	if (node.value !== null) return node.value;

	const raw = node.raw;
	// Long bracket strings
	const longMatch = raw.match(/^\[=*\[([\s\S]*)\]=*\]$/);
	if (longMatch) return longMatch[1];

	// Quoted strings
	if ((raw.startsWith('"') && raw.endsWith('"')) || (raw.startsWith("'") && raw.endsWith("'"))) {
		return raw
			.slice(1, -1)
			.replace(/\\n/g, '\n')
			.replace(/\\r/g, '\r')
			.replace(/\\t/g, '\t')
			.replace(/\\"/g, '"')
			.replace(/\\'/g, "'")
			.replace(/\\\\/g, '\\');
	}

	return raw;
}
