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

// Render prints the room header, status HUD, narration, and prompt.
// During combat, an enemy list with HP bars is shown. The HUD always
// displays the hero's HP/MP. All output fits within 80 columns.
func (c *CLI) Render(ctx context.Context, state model.GameState, narration string) error {
	c.startScanner(ctx)

	// Room header.
	if _, err := fmt.Fprintf(c.out, "\n[%s]\n", state.Dungeon.CurrentRoom); err != nil {
		return err
	}

	// Status HUD: HP/MP bar.
	if len(state.Party) > 0 {
		if _, err := fmt.Fprintln(c.out, formatHUD(state.Party[0])); err != nil {
			return err
		}
	}

	// Enemy list during combat.
	if state.Dungeon.Combat.Active {
		for _, enemy := range state.Dungeon.Combat.Enemies {
			if enemy.HP > 0 {
				if _, err := fmt.Fprintln(c.out, formatEnemyLine(enemy)); err != nil {
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
	_, err := fmt.Fprint(c.out, "> ")
	return err
}

// formatHUD returns a one-line status string like "HP 15/20 [████████░░] MP 3/5".
func formatHUD(char model.Character) string {
	hp := formatBar("HP", char.HP, char.MaxHP)
	if char.MaxMP > 0 {
		return hp + "  " + formatBar("MP", char.MP, char.MaxMP)
	}
	return hp
}

// formatBar renders "LABEL cur/max [████░░░░░░]" using a 10-character bar.
func formatBar(label string, cur, max int) string {
	const barWidth = 10
	if cur < 0 {
		cur = 0
	}
	filled := 0
	if max > 0 {
		filled = (cur * barWidth) / max
		if filled > barWidth {
			filled = barWidth
		}
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	return fmt.Sprintf("%s %d/%d [%s]", label, cur, max, bar)
}

// formatEnemyLine returns "  Goblin HP 8/8 [██████████]" for an enemy.
func formatEnemyLine(enemy model.EnemyInstance) string {
	return "  " + enemy.Name + " " + formatBar("HP", enemy.HP, enemy.MaxHP)
}

// Events returns the channel on which InputEvents are sent.
func (c *CLI) Events() <-chan model.InputEvent {
	return c.events
}
