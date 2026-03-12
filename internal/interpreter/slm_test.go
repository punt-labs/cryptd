package interpreter_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/punt-labs/cryptd/internal/inference"
	"github.com/punt-labs/cryptd/internal/interpreter"
	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newSLM(responses []string) (*interpreter.SLM, *testutil.FakeSLMServer) {
	srv := testutil.NewFakeSLMServer(responses)
	client := inference.NewClient(srv.URL(), "test-model", 5*time.Second)
	return interpreter.NewSLM(client, interpreter.NewRules()), srv
}

func TestSLM_RulesOnlyForNoIDActions(t *testing.T) {
	// Actions without IDs (move, look, inventory, help, quit) skip SLM entirely.
	slm, srv := newSLM(nil) // no SLM responses configured
	defer srv.Close()

	tests := []struct {
		input    string
		wantType string
	}{
		{"go north", "move"},
		{"n", "move"},
		{"l", "look"},
		{"look", "look"},
		{"i", "inventory"},
		{"inventory", "inventory"},
		{"?", "help"},
		{"help", "help"},
		{"quit", "quit"},
		{"defend", "defend"},
		{"flee", "flee"},
	}
	for _, tt := range tests {
		action, err := slm.Interpret(context.Background(), tt.input, model.GameState{})
		require.NoError(t, err)
		assert.Equal(t, tt.wantType, action.Type, "input: %q", tt.input)
	}
	assert.Empty(t, srv.Calls(), "SLM should not be called for actions without IDs")
}

func TestSLM_ItemActionsCallSLM(t *testing.T) {
	// "take sword" — rules parses it but SLM is called to resolve the item ID.
	slm, srv := newSLM([]string{`{"type":"take","item_id":"short_sword"}`})
	defer srv.Close()

	state := model.GameState{
		Dungeon: model.DungeonState{
			CurrentRoom: "entrance",
			RoomState: map[string]model.RoomState{
				"entrance": {Items: []string{"short_sword"}},
			},
		},
	}
	action, err := slm.Interpret(context.Background(), "take sword", state)
	require.NoError(t, err)
	assert.Equal(t, "take", action.Type)
	assert.Equal(t, "short_sword", action.ItemID, "SLM should resolve partial name")
	assert.Len(t, srv.Calls(), 1, "SLM should be called for item actions")
}

func TestSLM_ItemActionFallsBackToRulesOnSLMFailure(t *testing.T) {
	// SLM fails — fall back to rules result.
	slm, srv := newSLM(nil) // no responses → 503
	defer srv.Close()

	action, err := slm.Interpret(context.Background(), "take sword", model.GameState{})
	require.NoError(t, err)
	assert.Equal(t, "take", action.Type)
	assert.Equal(t, "sword", action.ItemID, "should fall back to rules ID")
}

func TestSLM_FallsToSLMForNaturalLanguage(t *testing.T) {
	slm, srv := newSLM([]string{`{"type":"move","direction":"north"}`})
	defer srv.Close()

	// "proceed northward" — rules returns unknown, SLM handles it.
	action, err := slm.Interpret(context.Background(), "proceed northward", model.GameState{})
	require.NoError(t, err)
	assert.Equal(t, "move", action.Type)
	assert.Equal(t, "north", action.Direction)
	assert.Len(t, srv.Calls(), 1, "SLM should be called for natural language")
}

func TestSLM_ContextInjectedInUserMessage(t *testing.T) {
	slm, srv := newSLM([]string{`{"type":"take","item_id":"rusty_key"}`})
	defer srv.Close()

	state := model.GameState{
		Dungeon: model.DungeonState{
			CurrentRoom: "entrance",
			RoomState: map[string]model.RoomState{
				"entrance": {Items: []string{"rusty_key"}},
			},
			Exits: []string{"south"},
		},
	}
	// "snag that key" — rules doesn't know "snag", so it goes to SLM.
	action, err := slm.Interpret(context.Background(), "snag that key", state)
	require.NoError(t, err)
	assert.Equal(t, "take", action.Type)
	assert.Equal(t, "rusty_key", action.ItemID)

	// Verify context was sent.
	calls := srv.Calls()
	require.Len(t, calls, 1)
	var msg struct{ Content string }
	require.NoError(t, json.Unmarshal(calls[0].Messages[1], &msg))
	assert.Contains(t, msg.Content, "Room: entrance")
	assert.Contains(t, msg.Content, "Items here: rusty_key")
	assert.Contains(t, msg.Content, "Player input: snag that key")
}

func TestSLM_CastSpellViaSLM(t *testing.T) {
	// "hurl a fireball at the goblin" — rules can't parse it, SLM handles it.
	slm, srv := newSLM([]string{`{"type":"cast","spell_id":"fireball","target":"goblin_0"}`})
	defer srv.Close()

	action, err := slm.Interpret(context.Background(), "hurl a fireball at the goblin", model.GameState{})
	require.NoError(t, err)
	assert.Equal(t, "cast", action.Type)
	assert.Equal(t, "fireball", action.SpellID)
	assert.Equal(t, "goblin_0", action.Target)
}

func TestSLM_ReturnsUnknownOnSLMFailure(t *testing.T) {
	// Empty responses causes FakeSLMServer to return 503.
	// "head to the northern passage" — rules returns unknown, SLM fails too.
	slm, srv := newSLM(nil)
	defer srv.Close()

	action, err := slm.Interpret(context.Background(), "head to the northern passage", model.GameState{})
	require.NoError(t, err)
	assert.Equal(t, "unknown", action.Type)
}

func TestSLM_ReturnsUnknownOnMalformedJSON(t *testing.T) {
	slm, srv := newSLM([]string{"this is not json at all"})
	defer srv.Close()

	// "what do I see around me?" — rules returns unknown, SLM returns garbage.
	action, err := slm.Interpret(context.Background(), "what do I see around me?", model.GameState{})
	require.NoError(t, err)
	assert.Equal(t, "unknown", action.Type)
}

func TestSLM_ReturnsUnknownOnInvalidActionType(t *testing.T) {
	slm, srv := newSLM([]string{`{"type":"dance"}`})
	defer srv.Close()

	action, err := slm.Interpret(context.Background(), "dance with the moon", model.GameState{})
	require.NoError(t, err)
	assert.Equal(t, "unknown", action.Type)
}

func TestSLM_StripsMarkdownFences(t *testing.T) {
	slm, srv := newSLM([]string{"```json\n{\"type\":\"move\",\"direction\":\"south\"}\n```"})
	defer srv.Close()

	action, err := slm.Interpret(context.Background(), "head south", model.GameState{})
	require.NoError(t, err)
	assert.Equal(t, "move", action.Type)
	assert.Equal(t, "south", action.Direction)
}

func TestSLM_SendsCorrectPrompt(t *testing.T) {
	slm, srv := newSLM([]string{`{"type":"take","item_id":"rusty_key"}`})
	defer srv.Close()

	// "snatch the rusty key" — rules doesn't know "snatch", goes to SLM.
	state := model.GameState{
		Dungeon: model.DungeonState{
			CurrentRoom: "entrance",
			RoomState: map[string]model.RoomState{
				"entrance": {Items: []string{"rusty_key"}},
			},
			Exits: []string{"south"},
		},
	}
	_, err := slm.Interpret(context.Background(), "snatch the rusty key", state)
	require.NoError(t, err)

	calls := srv.Calls()
	require.Len(t, calls, 1)
	assert.Equal(t, "test-model", calls[0].Model)
	require.Len(t, calls[0].Messages, 2)

	// Verify message roles and contents.
	var msg0, msg1 struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	require.NoError(t, json.Unmarshal(calls[0].Messages[0], &msg0))
	require.NoError(t, json.Unmarshal(calls[0].Messages[1], &msg1))
	assert.Equal(t, "system", msg0.Role)
	assert.Contains(t, msg0.Content, "text adventure command parser")
	assert.Equal(t, "user", msg1.Role)
	assert.Contains(t, msg1.Content, "Room: entrance")
	assert.Contains(t, msg1.Content, "Items here: rusty_key")
	assert.Contains(t, msg1.Content, "Player input: snatch the rusty key")

	// Verify temperature is set to 0 for deterministic parsing.
	require.NotNil(t, calls[0].Temperature)
	assert.Equal(t, 0.0, *calls[0].Temperature)
}

func TestSLM_PropagatesContextCancellation(t *testing.T) {
	slm, srv := newSLM([]string{`{"type":"look"}`})
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	// Should propagate cancellation, not fall back.
	_, err := slm.Interpret(ctx, "what do I see?", model.GameState{})
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}
