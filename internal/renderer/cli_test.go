package renderer_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

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

	// Goroutine reads input asynchronously; wait with a short timeout.
	select {
	case ev := <-r.Events():
		assert.Equal(t, "input", ev.Type)
		assert.Equal(t, "go south", ev.Payload)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for input event")
	}
}

func TestCLIRenderer_EOFClosesEvents(t *testing.T) {
	var out bytes.Buffer
	in := strings.NewReader("") // immediate EOF
	r := renderer.NewCLI(&out, in)

	_ = r.Render(context.Background(), model.GameState{}, "")

	// With an empty reader the goroutine closes the channel on EOF.
	select {
	case ev, ok := <-r.Events():
		if ok {
			assert.Equal(t, "quit", ev.Type)
		}
		// ok==false means channel closed — that's the expected path.
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for events channel to close")
	}
}

func TestCLIRenderer_ContextCancelStopsScanner(t *testing.T) {
	// Use a pipe so Scan() can block. We write a line, then cancel ctx so the
	// goroutine exits via ctx.Done() on the send select rather than blocking forever.
	pr, pw := io.Pipe()
	defer pr.Close()
	defer pw.Close()

	var out bytes.Buffer
	r := renderer.NewCLI(&out, pr)

	ctx, cancel := context.WithCancel(context.Background())
	_ = r.Render(ctx, model.GameState{}, "")

	// Write a line so the scanner goroutine unblocks from Scan(), then cancel
	// so the send is dropped and the goroutine exits.
	_, _ = fmt.Fprintln(pw, "some input")
	cancel()

	// After cancellation the scanner goroutine must exit and close events.
	select {
	case <-r.Events():
		// event received or channel closed — either is acceptable
	case <-time.After(time.Second):
		t.Fatal("timed out: scanner goroutine did not exit after context cancellation")
	}
}

// errWriter always returns an error on write.
type errWriter struct{ err error }

func (w *errWriter) Write([]byte) (int, error) { return 0, w.err }

func TestCLIRenderer_RenderReturnsWriteError(t *testing.T) {
	ew := &errWriter{err: fmt.Errorf("write failed")}
	in := strings.NewReader("")
	r := renderer.NewCLI(ew, in)

	err := r.Render(context.Background(), model.GameState{Dungeon: model.DungeonState{CurrentRoom: "entrance"}}, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write failed")
}
