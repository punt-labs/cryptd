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

// CLI is a basic text renderer for `cryptd serve -t` testing mode.
// It writes plain text to out and reads lines from in via bufio.Scanner.
type CLI struct {
	out       io.Writer
	in        io.Reader
	events    chan model.InputEvent
	startOnce sync.Once
}

// NewCLI creates a CLIRenderer that writes to out and reads from in.
func NewCLI(out io.Writer, in io.Reader) *CLI {
	return &CLI{
		out:    out,
		in:     in,
		events: make(chan model.InputEvent, 1),
	}
}

// startReader launches the background input reader exactly once.
func (c *CLI) startReader(ctx context.Context) {
	c.startOnce.Do(func() {
		scanner := bufio.NewScanner(c.in)
		go func() {
			defer close(c.events)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" {
					continue
				}
				ev := model.InputEvent{Type: "input", Payload: line}
				select {
				case c.events <- ev:
				case <-ctx.Done():
					return
				}
			}
			if err := scanner.Err(); err != nil {
				select {
				case c.events <- model.InputEvent{Type: "error", Payload: err.Error()}:
				case <-ctx.Done():
				}
			}
		}()
	})
}

// Render prints the room header, status line, narration, and prompt.
func (c *CLI) Render(ctx context.Context, state model.GameState, narration string) error {
	c.startReader(ctx)

	// Room header.
	if _, err := fmt.Fprintf(c.out, "\n[%s]\n", state.Dungeon.CurrentRoom); err != nil {
		return err
	}

	// Status line: simple HP (and MP if applicable).
	if len(state.Party) > 0 {
		if _, err := fmt.Fprintln(c.out, formatStatus(state.Party[0])); err != nil {
			return err
		}
	}

	// Enemy list during combat.
	if state.Dungeon.Combat.Active {
		for _, enemy := range state.Dungeon.Combat.Enemies {
			if enemy.HP > 0 {
				if _, err := fmt.Fprintf(c.out, "  %s HP: %d/%d\n", enemy.Name, enemy.HP, enemy.MaxHP); err != nil {
					return err
				}
			}
		}
	}

	// Narration.
	if narration != "" {
		if _, err := fmt.Fprintln(c.out, narration); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprint(c.out, "> "); err != nil {
		return err
	}
	return nil
}

// formatStatus returns a simple status string like "HP: 15/20  MP: 3/5".
func formatStatus(char model.Character) string {
	hp := fmt.Sprintf("HP: %d/%d", char.HP, char.MaxHP)
	if char.MaxMP > 0 {
		return hp + fmt.Sprintf("  MP: %d/%d", char.MP, char.MaxMP)
	}
	return hp
}

// Events returns the channel on which InputEvents are sent.
func (c *CLI) Events() <-chan model.InputEvent {
	return c.events
}
