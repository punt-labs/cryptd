// Package renderer provides Renderer implementations.
package renderer

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/punt-labs/cryptd/internal/model"
)

// CLI renders to a writer (stdout) and reads input from a reader (stdin).
// Stdin scanning runs in a dedicated goroutine so Render never blocks on input.
type CLI struct {
	out       io.Writer
	inScanner *bufio.Scanner
	events    chan model.InputEvent
	startOnce sync.Once
}

// NewCLI creates a CLIRenderer that writes to out and reads from in.
func NewCLI(out io.Writer, in io.Reader) *CLI {
	return &CLI{
		out:       out,
		inScanner: bufio.NewScanner(in),
		events:    make(chan model.InputEvent, 1),
	}
}

// startScanner launches the background stdin reader exactly once.
// The goroutine stops sending when ctx is cancelled. The underlying reader
// is NOT closed — callers are responsible for its lifetime.
func (c *CLI) startScanner(ctx context.Context) {
	c.startOnce.Do(func() {
		go func() {
			defer close(c.events)
			for c.inScanner.Scan() {
				ev := model.InputEvent{Type: "input", Payload: strings.TrimSpace(c.inScanner.Text())}
				select {
				case c.events <- ev:
				case <-ctx.Done():
					return
				}
			}
			if err := c.inScanner.Err(); err != nil {
				select {
				case c.events <- model.InputEvent{Type: "error", Payload: err.Error()}:
				case <-ctx.Done():
				}
			}
		}()
	})
}

// Render prints the current room ID and narration, then shows a prompt.
// Input arrives asynchronously via Events(). The ctx passed to the first call
// to Render governs the scanner goroutine lifetime.
func (c *CLI) Render(ctx context.Context, state model.GameState, narration string) error {
	c.startScanner(ctx)
	if _, err := fmt.Fprintf(c.out, "\n[%s]\n", state.Dungeon.CurrentRoom); err != nil {
		return err
	}
	if narration != "" {
		if _, err := fmt.Fprintln(c.out, narration); err != nil {
			return err
		}
	}
	_, err := fmt.Fprint(c.out, "> ")
	return err
}

// Events returns the channel on which InputEvents are sent.
func (c *CLI) Events() <-chan model.InputEvent {
	return c.events
}
