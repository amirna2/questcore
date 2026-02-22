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
