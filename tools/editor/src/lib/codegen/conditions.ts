/**
 * Serialize Condition types to Lua constructor calls.
 */

import type { Condition } from '../model/export-types.js';
import { luaString, luaValue } from './lua-utils.js';

export function generateCondition(c: Condition): string {
	switch (c.type) {
		case 'has_item':
			return `HasItem(${luaString(c.entity)})`;
		case 'flag_set':
			return `FlagSet(${luaString(c.flag)})`;
		case 'flag_not':
			return `FlagNot(${luaString(c.flag)})`;
		case 'flag_is':
			return `FlagIs(${luaString(c.flag)}, ${luaValue(c.value)})`;
		case 'in_room':
			return `InRoom(${luaString(c.room)})`;
		case 'prop_is':
			return `PropIs(${luaString(c.entity)}, ${luaString(c.prop)}, ${luaValue(c.value)})`;
		case 'counter_gt':
			return `CounterGt(${luaString(c.counter)}, ${c.value})`;
		case 'counter_lt':
			return `CounterLt(${luaString(c.counter)}, ${c.value})`;
		case 'in_combat':
			return `InCombat()`;
		case 'not':
			return `Not(${generateCondition(c.condition)})`;
	}
}
