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

func newLLM(responses []string) (*interpreter.LLM, *testutil.FakeSLMServer) {
	srv := testutil.NewFakeSLMServer(responses)
	client := inference.NewClientWithOpts(srv.URL(), "claude-sonnet-4-20250514",
		inference.WithAPIKey("sk-test"),
		inference.WithTimeout(5*time.Second),
	)
	return interpreter.NewLLM(client, interpreter.NewRules()), srv
}

func TestLLM_RulesOnlyForNoIDActions(t *testing.T) {
	llm, srv := newLLM(nil)
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
		{"quit", "quit"},
		{"defend", "defend"},
		{"flee", "flee"},
	}
	for _, tt := range tests {
		action, err := llm.Interpret(context.Background(), tt.input, model.GameState{})
		require.NoError(t, err)
		assert.Equal(t, tt.wantType, action.Type, "input: %q", tt.input)
	}
	assert.Empty(t, srv.Calls(), "LLM should not be called for actions without IDs")
}

func TestLLM_ItemActionsCallLLM(t *testing.T) {
	llm, srv := newLLM([]string{`{"type":"take","item_id":"short_sword"}`})
	defer srv.Close()

	state := model.GameState{
		Dungeon: model.DungeonState{
			CurrentRoom: "entrance",
			RoomState: map[string]model.RoomState{
				"entrance": {Items: []string{"short_sword"}},
			},
		},
	}
	action, err := llm.Interpret(context.Background(), "take sword", state)
	require.NoError(t, err)
	assert.Equal(t, "take", action.Type)
	assert.Equal(t, "short_sword", action.ItemID)
	assert.Len(t, srv.Calls(), 1)
}

func TestLLM_MovementViaNaturalLanguage(t *testing.T) {
	llm, srv := newLLM([]string{`{"type":"move","direction":"north"}`})
	defer srv.Close()

	action, err := llm.Interpret(context.Background(), "proceed northward", model.GameState{})
	require.NoError(t, err)
	assert.Equal(t, "move", action.Type)
	assert.Equal(t, "north", action.Direction)
}

func TestLLM_NilClientFallback(t *testing.T) {
	llm := interpreter.NewLLM(nil, interpreter.NewRules())

	// "grab the key" — rules knows "grab" as take, so it returns take with
	// item_id "key". With nil client, the LLM call fails and we fall back to
	// the rules result.
	action, err := llm.Interpret(context.Background(), "grab the key", model.GameState{})
	require.NoError(t, err)
	assert.Equal(t, "take", action.Type)
	assert.Equal(t, "key", action.ItemID)
}

func TestLLM_FallbackOnNetworkError(t *testing.T) {
	llm, srv := newLLM(nil) // no responses → 503
	defer srv.Close()

	action, err := llm.Interpret(context.Background(), "take sword", model.GameState{})
	require.NoError(t, err)
	assert.Equal(t, "take", action.Type)
	assert.Equal(t, "sword", action.ItemID, "should fall back to rules ID")
}

func TestLLM_FallbackOnJSONParseError(t *testing.T) {
	llm, srv := newLLM([]string{"this is not json"})
	defer srv.Close()

	action, err := llm.Interpret(context.Background(), "head to the northern passage", model.GameState{})
	require.NoError(t, err)
	assert.Equal(t, "unknown", action.Type)
}

func TestLLM_PropagatesContextCancellation(t *testing.T) {
	llm, srv := newLLM([]string{`{"type":"look"}`})
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := llm.Interpret(ctx, "what do I see?", model.GameState{})
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestLLM_SendsCorrectPromptAndOptions(t *testing.T) {
	llm, srv := newLLM([]string{`{"type":"take","item_id":"rusty_key"}`})
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
	_, err := llm.Interpret(context.Background(), "snatch the rusty key", state)
	require.NoError(t, err)

	calls := srv.Calls()
	require.Len(t, calls, 1)
	assert.Equal(t, "claude-sonnet-4-20250514", calls[0].Model)
	require.Len(t, calls[0].Messages, 2)

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
	assert.Contains(t, msg1.Content, "Player input: snatch the rusty key")

	// Temperature 0.0 for deterministic parsing.
	require.NotNil(t, calls[0].Temperature)
	assert.Equal(t, 0.0, *calls[0].Temperature)

	// MaxTokens 150 for LLM tier.
	assert.Equal(t, 150, calls[0].MaxTokens)
}

func TestLLM_SendsAuthHeader(t *testing.T) {
	llm, srv := newLLM([]string{`{"type":"look"}`})
	defer srv.Close()

	// "what's around me?" — rules returns unknown, LLM handles it.
	_, err := llm.Interpret(context.Background(), "what's around me?", model.GameState{})
	require.NoError(t, err)

	calls := srv.Calls()
	require.Len(t, calls, 1)
	assert.Equal(t, "Bearer sk-test", calls[0].AuthHeader)
}
