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

func TestSLM_TimeoutFallsBackToRules(t *testing.T) {
	srv := testutil.NewFakeSLMServer([]string{`{"type":"look"}`})
	defer srv.Close()
	srv.SetDelay(2 * time.Second) // slow server

	client := inference.NewClient(srv.URL(), "test-model", 50*time.Millisecond)
	slm := interpreter.NewSLM(client, interpreter.NewRules())

	// "go north" is understood by Rules interpreter.
	action, err := slm.Interpret(context.Background(), "go north", model.GameState{})
	require.NoError(t, err)
	assert.Equal(t, "move", action.Type)
	assert.Equal(t, "north", action.Direction)
}

func TestSLM_PartialFailure(t *testing.T) {
	// First call succeeds, then server starts timing out.
	srv := testutil.NewFakeSLMServer([]string{`{"type":"look"}`})
	defer srv.Close()

	client := inference.NewClient(srv.URL(), "test-model", 50*time.Millisecond)
	slm := interpreter.NewSLM(client, interpreter.NewRules())

	// First call: SLM responds.
	action, err := slm.Interpret(context.Background(), "look around", model.GameState{})
	require.NoError(t, err)
	assert.Equal(t, "look", action.Type)

	// Introduce delay — subsequent calls time out and fall back.
	srv.SetDelay(2 * time.Second)

	action, err = slm.Interpret(context.Background(), "go south", model.GameState{})
	require.NoError(t, err)
	assert.Equal(t, "move", action.Type)
	assert.Equal(t, "south", action.Direction)
}
