package narrator_test

import (
	"context"
	"testing"

	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/narrator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateNarrator_AllEventTypes(t *testing.T) {
	n := narrator.NewTemplate()
	state := model.GameState{}

	events := []string{"moved", "looked", "unknown_action", "quit", "locked_door", "no_exit"}
	for _, evType := range events {
		t.Run(evType, func(t *testing.T) {
			text, err := n.Narrate(context.Background(), model.EngineEvent{Type: evType}, state)
			require.NoError(t, err)
			assert.NotEmpty(t, text, "expected non-empty narration for event %q", evType)
		})
	}
}

func TestTemplateNarrator_MovedContainsRoom(t *testing.T) {
	n := narrator.NewTemplate()
	text, err := n.Narrate(context.Background(), model.EngineEvent{
		Type: "moved",
		Room: "goblin_lair",
	}, model.GameState{})
	require.NoError(t, err)
	assert.Contains(t, text, "goblin_lair")
}

func TestTemplateNarrator_LookedWithRoomName(t *testing.T) {
	n := narrator.NewTemplate()
	text, err := n.Narrate(context.Background(), model.EngineEvent{
		Type: "looked",
		Room: "entrance",
	}, model.GameState{})
	require.NoError(t, err)
	assert.Contains(t, text, "entrance")
}

func TestTemplateNarrator_UnknownEventFallback(t *testing.T) {
	n := narrator.NewTemplate()
	text, err := n.Narrate(context.Background(), model.EngineEvent{Type: "some_future_event"}, model.GameState{})
	require.NoError(t, err)
	assert.NotEmpty(t, text)
}
