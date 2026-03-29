package interpreter_test

import (
	"context"
	"testing"
	"time"

	"github.com/punt-labs/cryptd/internal/inference"
	"github.com/punt-labs/cryptd/internal/interpreter"
	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLLM_TimeoutFallsBackToRules(t *testing.T) {
	srv := testutil.NewFakeSLMServer([]string{`{"type":"look"}`})
	defer srv.Close()
	srv.SetDelay(2 * time.Second) // slow server

	client := inference.NewClientWithOpts(srv.URL(), "test-model",
		inference.WithAPIKey("test-key"),
		inference.WithTimeout(50*time.Millisecond),
	)
	llm := interpreter.NewLLM(client, interpreter.NewRules())

	// "go north" is understood by Rules interpreter — LLM is never called.
	action, err := llm.Interpret(context.Background(), "go north", model.GameState{})
	require.NoError(t, err)
	assert.Equal(t, "move", action.Type)
	assert.Equal(t, "north", action.Direction)
}

func TestLLM_TimeoutOnUnknownInputFallsBack(t *testing.T) {
	srv := testutil.NewFakeSLMServer([]string{`{"type":"look"}`})
	defer srv.Close()
	srv.SetDelay(2 * time.Second) // slow server

	client := inference.NewClientWithOpts(srv.URL(), "test-model",
		inference.WithAPIKey("test-key"),
		inference.WithTimeout(50*time.Millisecond),
	)
	llm := interpreter.NewLLM(client, interpreter.NewRules())

	// "survey my surroundings" is unknown to Rules. LLM times out,
	// so we get back the rules fallback result (unknown action).
	action, err := llm.Interpret(context.Background(), "survey my surroundings", model.GameState{})
	require.NoError(t, err)
	assert.Equal(t, "unknown", action.Type)
}

func TestLLM_PartialFailure(t *testing.T) {
	// First call succeeds, then server starts timing out.
	srv := testutil.NewFakeSLMServer([]string{`{"type":"look"}`})
	defer srv.Close()

	client := inference.NewClientWithOpts(srv.URL(), "test-model",
		inference.WithAPIKey("test-key"),
		inference.WithTimeout(50*time.Millisecond),
	)
	llm := interpreter.NewLLM(client, interpreter.NewRules())

	// First call: rules doesn't recognize this, LLM responds with "look".
	action, err := llm.Interpret(context.Background(), "survey my surroundings", model.GameState{})
	require.NoError(t, err)
	assert.Equal(t, "look", action.Type)

	// Introduce delay — subsequent LLM calls time out and fall back to rules.
	srv.SetDelay(2 * time.Second)

	action, err = llm.Interpret(context.Background(), "go south", model.GameState{})
	require.NoError(t, err)
	assert.Equal(t, "move", action.Type)
	assert.Equal(t, "south", action.Direction)
}

func TestLLM_AuthHeaderSent(t *testing.T) {
	srv := testutil.NewFakeSLMServer([]string{`{"type":"look"}`})
	defer srv.Close()

	client := inference.NewClientWithOpts(srv.URL(), "test-model",
		inference.WithAPIKey("test-key"),
		inference.WithTimeout(5*time.Second),
	)
	llm := interpreter.NewLLM(client, interpreter.NewRules())

	// Use input rules doesn't handle, so the LLM is actually called.
	_, err := llm.Interpret(context.Background(), "survey my surroundings", model.GameState{})
	require.NoError(t, err)

	calls := srv.Calls()
	require.Len(t, calls, 1)
	assert.Equal(t, "Bearer test-key", calls[0].AuthHeader)
}
