package engine_test

import (
	"testing"

	"github.com/punt-labs/cryptd/internal/engine"
	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/scenario"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func spellScenario() *scenario.Scenario {
	return &scenario.Scenario{
		ID:           "spell-test",
		StartingRoom: "arena",
		Rooms: map[string]*scenario.Room{
			"arena": {
				Name:    "Arena",
				Enemies: []string{"goblin"},
			},
		},
		Enemies: map[string]*scenario.EnemyTemplate{
			"goblin": {Name: "Goblin", HP: 8, Attack: "1d4", AI: "aggressive"},
		},
		Spells: map[string]*scenario.SpellTemplate{
			"fireball": {Name: "Fireball", MP: 3, Effect: "damage", Power: "2d6", Classes: []string{"mage", "priest"}},
			"heal":     {Name: "Heal", MP: 2, Effect: "heal", Power: "1d6+2", Classes: []string{"priest", "mage"}},
		},
		Items: map[string]*scenario.ScenarioItem{},
	}
}

func mageGame(t *testing.T) (*engine.Engine, model.GameState) {
	t.Helper()
	s := spellScenario()
	e := engine.New(s)
	char := model.Character{
		ID: "c1", Name: "Elara", Class: "mage", Level: 1,
		HP: 20, MaxHP: 20, MP: 10, MaxMP: 10,
		Stats: model.Stats{STR: 8, DEX: 10, CON: 10, INT: 16, WIS: 12, CHA: 10},
	}
	state, err := e.NewGame(char)
	require.NoError(t, err)
	return e, state
}

func TestCastSpell_DamageInCombat(t *testing.T) {
	e, state := mageGame(t)
	_, err := e.StartCombat(&state)
	require.NoError(t, err)

	skipToHeroTurn(t, e, &state)
	require.True(t, state.Dungeon.Combat.Active)

	result, err := e.CastSpell(&state, "fireball", "goblin_0")
	require.NoError(t, err)
	assert.Equal(t, "Fireball", result.SpellName)
	assert.Equal(t, "damage", result.Effect)
	assert.Equal(t, 3, result.MPCost)
	assert.Greater(t, result.Power, 0)
	assert.Equal(t, 7, state.Party[0].MP) // 10 - 3
}

func TestCastSpell_HealOutOfCombat(t *testing.T) {
	e, state := mageGame(t)

	// Damage the hero first.
	state.Party[0].HP = 10

	result, err := e.CastSpell(&state, "heal", "")
	require.NoError(t, err)
	assert.Equal(t, "Heal", result.SpellName)
	assert.Equal(t, "heal", result.Effect)
	assert.Greater(t, result.Power, 0)
	assert.Greater(t, state.Party[0].HP, 10)
	assert.Equal(t, 8, state.Party[0].MP) // 10 - 2
}

func TestCastSpell_HealCapsAtMax(t *testing.T) {
	e, state := mageGame(t)

	// Hero at full HP — heal should not exceed MaxHP.
	result, err := e.CastSpell(&state, "heal", "")
	require.NoError(t, err)
	assert.Equal(t, state.Party[0].MaxHP, state.Party[0].HP)
	assert.Greater(t, result.Power, 0)
}

func TestCastSpell_ClassGate_FighterCantCast(t *testing.T) {
	s := spellScenario()
	e := engine.New(s)
	char := model.Character{
		ID: "c1", Name: "Conan", Class: "fighter", Level: 1,
		HP: 30, MaxHP: 30, MP: 0, MaxMP: 0,
		Stats: model.Stats{STR: 16, DEX: 12, CON: 14, INT: 8, WIS: 8, CHA: 10},
	}
	state, err := e.NewGame(char)
	require.NoError(t, err)

	_, err = e.CastSpell(&state, "fireball", "")
	require.Error(t, err)
	var notCaster *engine.NotCasterError
	require.ErrorAs(t, err, &notCaster)
	assert.Contains(t, notCaster.Error(), "fighter")
}

func TestCastSpell_InsufficientMP(t *testing.T) {
	e, state := mageGame(t)
	state.Party[0].MP = 1 // Fireball costs 3

	_, err := e.CastSpell(&state, "fireball", "")
	require.Error(t, err)
	var insuffMP *engine.InsufficientMPError
	require.ErrorAs(t, err, &insuffMP)
	assert.Equal(t, 1, insuffMP.Have)
	assert.Equal(t, 3, insuffMP.Need)
}

func TestCastSpell_UnknownSpell(t *testing.T) {
	e, state := mageGame(t)

	_, err := e.CastSpell(&state, "lightning", "")
	require.Error(t, err)
	var unknown *engine.UnknownSpellError
	require.ErrorAs(t, err, &unknown)
}

func TestCastSpell_DamageRequiresCombat(t *testing.T) {
	e, state := mageGame(t)
	// Not in combat — damage spell should fail.
	_, err := e.CastSpell(&state, "fireball", "")
	require.Error(t, err)
	var notInCombat *engine.NotInCombatError
	require.ErrorAs(t, err, &notInCombat)
}

func TestCastSpell_DamageKillsEnemy(t *testing.T) {
	e, state := mageGame(t)
	_, err := e.StartCombat(&state)
	require.NoError(t, err)

	// Set enemy to 1 HP for guaranteed kill.
	state.Dungeon.Combat.Enemies[0].HP = 1

	skipToHeroTurn(t, e, &state)

	result, err := e.CastSpell(&state, "fireball", "goblin_0")
	require.NoError(t, err)
	assert.Equal(t, "damage", result.Effect)
	assert.False(t, state.Dungeon.Combat.Active)
	assert.True(t, state.Dungeon.RoomState["arena"].Cleared)
	assert.Equal(t, 8, state.Party[0].XP) // goblin MaxHP = 8
}

func TestCastSpell_HealInCombatConsumesTurn(t *testing.T) {
	e, state := mageGame(t)
	_, err := e.StartCombat(&state)
	require.NoError(t, err)

	skipToHeroTurn(t, e, &state)
	require.True(t, e.IsHeroTurn(&state))

	_, err = e.CastSpell(&state, "heal", "")
	require.NoError(t, err)

	// Turn should have advanced past the hero.
	assert.False(t, e.IsHeroTurn(&state))
}

func TestCastSpell_DeadHeroCannotCast(t *testing.T) {
	e, state := mageGame(t)
	state.Party[0].HP = 0

	_, err := e.CastSpell(&state, "heal", "")
	require.Error(t, err)
	var heroDead *engine.HeroDeadError
	require.ErrorAs(t, err, &heroDead)
	// MP must not have been deducted.
	assert.Equal(t, 10, state.Party[0].MP)
}

func TestCastSpell_ErrorMessages(t *testing.T) {
	assert.Contains(t, (&engine.NotCasterError{Class: "thief"}).Error(), "thief")
	assert.Contains(t, (&engine.InsufficientMPError{Have: 1, Need: 5}).Error(), "1")
	assert.Contains(t, (&engine.InsufficientMPError{Have: 1, Need: 5}).Error(), "5")
	assert.Contains(t, (&engine.UnknownSpellError{SpellID: "zap"}).Error(), "zap")
}
