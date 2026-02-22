package loader

import (
	"testing"

	lua "github.com/yuin/gopher-lua"
)

// newTestVM creates a sandboxed Lua VM with the API registered and a fresh collector.
func newTestVM() (*lua.LState, *collector) {
	L := lua.NewState(lua.Options{SkipOpenLibs: true})
	openSafeLibs(L)
	sandbox(L)
	coll := &collector{}
	registerAPI(L, coll)
	return L, coll
}

func TestCompileGame(t *testing.T) {
	L, _ := newTestVM()
	defer L.Close()

	if err := L.DoString(`
		return {
			title = "Test Game",
			author = "Author",
			version = "1.0",
			start = "hall",
			intro = "Welcome!"
		}
	`); err != nil {
		t.Fatal(err)
	}

	tbl := L.CheckTable(-1)
	game := compileGame(tbl)

	if game.Title != "Test Game" {
		t.Errorf("Title = %q, want %q", game.Title, "Test Game")
	}
	if game.Author != "Author" {
		t.Errorf("Author = %q, want %q", game.Author, "Author")
	}
	if game.Version != "1.0" {
		t.Errorf("Version = %q, want %q", game.Version, "1.0")
	}
	if game.Start != "hall" {
		t.Errorf("Start = %q, want %q", game.Start, "hall")
	}
	if game.Intro != "Welcome!" {
		t.Errorf("Intro = %q, want %q", game.Intro, "Welcome!")
	}
}

func TestCompileRoom_WithExitsAndFallbacks(t *testing.T) {
	L, coll := newTestVM()
	defer L.Close()

	if err := L.DoString(`
		local r = Rule("room_rule",
			When { verb = "look" },
			Then { Say("You see a room.") }
		)
		Room "hall" {
			description = "A grand hall.",
			exits = { north = "garden", south = "cellar" },
			fallbacks = { push = "Nothing to push." },
			rules = { r }
		}
	`); err != nil {
		t.Fatal(err)
	}

	if len(coll.rooms) != 1 {
		t.Fatalf("expected 1 room, got %d", len(coll.rooms))
	}

	room, scopedIDs, err := compileRoom(coll.rooms[0])
	if err != nil {
		t.Fatal(err)
	}

	if room.ID != "hall" {
		t.Errorf("ID = %q, want %q", room.ID, "hall")
	}
	if room.Description != "A grand hall." {
		t.Errorf("Description = %q, want %q", room.Description, "A grand hall.")
	}
	if room.Exits["north"] != "garden" {
		t.Errorf("Exits[north] = %q, want %q", room.Exits["north"], "garden")
	}
	if room.Exits["south"] != "cellar" {
		t.Errorf("Exits[south] = %q, want %q", room.Exits["south"], "cellar")
	}
	if room.Fallbacks["push"] != "Nothing to push." {
		t.Errorf("Fallbacks[push] = %q, want %q", room.Fallbacks["push"], "Nothing to push.")
	}
	if len(scopedIDs) != 1 || scopedIDs[0] != "room_rule" {
		t.Errorf("scopedIDs = %v, want [room_rule]", scopedIDs)
	}
}

func TestCompileEntity_ItemDefaultTakeable(t *testing.T) {
	L, coll := newTestVM()
	defer L.Close()

	if err := L.DoString(`
		Item "key" {
			name = "rusty key",
			description = "An old key.",
			location = "hall"
		}
	`); err != nil {
		t.Fatal(err)
	}

	entity, _, err := compileEntity(coll.entities[0])
	if err != nil {
		t.Fatal(err)
	}

	if entity.Kind != "item" {
		t.Errorf("Kind = %q, want %q", entity.Kind, "item")
	}
	if entity.Props["takeable"] != true {
		t.Errorf("Props[takeable] = %v, want true", entity.Props["takeable"])
	}
	if entity.Props["name"] != "rusty key" {
		t.Errorf("Props[name] = %v, want %q", entity.Props["name"], "rusty key")
	}
}

func TestCompileEntity_ItemExplicitTakeableFalse(t *testing.T) {
	L, coll := newTestVM()
	defer L.Close()

	if err := L.DoString(`
		Item "statue" {
			name = "statue",
			takeable = false
		}
	`); err != nil {
		t.Fatal(err)
	}

	entity, _, err := compileEntity(coll.entities[0])
	if err != nil {
		t.Fatal(err)
	}

	if entity.Props["takeable"] != false {
		t.Errorf("Props[takeable] = %v, want false", entity.Props["takeable"])
	}
}

func TestCompileEntity_NPCWithTopics(t *testing.T) {
	L, coll := newTestVM()
	defer L.Close()

	if err := L.DoString(`
		NPC "guard" {
			name = "guard",
			location = "hall",
			topics = {
				greet = {
					text = "Hello!",
					effects = { SetFlag("met_guard", true) }
				},
				quest = {
					text = "Find the gem.",
					requires = { FlagSet("met_guard") },
					effects = { SetFlag("quest_given", true) }
				}
			}
		}
	`); err != nil {
		t.Fatal(err)
	}

	entity, _, err := compileEntity(coll.entities[0])
	if err != nil {
		t.Fatal(err)
	}

	if entity.Kind != "npc" {
		t.Errorf("Kind = %q, want %q", entity.Kind, "npc")
	}
	if len(entity.Topics) != 2 {
		t.Fatalf("expected 2 topics, got %d", len(entity.Topics))
	}
	if entity.Topics["greet"].Text != "Hello!" {
		t.Errorf("greet.Text = %q, want %q", entity.Topics["greet"].Text, "Hello!")
	}
	if entity.Topics["quest"].Text != "Find the gem." {
		t.Errorf("quest.Text = %q, want %q", entity.Topics["quest"].Text, "Find the gem.")
	}
	if len(entity.Topics["quest"].Requires) != 1 {
		t.Fatalf("quest.Requires length = %d, want 1", len(entity.Topics["quest"].Requires))
	}
	if entity.Topics["quest"].Requires[0].Type != "flag_set" {
		t.Errorf("quest.Requires[0].Type = %q, want %q",
			entity.Topics["quest"].Requires[0].Type, "flag_set")
	}
}

func TestCompileEntity_GenericEntity(t *testing.T) {
	L, coll := newTestVM()
	defer L.Close()

	if err := L.DoString(`
		Entity "lever" {
			name = "lever",
			description = "A rusty lever.",
			location = "hall",
			pulled = false
		}
	`); err != nil {
		t.Fatal(err)
	}

	entity, _, err := compileEntity(coll.entities[0])
	if err != nil {
		t.Fatal(err)
	}

	if entity.Kind != "entity" {
		t.Errorf("Kind = %q, want %q", entity.Kind, "entity")
	}
	if entity.Props["pulled"] != false {
		t.Errorf("Props[pulled] = %v, want false", entity.Props["pulled"])
	}
	// Generic entities should NOT get default takeable.
	if _, ok := entity.Props["takeable"]; ok {
		t.Error("generic entity should not have default takeable")
	}
}

func TestCompileConditions_AllTypes(t *testing.T) {
	L, _ := newTestVM()
	defer L.Close()

	tests := []struct {
		lua      string
		wantType string
		checkKey string
		wantVal  any
	}{
		{`HasItem("key")`, "has_item", "item", "key"},
		{`FlagSet("door_open")`, "flag_set", "flag", "door_open"},
		{`FlagNot("dead")`, "flag_not", "flag", "dead"},
		{`FlagIs("verbose", true)`, "flag_is", "flag", "verbose"},
		{`InRoom("hall")`, "in_room", "room", "hall"},
		{`PropIs("door", "locked", true)`, "prop_is", "entity", "door"},
		{`CounterGt("turns", 5)`, "counter_gt", "counter", "turns"},
		{`CounterLt("health", 3)`, "counter_lt", "counter", "health"},
		{`Not(FlagSet("done"))`, "not", "", nil},
	}

	for _, tt := range tests {
		t.Run(tt.wantType, func(t *testing.T) {
			if err := L.DoString("return " + tt.lua); err != nil {
				t.Fatal(err)
			}
			tbl := L.CheckTable(-1)
			L.Pop(1)

			cond := compileCondition(tbl)
			if cond.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", cond.Type, tt.wantType)
			}
			if tt.wantType == "not" {
				if cond.Inner == nil {
					t.Error("Not condition: Inner is nil")
				} else if cond.Inner.Type != "flag_set" {
					t.Errorf("Not inner Type = %q, want flag_set", cond.Inner.Type)
				}
				if !cond.Negate {
					t.Error("Not condition: Negate should be true")
				}
			} else if tt.checkKey != "" {
				got, ok := cond.Params[tt.checkKey]
				if !ok {
					t.Errorf("missing param %q", tt.checkKey)
				} else if got != tt.wantVal {
					t.Errorf("Params[%q] = %v, want %v", tt.checkKey, got, tt.wantVal)
				}
			}
		})
	}
}

func TestCompileEffects_AllTypes(t *testing.T) {
	L, _ := newTestVM()
	defer L.Close()

	tests := []struct {
		lua      string
		wantType string
		checkKey string
		wantVal  any
	}{
		{`Say("hello")`, "say", "text", "hello"},
		{`GiveItem("key")`, "give_item", "item", "key"},
		{`RemoveItem("key")`, "remove_item", "item", "key"},
		{`SetFlag("done", true)`, "set_flag", "flag", "done"},
		{`IncCounter("score", 10)`, "inc_counter", "counter", "score"},
		{`SetCounter("lives", 3)`, "set_counter", "counter", "lives"},
		{`SetProp("door", "locked", false)`, "set_prop", "entity", "door"},
		{`MoveEntity("guard", "hall")`, "move_entity", "entity", "guard"},
		{`MovePlayer("garden")`, "move_player", "room", "garden"},
		{`OpenExit("hall", "north", "garden")`, "open_exit", "room", "hall"},
		{`CloseExit("hall", "north")`, "close_exit", "room", "hall"},
		{`EmitEvent("explosion")`, "emit_event", "event", "explosion"},
		{`StartDialogue("guard")`, "start_dialogue", "npc", "guard"},
		{`Stop()`, "stop", "", nil},
	}

	for _, tt := range tests {
		t.Run(tt.wantType, func(t *testing.T) {
			if err := L.DoString("return " + tt.lua); err != nil {
				t.Fatal(err)
			}
			tbl := L.CheckTable(-1)
			L.Pop(1)

			eff := compileEffect(tbl)
			if eff.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", eff.Type, tt.wantType)
			}
			if tt.checkKey != "" {
				got, ok := eff.Params[tt.checkKey]
				if !ok {
					t.Errorf("missing param %q", tt.checkKey)
				} else if got != tt.wantVal {
					t.Errorf("Params[%q] = %v (%T), want %v (%T)",
						tt.checkKey, got, got, tt.wantVal, tt.wantVal)
				}
			}
		})
	}
}

func TestCompileMatchCriteria_VerbOnly(t *testing.T) {
	L, _ := newTestVM()
	defer L.Close()

	if err := L.DoString(`return { verb = "look" }`); err != nil {
		t.Fatal(err)
	}
	tbl := L.CheckTable(-1)

	mc := compileMatchCriteria(tbl)
	if mc.Verb != "look" {
		t.Errorf("Verb = %q, want %q", mc.Verb, "look")
	}
	if mc.Object != "" {
		t.Errorf("Object = %q, want empty", mc.Object)
	}
}

func TestCompileMatchCriteria_Full(t *testing.T) {
	L, _ := newTestVM()
	defer L.Close()

	if err := L.DoString(`
		return {
			verb = "use",
			object = "key",
			target = "door",
			object_kind = "item",
			object_prop = { shiny = true },
			target_prop = { locked = true }
		}
	`); err != nil {
		t.Fatal(err)
	}
	tbl := L.CheckTable(-1)

	mc := compileMatchCriteria(tbl)
	if mc.Verb != "use" {
		t.Errorf("Verb = %q, want %q", mc.Verb, "use")
	}
	if mc.Object != "key" {
		t.Errorf("Object = %q, want %q", mc.Object, "key")
	}
	if mc.Target != "door" {
		t.Errorf("Target = %q, want %q", mc.Target, "door")
	}
	if mc.ObjectKind != "item" {
		t.Errorf("ObjectKind = %q, want %q", mc.ObjectKind, "item")
	}
	if mc.ObjectProp["shiny"] != true {
		t.Errorf("ObjectProp[shiny] = %v, want true", mc.ObjectProp["shiny"])
	}
	if mc.TargetProp["locked"] != true {
		t.Errorf("TargetProp[locked] = %v, want true", mc.TargetProp["locked"])
	}
}

func TestCompileRule_ScopeResolution(t *testing.T) {
	L, coll := newTestVM()
	defer L.Close()

	if err := L.DoString(`
		local r = Rule("scoped_rule",
			When { verb = "look" },
			Then { Say("You look around.") }
		)
		Room "hall" {
			description = "A hall.",
			rules = { r }
		}
	`); err != nil {
		t.Fatal(err)
	}

	// Before markScopedRules, rule scope should be "global".
	if coll.rules[0].scope != "global" {
		t.Errorf("initial scope = %q, want %q", coll.rules[0].scope, "global")
	}

	// Compile the room, which returns scoped IDs.
	_, scopedIDs, err := compileRoom(coll.rooms[0])
	if err != nil {
		t.Fatal(err)
	}

	// Mark scoped rules.
	markScopedRules(coll, scopedIDs, "room:hall")

	if coll.rules[0].scope != "room:hall" {
		t.Errorf("after marking, scope = %q, want %q", coll.rules[0].scope, "room:hall")
	}

	// Compile the rule and verify scope.
	rule, err := compileRule(coll.rules[0])
	if err != nil {
		t.Fatal(err)
	}
	if rule.Scope != "room:hall" {
		t.Errorf("compiled scope = %q, want %q", rule.Scope, "room:hall")
	}
}

func TestCompileRule_WithConditions(t *testing.T) {
	L, coll := newTestVM()
	defer L.Close()

	if err := L.DoString(`
		Rule("guarded_rule",
			When { verb = "take", object = "gem" },
			{ HasItem("key"), FlagSet("door_open") },
			Then { Say("You take the gem."), GiveItem("gem") }
		)
	`); err != nil {
		t.Fatal(err)
	}

	if len(coll.rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(coll.rules))
	}

	rule, err := compileRule(coll.rules[0])
	if err != nil {
		t.Fatal(err)
	}

	if len(rule.Conditions) != 2 {
		t.Fatalf("expected 2 conditions, got %d", len(rule.Conditions))
	}
	if rule.Conditions[0].Type != "has_item" {
		t.Errorf("cond[0].Type = %q, want %q", rule.Conditions[0].Type, "has_item")
	}
	if rule.Conditions[1].Type != "flag_set" {
		t.Errorf("cond[1].Type = %q, want %q", rule.Conditions[1].Type, "flag_set")
	}
	if len(rule.Effects) != 2 {
		t.Fatalf("expected 2 effects, got %d", len(rule.Effects))
	}
}

func TestCompileRule_WithoutConditions(t *testing.T) {
	L, coll := newTestVM()
	defer L.Close()

	if err := L.DoString(`
		Rule("simple",
			When { verb = "look" },
			Then { Say("You see nothing special.") }
		)
	`); err != nil {
		t.Fatal(err)
	}

	rule, err := compileRule(coll.rules[0])
	if err != nil {
		t.Fatal(err)
	}

	if len(rule.Conditions) != 0 {
		t.Errorf("expected 0 conditions, got %d", len(rule.Conditions))
	}
	if len(rule.Effects) != 1 {
		t.Fatalf("expected 1 effect, got %d", len(rule.Effects))
	}
	if rule.Effects[0].Type != "say" {
		t.Errorf("effect type = %q, want %q", rule.Effects[0].Type, "say")
	}
}

func TestCompileHandler(t *testing.T) {
	L, coll := newTestVM()
	defer L.Close()

	if err := L.DoString(`
		On("door_opened", {
			conditions = { InRoom("hall") },
			effects = { Say("The door creaks open.") }
		})
	`); err != nil {
		t.Fatal(err)
	}

	if len(coll.handlers) != 1 {
		t.Fatalf("expected 1 handler, got %d", len(coll.handlers))
	}

	handler, err := compileHandler(coll.handlers[0])
	if err != nil {
		t.Fatal(err)
	}

	if handler.EventType != "door_opened" {
		t.Errorf("EventType = %q, want %q", handler.EventType, "door_opened")
	}
	if len(handler.Conditions) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(handler.Conditions))
	}
	if handler.Conditions[0].Type != "in_room" {
		t.Errorf("condition type = %q, want %q", handler.Conditions[0].Type, "in_room")
	}
	if len(handler.Effects) != 1 {
		t.Fatalf("expected 1 effect, got %d", len(handler.Effects))
	}
}

func TestSourceOrder_AutoIncrement(t *testing.T) {
	L, coll := newTestVM()
	defer L.Close()

	if err := L.DoString(`
		Rule("first", When { verb = "look" }, Then { Say("1") })
		Rule("second", When { verb = "take" }, Then { Say("2") })
		Rule("third", When { verb = "drop" }, Then { Say("3") })
	`); err != nil {
		t.Fatal(err)
	}

	if len(coll.rules) != 3 {
		t.Fatalf("expected 3 rules, got %d", len(coll.rules))
	}

	for i, raw := range coll.rules {
		if raw.order != i+1 {
			t.Errorf("rule %d order = %d, want %d", i, raw.order, i+1)
		}
	}
}
