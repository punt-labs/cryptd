// Package renderer provides Renderer implementations.
package renderer

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/ergochat/readline"
	"github.com/punt-labs/cryptd/internal/model"
	"golang.org/x/term"
)

// CLI renders to a writer (stdout) and reads input from a reader (stdin).
// When stdin is a terminal, readline provides line editing and command history.
// Otherwise (pipes, tests, autoplay), a plain bufio.Scanner is used.
type CLI struct {
	out         io.Writer
	in          io.Reader
	events      chan model.InputEvent
	readGate    chan struct{} // signals the reader goroutine that Render is done
	startOnce   sync.Once
	useReadline bool
}

// NewCLI creates a CLIRenderer that writes to out and reads from in.
func NewCLI(out io.Writer, in io.Reader) *CLI {
	return &CLI{
		out:      out,
		in:       in,
		events:   make(chan model.InputEvent, 1),
		readGate: make(chan struct{}, 1),
	}
}

// startReader launches the background input reader exactly once.
// Uses readline when stdin is a terminal, bufio.Scanner otherwise.
func (c *CLI) startReader(ctx context.Context) {
	c.startOnce.Do(func() {
		if f, ok := c.in.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
			c.useReadline = true
			c.startReadline(ctx)
		} else {
			c.startScanner(ctx)
		}
	})
}

// startReadline launches readline in a background goroutine.
// The goroutine waits on readGate before each Readline() call so output
// from Render completes before the prompt appears. A helper goroutine
// closes the readline instance when ctx is cancelled, unblocking any
// in-progress Readline() call.
func (c *CLI) startReadline(ctx context.Context) {
	rl, err := readline.NewFromConfig(&readline.Config{
		Prompt:          "> ",
		InterruptPrompt: "^C",
		EOFPrompt:       "quit",
		Stdin:           c.in,
		Stdout:          c.out,
		Stderr:          c.out,
	})
	if err != nil {
		c.useReadline = false
		c.startScanner(ctx)
		return
	}

	// Close readline on context cancellation to unblock Readline().
	go func() {
		<-ctx.Done()
		rl.Close()
	}()

	go func() {
		defer close(c.events)
		for {
			// Wait for Render to finish before showing the prompt.
			select {
			case <-c.readGate:
			case <-ctx.Done():
				return
			}

			line, err := rl.Readline()
			if err != nil {
				if errors.Is(err, readline.ErrInterrupt) || errors.Is(err, io.EOF) {
					select {
					case c.events <- model.InputEvent{Type: "quit"}:
					case <-ctx.Done():
					}
				} else {
					select {
					case c.events <- model.InputEvent{Type: "error", Payload: err.Error()}:
					case <-ctx.Done():
					}
				}
				return
			}
			line = strings.TrimSpace(line)
			if line == "" {
				// Re-signal the gate so readline loops immediately on empty input.
				select {
				case c.readGate <- struct{}{}:
				default:
				}
				continue
			}
			ev := model.InputEvent{Type: "input", Payload: line}
			select {
			case c.events <- ev:
			case <-ctx.Done():
				return
			}
		}
	}()
}

// startScanner launches a plain bufio.Scanner in a background goroutine.
func (c *CLI) startScanner(ctx context.Context) {
	scanner := bufio.NewScanner(c.in)
	go func() {
		defer close(c.events)
		for scanner.Scan() {
			ev := model.InputEvent{Type: "input", Payload: strings.TrimSpace(scanner.Text())}
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
}

// Render prints the room header, status HUD, narration, and prompt.
// During combat, an enemy list with HP bars is shown. The HUD displays
// the hero's HP bar, plus MP bar when MaxMP > 0.
// When readline is active, Render skips the prompt — readline displays its
// own prompt when the reader goroutine is unblocked via readGate.
func (c *CLI) Render(ctx context.Context, state model.GameState, narration string) error {
	c.startReader(ctx)

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

	if c.useReadline {
		// Signal the readline goroutine that output is done.
		select {
		case c.readGate <- struct{}{}:
		default:
		}
	} else {
		if _, err := fmt.Fprint(c.out, "> "); err != nil {
			return err
		}
	}
	return nil
}

// formatHUD returns a one-line status string like "HP 15/20 [████████░░]  MP 3/5 [██████░░░░]".
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
