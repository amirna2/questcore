/**
 * Minimal type definitions for luaparse AST nodes we care about.
 * luaparse doesn't ship TypeScript types, so we define what we need.
 */

export interface LuaChunk {
	type: 'Chunk';
	body: LuaStatement[];
}

export type LuaStatement = LuaCallStatement | LuaAssignmentStatement | LuaLocalStatement;

export interface LuaCallStatement {
	type: 'CallStatement';
	expression: LuaExpression;
}

export interface LuaAssignmentStatement {
	type: 'AssignmentStatement';
	variables: LuaExpression[];
	init: LuaExpression[];
}

export interface LuaLocalStatement {
	type: 'LocalStatement';
	variables: LuaIdentifier[];
	init: LuaExpression[];
}

export type LuaExpression =
	| LuaCallExpression
	| LuaStringCallExpression
	| LuaTableCallExpression
	| LuaStringLiteral
	| LuaNumericLiteral
	| LuaBooleanLiteral
	| LuaNilLiteral
	| LuaIdentifier
	| LuaTableConstructor
	| LuaUnaryExpression;

export interface LuaCallExpression {
	type: 'CallExpression';
	base: LuaExpression;
	arguments: LuaExpression[];
}

export interface LuaStringCallExpression {
	type: 'StringCallExpression';
	base: LuaExpression;
	argument: LuaStringLiteral;
}

export interface LuaTableCallExpression {
	type: 'TableCallExpression';
	base: LuaExpression;
	arguments: LuaTableConstructor;
}

export interface LuaStringLiteral {
	type: 'StringLiteral';
	value: string | null; // null when encodingMode is 'none' (default)
	raw: string;
}

export interface LuaNumericLiteral {
	type: 'NumericLiteral';
	value: number;
	raw: string;
}

export interface LuaBooleanLiteral {
	type: 'BooleanLiteral';
	value: boolean;
	raw: string;
}

export interface LuaNilLiteral {
	type: 'NilLiteral';
	raw: string;
}

export interface LuaIdentifier {
	type: 'Identifier';
	name: string;
}

export interface LuaTableConstructor {
	type: 'TableConstructorExpression';
	fields: LuaTableField[];
}

export type LuaTableField = LuaTableKeyString | LuaTableKey | LuaTableValue;

export interface LuaTableKeyString {
	type: 'TableKeyString';
	key: LuaIdentifier;
	value: LuaExpression;
}

export interface LuaTableKey {
	type: 'TableKey';
	key: LuaExpression;
	value: LuaExpression;
}

export interface LuaTableValue {
	type: 'TableValue';
	value: LuaExpression;
}

export interface LuaUnaryExpression {
	type: 'UnaryExpression';
	operator: string;
	argument: LuaExpression;
}
