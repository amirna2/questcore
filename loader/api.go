package loader

import (
	lua "github.com/yuin/gopher-lua"
)

// registerAPI registers all Lua constructors and helpers as globals.
func registerAPI(L *lua.LState, coll *collector) {
	registerConstructors(L, coll)
	registerConditionHelpers(L)
	registerEffectHelpers(L)
}

func registerConstructors(L *lua.LState, coll *collector) {
	// Game { title = "...", ... }
	L.SetGlobal("Game", L.NewFunction(func(L *lua.LState) int {
		tbl := L.CheckTable(1)
		coll.game = tbl
		return 0
	}))

	// Room "id" { ... } — curried: Room("id") returns a function that takes a table.
	L.SetGlobal("Room", L.NewFunction(func(L *lua.LState) int {
		id := L.CheckString(1)
		L.Push(L.NewFunction(func(L *lua.LState) int {
			tbl := L.CheckTable(1)
			coll.rooms = append(coll.rooms, rawRoom{id: id, table: tbl})
			return 0
		}))
		return 1
	}))

	// Item "id" { ... } — curried, kind = "item".
	L.SetGlobal("Item", L.NewFunction(func(L *lua.LState) int {
		id := L.CheckString(1)
		L.Push(L.NewFunction(func(L *lua.LState) int {
			tbl := L.CheckTable(1)
			coll.entities = append(coll.entities, rawEntity{id: id, kind: "item", table: tbl})
			return 0
		}))
		return 1
	}))

	// NPC "id" { ... } — curried, kind = "npc".
	L.SetGlobal("NPC", L.NewFunction(func(L *lua.LState) int {
		id := L.CheckString(1)
		L.Push(L.NewFunction(func(L *lua.LState) int {
			tbl := L.CheckTable(1)
			coll.entities = append(coll.entities, rawEntity{id: id, kind: "npc", table: tbl})
			return 0
		}))
		return 1
	}))

	// Entity "id" { ... } — curried, kind = "entity".
	L.SetGlobal("Entity", L.NewFunction(func(L *lua.LState) int {
		id := L.CheckString(1)
		L.Push(L.NewFunction(func(L *lua.LState) int {
			tbl := L.CheckTable(1)
			coll.entities = append(coll.entities, rawEntity{id: id, kind: "entity", table: tbl})
			return 0
		}))
		return 1
	}))

	// Rule("id", when, conditions, then)
	// conditions may be nil.
	// Returns a marker table with __rule_id for scoping.
	L.SetGlobal("Rule", L.NewFunction(func(L *lua.LState) int {
		id := L.CheckString(1)
		when := L.CheckTable(2)

		var conditions *lua.LTable
		// Arg 3 can be a conditions table or nil (if nil, arg 4 is the then table at position 3).
		// We need to handle: Rule("id", when, conds, then) and Rule("id", when, then)
		arg3 := L.Get(3)
		arg4 := L.Get(4)

		var thenTbl *lua.LTable
		if arg4 != lua.LNil {
			// 4-arg form: Rule("id", when, conditions, then)
			if t, ok := arg3.(*lua.LTable); ok {
				conditions = t
			}
			thenTbl = L.CheckTable(4)
		} else {
			// 3-arg form: Rule("id", when, then)
			thenTbl = L.CheckTable(3)
		}

		order := coll.nextSourceOrder()
		coll.rules = append(coll.rules, rawRule{
			id:         id,
			when:       when,
			conditions: conditions,
			then:       thenTbl,
			scope:      "global",
			order:      order,
		})

		// Return a marker table so rooms/entities can reference this rule.
		marker := L.NewTable()
		marker.RawSetString("__rule_id", lua.LString(id))
		L.Push(marker)
		return 1
	}))

	// On("event_type", { conditions = {...}, effects = {...} })
	L.SetGlobal("On", L.NewFunction(func(L *lua.LState) int {
		eventType := L.CheckString(1)
		tbl := L.CheckTable(2)
		coll.handlers = append(coll.handlers, rawHandler{eventType: eventType, table: tbl})
		return 0
	}))

	// When { verb = "..." } — pass-through, returns the table.
	L.SetGlobal("When", L.NewFunction(func(L *lua.LState) int {
		tbl := L.CheckTable(1)
		L.Push(tbl)
		return 1
	}))

	// Then { effect1, effect2, ... } — pass-through, returns the table.
	L.SetGlobal("Then", L.NewFunction(func(L *lua.LState) int {
		tbl := L.CheckTable(1)
		L.Push(tbl)
		return 1
	}))
}

func registerConditionHelpers(L *lua.LState) {
	// HasItem("key")
	L.SetGlobal("HasItem", L.NewFunction(func(L *lua.LState) int {
		item := L.CheckString(1)
		tbl := L.NewTable()
		tbl.RawSetString("type", lua.LString("has_item"))
		tbl.RawSetString("item", lua.LString(item))
		L.Push(tbl)
		return 1
	}))

	// FlagSet("flag")
	L.SetGlobal("FlagSet", L.NewFunction(func(L *lua.LState) int {
		flag := L.CheckString(1)
		tbl := L.NewTable()
		tbl.RawSetString("type", lua.LString("flag_set"))
		tbl.RawSetString("flag", lua.LString(flag))
		L.Push(tbl)
		return 1
	}))

	// FlagNot("flag")
	L.SetGlobal("FlagNot", L.NewFunction(func(L *lua.LState) int {
		flag := L.CheckString(1)
		tbl := L.NewTable()
		tbl.RawSetString("type", lua.LString("flag_not"))
		tbl.RawSetString("flag", lua.LString(flag))
		L.Push(tbl)
		return 1
	}))

	// FlagIs("flag", value)
	L.SetGlobal("FlagIs", L.NewFunction(func(L *lua.LState) int {
		flag := L.CheckString(1)
		value := L.CheckBool(2)
		tbl := L.NewTable()
		tbl.RawSetString("type", lua.LString("flag_is"))
		tbl.RawSetString("flag", lua.LString(flag))
		tbl.RawSetString("value", lua.LBool(value))
		L.Push(tbl)
		return 1
	}))

	// InRoom("room_id")
	L.SetGlobal("InRoom", L.NewFunction(func(L *lua.LState) int {
		room := L.CheckString(1)
		tbl := L.NewTable()
		tbl.RawSetString("type", lua.LString("in_room"))
		tbl.RawSetString("room", lua.LString(room))
		L.Push(tbl)
		return 1
	}))

	// PropIs("entity", "prop", value)
	L.SetGlobal("PropIs", L.NewFunction(func(L *lua.LState) int {
		entity := L.CheckString(1)
		prop := L.CheckString(2)
		value := L.Get(3)
		tbl := L.NewTable()
		tbl.RawSetString("type", lua.LString("prop_is"))
		tbl.RawSetString("entity", lua.LString(entity))
		tbl.RawSetString("prop", lua.LString(prop))
		tbl.RawSetString("value", value)
		L.Push(tbl)
		return 1
	}))

	// CounterGt("counter", value)
	L.SetGlobal("CounterGt", L.NewFunction(func(L *lua.LState) int {
		counter := L.CheckString(1)
		value := L.CheckNumber(2)
		tbl := L.NewTable()
		tbl.RawSetString("type", lua.LString("counter_gt"))
		tbl.RawSetString("counter", lua.LString(counter))
		tbl.RawSetString("value", value)
		L.Push(tbl)
		return 1
	}))

	// CounterLt("counter", value)
	L.SetGlobal("CounterLt", L.NewFunction(func(L *lua.LState) int {
		counter := L.CheckString(1)
		value := L.CheckNumber(2)
		tbl := L.NewTable()
		tbl.RawSetString("type", lua.LString("counter_lt"))
		tbl.RawSetString("counter", lua.LString(counter))
		tbl.RawSetString("value", value)
		L.Push(tbl)
		return 1
	}))

	// Not(condition)
	L.SetGlobal("Not", L.NewFunction(func(L *lua.LState) int {
		inner := L.CheckTable(1)
		tbl := L.NewTable()
		tbl.RawSetString("type", lua.LString("not"))
		tbl.RawSetString("inner", inner)
		L.Push(tbl)
		return 1
	}))
}

func registerEffectHelpers(L *lua.LState) {
	// Say("text")
	L.SetGlobal("Say", L.NewFunction(func(L *lua.LState) int {
		text := L.CheckString(1)
		tbl := L.NewTable()
		tbl.RawSetString("type", lua.LString("say"))
		tbl.RawSetString("text", lua.LString(text))
		L.Push(tbl)
		return 1
	}))

	// GiveItem("id")
	L.SetGlobal("GiveItem", L.NewFunction(func(L *lua.LState) int {
		item := L.CheckString(1)
		tbl := L.NewTable()
		tbl.RawSetString("type", lua.LString("give_item"))
		tbl.RawSetString("item", lua.LString(item))
		L.Push(tbl)
		return 1
	}))

	// RemoveItem("id")
	L.SetGlobal("RemoveItem", L.NewFunction(func(L *lua.LState) int {
		item := L.CheckString(1)
		tbl := L.NewTable()
		tbl.RawSetString("type", lua.LString("remove_item"))
		tbl.RawSetString("item", lua.LString(item))
		L.Push(tbl)
		return 1
	}))

	// SetFlag("flag", value)
	L.SetGlobal("SetFlag", L.NewFunction(func(L *lua.LState) int {
		flag := L.CheckString(1)
		value := L.CheckBool(2)
		tbl := L.NewTable()
		tbl.RawSetString("type", lua.LString("set_flag"))
		tbl.RawSetString("flag", lua.LString(flag))
		tbl.RawSetString("value", lua.LBool(value))
		L.Push(tbl)
		return 1
	}))

	// IncCounter("counter", amount)
	L.SetGlobal("IncCounter", L.NewFunction(func(L *lua.LState) int {
		counter := L.CheckString(1)
		amount := L.CheckNumber(2)
		tbl := L.NewTable()
		tbl.RawSetString("type", lua.LString("inc_counter"))
		tbl.RawSetString("counter", lua.LString(counter))
		tbl.RawSetString("amount", amount)
		L.Push(tbl)
		return 1
	}))

	// SetCounter("counter", value)
	L.SetGlobal("SetCounter", L.NewFunction(func(L *lua.LState) int {
		counter := L.CheckString(1)
		value := L.CheckNumber(2)
		tbl := L.NewTable()
		tbl.RawSetString("type", lua.LString("set_counter"))
		tbl.RawSetString("counter", lua.LString(counter))
		tbl.RawSetString("value", value)
		L.Push(tbl)
		return 1
	}))

	// SetProp("entity", "prop", value)
	L.SetGlobal("SetProp", L.NewFunction(func(L *lua.LState) int {
		entity := L.CheckString(1)
		prop := L.CheckString(2)
		value := L.Get(3)
		tbl := L.NewTable()
		tbl.RawSetString("type", lua.LString("set_prop"))
		tbl.RawSetString("entity", lua.LString(entity))
		tbl.RawSetString("prop", lua.LString(prop))
		tbl.RawSetString("value", value)
		L.Push(tbl)
		return 1
	}))

	// MoveEntity("entity", "room")
	L.SetGlobal("MoveEntity", L.NewFunction(func(L *lua.LState) int {
		entity := L.CheckString(1)
		room := L.CheckString(2)
		tbl := L.NewTable()
		tbl.RawSetString("type", lua.LString("move_entity"))
		tbl.RawSetString("entity", lua.LString(entity))
		tbl.RawSetString("room", lua.LString(room))
		L.Push(tbl)
		return 1
	}))

	// MovePlayer("room")
	L.SetGlobal("MovePlayer", L.NewFunction(func(L *lua.LState) int {
		room := L.CheckString(1)
		tbl := L.NewTable()
		tbl.RawSetString("type", lua.LString("move_player"))
		tbl.RawSetString("room", lua.LString(room))
		L.Push(tbl)
		return 1
	}))

	// OpenExit("room", "direction", "target")
	L.SetGlobal("OpenExit", L.NewFunction(func(L *lua.LState) int {
		room := L.CheckString(1)
		direction := L.CheckString(2)
		target := L.CheckString(3)
		tbl := L.NewTable()
		tbl.RawSetString("type", lua.LString("open_exit"))
		tbl.RawSetString("room", lua.LString(room))
		tbl.RawSetString("direction", lua.LString(direction))
		tbl.RawSetString("target", lua.LString(target))
		L.Push(tbl)
		return 1
	}))

	// CloseExit("room", "direction")
	L.SetGlobal("CloseExit", L.NewFunction(func(L *lua.LState) int {
		room := L.CheckString(1)
		direction := L.CheckString(2)
		tbl := L.NewTable()
		tbl.RawSetString("type", lua.LString("close_exit"))
		tbl.RawSetString("room", lua.LString(room))
		tbl.RawSetString("direction", lua.LString(direction))
		L.Push(tbl)
		return 1
	}))

	// EmitEvent("type")
	L.SetGlobal("EmitEvent", L.NewFunction(func(L *lua.LState) int {
		event := L.CheckString(1)
		tbl := L.NewTable()
		tbl.RawSetString("type", lua.LString("emit_event"))
		tbl.RawSetString("event", lua.LString(event))
		L.Push(tbl)
		return 1
	}))

	// StartDialogue("npc")
	L.SetGlobal("StartDialogue", L.NewFunction(func(L *lua.LState) int {
		npc := L.CheckString(1)
		tbl := L.NewTable()
		tbl.RawSetString("type", lua.LString("start_dialogue"))
		tbl.RawSetString("npc", lua.LString(npc))
		L.Push(tbl)
		return 1
	}))

	// Stop()
	L.SetGlobal("Stop", L.NewFunction(func(L *lua.LState) int {
		tbl := L.NewTable()
		tbl.RawSetString("type", lua.LString("stop"))
		L.Push(tbl)
		return 1
	}))
}
