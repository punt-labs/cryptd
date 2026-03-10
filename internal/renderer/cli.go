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
// Stdin scanning runs in a dedicated goroutine so Render never blocks on
// input, and context cancellation in the game loop is honoured promptly.
type CLI struct {
	out       io.Writer
	in        *bufio.Scanner
	events    chan model.InputEvent
	startOnce sync.Once
}

// NewCLI creates a CLIRenderer that writes to out and reads from in.
func NewCLI(out io.Writer, in io.Reader) *CLI {
	return &CLI{
		out:    out,
		in:     bufio.NewScanner(in),
		events: make(chan model.InputEvent, 1),
	}
}

// startScanner launches the background stdin reader exactly once.
func (c *CLI) startScanner() {
	c.startOnce.Do(func() {
		go func() {
			for c.in.Scan() {
				c.events <- model.InputEvent{Type: "input", Payload: strings.TrimSpace(c.in.Text())}
			}
			close(c.events)
		}()
	})
}

// Render prints the current room description and narration, then shows a
// prompt. Input arrives asynchronously via Events().
func (c *CLI) Render(_ context.Context, state model.GameState, narration string) error {
	c.startScanner()
	fmt.Fprintf(c.out, "\n[%s]\n", state.Dungeon.CurrentRoom)
	if narration != "" {
		fmt.Fprintln(c.out, narration)
	}
	fmt.Fprint(c.out, "> ")
	return nil
}

// Events returns the channel on which InputEvents are sent.
func (c *CLI) Events() <-chan model.InputEvent {
	return c.events
}
