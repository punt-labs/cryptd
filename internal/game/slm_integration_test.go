//go:build integration

package game_test

import (
	"context"
	"testing"
	"time"

	"github.com/punt-labs/cryptd/internal/engine"
	"github.com/punt-labs/cryptd/internal/game"
	"github.com/punt-labs/cryptd/internal/inference"
	"github.com/punt-labs/cryptd/internal/interpreter"
	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/narrator"
	"github.com/punt-labs/cryptd/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newSLMStack creates an SLM interpreter + narrator backed by a FakeSLMServer,
// with Rules/Template fallbacks. Returns both components and the server for
// cleanup and inspection.
func newSLMStack(t *testing.T, responses []string) (*interpreter.SLM, *narrator.SLM, *testutil.FakeSLMServer) {
	t.Helper()
	srv := testutil.NewFakeSLMServer(responses)
	client := inference.NewClient(srv.URL(), "test-model", 5*time.Second)
	interp := interpreter.NewSLM(client, interpreter.NewRules())
	narr := narrator.NewSLM(client, narrator.NewTemplate())
	return interp, narr, srv
}

func TestSLMLoop_HappyPath(t *testing.T) {
	// SLM responds with valid actions and atmospheric narration.
	interp, narr, srv := newSLMStack(t, []string{
		// Interpreter responses (JSON actions).
		`{"type":"look"}`,
		`{"type":"quit"}`,
		// Narrator responses (atmospheric prose) — interleaved by the game loop.
		"The entrance hall stretches before you, shadows dancing on the walls.",
		"A final look reveals nothing new.",
	})
	defer srv.Close()

	eng := engine.New(loadScenario(t))
	state := newState(t, eng)

	inputs := []string{"look around the room", "time to leave"}
	fake := &fakeRenderer{events: make(chan model.InputEvent, len(inputs))}
	for _, inp := range inputs {
		fake.events <- model.InputEvent{Type: "input", Payload: inp}
	}

	err := game.NewLoop(eng, interp, narr, fake).Run(context.Background(), &state)
	require.NoError(t, err)

	// SLM was called for interpretation.
	calls := srv.Calls()
	assert.GreaterOrEqual(t, len(calls), 2, "expected at least 2 SLM calls (interp + narr)")
}

func TestSLMLoop_FallbackOnServerDown(t *testing.T) {
	// Start with a working server, then shut it down mid-session.
	interp, narr, srv := newSLMStack(t, []string{
		`{"type":"look"}`,
		"You see a dimly lit chamber.",
	})

	eng := engine.New(loadScenario(t))
	state := newState(t, eng)

	// First command: "look" — SLM handles it.
	// Second command: "go south" — server is down, falls back to Rules interpreter.
	// Third command: "quit" — server still down, falls back to Rules interpreter.
	inputs := []string{"look around", "go south", "quit"}
	fake := &fakeRenderer{events: make(chan model.InputEvent, len(inputs))}
	for _, inp := range inputs {
		fake.events <- model.InputEvent{Type: "input", Payload: inp}
	}

	// Shut down the SLM server after the first response is consumed.
	// The FakeSLMServer cycles through responses, so after "look" + narration,
	// close the server to force fallback on subsequent calls.
	go func() {
		// Wait for at least one call, then close.
		for {
			if len(srv.Calls()) >= 1 {
				srv.Close()
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()

	err := game.NewLoop(eng, interp, narr, fake).Run(context.Background(), &state)
	require.NoError(t, err)

	// The loop completed despite the server going down — fallback worked.
	// Player moved south (Rules interpreter understood "go south").
	assert.Contains(t, state.Dungeon.VisitedRooms, "goblin_lair")
}

func TestSLMLoop_TimeoutFallback(t *testing.T) {
	// Server responds slowly; client timeout triggers fallback.
	srv := testutil.NewFakeSLMServer([]string{`{"type":"look"}`, "Atmospheric text."})
	defer srv.Close()
	srv.SetDelay(2 * time.Second) // 2s delay

	// Client with 50ms timeout — will always time out.
	client := inference.NewClient(srv.URL(), "test-model", 50*time.Millisecond)
	interp := interpreter.NewSLM(client, interpreter.NewRules())
	narr := narrator.NewSLM(client, narrator.NewTemplate())

	eng := engine.New(loadScenario(t))
	state := newState(t, eng)

	inputs := []string{"look", "quit"}
	fake := &fakeRenderer{events: make(chan model.InputEvent, len(inputs))}
	for _, inp := range inputs {
		fake.events <- model.InputEvent{Type: "input", Payload: inp}
	}

	err := game.NewLoop(eng, interp, narr, fake).Run(context.Background(), &state)
	require.NoError(t, err)

	// Loop completed using fallback — SLM calls all timed out.
	// Renders should contain template narrator output (not SLM prose).
	assert.NotEmpty(t, fake.renders)
}
