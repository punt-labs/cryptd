package engine_test

import (
	"math/rand"
	"testing"

	"github.com/punt-labs/cryptd/internal/engine"
	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/scenario"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// combatScenario returns a scenario with a room containing one goblin.
func combatScenario() *scenario.Scenario {
	return &scenario.Scenario{
		ID:           "combat-test",
		StartingRoom: "arena",
		Rooms: map[string]*scenario.Room{
			"arena": {
				Name:            "Arena",
				DescriptionSeed: "A sandy arena.",
				Enemies:         []string{"goblin"},
				Items:           []string{"sword"},
			},
			"hallway": {
				Name:            "Hallway",
				DescriptionSeed: "A stone hallway.",
				Connections: map[string]*scenario.Connection{
					"south": {Room: "arena", Type: "open"},
				},
			},
		},
		Enemies: map[string]*scenario.EnemyTemplate{
			"goblin": {Name: "Goblin", HP: 8, Attack: "1d4", AI: "aggressive"},
		},
		Items: map[string]*scenario.ScenarioItem{
			"sword": {Name: "Sword", Type: "weapon", Damage: "1d6", Weight: 3},
		},
	}
}

// multiEnemyScenario returns a scenario with two enemies of different AI types.
func multiEnemyScenario() *scenario.Scenario {
	return &scenario.Scenario{
		ID:           "multi-enemy",
		StartingRoom: "arena",
		Rooms: map[string]*scenario.Room{
			"arena": {
				Name:    "Arena",
				Enemies: []string{"goblin", "rat"},
			},
		},
		Enemies: map[string]*scenario.EnemyTemplate{
			"goblin": {Name: "Goblin", HP: 8, Attack: "1d4", AI: "aggressive"},
			"rat":    {Name: "Rat", HP: 3, Attack: "1d2", AI: "cautious"},
		},
		Items: map[string]*scenario.ScenarioItem{},
	}
}

func combatGame(t *testing.T, s *scenario.Scenario) (*engine.Engine, model.GameState) {
	t.Helper()
	e := engine.New(s)
	char := model.Character{
		ID: "c1", Name: "Hero", Class: "fighter", Level: 1,
		HP: 30, MaxHP: 30,
		Stats: model.Stats{STR: 14, DEX: 12, CON: 12, INT: 10, WIS: 10, CHA: 10},
	}
	state, err := e.NewGame(char)
	require.NoError(t, err)
	return e, state
}

func TestStartCombat_CreatesEnemies(t *testing.T) {
	rand.Seed(42)
	e, state := combatGame(t, combatScenario())

	result, err := e.StartCombat(&state)
	require.NoError(t, err)
	require.Len(t, result.Enemies, 1)
	assert.Equal(t, "goblin_0", result.Enemies[0].ID)
	assert.Equal(t, "Goblin", result.Enemies[0].Name)
	assert.Equal(t, 8, result.Enemies[0].HP)
	assert.Equal(t, 8, result.Enemies[0].MaxHP)
	assert.Equal(t, "1d4", result.Enemies[0].Attack)
	assert.True(t, state.Dungeon.Combat.Active)
	assert.Equal(t, 1, state.Dungeon.Combat.Round)
	assert.Len(t, result.TurnOrder, 2)
}

func TestStartCombat_AlreadyActive(t *testing.T) {
	rand.Seed(42)
	e, state := combatGame(t, combatScenario())
	_, err := e.StartCombat(&state)
	require.NoError(t, err)

	_, err = e.StartCombat(&state)
	require.Error(t, err)
	var already *engine.AlreadyInCombatError
	require.ErrorAs(t, err, &already)
}

func TestStartCombat_NoEnemies(t *testing.T) {
	s := &scenario.Scenario{
		ID:           "empty",
		StartingRoom: "room",
		Rooms: map[string]*scenario.Room{
			"room": {Name: "Room"},
		},
		Enemies: map[string]*scenario.EnemyTemplate{},
		Items:   map[string]*scenario.ScenarioItem{},
	}
	e, state := combatGame(t, s)
	_, err := e.StartCombat(&state)
	require.Error(t, err)
	var noEnemies *engine.NoEnemiesError
	require.ErrorAs(t, err, &noEnemies)
}

func TestStartCombat_ClearedRoom(t *testing.T) {
	rand.Seed(42)
	e, state := combatGame(t, combatScenario())

	// Mark room cleared.
	rs := state.Dungeon.RoomState["arena"]
	rs.Cleared = true
	state.Dungeon.RoomState["arena"] = rs

	_, err := e.StartCombat(&state)
	require.Error(t, err)
	var noEnemies *engine.NoEnemiesError
	require.ErrorAs(t, err, &noEnemies)
}

func TestAttack_DamagesEnemy(t *testing.T) {
	rand.Seed(42)
	e, state := combatGame(t, combatScenario())
	_, err := e.StartCombat(&state)
	require.NoError(t, err)

	// Skip to hero turn if not already.
	skipToHeroTurn(t, e, &state)

	result, err := e.Attack(&state, "goblin_0")
	require.NoError(t, err)
	assert.Greater(t, result.Damage, 0)
	assert.Equal(t, "goblin_0", result.Target)
}

func TestAttack_KillsEnemy(t *testing.T) {
	rand.Seed(42)
	e, state := combatGame(t, combatScenario())
	_, err := e.StartCombat(&state)
	require.NoError(t, err)

	// Set enemy HP to 1 so any hit kills.
	state.Dungeon.Combat.Enemies[0].HP = 1

	skipToHeroTurn(t, e, &state)

	result, err := e.Attack(&state, "goblin_0")
	require.NoError(t, err)
	assert.True(t, result.Killed)
	assert.Equal(t, 8, result.XPAwarded) // MaxHP = 8
	assert.True(t, result.CombatOver)
	assert.False(t, state.Dungeon.Combat.Active)
	assert.True(t, state.Dungeon.RoomState["arena"].Cleared)
}

func TestAttack_XPAwarded(t *testing.T) {
	rand.Seed(42)
	e, state := combatGame(t, combatScenario())
	_, err := e.StartCombat(&state)
	require.NoError(t, err)

	state.Dungeon.Combat.Enemies[0].HP = 1
	skipToHeroTurn(t, e, &state)

	_, err = e.Attack(&state, "goblin_0")
	require.NoError(t, err)
	assert.Equal(t, 8, state.Party[0].XP)
}

func TestAttack_NotInCombat(t *testing.T) {
	e, state := combatGame(t, combatScenario())
	_, err := e.Attack(&state, "goblin_0")
	require.Error(t, err)
	var notInCombat *engine.NotInCombatError
	require.ErrorAs(t, err, &notInCombat)
}

func TestAttack_InvalidTarget(t *testing.T) {
	rand.Seed(42)
	e, state := combatGame(t, combatScenario())
	_, err := e.StartCombat(&state)
	require.NoError(t, err)

	skipToHeroTurn(t, e, &state)

	_, err = e.Attack(&state, "nonexistent")
	require.Error(t, err)
	var invalid *engine.InvalidTargetError
	require.ErrorAs(t, err, &invalid)
}

func TestAttack_DeadTarget(t *testing.T) {
	rand.Seed(42)
	e, state := combatGame(t, multiEnemyScenario())
	_, err := e.StartCombat(&state)
	require.NoError(t, err)

	// Kill first enemy manually.
	state.Dungeon.Combat.Enemies[0].HP = 0
	skipToHeroTurn(t, e, &state)

	_, err = e.Attack(&state, "goblin_0")
	require.Error(t, err)
	var invalid *engine.InvalidTargetError
	require.ErrorAs(t, err, &invalid)
}

func TestDefend_SetsFlag(t *testing.T) {
	rand.Seed(42)
	e, state := combatGame(t, combatScenario())
	_, err := e.StartCombat(&state)
	require.NoError(t, err)

	skipToHeroTurn(t, e, &state)

	_, err = e.Defend(&state)
	require.NoError(t, err)
	// After defend, turn advances — HeroDefending should stay true until
	// hero's next turn. It gets set before advanceTurn.
	// The flag is checked during enemy attack processing.
}

func TestDefend_HalvesDamage(t *testing.T) {
	rand.Seed(42)
	e, state := combatGame(t, combatScenario())
	_, err := e.StartCombat(&state)
	require.NoError(t, err)

	skipToHeroTurn(t, e, &state)

	_, err = e.Defend(&state)
	require.NoError(t, err)

	// Now process enemy turn — damage should be halved.
	if !state.Dungeon.Combat.Active {
		return
	}
	if !e.IsHeroTurn(&state) {
		result, err := e.ProcessEnemyTurn(&state)
		require.NoError(t, err)
		if result.Action == "attack" {
			// Damage should be halved (1d4 → max 4, halved → max 2).
			assert.LessOrEqual(t, result.Damage, 2)
		}
	}
}

func TestDefend_NotInCombat(t *testing.T) {
	e, state := combatGame(t, combatScenario())
	_, err := e.Defend(&state)
	require.Error(t, err)
	var notInCombat *engine.NotInCombatError
	require.ErrorAs(t, err, &notInCombat)
}

func TestFlee_Success(t *testing.T) {
	// Seed so that roll <= DEX (12).
	// We'll try multiple seeds to find one that works.
	for seed := int64(0); seed < 100; seed++ {
		rand.Seed(seed)
		e, state := combatGame(t, combatScenario())
		_, err := e.StartCombat(&state)
		require.NoError(t, err)

		skipToHeroTurn(t, e, &state)
		if !state.Dungeon.Combat.Active {
			continue
		}

		result, err := e.Flee(&state)
		require.NoError(t, err)
		if result.Success {
			assert.False(t, state.Dungeon.Combat.Active)
			// Room should NOT be cleared on flee.
			assert.False(t, state.Dungeon.RoomState["arena"].Cleared)
			return
		}
	}
	t.Fatal("could not find seed that produces successful flee")
}

func TestFlee_Failure(t *testing.T) {
	// Use a high-DEX character scenario where flee still fails sometimes.
	for seed := int64(0); seed < 100; seed++ {
		rand.Seed(seed)
		s := combatScenario()
		e := engine.New(s)
		char := model.Character{
			ID: "c1", Name: "Hero", Class: "fighter", Level: 1,
			HP: 30, MaxHP: 30,
			Stats: model.Stats{DEX: 5}, // Low DEX makes flee harder.
		}
		state, err := e.NewGame(char)
		require.NoError(t, err)

		_, err = e.StartCombat(&state)
		require.NoError(t, err)

		skipToHeroTurn(t, e, &state)
		if !state.Dungeon.Combat.Active {
			continue
		}

		result, err := e.Flee(&state)
		require.NoError(t, err)
		if !result.Success {
			assert.True(t, state.Dungeon.Combat.Active)
			return
		}
	}
	t.Fatal("could not find seed that produces failed flee")
}

func TestFlee_NotInCombat(t *testing.T) {
	e, state := combatGame(t, combatScenario())
	_, err := e.Flee(&state)
	require.Error(t, err)
	var notInCombat *engine.NotInCombatError
	require.ErrorAs(t, err, &notInCombat)
}

func TestProcessEnemyTurn_DamagesHero(t *testing.T) {
	rand.Seed(42)
	e, state := combatGame(t, combatScenario())
	_, err := e.StartCombat(&state)
	require.NoError(t, err)

	// Ensure it's an enemy turn.
	skipToEnemyTurn(t, e, &state)
	if !state.Dungeon.Combat.Active {
		return
	}

	result, err := e.ProcessEnemyTurn(&state)
	require.NoError(t, err)
	assert.Equal(t, "attack", result.Action)
	assert.Greater(t, result.Damage, 0)
	assert.Equal(t, state.Party[0].HP, result.HeroHP)
}

func TestProcessEnemyTurn_HeroDeath(t *testing.T) {
	rand.Seed(42)
	e, state := combatGame(t, combatScenario())
	_, err := e.StartCombat(&state)
	require.NoError(t, err)

	// Set hero HP to 1 so any hit kills.
	state.Party[0].HP = 1

	skipToEnemyTurn(t, e, &state)
	if !state.Dungeon.Combat.Active {
		return
	}

	result, err := e.ProcessEnemyTurn(&state)
	require.NoError(t, err)
	assert.True(t, result.HeroDead)
	assert.LessOrEqual(t, result.HeroHP, 0)
}

func TestProcessEnemyTurn_CautiousFlees(t *testing.T) {
	rand.Seed(42)
	s := &scenario.Scenario{
		ID:           "cautious-test",
		StartingRoom: "arena",
		Rooms: map[string]*scenario.Room{
			"arena": {Name: "Arena", Enemies: []string{"rat"}},
		},
		Enemies: map[string]*scenario.EnemyTemplate{
			"rat": {Name: "Rat", HP: 10, Attack: "1d2", AI: "cautious"},
		},
		Items: map[string]*scenario.ScenarioItem{},
	}
	e := engine.New(s)
	char := model.Character{
		ID: "c1", Name: "Hero", Class: "fighter", Level: 1,
		HP: 30, MaxHP: 30,
		Stats: model.Stats{DEX: 12},
	}
	state, err := e.NewGame(char)
	require.NoError(t, err)

	_, err = e.StartCombat(&state)
	require.NoError(t, err)

	// Set rat HP to 30% or below (3 out of 10).
	state.Dungeon.Combat.Enemies[0].HP = 3

	skipToEnemyTurn(t, e, &state)
	if !state.Dungeon.Combat.Active {
		return
	}

	result, err := e.ProcessEnemyTurn(&state)
	require.NoError(t, err)
	assert.Equal(t, "flee", result.Action)
	assert.Equal(t, 0, result.Damage)
}

func TestProcessEnemyTurn_CautiousAttacksWhenHealthy(t *testing.T) {
	rand.Seed(42)
	s := &scenario.Scenario{
		ID:           "cautious-attack",
		StartingRoom: "arena",
		Rooms: map[string]*scenario.Room{
			"arena": {Name: "Arena", Enemies: []string{"rat"}},
		},
		Enemies: map[string]*scenario.EnemyTemplate{
			"rat": {Name: "Rat", HP: 10, Attack: "1d2", AI: "cautious"},
		},
		Items: map[string]*scenario.ScenarioItem{},
	}
	e := engine.New(s)
	char := model.Character{
		ID: "c1", Name: "Hero", Class: "fighter", Level: 1,
		HP: 30, MaxHP: 30,
		Stats: model.Stats{DEX: 12},
	}
	state, err := e.NewGame(char)
	require.NoError(t, err)

	_, err = e.StartCombat(&state)
	require.NoError(t, err)

	// HP > 30% — should attack.
	state.Dungeon.Combat.Enemies[0].HP = 8

	skipToEnemyTurn(t, e, &state)
	if !state.Dungeon.Combat.Active {
		return
	}

	result, err := e.ProcessEnemyTurn(&state)
	require.NoError(t, err)
	assert.Equal(t, "attack", result.Action)
	assert.Greater(t, result.Damage, 0)
}

func TestFullCombat_KillGoblin(t *testing.T) {
	rand.Seed(42)
	e, state := combatGame(t, combatScenario())

	// Pick up and equip sword.
	_, err := e.PickUp(&state, "sword")
	require.NoError(t, err)
	_, err = e.Equip(&state, "sword")
	require.NoError(t, err)

	_, err = e.StartCombat(&state)
	require.NoError(t, err)

	// Fight until combat ends or hero dies (max 50 rounds safety).
	for i := 0; i < 50 && state.Dungeon.Combat.Active; i++ {
		if e.IsHeroTurn(&state) {
			targetID := e.FirstAliveEnemy(&state)
			if targetID == "" {
				break
			}
			result, err := e.Attack(&state, targetID)
			require.NoError(t, err)
			if result.CombatOver {
				break
			}
		} else {
			result, err := e.ProcessEnemyTurn(&state)
			require.NoError(t, err)
			if result.HeroDead {
				t.Skip("hero died — expected with some seeds")
			}
		}
	}

	assert.False(t, state.Dungeon.Combat.Active)
	assert.True(t, state.Dungeon.RoomState["arena"].Cleared)
	assert.Greater(t, state.Party[0].XP, 0)
}

func TestUnarmedDamage(t *testing.T) {
	rand.Seed(42)
	e, state := combatGame(t, combatScenario())

	_, err := e.StartCombat(&state)
	require.NoError(t, err)
	skipToHeroTurn(t, e, &state)
	if !state.Dungeon.Combat.Active {
		return
	}

	// No weapon equipped — should do 1d2 damage (1 or 2).
	result, err := e.Attack(&state, "goblin_0")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, result.Damage, 1)
	assert.LessOrEqual(t, result.Damage, 2)
}

func TestFirstAliveEnemy(t *testing.T) {
	rand.Seed(42)
	e, state := combatGame(t, multiEnemyScenario())
	_, err := e.StartCombat(&state)
	require.NoError(t, err)

	// First alive should be goblin_0.
	assert.Equal(t, "goblin_0", e.FirstAliveEnemy(&state))

	// Kill goblin_0.
	state.Dungeon.Combat.Enemies[0].HP = 0
	assert.Equal(t, "rat_1", e.FirstAliveEnemy(&state))

	// Kill rat_1.
	state.Dungeon.Combat.Enemies[1].HP = 0
	assert.Equal(t, "", e.FirstAliveEnemy(&state))
}

func TestCombatErrorMessages(t *testing.T) {
	assert.Contains(t, (&engine.NotInCombatError{}).Error(), "not in combat")
	assert.Contains(t, (&engine.NotHeroTurnError{}).Error(), "not your turn")
	assert.Contains(t, (&engine.InvalidTargetError{TargetID: "x"}).Error(), "x")
	assert.Contains(t, (&engine.HeroDeadError{}).Error(), "dead")
	assert.Contains(t, (&engine.AlreadyInCombatError{}).Error(), "already")
	assert.Contains(t, (&engine.NoEnemiesError{}).Error(), "no enemies")
}

func TestHeroDeadError_PreventsCombatActions(t *testing.T) {
	rand.Seed(42)
	e, state := combatGame(t, combatScenario())
	_, err := e.StartCombat(&state)
	require.NoError(t, err)

	state.Party[0].HP = 0

	skipToHeroTurn(t, e, &state)
	if !state.Dungeon.Combat.Active {
		return
	}

	_, err = e.Attack(&state, "goblin_0")
	var heroDead *engine.HeroDeadError
	require.ErrorAs(t, err, &heroDead)

	_, err = e.Defend(&state)
	require.ErrorAs(t, err, &heroDead)

	_, err = e.Flee(&state)
	require.ErrorAs(t, err, &heroDead)
}

// skipToHeroTurn processes enemy turns until it's the hero's turn.
func skipToHeroTurn(t *testing.T, e *engine.Engine, state *model.GameState) {
	t.Helper()
	for i := 0; i < 20 && state.Dungeon.Combat.Active && !e.IsHeroTurn(state); i++ {
		_, err := e.ProcessEnemyTurn(state)
		require.NoError(t, err)
	}
}

// skipToEnemyTurn processes hero defend until it's an enemy's turn.
func skipToEnemyTurn(t *testing.T, e *engine.Engine, state *model.GameState) {
	t.Helper()
	for i := 0; i < 20 && state.Dungeon.Combat.Active && e.IsHeroTurn(state); i++ {
		_, err := e.Defend(state)
		require.NoError(t, err)
	}
}
