package narrator_test

import (
	"context"
	"testing"
	"time"

	"github.com/punt-labs/cryptd/internal/inference"
	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/narrator"
	"github.com/punt-labs/cryptd/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLLMNarrator_TimeoutFallsBackToTemplate(t *testing.T) {
	srv := testutil.NewFakeSLMServer([]string{"Atmospheric prose."})
	defer srv.Close()
	srv.SetDelay(2 * time.Second) // slow server

	client := inference.NewClientWithOpts(srv.URL(), "test-model",
		inference.WithAPIKey("test-key"),
		inference.WithTimeout(50*time.Millisecond),
	)
	llm := narrator.NewLLM(client, narrator.NewTemplate())

	event := model.EngineEvent{
		Type: "moved",
		Room: "entrance",
		Details: map[string]any{
			"description": "dark room",
			"exits":       []string{"south"},
		},
	}

	result, err := llm.Narrate(context.Background(), event, model.GameState{})
	require.NoError(t, err)
	// Template narrator output, not LLM.
	assert.Contains(t, result, "You enter entrance")
	assert.Contains(t, result, "Exits: south.")
}

func TestLLMNarrator_PartialFailure(t *testing.T) {
	srv := testutil.NewFakeSLMServer([]string{"The chamber glows with an eerie light."})
	defer srv.Close()

	client := inference.NewClientWithOpts(srv.URL(), "test-model",
		inference.WithAPIKey("test-key"),
		inference.WithTimeout(50*time.Millisecond),
	)
	llm := narrator.NewLLM(client, narrator.NewTemplate())

	event := model.EngineEvent{
		Type: "moved",
		Room: "entrance",
		Details: map[string]any{
			"description": "dark room",
			"exits":       []string{"south"},
		},
	}

	// First call: LLM responds with atmospheric prose.
	result, err := llm.Narrate(context.Background(), event, model.GameState{})
	require.NoError(t, err)
	assert.Contains(t, result, "eerie light")
	assert.Contains(t, result, "Exits: south.") // deterministic suffix

	// Introduce delay — subsequent calls time out and fall back.
	srv.SetDelay(2 * time.Second)

	result, err = llm.Narrate(context.Background(), event, model.GameState{})
	require.NoError(t, err)
	// Template narrator output.
	assert.Contains(t, result, "You enter entrance")
}

func TestLLMNarrator_AuthHeaderSent(t *testing.T) {
	srv := testutil.NewFakeSLMServer([]string{"A dark room awaits."})
	defer srv.Close()

	client := inference.NewClientWithOpts(srv.URL(), "test-model",
		inference.WithAPIKey("test-key"),
		inference.WithTimeout(5*time.Second),
	)
	llm := narrator.NewLLM(client, narrator.NewTemplate())

	event := model.EngineEvent{
		Type: "moved",
		Room: "entrance",
		Details: map[string]any{
			"description": "dark room",
			"exits":       []string{"south"},
		},
	}

	_, err := llm.Narrate(context.Background(), event, model.GameState{})
	require.NoError(t, err)

	calls := srv.Calls()
	require.Len(t, calls, 1)
	assert.Equal(t, "Bearer test-key", calls[0].AuthHeader)
}
