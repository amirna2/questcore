package loader

import (
	"strings"
	"testing"

	"github.com/nathoo/questcore/types"
)

func TestLoad_MinimalGame(t *testing.T) {
	defs, err := Load("testdata/minimal")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if defs.Game.Title != "Minimal Test Game" {
		t.Errorf("Title = %q, want %q", defs.Game.Title, "Minimal Test Game")
	}
	if defs.Game.Start != "hall" {
		t.Errorf("Start = %q, want %q", defs.Game.Start, "hall")
	}
	if _, ok := defs.Rooms["hall"]; !ok {
		t.Error("room 'hall' not found")
	}
	if defs.Rooms["hall"].Description != "A grand hall." {
		t.Errorf("hall description = %q, want %q",
			defs.Rooms["hall"].Description, "A grand hall.")
	}
}

func TestLoad_FullGame(t *testing.T) {
	defs, err := Load("testdata/full")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Game metadata.
	if defs.Game.Title != "Full Test Game" {
		t.Errorf("Title = %q", defs.Game.Title)
	}
	if defs.Game.Author != "Tester" {
		t.Errorf("Author = %q", defs.Game.Author)
	}
	if defs.Game.Start != "entrance" {
		t.Errorf("Start = %q", defs.Game.Start)
	}

	// Rooms.
	if len(defs.Rooms) != 3 {
		t.Errorf("expected 3 rooms, got %d", len(defs.Rooms))
	}
	entrance := defs.Rooms["entrance"]
	if entrance.Exits["north"] != "throne_room" {
		t.Errorf("entrance north exit = %q", entrance.Exits["north"])
	}
	if entrance.Fallbacks["push"] != "Nothing here to push." {
		t.Errorf("entrance fallback = %q", entrance.Fallbacks["push"])
	}

	// Room-scoped rule.
	if len(entrance.Rules) != 1 {
		t.Errorf("expected 1 room rule on entrance, got %d", len(entrance.Rules))
	} else {
		if entrance.Rules[0].ID != "examine_painting" {
			t.Errorf("room rule ID = %q, want examine_painting", entrance.Rules[0].ID)
		}
		if entrance.Rules[0].Scope != "room:entrance" {
			t.Errorf("room rule scope = %q, want room:entrance", entrance.Rules[0].Scope)
		}
	}

	// Items.
	key, ok := defs.Entities["rusty_key"]
	if !ok {
		t.Fatal("entity 'rusty_key' not found")
	}
	if key.Kind != "item" {
		t.Errorf("rusty_key Kind = %q", key.Kind)
	}
	if key.Props["takeable"] != true {
		t.Errorf("rusty_key takeable = %v, want true", key.Props["takeable"])
	}

	gem, ok := defs.Entities["gem"]
	if !ok {
		t.Fatal("entity 'gem' not found")
	}
	if gem.Props["takeable"] != false {
		t.Errorf("gem takeable = %v, want false", gem.Props["takeable"])
	}

	// NPCs with topics.
	guard, ok := defs.Entities["guard"]
	if !ok {
		t.Fatal("entity 'guard' not found")
	}
	if guard.Kind != "npc" {
		t.Errorf("guard Kind = %q", guard.Kind)
	}
	if len(guard.Topics) != 2 {
		t.Errorf("guard topics count = %d, want 2", len(guard.Topics))
	}
	if guard.Topics["greet"].Text != "Hello, traveler." {
		t.Errorf("guard greet text = %q", guard.Topics["greet"].Text)
	}
	if len(guard.Topics["quest"].Requires) != 1 {
		t.Errorf("guard quest requires = %d conditions", len(guard.Topics["quest"].Requires))
	}

	// Generic entity.
	painting, ok := defs.Entities["painting"]
	if !ok {
		t.Fatal("entity 'painting' not found")
	}
	if painting.Kind != "entity" {
		t.Errorf("painting Kind = %q", painting.Kind)
	}

	// Global rules.
	if len(defs.GlobalRules) < 2 {
		t.Errorf("expected at least 2 global rules, got %d", len(defs.GlobalRules))
	}

	// Find the take_gem rule with conditions.
	var takeGem *struct{ found bool }
	for _, r := range defs.GlobalRules {
		if r.ID == "take_gem" {
			if len(r.Conditions) != 1 {
				t.Errorf("take_gem conditions = %d, want 1", len(r.Conditions))
			}
			if len(r.Effects) != 4 {
				t.Errorf("take_gem effects = %d, want 4", len(r.Effects))
			}
			takeGem = &struct{ found bool }{true}
		}
	}
	if takeGem == nil {
		t.Error("global rule 'take_gem' not found")
	}

	// Handlers.
	if len(defs.Handlers) != 1 {
		t.Errorf("expected 1 handler, got %d", len(defs.Handlers))
	} else if defs.Handlers[0].EventType != "door_unlocked" {
		t.Errorf("handler event = %q", defs.Handlers[0].EventType)
	}
}

func TestLoad_InvalidRefs_Fails(t *testing.T) {
	_, err := Load("testdata/invalid_refs")
	if err == nil {
		t.Fatal("expected error for invalid references")
	}
	if !strings.Contains(err.Error(), "undefined room") {
		t.Errorf("error = %q, expected 'undefined room'", err.Error())
	}
}

func TestLoad_DuplicateRuleIDs_Fails(t *testing.T) {
	_, err := Load("testdata/duplicate_rules")
	if err == nil {
		t.Fatal("expected error for duplicate rule IDs")
	}
	if !strings.Contains(err.Error(), "duplicate rule ID") {
		t.Errorf("error = %q, expected 'duplicate rule ID'", err.Error())
	}
}

func TestLoad_BadLuaSyntax_Fails(t *testing.T) {
	_, err := Load("testdata/bad_lua")
	if err == nil {
		t.Fatal("expected error for bad Lua syntax")
	}
}

func TestLoad_NoGameDef_Fails(t *testing.T) {
	_, err := Load("testdata/no_game")
	if err == nil {
		t.Fatal("expected error for missing Game{} definition")
	}
	if !strings.Contains(err.Error(), "no Game{} definition") {
		t.Errorf("error = %q, expected 'no Game{} definition'", err.Error())
	}
}

func TestLoad_SandboxEnforced(t *testing.T) {
	// os library should not be available.
	L, _ := newTestVM()
	defer L.Close()

	err := L.DoString(`os.execute("echo pwned")`)
	if err == nil {
		t.Fatal("expected sandbox to block os.execute")
	}
}

func TestLoad_ItemDefaultTakeable(t *testing.T) {
	defs, err := Load("testdata/full")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// rusty_key has no explicit takeable, should default to true.
	key := defs.Entities["rusty_key"]
	if key.Props["takeable"] != true {
		t.Errorf("rusty_key takeable = %v, want true", key.Props["takeable"])
	}

	// gem has explicit takeable = false.
	gem := defs.Entities["gem"]
	if gem.Props["takeable"] != false {
		t.Errorf("gem takeable = %v, want false", gem.Props["takeable"])
	}
}

func TestLoad_RuleScopeResolution(t *testing.T) {
	defs, err := Load("testdata/full")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// examine_painting should be scoped to room:entrance.
	entrance := defs.Rooms["entrance"]
	found := false
	for _, r := range entrance.Rules {
		if r.ID == "examine_painting" {
			found = true
			if r.Scope != "room:entrance" {
				t.Errorf("examine_painting scope = %q, want room:entrance", r.Scope)
			}
		}
	}
	if !found {
		t.Error("examine_painting not found in entrance room rules")
	}

	// Global rules should not contain examine_painting.
	for _, r := range defs.GlobalRules {
		if r.ID == "examine_painting" {
			t.Error("examine_painting should not be in global rules")
		}
	}
}

func TestLoad_CombatGame(t *testing.T) {
	defs, err := Load("testdata/combat")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Player stats.
	if defs.Game.PlayerStats == nil {
		t.Fatal("expected PlayerStats to be set")
	}
	if defs.Game.PlayerStats["hp"] != 20 {
		t.Errorf("player hp = %d, want 20", defs.Game.PlayerStats["hp"])
	}
	if defs.Game.PlayerStats["attack"] != 5 {
		t.Errorf("player attack = %d, want 5", defs.Game.PlayerStats["attack"])
	}

	// Enemy entity.
	goblin, ok := defs.Entities["cave_goblin"]
	if !ok {
		t.Fatal("cave_goblin entity not found")
	}
	if goblin.Kind != "enemy" {
		t.Errorf("cave_goblin kind = %q, want enemy", goblin.Kind)
	}
	if goblin.Props["hp"] != 12 {
		t.Errorf("goblin hp = %v, want 12", goblin.Props["hp"])
	}
	if goblin.Props["attack"] != 4 {
		t.Errorf("goblin attack = %v, want 4", goblin.Props["attack"])
	}
	if goblin.Props["alive"] != true {
		t.Errorf("goblin alive = %v, want true", goblin.Props["alive"])
	}
	if goblin.Props["name"] != "Cave Goblin" {
		t.Errorf("goblin name = %v, want Cave Goblin", goblin.Props["name"])
	}

	// Behavior.
	behavior, ok := goblin.Props["behavior"].([]types.BehaviorEntry)
	if !ok {
		t.Fatal("goblin behavior is not []BehaviorEntry")
	}
	if len(behavior) != 3 {
		t.Fatalf("expected 3 behavior entries, got %d", len(behavior))
	}
	if behavior[0].Action != "attack" || behavior[0].Weight != 70 {
		t.Errorf("behavior[0] = %+v, want {attack, 70}", behavior[0])
	}

	// Loot.
	lootItems, ok := goblin.Props["loot_items"].([]types.LootEntry)
	if !ok {
		t.Fatal("goblin loot_items is not []LootEntry")
	}
	if len(lootItems) != 1 {
		t.Fatalf("expected 1 loot item, got %d", len(lootItems))
	}
	if lootItems[0].ItemID != "goblin_blade" || lootItems[0].Chance != 50 {
		t.Errorf("loot[0] = %+v, want {goblin_blade, 50}", lootItems[0])
	}
	if goblin.Props["loot_gold"] != 5 {
		t.Errorf("loot_gold = %v, want 5", goblin.Props["loot_gold"])
	}

	// Attack rule exists.
	found := false
	for _, r := range defs.GlobalRules {
		if r.ID == "attack_goblin" {
			found = true
			if r.When.Verb != "attack" {
				t.Errorf("rule verb = %q, want attack", r.When.Verb)
			}
			// Should have start_combat effect.
			hasCombat := false
			for _, eff := range r.Effects {
				if eff.Type == "start_combat" {
					hasCombat = true
				}
			}
			if !hasCombat {
				t.Error("attack_goblin rule should have start_combat effect")
			}
		}
	}
	if !found {
		t.Error("attack_goblin rule not found in global rules")
	}
}

func TestLoad_FileOrdering(t *testing.T) {
	files := sortedLuaFiles([]string{"rooms.lua", "game.lua", "items.lua", "npcs.lua"})
	if files[0] != "game.lua" {
		t.Errorf("first file = %q, want game.lua", files[0])
	}
	// Rest should be alphabetical.
	if files[1] != "items.lua" {
		t.Errorf("second file = %q, want items.lua", files[1])
	}
}
