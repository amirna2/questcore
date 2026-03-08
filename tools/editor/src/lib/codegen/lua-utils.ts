/**
 * Shared Lua code generation utilities.
 */

/** Escape a string for Lua double-quoted string literals. */
export function luaString(s: string): string {
	const escaped = s
		.replace(/\\/g, '\\\\')
		.replace(/"/g, '\\"')
		.replace(/\n/g, '\\n')
		.replace(/\r/g, '\\r')
		.replace(/\t/g, '\\t');
	return `"${escaped}"`;
}

/** Indent every line of a string by `level` indentation levels (4 spaces each). */
export function indent(s: string, level: number): string {
	const prefix = '    '.repeat(level);
	return s
		.split('\n')
		.map((line) => (line.trim() === '' ? '' : prefix + line))
		.join('\n');
}

/** Format a Lua value: string, number, or boolean. */
export function luaValue(v: string | number | boolean): string {
	if (typeof v === 'string') return luaString(v);
	if (typeof v === 'boolean') return v ? 'true' : 'false';
	return String(v);
}
