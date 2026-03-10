//go:build integration

package game_test

import (
	"context"
	"errors"
	"testing"

	"github.com/punt-labs/cryptd/internal/engine"
	"github.com/punt-labs/cryptd/internal/game"
	"github.com/punt-labs/cryptd/internal/interpreter"
	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/narrator"
	"github.com/punt-labs/cryptd/internal/scenario"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadScenario(t *testing.T) *scenario.Scenario {
	t.Helper()
	s, err := scenario.Load("../../testdata/scenarios/minimal.yaml")
	require.NoError(t, err)
	return s
}

func newState(t *testing.T, eng *engine.Engine) model.GameState {
	t.Helper()
	char := model.Character{ID: "c1", Name: "Hero", Class: "fighter", Level: 1, HP: 10, MaxHP: 10}
	state, err := eng.NewGame(char)
	require.NoError(t, err)
	return state
}

func TestHeadlessLoop_NewGameMoveLookMove(t *testing.T) {
	eng := engine.New(loadScenario(t))
	interp := interpreter.NewRules()
	narr := narrator.NewTemplate()

	inputs := []string{"go south", "look", "go north", "quit"}
	fake := &fakeRenderer{events: make(chan model.InputEvent, len(inputs))}
	for _, inp := range inputs {
		fake.events <- model.InputEvent{Type: "input", Payload: inp}
	}

	state := newState(t, eng)
	loop := game.NewLoop(eng, interp, narr, fake)
	err := loop.Run(context.Background(), &state)
	require.NoError(t, err)

	assert.Equal(t, "entrance", state.Dungeon.CurrentRoom)
	assert.Contains(t, state.Dungeon.VisitedRooms, "goblin_lair")
	assert.GreaterOrEqual(t, len(state.AdventureLog), 2)
}

func TestLoop_ContextCancellation(t *testing.T) {
	eng := engine.New(loadScenario(t))
	interp := interpreter.NewRules()
	narr := narrator.NewTemplate()

	fake := &fakeRenderer{events: make(chan model.InputEvent)} // never sends
	state := newState(t, eng)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := game.NewLoop(eng, interp, narr, fake).Run(ctx, &state)
	require.ErrorIs(t, err, context.Canceled)
}

func TestLoop_ChannelClose(t *testing.T) {
	eng := engine.New(loadScenario(t))
	interp := interpreter.NewRules()
	narr := narrator.NewTemplate()

	ch := make(chan model.InputEvent)
	fake := &fakeRenderer{events: ch}
	state := newState(t, eng)

	go func() { close(ch) }()

	err := game.NewLoop(eng, interp, narr, fake).Run(context.Background(), &state)
	require.NoError(t, err)
}

func TestLoop_QuitEvent(t *testing.T) {
	eng := engine.New(loadScenario(t))
	interp := interpreter.NewRules()
	narr := narrator.NewTemplate()

	ch := make(chan model.InputEvent, 1)
	ch <- model.InputEvent{Type: "quit"}
	fake := &fakeRenderer{events: ch}
	state := newState(t, eng)

	err := game.NewLoop(eng, interp, narr, fake).Run(context.Background(), &state)
	require.NoError(t, err)
}

func TestLoop_NarrateErrorPropagates(t *testing.T) {
	eng := engine.New(loadScenario(t))
	interp := interpreter.NewRules()

	badNarr := &countingErrNarrator{failAfter: 0, err: errors.New("narrate boom")}
	fake := &fakeRenderer{events: make(chan model.InputEvent, 1)}
	fake.events <- model.InputEvent{Type: "input", Payload: "look"}

	state := newState(t, eng)
	err := game.NewLoop(eng, interp, badNarr, fake).Run(context.Background(), &state)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "narrate boom")
}

func TestLoop_RenderErrorPropagates(t *testing.T) {
	eng := engine.New(loadScenario(t))
	interp := interpreter.NewRules()
	narr := narrator.NewTemplate()

	bad := &countingErrRenderer{failAfter: 0, err: errors.New("render boom"), events: make(chan model.InputEvent)}
	state := newState(t, eng)

	err := game.NewLoop(eng, interp, narr, bad).Run(context.Background(), &state)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "render boom")
}

func TestLoop_DispatchErrorPropagates(t *testing.T) {
	eng := engine.New(loadScenario(t))
	interp := interpreter.NewRules()

	// Succeed on initial narrate (before loop), fail on second call (inside dispatch).
	narr := &countingErrNarrator{failAfter: 1, err: errors.New("narrate in loop boom")}
	ch := make(chan model.InputEvent, 1)
	ch <- model.InputEvent{Type: "input", Payload: "look"}
	fake := &fakeRenderer{events: ch}
	state := newState(t, eng)

	err := game.NewLoop(eng, interp, narr, fake).Run(context.Background(), &state)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "narrate in loop boom")
}

func TestLoop_RenderErrorInLoop(t *testing.T) {
	eng := engine.New(loadScenario(t))
	interp := interpreter.NewRules()
	narr := narrator.NewTemplate()

	// Succeed on first render (before loop), fail on second (inside loop).
	bad := &countingErrRenderer{failAfter: 1, err: errors.New("render in loop boom")}
	bad.events = make(chan model.InputEvent, 1)
	bad.events <- model.InputEvent{Type: "input", Payload: "look"}

	state := newState(t, eng)
	err := game.NewLoop(eng, interp, narr, bad).Run(context.Background(), &state)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "render in loop boom")
}
type fakeRenderer struct {
	events  chan model.InputEvent
	renders []string
}

func (f *fakeRenderer) Render(_ context.Context, _ model.GameState, narration string) error {
	f.renders = append(f.renders, narration)
	return nil
}

func (f *fakeRenderer) Events() <-chan model.InputEvent { return f.events }

// errRenderer returns an error on first Render call.
type errRenderer struct {
	err    error
	events chan model.InputEvent
}

func (r *errRenderer) Render(_ context.Context, _ model.GameState, _ string) error {
	return r.err
}

func (r *errRenderer) Events() <-chan model.InputEvent { return r.events }

// countingErrNarrator succeeds for the first failAfter calls then returns err.
type countingErrNarrator struct {
	calls     int
	failAfter int
	err       error
}

func (n *countingErrNarrator) Narrate(_ context.Context, _ model.EngineEvent, _ model.GameState) (string, error) {
	n.calls++
	if n.calls > n.failAfter {
		return "", n.err
	}
	return "ok", nil
}

// countingErrRenderer succeeds for the first failAfter calls then returns err.
type countingErrRenderer struct {
	calls     int
	failAfter int
	err       error
	events    chan model.InputEvent
}

func (r *countingErrRenderer) Render(_ context.Context, _ model.GameState, _ string) error {
	r.calls++
	if r.calls > r.failAfter {
		return r.err
	}
	return nil
}

func (r *countingErrRenderer) Events() <-chan model.InputEvent { return r.events }

