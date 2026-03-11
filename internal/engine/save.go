package engine

import (
	"fmt"
	"path/filepath"
	"regexp"
	"time"

	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/save"
)

// validSlot matches slot names containing only alphanumeric characters,
// underscores, and hyphens. This prevents path traversal attacks.
var validSlot = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// InvalidSlotError is returned when a save slot name contains invalid characters.
type InvalidSlotError struct {
	Slot string
}

func (e *InvalidSlotError) Error() string {
	return fmt.Sprintf("invalid slot name %q: must contain only alphanumeric characters, underscores, and hyphens", e.Slot)
}

// ScenarioMismatchError is returned when loading a save from a different scenario.
type ScenarioMismatchError struct {
	SaveScenario   string
	EngineScenario string
}

func (e *ScenarioMismatchError) Error() string {
	return fmt.Sprintf("save is for scenario %q but engine is running %q", e.SaveScenario, e.EngineScenario)
}

// SaveResult holds the outcome of saving the game.
type SaveResult struct {
	Slot string
	Path string
}

// LoadResult holds the outcome of loading a saved game.
type LoadResult struct {
	Slot string
	Path string
}

// validateSlot normalises an empty slot to "quicksave" and rejects names
// that contain path separators, "..", or non-alphanumeric characters.
func validateSlot(slot string) (string, error) {
	if slot == "" {
		return "quicksave", nil
	}
	if !validSlot.MatchString(slot) {
		return "", &InvalidSlotError{Slot: slot}
	}
	return slot, nil
}

// SaveGame persists the current game state to a named save slot.
// The save directory is Engine.SaveDir (defaults to ".dungeon/saves").
func (e *Engine) SaveGame(state *model.GameState, slot string) (SaveResult, error) {
	slot, err := validateSlot(slot)
	if err != nil {
		return SaveResult{}, err
	}
	state.Timestamp = e.Now().UTC().Format(time.RFC3339)
	path := filepath.Join(e.saveDir(), slot+".json")
	if err := save.Save(*state, path); err != nil {
		return SaveResult{}, fmt.Errorf("save game: %w", err)
	}
	return SaveResult{Slot: slot, Path: path}, nil
}

// LoadGame reads a saved game state from a named slot. The returned state
// replaces the caller's current state entirely.
func (e *Engine) LoadGame(slot string) (model.GameState, LoadResult, error) {
	slot, err := validateSlot(slot)
	if err != nil {
		return model.GameState{}, LoadResult{}, err
	}
	path := filepath.Join(e.saveDir(), slot+".json")
	state, err := save.Load(path)
	if err != nil {
		return model.GameState{}, LoadResult{}, fmt.Errorf("load game: %w", err)
	}
	if state.Scenario != e.s.ID {
		return model.GameState{}, LoadResult{}, &ScenarioMismatchError{
			SaveScenario:   state.Scenario,
			EngineScenario: e.s.ID,
		}
	}
	return state, LoadResult{Slot: slot, Path: path}, nil
}

// saveDir returns the directory for save files.
func (e *Engine) saveDir() string {
	if e.SaveDir != "" {
		return e.SaveDir
	}
	return filepath.Join(".dungeon", "saves")
}
