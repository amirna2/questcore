/**
 * Serialize Effect types to Lua constructor calls.
 */

import type { Effect } from '../model/export-types.js';
import { luaString, luaValue } from './lua-utils.js';

export function generateEffect(e: Effect): string {
	switch (e.type) {
		case 'say':
			return `Say(${luaString(e.text)})`;
		case 'give_item':
			return `GiveItem(${luaString(e.entity)})`;
		case 'remove_item':
			return `RemoveItem(${luaString(e.entity)})`;
		case 'set_flag':
			return `SetFlag(${luaString(e.flag)}, ${luaValue(e.value)})`;
		case 'inc_counter':
			return `IncCounter(${luaString(e.counter)}, ${e.amount})`;
		case 'set_counter':
			return `SetCounter(${luaString(e.counter)}, ${e.value})`;
		case 'set_prop':
			return `SetProp(${luaString(e.entity)}, ${luaString(e.prop)}, ${luaValue(e.value)})`;
		case 'move_entity':
			return `MoveEntity(${luaString(e.entity)}, ${luaString(e.room)})`;
		case 'move_player':
			return `MovePlayer(${luaString(e.room)})`;
		case 'open_exit':
			return `OpenExit(${luaString(e.room)}, ${luaString(e.direction)}, ${luaString(e.target)})`;
		case 'close_exit':
			return `CloseExit(${luaString(e.room)}, ${luaString(e.direction)})`;
		case 'emit_event':
			return `EmitEvent(${luaString(e.event)})`;
		case 'start_dialogue':
			return `StartDialogue(${luaString(e.entity)})`;
		case 'start_combat':
			return `StartCombat(${luaString(e.entity)})`;
		case 'stop':
			return `Stop()`;
	}
}
