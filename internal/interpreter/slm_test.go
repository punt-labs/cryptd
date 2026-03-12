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

func TestSLM_MoveNorth(t *testing.T) {
	slm, srv := newSLM([]string{`{"type":"move","direction":"north"}`})
	defer srv.Close()

	action, err := slm.Interpret(context.Background(), "I want to go to the northern passage", model.GameState{})
	require.NoError(t, err)
	assert.Equal(t, "move", action.Type)
	assert.Equal(t, "north", action.Direction)
}

func TestSLM_Look(t *testing.T) {
	slm, srv := newSLM([]string{`{"type":"look"}`})
	defer srv.Close()

	action, err := slm.Interpret(context.Background(), "what do I see around me?", model.GameState{})
	require.NoError(t, err)
	assert.Equal(t, "look", action.Type)
}

func TestSLM_TakeItem(t *testing.T) {
	slm, srv := newSLM([]string{`{"type":"take","item_id":"rusty_key"}`})
	defer srv.Close()

	action, err := slm.Interpret(context.Background(), "pick up the rusty key from the floor", model.GameState{})
	require.NoError(t, err)
	assert.Equal(t, "take", action.Type)
	assert.Equal(t, "rusty_key", action.ItemID)
}

func TestSLM_Attack(t *testing.T) {
	slm, srv := newSLM([]string{`{"type":"attack","target":"goblin_0"}`})
	defer srv.Close()

	action, err := slm.Interpret(context.Background(), "hit the goblin with my sword", model.GameState{})
	require.NoError(t, err)
	assert.Equal(t, "attack", action.Type)
	assert.Equal(t, "goblin_0", action.Target)
}

func TestSLM_CastSpell(t *testing.T) {
	slm, srv := newSLM([]string{`{"type":"cast","spell_id":"fireball","target":"goblin_0"}`})
	defer srv.Close()

	action, err := slm.Interpret(context.Background(), "cast fireball at the goblin", model.GameState{})
	require.NoError(t, err)
	assert.Equal(t, "cast", action.Type)
	assert.Equal(t, "fireball", action.SpellID)
	assert.Equal(t, "goblin_0", action.Target)
}

func TestSLM_SaveLoad(t *testing.T) {
	slm, srv := newSLM([]string{`{"type":"save","target":"slot1"}`, `{"type":"load","target":"slot1"}`})
	defer srv.Close()

	action, err := slm.Interpret(context.Background(), "save my game to slot1", model.GameState{})
	require.NoError(t, err)
	assert.Equal(t, "save", action.Type)
	assert.Equal(t, "slot1", action.Target)

	action, err = slm.Interpret(context.Background(), "load slot1", model.GameState{})
	require.NoError(t, err)
	assert.Equal(t, "load", action.Type)
	assert.Equal(t, "slot1", action.Target)
}

func TestSLM_FallbackOnInferenceError(t *testing.T) {
	// Empty responses causes FakeSLMServer to return 503.
	slm, srv := newSLM(nil)
	defer srv.Close()

	// Should fall back to RulesInterpreter, which understands "go north".
	action, err := slm.Interpret(context.Background(), "go north", model.GameState{})
	require.NoError(t, err)
	assert.Equal(t, "move", action.Type)
	assert.Equal(t, "north", action.Direction)
}

func TestSLM_FallbackOnMalformedJSON(t *testing.T) {
	slm, srv := newSLM([]string{"this is not json at all"})
	defer srv.Close()

	// Should fall back to RulesInterpreter.
	action, err := slm.Interpret(context.Background(), "look", model.GameState{})
	require.NoError(t, err)
	assert.Equal(t, "look", action.Type)
}

func TestSLM_FallbackOnUnknownActionType(t *testing.T) {
	slm, srv := newSLM([]string{`{"type":"dance"}`})
	defer srv.Close()

	// "dance" is not a valid action type — falls back to rules.
	action, err := slm.Interpret(context.Background(), "look", model.GameState{})
	require.NoError(t, err)
	assert.Equal(t, "look", action.Type)
}

func TestSLM_FallbackOnMissingRequiredFields(t *testing.T) {
	for _, tt := range []struct {
		name     string
		response string
		input    string
	}{
		{"move without direction", `{"type":"move"}`, "go north"},
		{"move with invalid direction", `{"type":"move","direction":"sideways"}`, "go north"},
		{"take without item_id", `{"type":"take"}`, "take sword"},
		{"cast without spell_id", `{"type":"cast","target":"goblin_0"}`, "cast fireball"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			slm, srv := newSLM([]string{tt.response})
			defer srv.Close()

			// Should fall back to RulesInterpreter.
			action, err := slm.Interpret(context.Background(), tt.input, model.GameState{})
			require.NoError(t, err)
			assert.NotEqual(t, "", action.Type)
		})
	}
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
	slm, srv := newSLM([]string{`{"type":"look"}`})
	defer srv.Close()

	_, err := slm.Interpret(context.Background(), "look around", model.GameState{})
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
	assert.Equal(t, "look around", msg1.Content)

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
	_, err := slm.Interpret(ctx, "look", model.GameState{})
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}
