package engine_test

import (
	"testing"

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

func TestLook_ReturnsRoomInfo(t *testing.T) {
	e, state := newGame(t)
	result := e.Look(&state)
	assert.Equal(t, "entrance", result.Room)
	assert.NotEmpty(t, result.Description)
	assert.Contains(t, result.Exits, "south")
}
