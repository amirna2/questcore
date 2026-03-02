package state

import (
	"sort"
	"testing"

	"github.com/nathoo/questcore/types"
)

func testDefs() *Defs {
	return &Defs{
		Game: types.GameDef{
			Title:   "Test Game",
			Author:  "Test",
			Version: "0.1.0",
			Start:   "entrance",
		},
		Rooms: map[string]types.RoomDef{
			"entrance": {
				ID:          "entrance",
				Description: "The entrance.",
				Exits:       map[string]string{"north": "hall"},
			},
			"hall": {
				ID:          "hall",
				Description: "A grand hall.",
				Exits:       map[string]string{"south": "entrance"},
			},
		},
		Entities: map[string]types.EntityDef{
			"rusty_key": {
				ID:   "rusty_key",
				Kind: "item",
				Props: map[string]any{
					"name":        "Rusty Key",
					"description": "An old iron key.",
					"location":    "hall",
					"takeable":    true,
				},
			},
			"golden_key": {
				ID:   "golden_key",
				Kind: "item",
				Props: map[string]any{
					"name":     "Golden Key",
					"location": "entrance",
				},
			},
			"guard": {
				ID:   "guard",
				Kind: "npc",
				Props: map[string]any{
					"name":     "Old Guard",
					"location": "entrance",
				},
			},
		},
	}
}

func TestNewState_StartsAtStartRoom(t *testing.T) {
	defs := testDefs()
	s := NewState(defs)

	if s.Player.Location != "entrance" {
		t.Errorf("expected player at entrance, got %q", s.Player.Location)
	}
}

func TestNewState_EmptyInventory(t *testing.T) {
	defs := testDefs()
	s := NewState(defs)

	if len(s.Player.Inventory) != 0 {
		t.Errorf("expected empty inventory, got %v", s.Player.Inventory)
	}
}

func TestNewState_ZeroTurnCount(t *testing.T) {
	defs := testDefs()
	s := NewState(defs)

	if s.TurnCount != 0 {
		t.Errorf("expected turn 0, got %d", s.TurnCount)
	}
}

func TestGetFlag_UnsetReturnsFalse(t *testing.T) {
	s := &types.State{Flags: map[string]bool{}}

	if GetFlag(s, "nonexistent") {
		t.Error("expected unset flag to be false")
	}
}

func TestGetFlag_SetReturnsValue(t *testing.T) {
	s := &types.State{Flags: map[string]bool{"door_open": true}}

	if !GetFlag(s, "door_open") {
		t.Error("expected door_open to be true")
	}
}

func TestGetCounter_UnsetReturnsZero(t *testing.T) {
	s := &types.State{Counters: map[string]int{}}

	if got := GetCounter(s, "score"); got != 0 {
		t.Errorf("expected 0, got %d", got)
	}
}

func TestGetCounter_SetReturnsValue(t *testing.T) {
	s := &types.State{Counters: map[string]int{"score": 42}}

	if got := GetCounter(s, "score"); got != 42 {
		t.Errorf("expected 42, got %d", got)
	}
}

func TestHasItem_EmptyInventory(t *testing.T) {
	s := &types.State{Player: types.Player{Inventory: []string{}}}

	if HasItem(s, "key") {
		t.Error("expected empty inventory to not have key")
	}
}

func TestHasItem_ItemPresent(t *testing.T) {
	s := &types.State{Player: types.Player{Inventory: []string{"sword", "key"}}}

	if !HasItem(s, "key") {
		t.Error("expected inventory to contain key")
	}
}

func TestHasItem_ItemAbsent(t *testing.T) {
	s := &types.State{Player: types.Player{Inventory: []string{"sword"}}}

	if HasItem(s, "key") {
		t.Error("expected inventory to not contain key")
	}
}

func TestGetEntityProp_BaseDefinition(t *testing.T) {
	defs := testDefs()
	s := NewState(defs)

	val, ok := GetEntityProp(s, defs, "rusty_key", "name")
	if !ok {
		t.Fatal("expected to find name property")
	}
	if val != "Rusty Key" {
		t.Errorf("expected 'Rusty Key', got %v", val)
	}
}

func TestGetEntityProp_RuntimeOverride(t *testing.T) {
	defs := testDefs()
	s := NewState(defs)
	s.Entities["rusty_key"] = types.EntityState{
		Props: map[string]any{"name": "Shiny Key"},
	}

	val, ok := GetEntityProp(s, defs, "rusty_key", "name")
	if !ok {
		t.Fatal("expected to find name property")
	}
	if val != "Shiny Key" {
		t.Errorf("expected 'Shiny Key', got %v", val)
	}
}

func TestGetEntityProp_OverrideFallsBackToBase(t *testing.T) {
	defs := testDefs()
	s := NewState(defs)
	// Override a different property; "name" should still come from base.
	s.Entities["rusty_key"] = types.EntityState{
		Props: map[string]any{"shiny": true},
	}

	val, ok := GetEntityProp(s, defs, "rusty_key", "name")
	if !ok {
		t.Fatal("expected to find name property from base")
	}
	if val != "Rusty Key" {
		t.Errorf("expected 'Rusty Key', got %v", val)
	}
}

func TestGetEntityProp_UnknownEntity(t *testing.T) {
	defs := testDefs()
	s := NewState(defs)

	_, ok := GetEntityProp(s, defs, "nonexistent", "name")
	if ok {
		t.Error("expected unknown entity to return not found")
	}
}

func TestGetEntityProp_UnknownProp(t *testing.T) {
	defs := testDefs()
	s := NewState(defs)

	_, ok := GetEntityProp(s, defs, "rusty_key", "nonexistent")
	if ok {
		t.Error("expected unknown prop to return not found")
	}
}

func TestEntityLocation_BaseDefinition(t *testing.T) {
	defs := testDefs()
	s := NewState(defs)

	loc := EntityLocation(s, defs, "rusty_key")
	if loc != "hall" {
		t.Errorf("expected 'hall', got %q", loc)
	}
}

func TestEntityLocation_RuntimeOverride(t *testing.T) {
	defs := testDefs()
	s := NewState(defs)
	s.Entities["rusty_key"] = types.EntityState{Location: "entrance"}

	loc := EntityLocation(s, defs, "rusty_key")
	if loc != "entrance" {
		t.Errorf("expected 'entrance', got %q", loc)
	}
}

func TestEntityLocation_UnknownEntity(t *testing.T) {
	defs := testDefs()
	s := NewState(defs)

	loc := EntityLocation(s, defs, "nonexistent")
	if loc != "" {
		t.Errorf("expected empty string, got %q", loc)
	}
}

func TestEntitiesInRoom_FindsAll(t *testing.T) {
	defs := testDefs()
	s := NewState(defs)

	ids := EntitiesInRoom(s, defs, "entrance")
	sort.Strings(ids)
	expected := []string{"golden_key", "guard"}
	if len(ids) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, ids)
	}
	for i, id := range ids {
		if id != expected[i] {
			t.Errorf("expected %q at index %d, got %q", expected[i], i, id)
		}
	}
}

func TestEntitiesInRoom_ReflectsOverrides(t *testing.T) {
	defs := testDefs()
	s := NewState(defs)
	// Move guard to hall at runtime.
	s.Entities["guard"] = types.EntityState{Location: "hall"}

	ids := EntitiesInRoom(s, defs, "entrance")
	if len(ids) != 1 || ids[0] != "golden_key" {
		t.Errorf("expected [golden_key], got %v", ids)
	}

	hallIDs := EntitiesInRoom(s, defs, "hall")
	sort.Strings(hallIDs)
	expected := []string{"guard", "rusty_key"}
	if len(hallIDs) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, hallIDs)
	}
	for i, id := range hallIDs {
		if id != expected[i] {
			t.Errorf("expected %q at index %d, got %q", expected[i], i, id)
		}
	}
}

func TestEntitiesInRoom_EmptyRoom(t *testing.T) {
	defs := testDefs()
	s := NewState(defs)

	ids := EntitiesInRoom(s, defs, "nonexistent_room")
	if len(ids) != 0 {
		t.Errorf("expected no entities, got %v", ids)
	}
}

func TestRoomExits_BaseExits(t *testing.T) {
	defs := testDefs()
	s := NewState(defs)

	exits := RoomExits(s, defs, "entrance")
	if exits["north"] != "hall" {
		t.Errorf("expected north → hall, got %v", exits)
	}
}

func TestRoomExits_OpenExit(t *testing.T) {
	defs := testDefs()
	s := NewState(defs)
	s.Entities["room:hall"] = types.EntityState{
		Props: map[string]any{"exit:west": "treasury"},
	}

	exits := RoomExits(s, defs, "hall")
	if exits["west"] != "treasury" {
		t.Errorf("expected west → treasury, got %v", exits)
	}
	if exits["south"] != "entrance" {
		t.Errorf("expected south → entrance to still exist, got %v", exits)
	}
}

func TestRoomExits_CloseExit(t *testing.T) {
	defs := testDefs()
	s := NewState(defs)
	s.Entities["room:hall"] = types.EntityState{
		Props: map[string]any{"exit:south": ""},
	}

	exits := RoomExits(s, defs, "hall")
	if _, ok := exits["south"]; ok {
		t.Error("expected south exit to be closed")
	}
}

func TestRoomExits_UnknownRoom(t *testing.T) {
	defs := testDefs()
	s := NewState(defs)

	exits := RoomExits(s, defs, "nonexistent")
	if exits != nil {
		t.Errorf("expected nil, got %v", exits)
	}
}

// --- Combat state helpers ---

func TestInCombat_Active(t *testing.T) {
	s := &types.State{}
	s.Combat.Active = true
	if !InCombat(s) {
		t.Error("expected InCombat to return true")
	}
}

func TestInCombat_Inactive(t *testing.T) {
	s := &types.State{}
	if InCombat(s) {
		t.Error("expected InCombat to return false")
	}
}

func TestGetStat_Player(t *testing.T) {
	s := &types.State{
		Player: types.Player{Stats: map[string]int{"hp": 15, "attack": 5}},
	}
	defs := testDefs()

	v, ok := GetStat(s, defs, "player", "hp")
	if !ok || v != 15 {
		t.Errorf("expected hp=15, got %d (ok=%v)", v, ok)
	}

	v, ok = GetStat(s, defs, "player", "attack")
	if !ok || v != 5 {
		t.Errorf("expected attack=5, got %d (ok=%v)", v, ok)
	}
}

func TestGetStat_Player_Missing(t *testing.T) {
	s := &types.State{
		Player: types.Player{Stats: map[string]int{}},
	}
	defs := testDefs()

	_, ok := GetStat(s, defs, "player", "nonexistent")
	if ok {
		t.Error("expected missing stat to return false")
	}
}

func TestGetStat_Entity_FromDef(t *testing.T) {
	defs := &Defs{
		Entities: map[string]types.EntityDef{
			"goblin": {
				ID:   "goblin",
				Kind: "enemy",
				Props: map[string]any{
					"hp": 12, "attack": 4,
				},
			},
		},
	}
	s := &types.State{Entities: map[string]types.EntityState{}}

	v, ok := GetStat(s, defs, "goblin", "hp")
	if !ok || v != 12 {
		t.Errorf("expected hp=12, got %d (ok=%v)", v, ok)
	}
}

func TestGetStat_Entity_RuntimeOverride(t *testing.T) {
	defs := &Defs{
		Entities: map[string]types.EntityDef{
			"goblin": {
				ID:    "goblin",
				Kind:  "enemy",
				Props: map[string]any{"hp": 12},
			},
		},
	}
	s := &types.State{
		Entities: map[string]types.EntityState{
			"goblin": {Props: map[string]any{"hp": 5}},
		},
	}

	v, ok := GetStat(s, defs, "goblin", "hp")
	if !ok || v != 5 {
		t.Errorf("expected hp=5 (override), got %d (ok=%v)", v, ok)
	}
}

func TestGetStat_Entity_Missing(t *testing.T) {
	defs := testDefs()
	s := NewState(defs)

	_, ok := GetStat(s, defs, "rusty_key", "hp")
	if ok {
		t.Error("expected missing stat to return false")
	}
}

func TestGetStat_Entity_Float64Coercion(t *testing.T) {
	defs := &Defs{
		Entities: map[string]types.EntityDef{
			"goblin": {
				ID:    "goblin",
				Kind:  "enemy",
				Props: map[string]any{"hp": float64(8)},
			},
		},
	}
	s := &types.State{Entities: map[string]types.EntityState{}}

	v, ok := GetStat(s, defs, "goblin", "hp")
	if !ok || v != 8 {
		t.Errorf("expected hp=8, got %d (ok=%v)", v, ok)
	}
}

func TestSetStat_Player(t *testing.T) {
	s := &types.State{
		Player:   types.Player{Stats: map[string]int{"hp": 20}},
		Entities: map[string]types.EntityState{},
	}

	SetStat(s, "player", "hp", 15)
	if s.Player.Stats["hp"] != 15 {
		t.Errorf("expected hp=15, got %d", s.Player.Stats["hp"])
	}
}

func TestSetStat_Player_NilStats(t *testing.T) {
	s := &types.State{
		Player:   types.Player{},
		Entities: map[string]types.EntityState{},
	}

	SetStat(s, "player", "hp", 10)
	if s.Player.Stats["hp"] != 10 {
		t.Errorf("expected hp=10, got %d", s.Player.Stats["hp"])
	}
}

func TestSetStat_Entity(t *testing.T) {
	s := &types.State{
		Entities: map[string]types.EntityState{},
	}

	SetStat(s, "goblin", "hp", 5)
	es := s.Entities["goblin"]
	if es.Props["hp"] != 5 {
		t.Errorf("expected hp=5, got %v", es.Props["hp"])
	}
}

func TestSetStat_Entity_ExistingProps(t *testing.T) {
	s := &types.State{
		Entities: map[string]types.EntityState{
			"goblin": {Props: map[string]any{"alive": true}},
		},
	}

	SetStat(s, "goblin", "hp", 3)
	es := s.Entities["goblin"]
	if es.Props["hp"] != 3 {
		t.Errorf("expected hp=3, got %v", es.Props["hp"])
	}
	if es.Props["alive"] != true {
		t.Error("expected alive to still be true")
	}
}

func TestNewState_CopiesPlayerStats(t *testing.T) {
	defs := &Defs{
		Game: types.GameDef{
			Start:       "room1",
			PlayerStats: map[string]int{"hp": 20, "max_hp": 20, "attack": 5, "defense": 2},
		},
		Rooms:    map[string]types.RoomDef{"room1": {ID: "room1"}},
		Entities: map[string]types.EntityDef{},
	}

	s := NewState(defs)
	if s.Player.Stats["hp"] != 20 {
		t.Errorf("expected hp=20, got %d", s.Player.Stats["hp"])
	}
	if s.Player.Stats["attack"] != 5 {
		t.Errorf("expected attack=5, got %d", s.Player.Stats["attack"])
	}

	// Mutating state should not affect defs.
	s.Player.Stats["hp"] = 10
	if defs.Game.PlayerStats["hp"] != 20 {
		t.Error("mutating state should not affect defs")
	}
}

func TestNewState_NoPlayerStats(t *testing.T) {
	defs := testDefs() // no PlayerStats defined

	s := NewState(defs)
	if s.Player.Stats == nil {
		t.Fatal("expected Stats map to be initialized (not nil)")
	}
	if len(s.Player.Stats) != 0 {
		t.Errorf("expected empty Stats, got %v", s.Player.Stats)
	}
}
