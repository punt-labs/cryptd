package monkeytest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testScenarioDir = "../../testdata/scenarios"

func TestRunner_MinimalScenario(t *testing.T) {
	cfg := RunConfig{
		Scenario: "minimal",
		Class:    "fighter",
		Players:  10,
		MaxMoves: 50,
		Workers:  1, // deterministic
		Seed:     42,
	}
	report, err := Run(cfg, testScenarioDir, "minimal")
	require.NoError(t, err)

	assert.Equal(t, 10, report.Aggregate.Count)
	// All sessions should have executed.
	assert.Len(t, report.Sessions, 10)

	for _, s := range report.Sessions {
		assert.Greater(t, s.TotalMoves, 0, "each session should have at least 1 move")
		assert.GreaterOrEqual(t, s.RoomsVisited, 1, "should visit at least the starting room")
		assert.Equal(t, "fighter", s.Class)
	}

	// At least some sessions should have killed enemies (goblins in goblin_lair).
	totalKills := 0
	for _, s := range report.Sessions {
		totalKills += s.EnemiesKilled
	}
	assert.Greater(t, totalKills, 0, "some sessions should kill enemies")
}

func TestRunner_AllClasses(t *testing.T) {
	cfg := RunConfig{
		Scenario: "minimal",
		Class:    "all",
		Players:  8,
		MaxMoves: 50,
		Workers:  1,
		Seed:     99,
	}
	report, err := Run(cfg, testScenarioDir, "minimal")
	require.NoError(t, err)

	// Should have per-class breakdown.
	require.NotNil(t, report.PerClass)
	assert.Len(t, report.PerClass, 4) // fighter, mage, priest, thief

	// Each class should have 2 sessions (8 players / 4 classes).
	for _, cls := range AllClasses {
		agg, ok := report.PerClass[cls]
		require.True(t, ok, "missing class %s", cls)
		assert.Equal(t, 2, agg.Count)
	}
}

func TestRunner_MaxMovesRespected(t *testing.T) {
	cfg := RunConfig{
		Scenario: "minimal",
		Class:    "fighter",
		Players:  5,
		MaxMoves: 10,
		Workers:  1,
		Seed:     7,
	}
	report, err := Run(cfg, testScenarioDir, "minimal")
	require.NoError(t, err)

	for _, s := range report.Sessions {
		assert.LessOrEqual(t, s.TotalMoves, 10, "session should not exceed max moves")
	}
}

func TestRunner_ParallelWorkers(t *testing.T) {
	cfg := RunConfig{
		Scenario: "minimal",
		Class:    "fighter",
		Players:  20,
		MaxMoves: 30,
		Workers:  4,
		Seed:     42,
	}
	report, err := Run(cfg, testScenarioDir, "minimal")
	require.NoError(t, err)

	assert.Equal(t, 20, report.Aggregate.Count)
	// Verify all sessions completed (no zeros from uninitialized slots).
	for i, s := range report.Sessions {
		assert.Greater(t, s.TotalMoves, 0, "session %d should have moves", i)
	}
}

func TestCompute_Metrics(t *testing.T) {
	sessions := []SessionMetrics{
		{Survived: true, TotalMoves: 10, XPGained: 8, RoomsVisited: 3, EnemiesKilled: 1, FinalLevel: 1},
		{Survived: true, TotalMoves: 20, XPGained: 16, RoomsVisited: 4, EnemiesKilled: 2, FinalLevel: 1},
		{Survived: false, TotalMoves: 5, XPGained: 0, RoomsVisited: 2, EnemiesKilled: 0, FinalLevel: 1},
	}

	agg := Compute(sessions)
	assert.Equal(t, 3, agg.Count)
	assert.InDelta(t, 0.667, agg.SurvivalRate, 0.01)
	assert.InDelta(t, 11.67, agg.MeanMoves, 0.1)
	assert.InDelta(t, 8.0, agg.MeanXP, 0.1)
	assert.Equal(t, 10, agg.P50Moves)
}

func TestCompute_Empty(t *testing.T) {
	agg := Compute(nil)
	assert.Equal(t, 0, agg.Count)
}
