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

func newLLMNarrator(responses []string) (*narrator.LLM, *testutil.FakeSLMServer) {
	srv := testutil.NewFakeSLMServer(responses)
	client := inference.NewClientWithOpts(srv.URL(), "claude-sonnet-4-20250514",
		inference.WithAPIKey("sk-test"),
		inference.WithTimeout(5*time.Second),
	)
	return narrator.NewLLM(client, narrator.NewTemplate()), srv
}

func TestLLMNarrator_MovedEvent(t *testing.T) {
	llm, srv := newLLMNarrator([]string{"You step into a damp stone chamber. Water drips from the ceiling."})
	defer srv.Close()

	event := model.EngineEvent{
		Type: "moved",
		Room: "entrance",
		Details: map[string]any{
			"description": "dark stone chamber, dripping water",
			"exits":       []string{"north", "south"},
		},
	}

	result, err := llm.Narrate(context.Background(), event, model.GameState{})
	require.NoError(t, err)
	assert.Contains(t, result, "damp stone chamber")
	assert.Contains(t, result, "Exits: north, south.")
}

func TestLLMNarrator_LookedEvent(t *testing.T) {
	llm, srv := newLLMNarrator([]string{"The chamber stretches before you, shadows dancing on the walls."})
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

	result, err := llm.Narrate(context.Background(), event, model.GameState{})
	require.NoError(t, err)
	assert.Contains(t, result, "shadows dancing")
	assert.Contains(t, result, "Exits: east, west.")
	assert.Contains(t, result, "You see: rusty key.")
}

func TestLLMNarrator_CombatStarted(t *testing.T) {
	llm, srv := newLLMNarrator([]string{"A snarling goblin leaps from the shadows, blade drawn."})
	defer srv.Close()

	event := model.EngineEvent{
		Type:    "combat_started",
		Details: map[string]any{"enemy_names": "goblin"},
	}

	result, err := llm.Narrate(context.Background(), event, model.GameState{})
	require.NoError(t, err)
	assert.Contains(t, result, "goblin")
	assert.Len(t, srv.Calls(), 1)
}

func TestLLMNarrator_CombatWon(t *testing.T) {
	llm, srv := newLLMNarrator([]string{"Silence falls as the last foe crumbles."})
	defer srv.Close()

	event := model.EngineEvent{Type: "combat_won", Details: map[string]any{}}

	result, err := llm.Narrate(context.Background(), event, model.GameState{})
	require.NoError(t, err)
	assert.Contains(t, result, "foe crumbles")
}

func TestLLMNarrator_HeroDied(t *testing.T) {
	llm, srv := newLLMNarrator([]string{"Darkness closes in as your strength fades."})
	defer srv.Close()

	event := model.EngineEvent{Type: "hero_died", Details: map[string]any{}}

	result, err := llm.Narrate(context.Background(), event, model.GameState{})
	require.NoError(t, err)
	assert.Contains(t, result, "strength fades")
}

func TestLLMNarrator_LevelUp(t *testing.T) {
	llm, srv := newLLMNarrator([]string{"Power surges through your veins."})
	defer srv.Close()

	event := model.EngineEvent{
		Type:    "level_up",
		Details: map[string]any{"level": 3, "hp_gain": 5},
	}

	result, err := llm.Narrate(context.Background(), event, model.GameState{})
	require.NoError(t, err)
	assert.Contains(t, result, "Power surges")
	assert.Contains(t, result, "(Level 3, +5 HP)")
}

func TestLLMNarrator_Examined(t *testing.T) {
	llm, srv := newLLMNarrator([]string{"A battered blade with notches along its edge."})
	defer srv.Close()

	event := model.EngineEvent{
		Type: "examined",
		Details: map[string]any{
			"item_name":   "short sword",
			"description": "a short sword with a worn grip",
		},
	}

	result, err := llm.Narrate(context.Background(), event, model.GameState{})
	require.NoError(t, err)
	assert.Contains(t, result, "battered blade")
}

func TestLLMNarrator_ExaminedNoDescriptionFallback(t *testing.T) {
	llm, srv := newLLMNarrator([]string{"should not be called"})
	defer srv.Close()

	event := model.EngineEvent{
		Type:    "examined",
		Details: map[string]any{"item_name": "key"},
	}

	result, err := llm.Narrate(context.Background(), event, model.GameState{})
	require.NoError(t, err)
	assert.Contains(t, result, "nothing special about the key")
	assert.Empty(t, srv.Calls())
}

func TestLLMNarrator_TacticalEventsDelegateToTemplate(t *testing.T) {
	llm, srv := newLLMNarrator([]string{"should not be called"})
	defer srv.Close()

	tacticalEvents := []string{
		"attack_hit", "attack_kill", "enemy_attacks", "defend",
		"flee_success", "flee_fail", "spell_damage", "spell_heal",
		"picked_up", "dropped", "equipped", "unequipped",
		"not_in_combat", "in_combat", "unknown_action", "quit",
	}
	for _, eventType := range tacticalEvents {
		t.Run(eventType, func(t *testing.T) {
			event := model.EngineEvent{Type: eventType, Details: map[string]any{}}
			result, err := llm.Narrate(context.Background(), event, model.GameState{})
			require.NoError(t, err)
			assert.NotEmpty(t, result)
		})
	}
	assert.Empty(t, srv.Calls())
}

func TestLLMNarrator_EmptyDescriptionFallback(t *testing.T) {
	llm, srv := newLLMNarrator([]string{"should not be called"})
	defer srv.Close()

	event := model.EngineEvent{
		Type:    "moved",
		Room:    "entrance",
		Details: map[string]any{},
	}

	result, err := llm.Narrate(context.Background(), event, model.GameState{})
	require.NoError(t, err)
	assert.Contains(t, result, "You enter entrance")
	assert.Empty(t, srv.Calls())
}

func TestLLMNarrator_EmptyResponseFallback(t *testing.T) {
	llm, srv := newLLMNarrator([]string{"   "})
	defer srv.Close()

	event := model.EngineEvent{
		Type: "moved",
		Room: "entrance",
		Details: map[string]any{
			"description": "dark room",
		},
	}

	result, err := llm.Narrate(context.Background(), event, model.GameState{})
	require.NoError(t, err)
	assert.Contains(t, result, "You enter entrance")
}

func TestLLMNarrator_NilClientFallback(t *testing.T) {
	llm := narrator.NewLLM(nil, narrator.NewTemplate())

	event := model.EngineEvent{
		Type: "moved",
		Room: "entrance",
		Details: map[string]any{
			"description": "dark room",
		},
	}

	result, err := llm.Narrate(context.Background(), event, model.GameState{})
	require.NoError(t, err)
	assert.Contains(t, result, "You enter entrance")
}

func TestLLMNarrator_FallbackOnInferenceError(t *testing.T) {
	llm, srv := newLLMNarrator(nil) // 503
	defer srv.Close()

	event := model.EngineEvent{
		Type: "moved",
		Room: "entrance",
		Details: map[string]any{
			"description": "dark room",
		},
	}

	result, err := llm.Narrate(context.Background(), event, model.GameState{})
	require.NoError(t, err)
	assert.Contains(t, result, "You enter entrance")
}

func TestLLMNarrator_PropagatesContextCancellation(t *testing.T) {
	llm, srv := newLLMNarrator([]string{"ok"})
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

	_, err := llm.Narrate(ctx, event, model.GameState{})
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestLLMNarrator_SendsCorrectPromptAndOptions(t *testing.T) {
	llm, srv := newLLMNarrator([]string{"You stand in a dark room."})
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

	_, err := llm.Narrate(context.Background(), event, model.GameState{})
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
	assert.Contains(t, msg0.Content, "Text adventure narrator")
	assert.Equal(t, "user", msg1.Role)
	assert.Contains(t, msg1.Content, "Room: goblin_lair")
	assert.Contains(t, msg1.Content, "filthy cave, bones on floor")

	// Temperature 0.7 for creative narration.
	require.NotNil(t, calls[0].Temperature)
	assert.InDelta(t, 0.7, *calls[0].Temperature, 0.001)

	// MaxTokens 300 for LLM tier.
	assert.Equal(t, 300, calls[0].MaxTokens)
}
