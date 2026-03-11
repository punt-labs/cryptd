package engine_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/punt-labs/cryptd/internal/engine"
	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/scenario"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func saveScenario() *scenario.Scenario {
	return &scenario.Scenario{
		ID:           "save-test",
		StartingRoom: "entrance",
		Rooms: map[string]*scenario.Room{
			"entrance": {
				Name:    "Entrance",
				Enemies: []string{"goblin"},
			},
		},
		Enemies: map[string]*scenario.EnemyTemplate{
			"goblin": {Name: "Goblin", HP: 8, Attack: "1d4", AI: "aggressive"},
		},
		Items:  map[string]*scenario.ScenarioItem{},
		Spells: map[string]*scenario.SpellTemplate{},
	}
}

func TestSaveAndLoad_RoundTrip(t *testing.T) {
	e := engine.New(saveScenario())
	char := model.Character{
		ID: "c1", Name: "Hero", Class: "fighter", Level: 1,
		HP: 20, MaxHP: 20, MP: 0, MaxMP: 0, XP: 15,
		Stats: model.Stats{STR: 14, DEX: 12, CON: 12, INT: 10, WIS: 10, CHA: 10},
	}
	state, err := e.NewGame(char)
	require.NoError(t, err)

	// Mutate state to make it interesting: start combat.
	_, err = e.StartCombat(&state)
	require.NoError(t, err)
	require.True(t, state.Dungeon.Combat.Active)

	// Save to a temp directory.
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	result, err := e.SaveGame(&state, "test-slot")
	require.NoError(t, err)
	assert.Equal(t, "test-slot", result.Slot)

	// Verify file exists.
	_, err = os.Stat(filepath.Join(tmpDir, ".dungeon", "saves", "test-slot.json"))
	require.NoError(t, err)

	// Load it back.
	loaded, loadResult, err := e.LoadGame("test-slot")
	require.NoError(t, err)
	assert.Equal(t, "test-slot", loadResult.Slot)

	// Verify state is identical.
	assert.Equal(t, state.Scenario, loaded.Scenario)
	assert.Equal(t, state.Dungeon.CurrentRoom, loaded.Dungeon.CurrentRoom)
	assert.Equal(t, state.Dungeon.Combat.Active, loaded.Dungeon.Combat.Active)
	assert.Equal(t, len(state.Dungeon.Combat.Enemies), len(loaded.Dungeon.Combat.Enemies))
	assert.Equal(t, state.Party[0].HP, loaded.Party[0].HP)
	assert.Equal(t, state.Party[0].XP, loaded.Party[0].XP)
	assert.Equal(t, state.Party[0].Stats, loaded.Party[0].Stats)
}

func TestSaveGame_DefaultSlot(t *testing.T) {
	e := engine.New(saveScenario())
	char := model.Character{
		ID: "c1", Name: "Hero", Class: "fighter", Level: 1,
		HP: 20, MaxHP: 20,
		Stats: model.Stats{STR: 10, DEX: 10, CON: 10, INT: 10, WIS: 10, CHA: 10},
	}
	state, err := e.NewGame(char)
	require.NoError(t, err)

	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	result, err := e.SaveGame(&state, "")
	require.NoError(t, err)
	assert.Equal(t, "quicksave", result.Slot)
}

func TestLoadGame_NotFound(t *testing.T) {
	e := engine.New(saveScenario())

	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	_, _, err = e.LoadGame("nonexistent")
	require.Error(t, err)
}
