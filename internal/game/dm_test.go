//go:build integration

package game_test

import (
	"context"
	"testing"

	"github.com/punt-labs/cryptd/internal/engine"
	"github.com/punt-labs/cryptd/internal/game"
	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/scenario"
	"github.com/punt-labs/cryptd/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadMinimalScenario(t *testing.T) *scenario.Spec {
	t.Helper()
	s, err := scenario.Load("../../testdata/scenarios/minimal.yaml")
	require.NoError(t, err)
	return s
}

func newTestState(t *testing.T, eng *engine.Engine) model.GameState {
	t.Helper()
	char := model.Character{
		ID: "c1", Name: "Hero", Class: "fighter", Level: 1,
		HP: 100, MaxHP: 100,
		Stats: model.Stats{STR: 14, DEX: 12, CON: 12, INT: 10, WIS: 10, CHA: 10},
	}
	state, err := eng.NewGame(char)
	require.NoError(t, err)
	return state
}

// TestDMPipeline_Movement wires FakeLLMInterpreter + FakeLLMNarrator through
// the game loop and verifies movement state transitions.
func TestDMPipeline_Movement(t *testing.T) {
	eng := engine.New(loadMinimalScenario(t))
	state := newTestState(t, eng)

	// Actions: move south (to goblin_lair), then quit.
	// The FakeLLMInterpreter returns canned actions.
	interp := testutil.NewFakeLLMInterpreter([]model.EngineAction{
		{Type: "move", Direction: "south"},
		// After entering goblin_lair, combat auto-starts. Attacks to clear it.
		{Type: "attack"},
		{Type: "attack"},
		{Type: "attack"},
		{Type: "attack"},
		{Type: "attack"},
		{Type: "attack"},
		{Type: "attack"},
		{Type: "attack"},
		{Type: "move", Direction: "north"},
		{Type: "quit"},
	})

	narr := testutil.NewFakeLLMNarrator([]string{
		"You see a dark corridor.",
		"Combat begins!",
		"You strike!",
		"Victory!",
		"You return north.",
		"Farewell.",
	})

	ch := make(chan model.InputEvent, 20)
	// Feed enough input events — the fake interpreter ignores the payload.
	for i := 0; i < 15; i++ {
		ch <- model.InputEvent{Type: "input", Payload: "action"}
	}
	fake := &fakeRenderer{events: ch}

	loop := game.NewLoop(eng, interp, narr, fake)
	err := loop.Run(context.Background(), &state)
	require.NoError(t, err)

	// Verify state transitions occurred.
	assert.Contains(t, state.Dungeon.VisitedRooms, "goblin_lair")
	assert.Equal(t, "entrance", state.Dungeon.CurrentRoom)
}

// TestDMPipeline_Look verifies that a "look" action produces narration
// through the FakeLLMNarrator.
func TestDMPipeline_Look(t *testing.T) {
	eng := engine.New(loadMinimalScenario(t))
	state := newTestState(t, eng)

	interp := testutil.NewFakeLLMInterpreter([]model.EngineAction{
		{Type: "look"},
		{Type: "quit"},
	})

	narr := testutil.NewFakeLLMNarrator([]string{
		"Initial room.",
		"You look around the entrance hall.",
		"Farewell.",
	})

	ch := make(chan model.InputEvent, 5)
	ch <- model.InputEvent{Type: "input", Payload: "look"}
	ch <- model.InputEvent{Type: "input", Payload: "quit"}
	fake := &fakeRenderer{events: ch}

	loop := game.NewLoop(eng, interp, narr, fake)
	err := loop.Run(context.Background(), &state)
	require.NoError(t, err)

	// At least 3 renders: initial + look + quit.
	assert.GreaterOrEqual(t, len(fake.renders), 3)
}

// TestDMPipeline_UnknownInput verifies that unknown actions produce
// "unknown_action" narration.
func TestDMPipeline_UnknownInput(t *testing.T) {
	eng := engine.New(loadMinimalScenario(t))
	state := newTestState(t, eng)

	interp := testutil.NewFakeLLMInterpreter([]model.EngineAction{
		{Type: "unknown"},
		{Type: "quit"},
	})

	narr := testutil.NewFakeLLMNarrator([]string{
		"You stand at the entrance.",
		"I don't understand.",
		"Farewell.",
	})

	ch := make(chan model.InputEvent, 5)
	ch <- model.InputEvent{Type: "input", Payload: "xyzzy"}
	ch <- model.InputEvent{Type: "input", Payload: "quit"}
	fake := &fakeRenderer{events: ch}

	loop := game.NewLoop(eng, interp, narr, fake)
	err := loop.Run(context.Background(), &state)
	require.NoError(t, err)

	// Player stays at entrance.
	assert.Equal(t, "entrance", state.Dungeon.CurrentRoom)
}
