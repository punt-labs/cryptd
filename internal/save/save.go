// Package save handles persistence of GameState to JSON files.
// Save files live at .dungeon/saves/<slot>.json (DES-017).
package save

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/punt-labs/cryptd/internal/model"
)

// SchemaVersion is the current save file format version.
const SchemaVersion = "1.0"

// ErrVersionMismatch is returned when a save file's schema_version does not
// match SchemaVersion.
var ErrVersionMismatch = errors.New("unsupported schema_version")

// Save writes state to path as indented JSON.
// It creates any necessary parent directories.
// SchemaVersion is set on the state before writing.
func Save(state model.GameState, path string) error {
	state.SchemaVersion = SchemaVersion
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal game state: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create save directory: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write save file %s: %w", path, err)
	}
	return nil
}

// Load reads and deserialises a save file from path.
// Returns ErrVersionMismatch if the file's schema_version is not SchemaVersion.
// Unknown JSON fields are silently ignored (forward compatibility).
func Load(path string) (model.GameState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return model.GameState{}, fmt.Errorf("read save file %s: %w", path, err)
	}
	var state model.GameState
	if err := json.Unmarshal(data, &state); err != nil {
		return model.GameState{}, fmt.Errorf("parse save file %s: %w", path, err)
	}
	if state.SchemaVersion != SchemaVersion {
		return model.GameState{}, fmt.Errorf("%w: got %q, want %q", ErrVersionMismatch, state.SchemaVersion, SchemaVersion)
	}
	return state, nil
}
