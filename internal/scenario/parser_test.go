package scenario_test

import (
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/punt-labs/cryptd/internal/scenario"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testdataDir = "../../testdata/scenarios"

func TestLoad_ValidMinimal(t *testing.T) {
	s, err := scenario.Load(filepath.Join(testdataDir, "minimal.yaml"))
	require.NoError(t, err)

	assert.Equal(t, "minimal", s.ID)
	assert.Equal(t, "Minimal Dungeon", s.Title)
	assert.Equal(t, "entrance", s.StartingRoom)
	assert.Equal(t, "respawn", s.Death)

	require.Contains(t, s.Rooms, "entrance")
	assert.Equal(t, "Entrance Hall", s.Rooms["entrance"].Name)
	require.Contains(t, s.Rooms["entrance"].Connections, "south")
	assert.Equal(t, "goblin_lair", s.Rooms["entrance"].Connections["south"].Room)
	assert.Equal(t, "open", s.Rooms["entrance"].Connections["south"].Type)

	require.Contains(t, s.Enemies, "goblin")
	assert.Equal(t, 8, s.Enemies["goblin"].HP)
	assert.Equal(t, "1d4", s.Enemies["goblin"].Attack)

	require.Contains(t, s.Items, "rusty_key")
	assert.Equal(t, "Rusty Key", s.Items["rusty_key"].Name)
}

func TestLoad_SetsIDFromFilename(t *testing.T) {
	s, err := scenario.Load(filepath.Join(testdataDir, "minimal.yaml"))
	require.NoError(t, err)
	assert.Equal(t, "minimal", s.ID)
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := scenario.Load("nonexistent/path.yaml")
	require.Error(t, err)
}

func TestValidate_StartingRoomNotInRooms(t *testing.T) {
	s := &scenario.Scenario{
		Title:        "Test",
		StartingRoom: "nonexistent",
		Rooms:        map[string]*scenario.Room{"entrance": {Name: "Hall"}},
	}
	err := scenario.Validate(s)
	require.Error(t, err)
	var e *scenario.BrokenRoomRefError
	assert.True(t, errors.As(err, &e))
}

func TestErrorMessages(t *testing.T) {
	assert.Contains(t, (&scenario.MissingFieldError{Field: "starting_room"}).Error(), "starting_room")
	assert.Contains(t, (&scenario.BrokenRoomRefError{Room: "r", Dir: "n", Target: "x"}).Error(), "x")
	assert.Contains(t, (&scenario.UnknownEnemyError{Room: "r", EnemyID: "goblin"}).Error(), "goblin")

	inner := fmt.Errorf("bad dice")
	ide := &scenario.InvalidDiceError{Field: "attack", Value: "2x6", Err: inner}
	assert.Contains(t, ide.Error(), "2x6")
	assert.ErrorIs(t, ide, inner)
}

// Invalid fixture tests — each triggers a specific typed error.

func TestValidate_MissingStartingRoom(t *testing.T) {
	_, err := scenario.Load(filepath.Join(testdataDir, "invalid", "missing-starting-room.yaml"))
	require.Error(t, err)
	var e *scenario.MissingFieldError
	assert.True(t, errors.As(err, &e), "want MissingFieldError, got %T: %v", err, err)
	assert.Equal(t, "starting_room", e.Field)
}

func TestValidate_BrokenRoomRef(t *testing.T) {
	_, err := scenario.Load(filepath.Join(testdataDir, "invalid", "broken-room-ref.yaml"))
	require.Error(t, err)
	var e *scenario.BrokenRoomRefError
	assert.True(t, errors.As(err, &e), "want BrokenRoomRefError, got %T: %v", err, err)
}

func TestValidate_MissingRoomName(t *testing.T) {
	_, err := scenario.Load(filepath.Join(testdataDir, "invalid", "missing-room-name.yaml"))
	require.Error(t, err)
	var e *scenario.MissingFieldError
	assert.True(t, errors.As(err, &e), "want MissingFieldError, got %T: %v", err, err)
}

func TestValidate_UnknownEnemyTemplate(t *testing.T) {
	_, err := scenario.Load(filepath.Join(testdataDir, "invalid", "unknown-enemy-template.yaml"))
	require.Error(t, err)
	var e *scenario.UnknownEnemyError
	assert.True(t, errors.As(err, &e), "want UnknownEnemyError, got %T: %v", err, err)
}

func TestValidate_InvalidDiceEnemy(t *testing.T) {
	_, err := scenario.Load(filepath.Join(testdataDir, "invalid", "invalid-dice-enemy.yaml"))
	require.Error(t, err)
	var e *scenario.InvalidDiceError
	assert.True(t, errors.As(err, &e), "want InvalidDiceError, got %T: %v", err, err)
}

func TestValidate_InvalidDiceItem(t *testing.T) {
	_, err := scenario.Load(filepath.Join(testdataDir, "invalid", "invalid-dice-item.yaml"))
	require.Error(t, err)
	var e *scenario.InvalidDiceError
	assert.True(t, errors.As(err, &e), "want InvalidDiceError, got %T: %v", err, err)
}
