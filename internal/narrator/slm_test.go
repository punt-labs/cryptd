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
	assert.Contains(t, result, "Exits: north, south.")
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
	assert.Contains(t, result, "Exits: east, west.")
	assert.Contains(t, result, "You see: rusty key.")
}

func TestSLMNarrator_CombatStarted(t *testing.T) {
	slm, srv := newSLMNarrator([]string{"A snarling goblin leaps from the shadows, blade drawn."})
	defer srv.Close()

	event := model.EngineEvent{
		Type:    "combat_started",
		Details: map[string]any{"enemy_names": "goblin"},
	}

	result, err := slm.Narrate(context.Background(), event, model.GameState{})
	require.NoError(t, err)
	assert.Contains(t, result, "goblin")
	assert.Len(t, srv.Calls(), 1)
}

func TestSLMNarrator_CombatWon(t *testing.T) {
	slm, srv := newSLMNarrator([]string{"Silence falls as the last foe crumbles."})
	defer srv.Close()

	event := model.EngineEvent{Type: "combat_won", Details: map[string]any{}}

	result, err := slm.Narrate(context.Background(), event, model.GameState{})
	require.NoError(t, err)
	assert.Contains(t, result, "foe crumbles")
	assert.Len(t, srv.Calls(), 1)
}

func TestSLMNarrator_HeroDied(t *testing.T) {
	slm, srv := newSLMNarrator([]string{"Darkness closes in as your strength fades."})
	defer srv.Close()

	event := model.EngineEvent{Type: "hero_died", Details: map[string]any{}}

	result, err := slm.Narrate(context.Background(), event, model.GameState{})
	require.NoError(t, err)
	assert.Contains(t, result, "strength fades")
	assert.Len(t, srv.Calls(), 1)
}

func TestSLMNarrator_LevelUp(t *testing.T) {
	slm, srv := newSLMNarrator([]string{"Power surges through your veins."})
	defer srv.Close()

	event := model.EngineEvent{
		Type:    "level_up",
		Details: map[string]any{"level": 3, "hp_gain": 5},
	}

	result, err := slm.Narrate(context.Background(), event, model.GameState{})
	require.NoError(t, err)
	assert.Contains(t, result, "Power surges")
	assert.Contains(t, result, "(Level 3, +5 HP)")
	assert.Len(t, srv.Calls(), 1)
}

func TestSLMNarrator_Examined(t *testing.T) {
	slm, srv := newSLMNarrator([]string{"A battered blade with notches along its edge, clearly well-used."})
	defer srv.Close()

	event := model.EngineEvent{
		Type: "examined",
		Details: map[string]any{
			"item_name":   "short sword",
			"description": "a short sword with a worn grip",
		},
	}

	result, err := slm.Narrate(context.Background(), event, model.GameState{})
	require.NoError(t, err)
	assert.Contains(t, result, "battered blade")
	assert.Len(t, srv.Calls(), 1)
}

func TestSLMNarrator_ExaminedNoDescription(t *testing.T) {
	slm, srv := newSLMNarrator([]string{"should not be called"})
	defer srv.Close()

	event := model.EngineEvent{
		Type:    "examined",
		Details: map[string]any{"item_name": "key"},
	}

	result, err := slm.Narrate(context.Background(), event, model.GameState{})
	require.NoError(t, err)
	assert.Contains(t, result, "nothing special about the key")
	assert.Empty(t, srv.Calls())
}

func TestSLMNarrator_TacticalEventsDelegateToTemplate(t *testing.T) {
	slm, srv := newSLMNarrator([]string{"should not be called"})
	defer srv.Close()

	// These events are mechanical/tactical — always use template narrator.
	tacticalEvents := []string{
		"attack_hit", "attack_kill", "enemy_attacks", "defend",
		"flee_success", "flee_fail", "spell_damage", "spell_heal",
		"picked_up", "dropped", "equipped", "unequipped",
		"not_in_combat", "in_combat", "unknown_action", "quit",
	}
	for _, eventType := range tacticalEvents {
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

func TestSLMNarrator_HandlesAnySliceDetails(t *testing.T) {
	slm, srv := newSLMNarrator([]string{"A dank corridor stretches before you."})
	defer srv.Close()

	// Details with []any (as produced by JSON unmarshalling) instead of []string.
	event := model.EngineEvent{
		Type: "moved",
		Room: "corridor",
		Details: map[string]any{
			"description": "dank corridor",
			"exits":       []any{"north", "south"},
			"items":       []any{"torch"},
		},
	}

	result, err := slm.Narrate(context.Background(), event, model.GameState{})
	require.NoError(t, err)
	assert.Contains(t, result, "dank corridor")
	assert.Contains(t, result, "Exits: north, south.")
	assert.Contains(t, result, "You see: torch.")
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

func TestSLMNarrator_CombatStartedFallbackOnError(t *testing.T) {
	slm, srv := newSLMNarrator(nil) // 503
	defer srv.Close()

	event := model.EngineEvent{
		Type:    "combat_started",
		Details: map[string]any{"enemy_names": "goblin"},
	}

	result, err := slm.Narrate(context.Background(), event, model.GameState{})
	require.NoError(t, err)
	assert.Contains(t, result, "Combat begins")
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

func TestSLMNarrator_CombatStartedPrompt(t *testing.T) {
	slm, srv := newSLMNarrator([]string{"Steel rings against stone."})
	defer srv.Close()

	event := model.EngineEvent{
		Type:    "combat_started",
		Details: map[string]any{"enemy_names": "goblin, skeleton"},
	}

	_, err := slm.Narrate(context.Background(), event, model.GameState{})
	require.NoError(t, err)

	calls := srv.Calls()
	require.Len(t, calls, 1)

	var msg1 struct {
		Content string `json:"content"`
	}
	require.NoError(t, json.Unmarshal(calls[0].Messages[1], &msg1))
	assert.Contains(t, msg1.Content, "goblin, skeleton")
}
