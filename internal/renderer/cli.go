// Package renderer provides Renderer implementations.
package renderer

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/punt-labs/cryptd/internal/model"
)

// CLI renders to a writer (stdout) and reads input from a reader (stdin).
type CLI struct {
	out    io.Writer
	in     *bufio.Scanner
	events chan model.InputEvent
}

// NewCLI creates a CLIRenderer that writes to out and reads from in.
func NewCLI(out io.Writer, in io.Reader) *CLI {
	return &CLI{
		out:    out,
		in:     bufio.NewScanner(in),
		events: make(chan model.InputEvent, 1),
	}
}

// Render prints the current room and narration, then reads one line of input
// and sends it to the events channel.
func (c *CLI) Render(_ context.Context, state model.GameState, narration string) error {
	fmt.Fprintf(c.out, "\n[%s]\n", state.Dungeon.CurrentRoom)
	if narration != "" {
		fmt.Fprintln(c.out, narration)
	}
	fmt.Fprint(c.out, "> ")

	if c.in.Scan() {
		line := strings.TrimSpace(c.in.Text())
		c.events <- model.InputEvent{Type: "input", Payload: line}
	} else {
		c.events <- model.InputEvent{Type: "quit"}
	}
	return nil
}

// Events returns the channel on which InputEvents are sent.
func (c *CLI) Events() <-chan model.InputEvent {
	return c.events
}
