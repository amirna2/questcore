import { describe, it, expect } from 'vitest';
import { generateRooms } from '../../src/lib/codegen/rooms.js';
import type { Room } from '../../src/lib/model/export-types.js';

describe('generateRooms', () => {
	it('generates a simple room with exits', () => {
		const rooms: Room[] = [
			{
				id: 'castle_gates',
				description: 'You stand before the imposing castle gates.',
				exits: { north: 'great_hall' },
				fallbacks: {},
				rules: []
			}
		];

		const output = generateRooms(rooms);
		expect(output).toBe(
			`Room "castle_gates" {
    description = "You stand before the imposing castle gates.",
    exits = {
        north = "great_hall"
    },
}
`
		);
	});

	it('generates a room with fallbacks', () => {
		const rooms: Room[] = [
			{
				id: 'throne_room',
				description: 'The throne room is grand.',
				exits: { south: 'great_hall' },
				fallbacks: { take: 'Everything belongs to the king.' },
				rules: []
			}
		];

		const output = generateRooms(rooms);
		expect(output).toContain('fallbacks = {');
		expect(output).toContain('take = "Everything belongs to the king."');
	});

	it('omits empty exits and fallbacks', () => {
		const rooms: Room[] = [
			{
				id: 'void',
				description: 'An empty void.',
				exits: {},
				fallbacks: {},
				rules: []
			}
		];

		const output = generateRooms(rooms);
		expect(output).not.toContain('exits');
		expect(output).not.toContain('fallbacks');
	});

	it('generates multiple rooms separated by blank lines', () => {
		const rooms: Room[] = [
			{ id: 'a', description: 'Room A.', exits: {}, fallbacks: {}, rules: [] },
			{ id: 'b', description: 'Room B.', exits: {}, fallbacks: {}, rules: [] }
		];

		const output = generateRooms(rooms);
		expect(output).toContain('Room "a"');
		expect(output).toContain('Room "b"');
		// Separated by blank line
		expect(output).toContain('}\n\nRoom "b"');
	});

	it('escapes special characters in descriptions', () => {
		const rooms: Room[] = [
			{
				id: 'test',
				description: 'A room with "quotes" and\nnewlines.',
				exits: {},
				fallbacks: {},
				rules: []
			}
		];

		const output = generateRooms(rooms);
		expect(output).toContain('"A room with \\"quotes\\" and\\nnewlines."');
	});
});
