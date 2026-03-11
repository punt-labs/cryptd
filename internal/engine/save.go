package engine

import (
	"fmt"
	"path/filepath"

	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/save"
)

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

// SaveGame persists the current game state to a named save slot.
// The save directory defaults to ".dungeon/saves" relative to the working directory.
func (e *Engine) SaveGame(state *model.GameState, slot string) (SaveResult, error) {
	if slot == "" {
		slot = "quicksave"
	}
	state.Timestamp = e.Now().UTC().Format("2006-01-02T15:04:05Z")
	path := filepath.Join(".dungeon", "saves", slot+".json")
	if err := save.Save(*state, path); err != nil {
		return SaveResult{}, fmt.Errorf("save game: %w", err)
	}
	return SaveResult{Slot: slot, Path: path}, nil
}

// LoadGame reads a saved game state from a named slot. The returned state
// replaces the caller's current state entirely.
func (e *Engine) LoadGame(slot string) (model.GameState, LoadResult, error) {
	if slot == "" {
		slot = "quicksave"
	}
	path := filepath.Join(".dungeon", "saves", slot+".json")
	state, err := save.Load(path)
	if err != nil {
		return model.GameState{}, LoadResult{}, fmt.Errorf("load game: %w", err)
	}
	return state, LoadResult{Slot: slot, Path: path}, nil
}
