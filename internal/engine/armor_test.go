package engine_test

import (
	"testing"

	"github.com/punt-labs/cryptd/internal/engine"
	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/scenario"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func armorScenario() *scenario.Spec {
	return &scenario.Spec{
		ID:           "armor-test",
		StartingRoom: "arena",
		Rooms: map[string]*scenario.Room{
			"arena": {Name: "Arena", Enemies: []string{"goblin"}, Items: []string{"shield"}},
		},
		Items: map[string]*scenario.Item{
			"shield": {Name: "Shield", Type: "armor", Defense: 3, Weight: 5.0, Value: 15},
		},
		Enemies: map[string]*scenario.EnemyTemplate{
			"goblin": {Name: "Goblin", HP: 8, Attack: "1d4", AI: "aggressive"},
		},
		Spells: map[string]*scenario.SpellTemplate{},
	}
}

func TestArmorReducesDamage(t *testing.T) {
	s := armorScenario()
	e := engine.New(s)
	char := model.Character{
		ID: "hero", Name: "Tank", Class: "fighter", Level: 1,
		HP: 100, MaxHP: 100, // high HP to survive many rounds
		Stats: model.Stats{STR: 14, DEX: 12, CON: 12, INT: 10, WIS: 10, CHA: 10},
	}
	state, err := e.NewGame(char)
	require.NoError(t, err)

	// Pick up and equip armor.
	_, err = e.PickUp(&state, "shield")
	require.NoError(t, err)
	_, err = e.Equip(&state, "shield")
	require.NoError(t, err)
	assert.Equal(t, "shield", state.Party[0].Equipped.Armor)

	// Start combat.
	_, err = e.StartCombat(&state)
	require.NoError(t, err)

	// Process 20 enemy attack rounds to get a statistical sample.
	// With defense 3 and 1d4 attack (range 1-4), damage after reduction:
	// raw 1 → max(1, 1-3) = 1, raw 2 → 1, raw 3 → 1, raw 4 → 1
	// So ALL damage should be exactly 1 with defense 3 against 1d4.
	totalDamage := 0
	startHP := state.Party[0].HP
	for i := 0; i < 20; i++ {
		if !state.Dungeon.Combat.Active {
			break
		}
		if e.IsHeroTurn(&state) {
			// Hero attacks to advance turn.
			_, err := e.Attack(&state, e.FirstAliveEnemy(&state))
			if err != nil {
				break
			}
			continue
		}
		result, err := e.ProcessEnemyTurn(&state)
		if err != nil {
			break
		}
		if result.Action == "attack" {
			totalDamage += result.Damage
			// Every hit should be exactly 1 (defense 3 absorbs all of 1d4).
			assert.Equal(t, 1, result.Damage, "armor defense 3 should reduce 1d4 to minimum 1")
		}
	}

	actualDamage := startHP - state.Party[0].HP
	assert.Equal(t, totalDamage, actualDamage, "tracked damage should match HP loss")
}

func TestArmorPlusDefendStance(t *testing.T) {
	s := armorScenario()
	e := engine.New(s)
	char := model.Character{
		ID: "hero", Name: "Tank", Class: "fighter", Level: 1,
		HP: 100, MaxHP: 100,
		Stats: model.Stats{STR: 14, DEX: 12, CON: 12, INT: 10, WIS: 10, CHA: 10},
	}
	state, err := e.NewGame(char)
	require.NoError(t, err)

	_, err = e.PickUp(&state, "shield")
	require.NoError(t, err)
	_, err = e.Equip(&state, "shield")
	require.NoError(t, err)

	_, err = e.StartCombat(&state)
	require.NoError(t, err)

	// Skip to hero turn and defend.
	for !e.IsHeroTurn(&state) && state.Dungeon.Combat.Active {
		e.ProcessEnemyTurn(&state)
	}
	if state.Dungeon.Combat.Active {
		_, err = e.Defend(&state)
		require.NoError(t, err)

		// Next enemy attack should have defend (halve) + armor (subtract 3).
		// 1d4 raw → halved → then -3 → floor 1. All results should be 1.
		if state.Dungeon.Combat.Active && !e.IsHeroTurn(&state) {
			result, err := e.ProcessEnemyTurn(&state)
			require.NoError(t, err)
			if result.Action == "attack" {
				assert.Equal(t, 1, result.Damage, "defend + armor should reduce all 1d4 to 1")
			}
		}
	}
}

func TestNoArmorNoPenalty(t *testing.T) {
	// Without armor, damage should pass through at full (only defend stance reduces).
	//
	// This is a probabilistic test: with 1d4 (range 1-4), the probability of
	// a single roll being 1 is 1/4. To make all-ones essentially impossible
	// even on a deterministic CI RNG, we use a goblin with enough HP to give
	// us at least 25 enemy turns of damage samples. P(all 25 samples = 1)
	// = (1/4)^25 ≈ 9e-16, statistically impossible.
	s := armorScenario()
	// Override goblin HP for this test only — needs to survive long enough
	// for a statistically meaningful sample of attack rolls.
	s.Enemies["goblin"] = &scenario.EnemyTemplate{
		Name: "Tough Goblin", HP: 200, Attack: "1d4", AI: "aggressive",
	}
	e := engine.New(s)
	char := model.Character{
		ID: "hero", Name: "Naked", Class: "fighter", Level: 1,
		HP: 1000, MaxHP: 1000, // high HP to survive many rounds
		Stats: model.Stats{STR: 14, DEX: 12, CON: 12, INT: 10, WIS: 10, CHA: 10},
	}
	state, err := e.NewGame(char)
	require.NoError(t, err)

	// Don't equip armor.
	_, err = e.StartCombat(&state)
	require.NoError(t, err)

	// Collect a statistically robust sample of enemy damages.
	var damages []int
	for i := 0; i < 200; i++ {
		if !state.Dungeon.Combat.Active {
			break
		}
		if e.IsHeroTurn(&state) {
			_, err := e.Attack(&state, e.FirstAliveEnemy(&state))
			require.NoError(t, err, "iteration %d: hero attack failed unexpectedly", i)
			continue
		}
		result, err := e.ProcessEnemyTurn(&state)
		require.NoError(t, err, "iteration %d: enemy turn failed unexpectedly", i)
		if result.Action == "attack" {
			damages = append(damages, result.Damage)
		}
	}

	// We need at least 25 samples for the probabilistic assertion to be
	// meaningful. If we got fewer (e.g. hero died, combat ended early),
	// the test setup is broken — fail loudly rather than silently skipping.
	require.GreaterOrEqual(t, len(damages), 25,
		"setup error: needed >= 25 enemy damage samples, got %d (%v)", len(damages), damages)

	// Without armor, 1d4 should produce values 1-4. At least one should be > 1.
	// Probability of all 25+ samples being 1 with proper RNG: (1/4)^25 ≈ 9e-16.
	hasAboveOne := false
	for _, d := range damages {
		if d > 1 {
			hasAboveOne = true
			break
		}
	}
	assert.True(t, hasAboveOne,
		"without armor, 1d4 should sometimes deal > 1 damage (got %d samples, all=1: %v)",
		len(damages), damages)
}
