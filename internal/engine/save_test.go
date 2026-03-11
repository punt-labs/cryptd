package engine_test

import (
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

func saveEngine(t *testing.T) *engine.Engine {
	t.Helper()
	e := engine.New(saveScenario())
	e.SaveDir = t.TempDir()
	return e
}

func TestSaveAndLoad_RoundTrip(t *testing.T) {
	e := saveEngine(t)
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

	result, err := e.SaveGame(&state, "test-slot")
	require.NoError(t, err)
	assert.Equal(t, "test-slot", result.Slot)

	// Load it back.
	loaded, loadResult, err := e.LoadGame("test-slot")
	require.NoError(t, err)
	assert.Equal(t, "test-slot", loadResult.Slot)

	// Normalize fields set by save infrastructure before comparing.
	state.SchemaVersion = loaded.SchemaVersion
	state.Timestamp = loaded.Timestamp
	assert.Equal(t, state, loaded)
}

func TestSaveGame_DefaultSlot(t *testing.T) {
	e := saveEngine(t)
	char := model.Character{
		ID: "c1", Name: "Hero", Class: "fighter", Level: 1,
		HP: 20, MaxHP: 20,
		Stats: model.Stats{STR: 10, DEX: 10, CON: 10, INT: 10, WIS: 10, CHA: 10},
	}
	state, err := e.NewGame(char)
	require.NoError(t, err)

	result, err := e.SaveGame(&state, "")
	require.NoError(t, err)
	assert.Equal(t, "quicksave", result.Slot)
}

func TestLoadGame_NotFound(t *testing.T) {
	e := saveEngine(t)
	_, _, err := e.LoadGame("nonexistent")
	require.Error(t, err)
}

func TestSaveGame_InvalidSlot(t *testing.T) {
	e := saveEngine(t)
	char := model.Character{
		ID: "c1", Name: "Hero", Class: "fighter", Level: 1,
		HP: 20, MaxHP: 20,
		Stats: model.Stats{STR: 10, DEX: 10, CON: 10, INT: 10, WIS: 10, CHA: 10},
	}
	state, err := e.NewGame(char)
	require.NoError(t, err)

	for _, slot := range []string{"../escape", "a/b", "slot name", "slot.json", "../../etc/passwd"} {
		t.Run(slot, func(t *testing.T) {
			_, err := e.SaveGame(&state, slot)
			require.Error(t, err)
			var slotErr *engine.InvalidSlotError
			assert.ErrorAs(t, err, &slotErr)
		})
	}
}

func TestLoadGame_InvalidSlot(t *testing.T) {
	e := saveEngine(t)
	_, _, err := e.LoadGame("../escape")
	require.Error(t, err)
	var slotErr *engine.InvalidSlotError
	assert.ErrorAs(t, err, &slotErr)
}

func TestLoadGame_ScenarioMismatch(t *testing.T) {
	// Save with one scenario.
	e := saveEngine(t)
	char := model.Character{
		ID: "c1", Name: "Hero", Class: "fighter", Level: 1,
		HP: 20, MaxHP: 20,
		Stats: model.Stats{STR: 10, DEX: 10, CON: 10, INT: 10, WIS: 10, CHA: 10},
	}
	state, err := e.NewGame(char)
	require.NoError(t, err)
	_, err = e.SaveGame(&state, "mismatch")
	require.NoError(t, err)

	// Load with a different scenario.
	otherScenario := saveScenario()
	otherScenario.ID = "different-scenario"
	e2 := engine.New(otherScenario)
	e2.SaveDir = e.SaveDir

	_, _, err = e2.LoadGame("mismatch")
	require.Error(t, err)
	var mismatchErr *engine.ScenarioMismatchError
	assert.ErrorAs(t, err, &mismatchErr)
}

func TestSaveGame_ValidSlotNames(t *testing.T) {
	e := saveEngine(t)
	char := model.Character{
		ID: "c1", Name: "Hero", Class: "fighter", Level: 1,
		HP: 20, MaxHP: 20,
		Stats: model.Stats{STR: 10, DEX: 10, CON: 10, INT: 10, WIS: 10, CHA: 10},
	}
	state, err := e.NewGame(char)
	require.NoError(t, err)

	for _, slot := range []string{"slot1", "my-save", "SAVE_01", "a", "abc123"} {
		t.Run(slot, func(t *testing.T) {
			result, err := e.SaveGame(&state, slot)
			require.NoError(t, err)
			assert.Equal(t, slot, result.Slot)
		})
	}
}
