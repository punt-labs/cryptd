// Package engine implements the deterministic game rules machine.
// It knows nothing about interpreters, narrators, or renderers.
package engine

import (
	"fmt"
	"time"

	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/scenario"
)

// MoveResult holds the outcome of a successful move action.
type MoveResult struct {
	NewRoom string
	Exits   []string
	Items   []string
	Enemies []string
}

// LookResult holds the outcome of a look action.
type LookResult struct {
	Room        string
	Name        string
	Description string
	Exits       []string
	Items       []string
	Enemies     []string
}

// NoExitError is returned when the player moves in a direction with no connection.
type NoExitError struct {
	Direction string
}

func (e *NoExitError) Error() string {
	return fmt.Sprintf("no exit to the %s", e.Direction)
}

// LockedError is returned when the player tries to move through a locked door.
type LockedError struct {
	Direction string
	Room      string
}

func (e *LockedError) Error() string {
	return fmt.Sprintf("the way %s is locked", e.Direction)
}

// Engine is the deterministic rules machine. All game state transitions go
// through Engine methods. The Engine holds the scenario but never mutates it.
type Engine struct {
	s *scenario.Scenario
}

// New creates an Engine for the given scenario.
func New(s *scenario.Scenario) *Engine {
	return &Engine{s: s}
}

// NewGame initialises a fresh GameState for the scenario and character.
func (e *Engine) NewGame(char model.Character) (model.GameState, error) {
	state := model.GameState{
		Scenario: e.s.ID,
		PlayMode: "headless",
		Dungeon: model.DungeonState{
			CurrentRoom:  e.s.StartingRoom,
			VisitedRooms: []string{e.s.StartingRoom},
			RoomState:    make(map[string]model.RoomState),
		},
		Party: []model.Character{char},
	}
	return state, nil
}

// Move executes a move in the given direction, mutating state in place.
// Returns a MoveResult on success, or NoExitError / LockedError on failure.
func (e *Engine) Move(state *model.GameState, direction string) (MoveResult, error) {
	room, ok := e.s.Rooms[state.Dungeon.CurrentRoom]
	if !ok {
		return MoveResult{}, fmt.Errorf("current room %q not found in scenario", state.Dungeon.CurrentRoom)
	}

	conn, ok := room.Connections[direction]
	if !ok {
		return MoveResult{}, &NoExitError{Direction: direction}
	}

	if conn.Type == "locked" {
		return MoveResult{}, &LockedError{Direction: direction, Room: conn.Room}
	}

	// Hidden connections are undiscoverable until revealed; treat as no exit.
	if conn.Type == "hidden" {
		return MoveResult{}, &NoExitError{Direction: direction}
	}

	// Validate destination before mutating state so a broken scenario ref
	// cannot corrupt the in-progress game state.
	dest, ok := e.s.Rooms[conn.Room]
	if !ok {
		return MoveResult{}, fmt.Errorf("destination room %q not found in scenario", conn.Room)
	}

	state.Dungeon.CurrentRoom = conn.Room
	state.Dungeon.VisitedRooms = appendUnique(state.Dungeon.VisitedRooms, conn.Room)

	state.AdventureLog = append(state.AdventureLog, model.LogEntry{
		Text:      fmt.Sprintf("You move %s and enter %s.", direction, conn.Room),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})

	return MoveResult{
		NewRoom: conn.Room,
		Exits:   exitList(dest),
		Items:   dest.Items,
		Enemies: dest.Enemies,
	}, nil
}

// Look returns information about the current room without mutating state.
func (e *Engine) Look(state *model.GameState) LookResult {
	room, ok := e.s.Rooms[state.Dungeon.CurrentRoom]
	if !ok {
		return LookResult{Room: state.Dungeon.CurrentRoom}
	}
	return LookResult{
		Room:        state.Dungeon.CurrentRoom,
		Name:        room.Name,
		Description: room.DescriptionSeed,
		Exits:       exitList(room),
		Items:       room.Items,
		Enemies:     room.Enemies,
	}
}

func exitList(room *scenario.Room) []string {
	exits := make([]string, 0, len(room.Connections))
	for dir := range room.Connections {
		exits = append(exits, dir)
	}
	return exits
}

func appendUnique(slice []string, s string) []string {
	for _, v := range slice {
		if v == s {
			return slice
		}
	}
	return append(slice, s)
}
