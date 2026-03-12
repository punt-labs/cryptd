package narrator_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/punt-labs/cryptd/internal/inference"
	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/narrator"
	"github.com/punt-labs/cryptd/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newSLMNarrator(responses []string) (*narrator.SLM, *testutil.FakeSLMServer) {
	srv := testutil.NewFakeSLMServer(responses)
	client := inference.NewClient(srv.URL(), "test-model", 5*time.Second)
	return narrator.NewSLM(client, narrator.NewTemplate()), srv
}

func TestSLMNarrator_MovedEvent(t *testing.T) {
	slm, srv := newSLMNarrator([]string{"You step into a damp stone chamber. Water drips from the ceiling."})
	defer srv.Close()

	event := model.EngineEvent{
		Type: "moved",
		Room: "entrance",
		Details: map[string]any{
			"description": "dark stone chamber, dripping water",
			"exits":       []string{"north", "south"},
		},
	}

	result, err := slm.Narrate(context.Background(), event, model.GameState{})
	require.NoError(t, err)
	assert.Contains(t, result, "damp stone chamber")
}

func TestSLMNarrator_LookedEvent(t *testing.T) {
	slm, srv := newSLMNarrator([]string{"The chamber stretches before you, shadows dancing on the walls."})
	defer srv.Close()

	event := model.EngineEvent{
		Type: "looked",
		Room: "great_hall",
		Details: map[string]any{
			"description": "vast chamber, torchlit walls",
			"exits":       []string{"east", "west"},
			"items":       []string{"rusty key"},
		},
	}

	result, err := slm.Narrate(context.Background(), event, model.GameState{})
	require.NoError(t, err)
	assert.Contains(t, result, "shadows dancing")
}

func TestSLMNarrator_NonRoomEventDelegates(t *testing.T) {
	slm, srv := newSLMNarrator([]string{"should not be called"})
	defer srv.Close()

	for _, eventType := range []string{"attack_hit", "combat_started", "picked_up", "quit", "unknown_action"} {
		t.Run(eventType, func(t *testing.T) {
			event := model.EngineEvent{Type: eventType, Details: map[string]any{}}
			result, err := slm.Narrate(context.Background(), event, model.GameState{})
			require.NoError(t, err)
			assert.NotEmpty(t, result)
		})
	}

	// No SLM calls should have been made.
	assert.Empty(t, srv.Calls())
}

func TestSLMNarrator_EmptyDescriptionFallback(t *testing.T) {
	slm, srv := newSLMNarrator([]string{"should not be called"})
	defer srv.Close()

	event := model.EngineEvent{
		Type:    "moved",
		Room:    "entrance",
		Details: map[string]any{},
	}

	result, err := slm.Narrate(context.Background(), event, model.GameState{})
	require.NoError(t, err)
	assert.Contains(t, result, "You enter entrance")
	assert.Empty(t, srv.Calls())
}

func TestSLMNarrator_EmptyResponseFallback(t *testing.T) {
	slm, srv := newSLMNarrator([]string{"   "})
	defer srv.Close()

	event := model.EngineEvent{
		Type: "moved",
		Room: "entrance",
		Details: map[string]any{
			"description": "dark room",
		},
	}

	result, err := slm.Narrate(context.Background(), event, model.GameState{})
	require.NoError(t, err)
	assert.Contains(t, result, "You enter entrance")
}

func TestSLMNarrator_FallbackOnInferenceError(t *testing.T) {
	// Empty responses causes 503.
	slm, srv := newSLMNarrator(nil)
	defer srv.Close()

	event := model.EngineEvent{
		Type: "moved",
		Room: "entrance",
		Details: map[string]any{
			"description": "dark room",
		},
	}

	result, err := slm.Narrate(context.Background(), event, model.GameState{})
	require.NoError(t, err)
	assert.Contains(t, result, "You enter entrance")
}

func TestSLMNarrator_PropagatesContextCancellation(t *testing.T) {
	slm, srv := newSLMNarrator([]string{"ok"})
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	event := model.EngineEvent{
		Type: "moved",
		Room: "entrance",
		Details: map[string]any{
			"description": "dark room",
		},
	}

	_, err := slm.Narrate(ctx, event, model.GameState{})
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestSLMNarrator_SendsCorrectPrompt(t *testing.T) {
	slm, srv := newSLMNarrator([]string{"You stand in a dark room."})
	defer srv.Close()

	event := model.EngineEvent{
		Type: "moved",
		Room: "goblin_lair",
		Details: map[string]any{
			"description": "filthy cave, bones on floor",
			"exits":       []string{"north"},
			"items":       []string{"short sword"},
		},
	}

	_, err := slm.Narrate(context.Background(), event, model.GameState{})
	require.NoError(t, err)

	calls := srv.Calls()
	require.Len(t, calls, 1)
	require.Len(t, calls[0].Messages, 2)

	var msg0, msg1 struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	require.NoError(t, json.Unmarshal(calls[0].Messages[0], &msg0))
	require.NoError(t, json.Unmarshal(calls[0].Messages[1], &msg1))

	assert.Equal(t, "system", msg0.Role)
	assert.Contains(t, msg0.Content, "narrator for a text adventure")
	assert.Equal(t, "user", msg1.Role)
	assert.Contains(t, msg1.Content, "Room: goblin_lair")
	assert.Contains(t, msg1.Content, "filthy cave, bones on floor")
	assert.Contains(t, msg1.Content, "Exits: north")
	assert.Contains(t, msg1.Content, "Visible items: short sword")

	// Temperature should be 0.7 for creative narration.
	require.NotNil(t, calls[0].Temperature)
	assert.InDelta(t, 0.7, *calls[0].Temperature, 0.001)

	// MaxTokens should be 200 for concise narration.
	assert.Equal(t, 200, calls[0].MaxTokens)
}
