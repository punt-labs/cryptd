package engine_test

import (
	"testing"

	"github.com/punt-labs/cryptd/internal/engine"
	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/scenario"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func levelScenario() *scenario.Scenario {
	return &scenario.Scenario{
		ID:           "level-test",
		StartingRoom: "arena",
		Rooms: map[string]*scenario.Room{
			"arena": {Name: "Arena"},
		},
		Enemies: map[string]*scenario.EnemyTemplate{},
		Items:   map[string]*scenario.ScenarioItem{},
		Spells:  map[string]*scenario.SpellTemplate{},
	}
}

func newCharAt(class string, level, xp int) model.Character {
	return model.Character{
		ID: "c1", Name: "Hero", Class: class, Level: level,
		HP: 20, MaxHP: 20, MP: 10, MaxMP: 10, XP: xp,
		Stats: model.Stats{STR: 10, INT: 10, DEX: 10, CON: 10, WIS: 10, CHA: 10},
	}
}

func TestCheckLevelUp_FighterLevels(t *testing.T) {
	e := engine.New(levelScenario())
	// Fighter needs 20 XP for level 2.
	char := newCharAt("fighter", 1, 20)
	state, err := e.NewGame(char)
	require.NoError(t, err)

	result := e.CheckLevelUp(&state)
	assert.True(t, result.Leveled)
	assert.Equal(t, 2, result.NewLevel)
	assert.Equal(t, 2, state.Party[0].Level)
	assert.Equal(t, 8, result.HPGain)
	assert.Equal(t, 28, state.Party[0].MaxHP)
	assert.Equal(t, 28, state.Party[0].HP)
	assert.Equal(t, 0, result.MPGain)
	assert.Equal(t, 1, result.StatGain["STR"])
	assert.Equal(t, 1, result.StatGain["CON"])
	assert.Equal(t, 11, state.Party[0].Stats.STR)
	assert.Equal(t, 11, state.Party[0].Stats.CON)
}

func TestCheckLevelUp_MageLevels(t *testing.T) {
	e := engine.New(levelScenario())
	// Mage needs 30 XP for level 2.
	char := newCharAt("mage", 1, 30)
	state, err := e.NewGame(char)
	require.NoError(t, err)

	result := e.CheckLevelUp(&state)
	assert.True(t, result.Leveled)
	assert.Equal(t, 2, result.NewLevel)
	assert.Equal(t, 4, result.HPGain)
	assert.Equal(t, 4, result.MPGain)
	assert.Equal(t, 14, state.Party[0].MaxMP)
	assert.Equal(t, 14, state.Party[0].MP)
	assert.Equal(t, 1, result.StatGain["INT"])
	assert.Equal(t, 1, result.StatGain["WIS"])
}

func TestCheckLevelUp_NoLevelIfXPInsufficient(t *testing.T) {
	e := engine.New(levelScenario())
	char := newCharAt("fighter", 1, 19) // needs 20 for level 2
	state, err := e.NewGame(char)
	require.NoError(t, err)

	result := e.CheckLevelUp(&state)
	assert.False(t, result.Leveled)
	assert.Equal(t, 1, state.Party[0].Level)
}

func TestCheckLevelUp_MultiLevel(t *testing.T) {
	e := engine.New(levelScenario())
	// Fighter with 100 XP should reach level 4 (thresholds: 20, 50, 100).
	char := newCharAt("fighter", 1, 100)
	state, err := e.NewGame(char)
	require.NoError(t, err)

	result := e.CheckLevelUp(&state)
	assert.True(t, result.Leveled)
	assert.Equal(t, 4, result.NewLevel)
	assert.Equal(t, 4, state.Party[0].Level)
	// 3 levels × 8 HP = 24 HP gained
	assert.Equal(t, 24, result.HPGain)
	assert.Equal(t, 44, state.Party[0].MaxHP)
	// 3 levels × +1 STR = +3 STR
	assert.Equal(t, 3, result.StatGain["STR"])
	assert.Equal(t, 13, state.Party[0].Stats.STR)
}

func TestCheckLevelUp_CapsAtMaxLevel(t *testing.T) {
	e := engine.New(levelScenario())
	// Level 10 fighter with massive XP should not level past 10.
	char := newCharAt("fighter", 10, 99999)
	state, err := e.NewGame(char)
	require.NoError(t, err)

	result := e.CheckLevelUp(&state)
	assert.False(t, result.Leveled)
	assert.Equal(t, 10, state.Party[0].Level)
}

func TestCheckLevelUp_UnknownClassNoOp(t *testing.T) {
	e := engine.New(levelScenario())
	char := newCharAt("bard", 1, 9999)
	state, err := e.NewGame(char)
	require.NoError(t, err)

	result := e.CheckLevelUp(&state)
	assert.False(t, result.Leveled)
}

func TestCheckLevelUp_PriestGainsMP(t *testing.T) {
	e := engine.New(levelScenario())
	char := newCharAt("priest", 1, 25) // priest needs 25 for level 2
	state, err := e.NewGame(char)
	require.NoError(t, err)

	result := e.CheckLevelUp(&state)
	assert.True(t, result.Leveled)
	assert.Equal(t, 6, result.HPGain)
	assert.Equal(t, 3, result.MPGain)
	assert.Equal(t, 1, result.StatGain["WIS"])
	assert.Equal(t, 1, result.StatGain["CHA"])
}

func TestCheckLevelUp_ThiefStatGains(t *testing.T) {
	e := engine.New(levelScenario())
	char := newCharAt("thief", 1, 22) // thief needs 22 for level 2
	state, err := e.NewGame(char)
	require.NoError(t, err)

	result := e.CheckLevelUp(&state)
	assert.True(t, result.Leveled)
	assert.Equal(t, 6, result.HPGain)
	assert.Equal(t, 0, result.MPGain)
	assert.Equal(t, 1, result.StatGain["DEX"])
	assert.Equal(t, 1, result.StatGain["CHA"])
}
