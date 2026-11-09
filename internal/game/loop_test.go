package game_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/punt-labs/cryptd/internal/engine"
	"github.com/punt-labs/cryptd/internal/game"
	"github.com/punt-labs/cryptd/internal/interpreter"
	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/narrator"
	"github.com/punt-labs/cryptd/internal/save"
	"github.com/punt-labs/cryptd/internal/scenario"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// scriptRenderer feeds a fixed list of inputs to the loop and discards output.
type scriptRenderer struct {
	events chan model.InputEvent
}

func (r *scriptRenderer) Render(_ context.Context, _ model.GameState, _ string) error { return nil }
func (r *scriptRenderer) Events() <-chan model.InputEvent                             { return r.events }

func loopScenario(t *testing.T) *scenario.Spec {
	t.Helper()
	s, err := scenario.Load("../../testdata/scenarios/minimal.yaml")
	require.NoError(t, err)
	return s
}

// TestLoop_LoadRejectsHerolessSave covers the natural-language "load" path
// through Loop.dispatch (the sibling of the daemon load path). A crafted
// heroless save is rejected by engine.LoadGame, so the loop keeps the current
// hero instead of installing an empty-party state that would panic.
func TestLoop_LoadRejectsHerolessSave(t *testing.T) {
	eng := engine.New(loopScenario(t))
	eng.SaveDir = t.TempDir()

	// Write a schema-valid save for this scenario with no hero.
	heroless := model.GameState{
		Scenario: eng.Scenario().ID,
		Dungeon:  model.DungeonState{CurrentRoom: "entrance"},
		Party:    nil,
	}
	require.NoError(t, save.Save(heroless, filepath.Join(eng.SaveDir, "corrupt.json")))

	char := model.Character{
		ID: "c1", Name: "Hero", Class: "fighter", Level: 1,
		HP: 100, MaxHP: 100,
		Stats: model.Stats{STR: 14, DEX: 12, CON: 12, INT: 10, WIS: 10, CHA: 10},
	}
	state, err := eng.NewGame(char)
	require.NoError(t, err)

	inputs := []string{"load corrupt", "quit"}
	rend := &scriptRenderer{events: make(chan model.InputEvent, len(inputs))}
	for _, inp := range inputs {
		rend.events <- model.InputEvent{Type: "input", Payload: inp}
	}

	loop := game.NewLoop(eng, interpreter.NewRules(), narrator.NewTemplate(), rend)
	require.NoError(t, loop.Run(context.Background(), &state))

	// The heroless save was rejected, so the original hero is still present.
	require.Len(t, state.Party, 1)
	assert.Equal(t, "Hero", state.Party[0].Name)
}
