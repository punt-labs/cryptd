package renderer_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/renderer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCLIRenderer_RenderWritesOutput(t *testing.T) {
	var out bytes.Buffer
	in := strings.NewReader("look\n")
	r := renderer.NewCLI(&out, in)

	state := model.GameState{
		Dungeon: model.DungeonState{CurrentRoom: "entrance"},
	}
	err := r.Render(context.Background(), state, "You stand in the entrance hall.")
	require.NoError(t, err)

	output := out.String()
	assert.Contains(t, output, "entrance")
	assert.Contains(t, output, "You stand in the entrance hall.")
}

func TestCLIRenderer_EventsChannelReceivesInput(t *testing.T) {
	var out bytes.Buffer
	in := strings.NewReader("go south\n")
	r := renderer.NewCLI(&out, in)

	state := model.GameState{}
	err := r.Render(context.Background(), state, "")
	require.NoError(t, err)

	ev := <-r.Events()
	assert.Equal(t, "input", ev.Type)
	assert.Equal(t, "go south", ev.Payload)
}

func TestCLIRenderer_EOFClosesEvents(t *testing.T) {
	var out bytes.Buffer
	in := strings.NewReader("") // immediate EOF
	r := renderer.NewCLI(&out, in)

	_ = r.Render(context.Background(), model.GameState{}, "")

	ev, ok := <-r.Events()
	// Channel should either be closed or emit a quit event.
	if ok {
		assert.Equal(t, "quit", ev.Type)
	}
}
