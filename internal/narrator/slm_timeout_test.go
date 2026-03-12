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

func TestSLMNarrator_TimeoutFallsBackToTemplate(t *testing.T) {
	srv := testutil.NewFakeSLMServer([]string{"Atmospheric prose."})
	defer srv.Close()
	srv.SetDelay(2 * time.Second) // slow server

	client := inference.NewClient(srv.URL(), "test-model", 50*time.Millisecond)
	slm := narrator.NewSLM(client, narrator.NewTemplate())

	event := model.EngineEvent{
		Type: "moved",
		Room: "entrance",
		Details: map[string]any{
			"description": "dark room",
			"exits":       []string{"south"},
		},
	}

	result, err := slm.Narrate(context.Background(), event, model.GameState{})
	require.NoError(t, err)
	// Template narrator output, not SLM.
	assert.Contains(t, result, "You enter entrance")
	assert.Contains(t, result, "Exits: south.")
}

func TestSLMNarrator_PartialFailure(t *testing.T) {
	srv := testutil.NewFakeSLMServer([]string{"The chamber glows with an eerie light."})
	defer srv.Close()

	client := inference.NewClient(srv.URL(), "test-model", 50*time.Millisecond)
	slm := narrator.NewSLM(client, narrator.NewTemplate())

	event := model.EngineEvent{
		Type: "moved",
		Room: "entrance",
		Details: map[string]any{
			"description": "dark room",
			"exits":       []string{"south"},
		},
	}

	// First call: SLM responds with atmospheric prose.
	result, err := slm.Narrate(context.Background(), event, model.GameState{})
	require.NoError(t, err)
	assert.Contains(t, result, "eerie light")
	assert.Contains(t, result, "Exits: south.") // deterministic suffix

	// Introduce delay — subsequent calls time out and fall back.
	srv.SetDelay(2 * time.Second)

	result, err = slm.Narrate(context.Background(), event, model.GameState{})
	require.NoError(t, err)
	// Template narrator output.
	assert.Contains(t, result, "You enter entrance")
}
