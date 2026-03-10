package save_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/save"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sampleState() model.GameState {
	return model.GameState{
		PlayMode: "headless",
		Scenario: "minimal",
		Timestamp: time.Date(2026, 3, 10, 18, 0, 0, 0, time.UTC).Format(time.RFC3339),
		Party: []model.Character{
			{
				ID:    "aldric",
				Name:  "Aldric",
				Class: "fighter",
				Level: 3,
				HP:    45, MaxHP: 60,
				XP: 1240, Gold: 48,
				Stats:      model.Stats{STR: 16, INT: 10, DEX: 12, CON: 14, WIS: 9, CHA: 11},
				Inventory:  []model.Item{},
				Equipped:   model.Equipment{Weapon: "rusty_sword"},
				Conditions: []model.Condition{},
			},
		},
		Dungeon: model.DungeonState{
			CurrentRoom:  "goblin_lair",
			VisitedRooms: []string{"entrance", "corridor_a", "goblin_lair"},
			RoomState:    map[string]model.RoomState{"goblin_lair": {Cleared: true}},
		},
		AdventureLog: []model.LogEntry{
			{Text: "You enter the dungeon.", Timestamp: "2026-03-10T18:00:00Z"},
			{Text: "You move south.", Timestamp: "2026-03-10T18:01:00Z"},
			{Text: "A goblin attacks!", Timestamp: "2026-03-10T18:02:00Z"},
		},
	}
}

func TestSaveLoad_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "saves", "test.json")

	original := sampleState()
	require.NoError(t, save.Save(original, path))

	loaded, err := save.Load(path)
	require.NoError(t, err)

	// SchemaVersion is injected by Save
	original.SchemaVersion = save.SchemaVersion
	assert.Equal(t, original, loaded)
}

func TestSave_CreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "deep", "save.json")
	require.NoError(t, save.Save(sampleState(), path))
	_, err := os.Stat(path)
	assert.NoError(t, err)
}

func TestLoad_UnknownFieldIgnored(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "save.json")

	// Write JSON with an extra field that the struct doesn't know about
	raw := map[string]any{
		"schema_version": save.SchemaVersion,
		"play_mode":      "headless",
		"scenario":       "minimal",
		"timestamp":      "2026-03-10T18:00:00Z",
		"party":          []any{},
		"dungeon": map[string]any{
			"current_room":  "entrance",
			"visited_rooms": []string{"entrance"},
			"room_state":    map[string]any{},
		},
		"adventure_log": []any{},
		"future_field":   "some future value", // unknown field
	}
	data, err := json.MarshalIndent(raw, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0o644))

	loaded, err := save.Load(path)
	require.NoError(t, err, "unknown field should not cause an error")
	assert.Equal(t, "headless", loaded.PlayMode)
}

func TestLoad_VersionMismatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "save.json")

	raw := map[string]any{
		"schema_version": "99.0",
		"play_mode":      "headless",
		"party":          []any{},
		"dungeon":        map[string]any{"visited_rooms": []any{}, "room_state": map[string]any{}},
		"adventure_log":  []any{},
	}
	data, err := json.MarshalIndent(raw, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0o644))

	_, err = save.Load(path)
	require.Error(t, err)
	assert.ErrorIs(t, err, save.ErrVersionMismatch)
}

func TestLoad_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	require.NoError(t, os.WriteFile(path, []byte("{not valid json"), 0o644))
	_, err := save.Load(path)
	require.Error(t, err)
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := save.Load("nonexistent/path.json")
	require.Error(t, err)
}

func TestLoad_FixtureFile(t *testing.T) {
	state, err := save.Load("../../testdata/saves/fighter-level-3.json")
	require.NoError(t, err)
	require.Len(t, state.Party, 1)
	assert.Equal(t, 3, state.Party[0].Level)
	assert.Equal(t, "fighter", state.Party[0].Class)
	assert.Equal(t, "Aldric", state.Party[0].Name)
}

func TestSave_MkdirAllFails(t *testing.T) {
	dir := t.TempDir()
	// Create a regular file where we want a directory — MkdirAll will fail
	blocker := filepath.Join(dir, "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte(""), 0o644))
	path := filepath.Join(blocker, "save.json") // blocker is a file, not a dir
	err := save.Save(sampleState(), path)
	require.Error(t, err)
}

func TestSave_WriteFileFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "save.json")
	// Create the file and make it read-only, then try to overwrite it
	require.NoError(t, os.WriteFile(path, []byte("{}"), 0o444))
	// Make parent dir read-only too so WriteFile fails
	require.NoError(t, os.Chmod(dir, 0o555))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) }) // restore for cleanup
	err := save.Save(sampleState(), path)
	// On some systems this may succeed as root; skip if so
	if err == nil {
		t.Skip("could not make directory read-only (running as root?)")
	}
	require.Error(t, err)
}
