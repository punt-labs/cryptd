package engine_test

import (
	"testing"
	"time"

	"github.com/punt-labs/cryptd/internal/engine"
	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/scenario"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadMinimal(t *testing.T) *scenario.Scenario {
	t.Helper()
	s, err := scenario.Load("../../testdata/scenarios/minimal.yaml")
	require.NoError(t, err)
	return s
}

func newGame(t *testing.T) (*engine.Engine, model.GameState) {
	t.Helper()
	s := loadMinimal(t)
	char := model.Character{
		ID: "c1", Name: "Test", Class: "fighter", Level: 1,
		HP: 10, MaxHP: 10,
	}
	e := engine.New(s)
	state, err := e.NewGame(char)
	require.NoError(t, err)
	return e, state
}

func TestNewGame_SetsStartingRoom(t *testing.T) {
	_, state := newGame(t)
	assert.Equal(t, "entrance", state.Dungeon.CurrentRoom)
	assert.Contains(t, state.Dungeon.VisitedRooms, "entrance")
	require.Len(t, state.Party, 1)
	assert.Equal(t, "Test", state.Party[0].Name)
}

func TestMove_OpenDoor(t *testing.T) {
	e, state := newGame(t)
	result, err := e.Move(&state, "south")
	require.NoError(t, err)
	assert.Equal(t, "goblin_lair", result.NewRoom)
	assert.Contains(t, result.Exits, "north")
	assert.Equal(t, "goblin_lair", state.Dungeon.CurrentRoom)
	assert.Contains(t, state.Dungeon.VisitedRooms, "goblin_lair")
}

func TestMove_UpdatesFogOfWar(t *testing.T) {
	e, state := newGame(t)
	_, err := e.Move(&state, "south")
	require.NoError(t, err)
	assert.Contains(t, state.Dungeon.VisitedRooms, "entrance")
	assert.Contains(t, state.Dungeon.VisitedRooms, "goblin_lair")
}

func TestMove_UnknownDirection(t *testing.T) {
	e, state := newGame(t)
	_, err := e.Move(&state, "east")
	require.Error(t, err)
	var noExit *engine.NoExitError
	require.ErrorAs(t, err, &noExit)
	assert.Equal(t, "east", noExit.Direction)
}

func TestMove_LockedDoor(t *testing.T) {
	e, state := newGame(t)
	_, err := e.Move(&state, "west")
	require.Error(t, err)
	var locked *engine.LockedError
	require.ErrorAs(t, err, &locked)
}

func TestMove_AppendsLogEntry(t *testing.T) {
	e, state := newGame(t)
	_, err := e.Move(&state, "south")
	require.NoError(t, err)
	require.NotEmpty(t, state.AdventureLog)
	assert.Contains(t, state.AdventureLog[len(state.AdventureLog)-1].Text, "goblin_lair")
}

func TestMove_VisitedRoomsDeduped(t *testing.T) {
	e, state := newGame(t)
	_, err := e.Move(&state, "south")
	require.NoError(t, err)
	_, err = e.Move(&state, "north")
	require.NoError(t, err)
	_, err = e.Move(&state, "south")
	require.NoError(t, err)
	// entrance and goblin_lair should appear exactly once each.
	count := 0
	for _, r := range state.Dungeon.VisitedRooms {
		if r == "goblin_lair" {
			count++
		}
	}
	assert.Equal(t, 1, count, "goblin_lair should appear exactly once in VisitedRooms")
}

func TestErrorMessages(t *testing.T) {
	noExit := &engine.NoExitError{Direction: "up"}
	assert.Contains(t, noExit.Error(), "up")

	locked := &engine.LockedError{Direction: "west", Room: "vault"}
	assert.Contains(t, locked.Error(), "west")
}

func TestLook_UnknownRoomReturnsBareResult(t *testing.T) {
	e, state := newGame(t)
	state.Dungeon.CurrentRoom = "nonexistent_room"
	result := e.Look(&state)
	assert.Equal(t, "nonexistent_room", result.Room)
	assert.Empty(t, result.Name)
	assert.Empty(t, result.Exits)
}

func TestLook_HiddenConnectionsNotInExits(t *testing.T) {
	e, state := newGame(t)
	result := e.Look(&state)
	// entrance has north (hidden) — should not appear in exits
	for _, exit := range result.Exits {
		assert.NotEqual(t, "north", exit, "hidden connection 'north' should not appear in exits")
	}
	assert.Contains(t, result.Exits, "south")
}

func TestMove_HiddenIsNoExit(t *testing.T) {
	e, state := newGame(t)
	_, err := e.Move(&state, "north")
	require.Error(t, err)
	var noExit *engine.NoExitError
	require.ErrorAs(t, err, &noExit)
	assert.Equal(t, "north", noExit.Direction)
}

func TestMove_UsesInjectedClock(t *testing.T) {
	e, state := newGame(t)
	fixed := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	e.Now = func() time.Time { return fixed }

	_, err := e.Move(&state, "south")
	require.NoError(t, err)
	require.NotEmpty(t, state.AdventureLog)
	assert.Equal(t, "2025-01-01T00:00:00Z", state.AdventureLog[len(state.AdventureLog)-1].Timestamp)
}

func TestMove_UnknownConnectionType(t *testing.T) {
	// Build a synthetic scenario with a "bogus" connection type.
	s := &scenario.Scenario{
		ID:           "synthetic",
		StartingRoom: "start",
		Rooms: map[string]*scenario.Room{
			"start": {
				Name: "Start",
				Connections: map[string]*scenario.Connection{
					"south": {Room: "end", Type: "bogus"},
				},
			},
			"end": {Name: "End"},
		},
	}
	e := engine.New(s)
	char := model.Character{ID: "c1", Name: "Test", Class: "fighter", Level: 1, HP: 10, MaxHP: 10}
	state, err := e.NewGame(char)
	require.NoError(t, err)

	_, err = e.Move(&state, "south")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bogus")
}

func TestLook_ReturnsRoomInfo(t *testing.T) {
	e, state := newGame(t)
	result := e.Look(&state)
	assert.Equal(t, "entrance", result.Room)
	assert.NotEmpty(t, result.Description)
	assert.Contains(t, result.Exits, "south")
}

// --- Inventory tests ---

func TestNewGame_SeedsRoomItems(t *testing.T) {
	_, state := newGame(t)
	// entrance has short_sword, goblin_lair has rusty_key
	assert.Contains(t, state.Dungeon.RoomState["entrance"].Items, "short_sword")
	assert.Contains(t, state.Dungeon.RoomState["goblin_lair"].Items, "rusty_key")
	assert.Contains(t, state.Dungeon.RoomState["vault"].Items, "gold_coin")
	// secret_room has no items
	assert.Empty(t, state.Dungeon.RoomState["secret_room"].Items)
}

func TestPickUp_Success(t *testing.T) {
	e, state := newGame(t)
	result, err := e.PickUp(&state, "short_sword")
	require.NoError(t, err)
	assert.Equal(t, "Short Sword", result.Item.Name)
	assert.Equal(t, "weapon", result.Item.Type)
	// Item removed from room, added to inventory.
	assert.NotContains(t, state.Dungeon.RoomState["entrance"].Items, "short_sword")
	require.Len(t, state.Party[0].Inventory, 1)
	assert.Equal(t, "short_sword", state.Party[0].Inventory[0].ID)
	// Log entry appended.
	require.NotEmpty(t, state.AdventureLog)
	assert.Contains(t, state.AdventureLog[len(state.AdventureLog)-1].Text, "Short Sword")
}

func TestPickUp_ItemNotInRoom(t *testing.T) {
	e, state := newGame(t)
	_, err := e.PickUp(&state, "nonexistent")
	require.Error(t, err)
	var notInRoom *engine.ItemNotInRoomError
	require.ErrorAs(t, err, &notInRoom)
	assert.Equal(t, "nonexistent", notInRoom.ItemID)
}

func TestPickUp_TooHeavy(t *testing.T) {
	s := &scenario.Scenario{
		ID:           "heavy",
		StartingRoom: "room",
		Rooms: map[string]*scenario.Room{
			"room": {Name: "Room", Items: []string{"boulder"}},
		},
		Items: map[string]*scenario.ScenarioItem{
			"boulder": {Name: "Boulder", Type: "misc", Weight: model.MaxCarryWeight + 1},
		},
	}
	e := engine.New(s)
	char := model.Character{ID: "c1", Name: "Test", Class: "fighter", Level: 1, HP: 10, MaxHP: 10}
	state, err := e.NewGame(char)
	require.NoError(t, err)

	_, err = e.PickUp(&state, "boulder")
	require.Error(t, err)
	var heavy *engine.TooHeavyError
	require.ErrorAs(t, err, &heavy)
	// Item stays in room.
	assert.Contains(t, state.Dungeon.RoomState["room"].Items, "boulder")
}

func TestDrop_Success(t *testing.T) {
	e, state := newGame(t)
	_, err := e.PickUp(&state, "short_sword")
	require.NoError(t, err)

	result, err := e.Drop(&state, "short_sword")
	require.NoError(t, err)
	assert.Equal(t, "Short Sword", result.Item.Name)
	assert.Empty(t, state.Party[0].Inventory)
	assert.Contains(t, state.Dungeon.RoomState["entrance"].Items, "short_sword")
}

func TestDrop_NotInInventory(t *testing.T) {
	e, state := newGame(t)
	_, err := e.Drop(&state, "short_sword")
	require.Error(t, err)
	var notInInv *engine.ItemNotInInventoryError
	require.ErrorAs(t, err, &notInInv)
}

func TestEquip_Success(t *testing.T) {
	e, state := newGame(t)
	_, err := e.PickUp(&state, "short_sword")
	require.NoError(t, err)

	result, err := e.Equip(&state, "short_sword")
	require.NoError(t, err)
	assert.Equal(t, "weapon", result.Slot)
	assert.Equal(t, "short_sword", state.Party[0].Equipped.Weapon)
	assert.Empty(t, state.Party[0].Inventory, "item should be removed from inventory after equipping")
}

func TestEquip_NotInInventory(t *testing.T) {
	e, state := newGame(t)
	_, err := e.Equip(&state, "short_sword")
	require.Error(t, err)
	var notInInv *engine.ItemNotInInventoryError
	require.ErrorAs(t, err, &notInInv)
}

func TestEquip_NotEquippable(t *testing.T) {
	e, state := newGame(t)
	// Move to goblin_lair and pick up rusty_key (type: key)
	_, err := e.Move(&state, "south")
	require.NoError(t, err)
	_, err = e.PickUp(&state, "rusty_key")
	require.NoError(t, err)

	_, err = e.Equip(&state, "rusty_key")
	require.Error(t, err)
	var notEquippable *engine.NotEquippableError
	require.ErrorAs(t, err, &notEquippable)
	assert.Equal(t, "key", notEquippable.ItemType)
}

func TestEquip_SlotOccupied(t *testing.T) {
	s := &scenario.Scenario{
		ID:           "two-weapons",
		StartingRoom: "room",
		Rooms: map[string]*scenario.Room{
			"room": {Name: "Room", Items: []string{"sword_a", "sword_b"}},
		},
		Items: map[string]*scenario.ScenarioItem{
			"sword_a": {Name: "Sword A", Type: "weapon", Damage: "1d6", Weight: 3},
			"sword_b": {Name: "Sword B", Type: "weapon", Damage: "1d8", Weight: 4},
		},
	}
	e := engine.New(s)
	char := model.Character{ID: "c1", Name: "Test", Class: "fighter", Level: 1, HP: 10, MaxHP: 10}
	state, err := e.NewGame(char)
	require.NoError(t, err)

	_, err = e.PickUp(&state, "sword_a")
	require.NoError(t, err)
	_, err = e.PickUp(&state, "sword_b")
	require.NoError(t, err)
	_, err = e.Equip(&state, "sword_a")
	require.NoError(t, err)

	_, err = e.Equip(&state, "sword_b")
	require.Error(t, err)
	var occupied *engine.SlotOccupiedError
	require.ErrorAs(t, err, &occupied)
	assert.Equal(t, "weapon", occupied.Slot)
}

func TestUnequip_Success(t *testing.T) {
	e, state := newGame(t)
	_, err := e.PickUp(&state, "short_sword")
	require.NoError(t, err)
	_, err = e.Equip(&state, "short_sword")
	require.NoError(t, err)

	result, err := e.Unequip(&state, "weapon")
	require.NoError(t, err)
	assert.Equal(t, "Short Sword", result.Item.Name)
	assert.Equal(t, "", state.Party[0].Equipped.Weapon)
	require.Len(t, state.Party[0].Inventory, 1)
	assert.Equal(t, "short_sword", state.Party[0].Inventory[0].ID)
}

func TestUnequip_SlotEmpty(t *testing.T) {
	e, state := newGame(t)
	_, err := e.Unequip(&state, "weapon")
	require.Error(t, err)
	var empty *engine.SlotEmptyError
	require.ErrorAs(t, err, &empty)
}

func TestExamine_InInventory(t *testing.T) {
	e, state := newGame(t)
	_, err := e.PickUp(&state, "short_sword")
	require.NoError(t, err)

	result, err := e.Examine(&state, "short_sword")
	require.NoError(t, err)
	assert.Equal(t, "Short Sword", result.Item.Name)
	assert.Equal(t, "A simple but serviceable blade.", result.Item.Description)
}

func TestExamine_InRoom(t *testing.T) {
	e, state := newGame(t)
	result, err := e.Examine(&state, "short_sword")
	require.NoError(t, err)
	assert.Equal(t, "Short Sword", result.Item.Name)
}

func TestExamine_NotFound(t *testing.T) {
	e, state := newGame(t)
	_, err := e.Examine(&state, "nonexistent")
	require.Error(t, err)
	var notInRoom *engine.ItemNotInRoomError
	require.ErrorAs(t, err, &notInRoom)
}

func TestInventory_Empty(t *testing.T) {
	e, state := newGame(t)
	result := e.Inventory(&state)
	assert.Empty(t, result.Items)
	assert.Equal(t, float64(0), result.Weight)
	assert.Equal(t, model.MaxCarryWeight, result.Capacity)
}

func TestInventory_AfterPickUp(t *testing.T) {
	e, state := newGame(t)
	_, err := e.PickUp(&state, "short_sword")
	require.NoError(t, err)

	result := e.Inventory(&state)
	require.Len(t, result.Items, 1)
	assert.Equal(t, "short_sword", result.Items[0].ID)
	assert.Equal(t, 3.0, result.Weight)
}

func TestLook_ReturnsRoomItemsFromMutableState(t *testing.T) {
	e, state := newGame(t)
	result := e.Look(&state)
	assert.Contains(t, result.Items, "short_sword")

	// Pick up item — look should no longer show it.
	_, err := e.PickUp(&state, "short_sword")
	require.NoError(t, err)
	result = e.Look(&state)
	assert.NotContains(t, result.Items, "short_sword")
}

func TestMove_ReturnsRoomItemsFromMutableState(t *testing.T) {
	e, state := newGame(t)
	// Drop an item in entrance, move south, move back — should see dropped item.
	_, err := e.PickUp(&state, "short_sword")
	require.NoError(t, err)
	_, err = e.Move(&state, "south")
	require.NoError(t, err)
	_, err = e.Drop(&state, "short_sword")
	require.NoError(t, err)

	// Move back and forth to verify items travel with the room.
	_, err = e.Move(&state, "north")
	require.NoError(t, err)
	result, err := e.Move(&state, "south")
	require.NoError(t, err)
	assert.Contains(t, result.Items, "short_sword")
}

func TestInventoryErrorMessages(t *testing.T) {
	assert.Contains(t, (&engine.ItemNotInRoomError{ItemID: "key"}).Error(), "key")
	assert.Contains(t, (&engine.ItemNotInInventoryError{ItemID: "key"}).Error(), "key")
	assert.Contains(t, (&engine.TooHeavyError{ItemID: "rock", Weight: 10, Capacity: 50, Current: 45}).Error(), "rock")
	assert.Contains(t, (&engine.NotEquippableError{ItemID: "key", ItemType: "key"}).Error(), "key")
	assert.Contains(t, (&engine.SlotOccupiedError{Slot: "weapon", OccupiedBy: "sword"}).Error(), "weapon")
	assert.Contains(t, (&engine.SlotEmptyError{Slot: "weapon"}).Error(), "weapon")
}
