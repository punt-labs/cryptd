package scenario

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/punt-labs/cryptd/internal/dice"
	"gopkg.in/yaml.v3"
)

// MissingFieldError is returned when a required field is absent or empty.
type MissingFieldError struct {
	Field string
}

func (e *MissingFieldError) Error() string {
	return fmt.Sprintf("missing required field: %s", e.Field)
}

// BrokenRoomRefError is returned when a connection points to a nonexistent room.
type BrokenRoomRefError struct {
	Room   string
	Dir    string
	Target string
}

func (e *BrokenRoomRefError) Error() string {
	return fmt.Sprintf("room %q connection %q references unknown room %q", e.Room, e.Dir, e.Target)
}

// UnknownEnemyError is returned when a room references an enemy template that
// does not exist in the scenario's enemies map.
type UnknownEnemyError struct {
	Room    string
	EnemyID string
}

func (e *UnknownEnemyError) Error() string {
	return fmt.Sprintf("room %q references unknown enemy template %q", e.Room, e.EnemyID)
}

// InvalidDiceError is returned when a dice-notation field cannot be parsed.
type InvalidDiceError struct {
	Field string
	Value string
	Err   error
}

func (e *InvalidDiceError) Error() string {
	return fmt.Sprintf("invalid dice notation in %s %q: %v", e.Field, e.Value, e.Err)
}

func (e *InvalidDiceError) Unwrap() error { return e.Err }

// Load reads a scenario YAML file, sets ID from the filename, and validates it.
// Returns the first validation error encountered.
func Load(path string) (*Spec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read scenario %s: %w", path, err)
	}

	var s Spec
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse scenario %s: %w", path, err)
	}

	base := filepath.Base(path)
	s.ID = strings.TrimSuffix(base, filepath.Ext(base))

	if err := Validate(&s); err != nil {
		return nil, err
	}
	return &s, nil
}

// Validate checks the semantic integrity of a parsed Spec.
// Returns the first error found, typed as one of MissingFieldError,
// BrokenRoomRefError, UnknownEnemyError, or InvalidDiceError.
func Validate(s *Spec) error {
	if s.StartingRoom == "" {
		return &MissingFieldError{Field: "starting_room"}
	}
	if _, ok := s.Rooms[s.StartingRoom]; !ok {
		return &BrokenRoomRefError{Room: "(scenario)", Dir: "starting_room", Target: s.StartingRoom}
	}

	for roomID, room := range s.Rooms {
		if room.Name == "" {
			return &MissingFieldError{Field: fmt.Sprintf("rooms.%s.name", roomID)}
		}
		for dir, conn := range room.Connections {
			if _, ok := s.Rooms[conn.Room]; !ok {
				return &BrokenRoomRefError{Room: roomID, Dir: dir, Target: conn.Room}
			}
		}
		for _, enemyID := range room.Enemies {
			if _, ok := s.Enemies[enemyID]; !ok {
				return &UnknownEnemyError{Room: roomID, EnemyID: enemyID}
			}
		}
	}

	for id, enemy := range s.Enemies {
		if enemy.Attack != "" {
			if _, err := dice.Parse(enemy.Attack); err != nil {
				return &InvalidDiceError{Field: fmt.Sprintf("enemies.%s.attack", id), Value: enemy.Attack, Err: err}
			}
		}
	}

	for id, item := range s.Items {
		if item.Damage != "" {
			if _, err := dice.Parse(item.Damage); err != nil {
				return &InvalidDiceError{Field: fmt.Sprintf("items.%s.damage", id), Value: item.Damage, Err: err}
			}
		}
	}

	return nil
}
