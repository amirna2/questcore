import { describe, it, expect } from 'vitest';
// @ts-expect-error luaparse has no TypeScript declarations
import * as luaparse from 'luaparse';
import { walkChunk } from '../../src/lib/parser/walker.js';
import type { LuaChunk } from '../../src/lib/parser/lua-types.js';

function parse(source: string): LuaChunk {
	return luaparse.parse(source) as LuaChunk;
}

describe('walkChunk', () => {
	it('extracts a Game declaration', () => {
		const chunk = parse('Game { title = "Test" }');
		const { declarations, warnings } = walkChunk(chunk);

		expect(warnings).toHaveLength(0);
		expect(declarations).toHaveLength(1);
		expect(declarations[0].kind).toBe('Game');
		expect(declarations[0].id).toBeNull();
	});

	it('extracts a Room declaration with curried syntax', () => {
		const chunk = parse('Room "library" { description = "Books everywhere." }');
		const { declarations } = walkChunk(chunk);

		expect(declarations).toHaveLength(1);
		expect(declarations[0].kind).toBe('Room');
		expect(declarations[0].id).toBe('library');
	});

	it('extracts Item, NPC, Enemy declarations', () => {
		const source = `
			Item "key" { name = "rusty key", location = "hall" }
			NPC "guard" { name = "Guard", location = "gate" }
			Enemy "goblin" { name = "Goblin", location = "cave" }
		`;
		const { declarations } = walkChunk(parse(source));

		expect(declarations).toHaveLength(3);
		expect(declarations[0].kind).toBe('Item');
		expect(declarations[0].id).toBe('key');
		expect(declarations[1].kind).toBe('NPC');
		expect(declarations[2].kind).toBe('Enemy');
	});

	it('extracts a Rule declaration', () => {
		const source = `
			Rule("test_rule",
				When { verb = "look", object = "book" },
				{ HasItem("book") },
				Then { Say("You read it.") }
			)
		`;
		const { declarations } = walkChunk(parse(source));

		expect(declarations).toHaveLength(1);
		expect(declarations[0].kind).toBe('Rule');
		expect(declarations[0].id).toBe('test_rule');
		expect(declarations[0].ruleArgs).toBeDefined();
		expect(declarations[0].ruleArgs!.conditions).toHaveLength(1);
	});

	it('extracts a Rule without conditions', () => {
		const source = `
			Rule("simple",
				When { verb = "look" },
				Then { Say("You look around.") }
			)
		`;
		const { declarations } = walkChunk(parse(source));

		expect(declarations).toHaveLength(1);
		expect(declarations[0].ruleArgs!.conditions).toHaveLength(0);
	});

	it('extracts an On event handler', () => {
		const source = `
			On("room_entered", {
				conditions = { InRoom("cave") },
				effects = { Say("Spooky!") }
			})
		`;
		const { declarations } = walkChunk(parse(source));

		expect(declarations).toHaveLength(1);
		expect(declarations[0].kind).toBe('On');
		expect(declarations[0].id).toBe('room_entered');
	});

	it('warns on unrecognized statements', () => {
		const source = `
			local x = 5
			Room "test" { description = "ok" }
		`;
		const { declarations, warnings } = walkChunk(parse(source));

		expect(declarations).toHaveLength(1);
		expect(warnings).toHaveLength(1);
		expect(warnings[0]).toContain('Unrecognized');
	});

	it('handles multiple declarations in one file', () => {
		const source = `
			Room "a" { description = "Room A" }
			Room "b" { description = "Room B" }
			Item "key" { name = "key", location = "a" }
		`;
		const { declarations } = walkChunk(parse(source));

		expect(declarations).toHaveLength(3);
	});
});
