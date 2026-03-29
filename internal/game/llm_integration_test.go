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

func TestLLMLoop_HappyPath(t *testing.T) {
	// Separate servers for interpreter and narrator avoid response ordering issues.
	interpSrv := testutil.NewFakeSLMServer([]string{
		`{"type":"look"}`,
		`{"type":"quit"}`,
	})
	defer interpSrv.Close()

	narrSrv := testutil.NewFakeSLMServer([]string{
		"The entrance hall stretches before you, shadows dancing on the walls.",
	})
	defer narrSrv.Close()

	interpClient := inference.NewClientWithOpts(interpSrv.URL(), "test-model",
		inference.WithAPIKey("test-key"),
		inference.WithTimeout(5*time.Second),
	)
	narrClient := inference.NewClientWithOpts(narrSrv.URL(), "test-model",
		inference.WithAPIKey("test-key"),
		inference.WithTimeout(5*time.Second),
	)
	interp := interpreter.NewLLM(interpClient, interpreter.NewRules())
	narr := narrator.NewLLM(narrClient, narrator.NewTemplate())

	eng := engine.New(loadScenario(t))
	state := newState(t, eng)

	// Inputs must be unrecognisable by the Rules interpreter so the LLM
	// is actually invoked. Rules handles "look" directly — use natural-
	// language phrases that only the LLM can parse.
	inputs := []string{"survey my surroundings", "time to leave"}
	fake := &fakeRenderer{events: make(chan model.InputEvent, len(inputs))}
	for _, inp := range inputs {
		fake.events <- model.InputEvent{Type: "input", Payload: inp}
	}

	err := game.NewLoop(eng, interp, narr, fake).Run(context.Background(), &state)
	require.NoError(t, err)

	// Interpreter was called for both inputs (Rules returned "unknown",
	// so the LLM was consulted each time).
	interpCalls := interpSrv.Calls()
	assert.Equal(t, 2, len(interpCalls), "expected 2 interpreter LLM calls")

	// Verify auth headers were sent.
	for i, call := range interpCalls {
		assert.Equal(t, "Bearer test-key", call.AuthHeader, "interpreter call %d missing auth header", i)
	}

	// Narrator was called for room events (initial look + dispatched look).
	narrCalls := narrSrv.Calls()
	assert.GreaterOrEqual(t, len(narrCalls), 1, "expected at least 1 narrator LLM call")

	for i, call := range narrCalls {
		assert.Equal(t, "Bearer test-key", call.AuthHeader, "narrator call %d missing auth header", i)
	}
}

func TestLLMLoop_FallbackOnServerDown(t *testing.T) {
	// Single server for both interpreter and narrator. Close it before
	// running the game loop — both components fall back immediately.
	srv := testutil.NewFakeSLMServer([]string{`{"type":"look"}`})
	srv.Close() // server is already down

	client := inference.NewClientWithOpts(srv.URL(), "test-model",
		inference.WithAPIKey("test-key"),
		inference.WithTimeout(1*time.Second),
	)
	interp := interpreter.NewLLM(client, interpreter.NewRules())
	narr := narrator.NewLLM(client, narrator.NewTemplate())

	eng := engine.New(loadScenario(t))
	state := newState(t, eng)

	inputs := []string{"look", "quit"}
	fake := &fakeRenderer{events: make(chan model.InputEvent, len(inputs))}
	for _, inp := range inputs {
		fake.events <- model.InputEvent{Type: "input", Payload: inp}
	}

	err := game.NewLoop(eng, interp, narr, fake).Run(context.Background(), &state)
	require.NoError(t, err)

	// The loop completed using fallback narration and interpretation.
	assert.GreaterOrEqual(t, len(fake.renders), 2)
}

func TestLLMLoop_TimeoutFallback(t *testing.T) {
	// Both servers respond slowly; client timeout triggers fallback.
	interpSrv := testutil.NewFakeSLMServer([]string{`{"type":"look"}`})
	defer interpSrv.Close()
	interpSrv.SetDelay(2 * time.Second)

	narrSrv := testutil.NewFakeSLMServer([]string{"Atmospheric text."})
	defer narrSrv.Close()
	narrSrv.SetDelay(2 * time.Second)

	// Clients with 50ms timeout — will always time out.
	interpClient := inference.NewClientWithOpts(interpSrv.URL(), "test-model",
		inference.WithAPIKey("test-key"),
		inference.WithTimeout(50*time.Millisecond),
	)
	narrClient := inference.NewClientWithOpts(narrSrv.URL(), "test-model",
		inference.WithAPIKey("test-key"),
		inference.WithTimeout(50*time.Millisecond),
	)
	interp := interpreter.NewLLM(interpClient, interpreter.NewRules())
	narr := narrator.NewLLM(narrClient, narrator.NewTemplate())

	eng := engine.New(loadScenario(t))
	state := newState(t, eng)

	inputs := []string{"look", "quit"}
	fake := &fakeRenderer{events: make(chan model.InputEvent, len(inputs))}
	for _, inp := range inputs {
		fake.events <- model.InputEvent{Type: "input", Payload: inp}
	}

	err := game.NewLoop(eng, interp, narr, fake).Run(context.Background(), &state)
	require.NoError(t, err)

	// Loop completed using fallback — LLM calls all timed out.
	assert.NotEmpty(t, fake.renders)
}
