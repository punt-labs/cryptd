package renderer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/punt-labs/cryptd/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newReadlineCLI creates a CLI and forces the readline path (bypassing
// terminal detection). The caller must signal readGate to unblock the
// readline goroutine before each input line.
func newReadlineCLI(out io.Writer, in io.Reader) *CLI {
	c := &CLI{
		out:         out,
		in:          in,
		events:      make(chan model.InputEvent, 1),
		readGate:    make(chan struct{}, 1),
		useReadline: true,
	}
	return c
}

func TestReadline_InputEvent(t *testing.T) {
	pr, pw := io.Pipe()
	defer pr.Close()
	defer pw.Close()

	var out bytes.Buffer
	c := newReadlineCLI(&out, pr)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c.startReadline(ctx)

	// Signal readGate so readline can prompt.
	c.readGate <- struct{}{}

	_, err := fmt.Fprintln(pw, "look around")
	require.NoError(t, err)

	select {
	case ev := <-c.events:
		assert.Equal(t, "input", ev.Type)
		assert.Equal(t, "look around", ev.Payload)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for input event")
	}
}

func TestReadline_EOFSendsQuit(t *testing.T) {
	pr, pw := io.Pipe()

	var out bytes.Buffer
	c := newReadlineCLI(&out, pr)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c.startReadline(ctx)

	// Signal readGate then close write end to trigger EOF.
	c.readGate <- struct{}{}
	pw.Close()

	select {
	case ev, ok := <-c.events:
		if ok {
			assert.Equal(t, "quit", ev.Type)
		}
		// Channel closed is also acceptable.
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for quit event")
	}
}

func TestReadline_ContextCancelUnblocks(t *testing.T) {
	pr, pw := io.Pipe()
	defer pr.Close()
	defer pw.Close()

	var out bytes.Buffer
	c := newReadlineCLI(&out, pr)

	ctx, cancel := context.WithCancel(context.Background())
	c.startReadline(ctx)

	// Signal readGate so readline blocks on Readline().
	c.readGate <- struct{}{}

	// Cancel context — should unblock Readline() via rl.Close().
	cancel()

	// Events channel must close.
	select {
	case _, ok := <-c.events:
		if ok {
			// Drain any event, then wait for close.
			select {
			case <-c.events:
			case <-time.After(2 * time.Second):
				t.Fatal("events channel did not close after cancel")
			}
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for events channel to close")
	}
}

func TestReadline_EmptyLineReSignalsGate(t *testing.T) {
	pr, pw := io.Pipe()
	defer pr.Close()
	defer pw.Close()

	var out bytes.Buffer
	c := newReadlineCLI(&out, pr)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c.startReadline(ctx)

	// Signal readGate, send empty line, then a real command.
	c.readGate <- struct{}{}

	_, err := fmt.Fprintln(pw, "")
	require.NoError(t, err)

	// The empty line should re-signal readGate internally.
	// Send a real line — the goroutine should read it without another external gate signal.
	_, err = fmt.Fprintln(pw, "attack goblin")
	require.NoError(t, err)

	select {
	case ev := <-c.events:
		assert.Equal(t, "input", ev.Type)
		assert.Equal(t, "attack goblin", ev.Payload)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for input event after empty line")
	}
}
