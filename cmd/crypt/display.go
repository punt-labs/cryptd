package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/protocol"
)

const barWidth = 10

// formatBar returns a labelled bar like "HP 15/20 [████████░░]".
func formatBar(label string, cur, max int) string {
	if max <= 0 {
		return fmt.Sprintf("%s %d/%d [%s]", label, cur, max, strings.Repeat("░", barWidth))
	}
	filled := cur * barWidth / max
	if filled < 0 {
		filled = 0
	}
	if filled > barWidth {
		filled = barWidth
	}
	return fmt.Sprintf("%s %d/%d [%s%s]",
		label, cur, max,
		strings.Repeat("█", filled),
		strings.Repeat("░", barWidth-filled))
}

// formatHUD returns a compact status line for a hero character.
// Shows HP bar always; MP bar only when MaxMP > 0.
func formatHUD(char model.Character) string {
	s := formatBar("HP", char.HP, char.MaxHP)
	if char.MaxMP > 0 {
		s += "  " + formatBar("MP", char.MP, char.MaxMP)
	}
	return s
}

// formatEnemyLine returns an indented enemy status line.
func formatEnemyLine(enemy model.EnemyInstance) string {
	return "  " + formatBar(enemy.Name, enemy.HP, enemy.MaxHP)
}

// displayPlayResponse renders a typed PlayResponse with room header, HUD,
// enemy list, narration text, and death notice. Returns true if the server
// signaled quit.
func displayPlayResponse(out io.Writer, resp protocol.PlayResponse) bool {
	// Room header.
	if resp.State != nil && resp.State.Dungeon.CurrentRoom != "" {
		fmt.Fprintf(out, "\n[%s]\n", resp.State.Dungeon.CurrentRoom)
	}

	// Hero HUD.
	if resp.State != nil && len(resp.State.Party) > 0 {
		fmt.Fprintln(out, formatHUD(resp.State.Party[0]))
	}

	// Enemy list during combat.
	if resp.State != nil && resp.State.Dungeon.Combat.Active {
		for _, e := range resp.State.Dungeon.Combat.Enemies {
			if e.HP > 0 {
				fmt.Fprintln(out, formatEnemyLine(e))
			}
		}
	}

	// Narration text.
	if resp.Text != "" {
		fmt.Fprintln(out, resp.Text)
	}

	// Death notice.
	if resp.Dead {
		fmt.Fprintln(out, "\nYou have been slain. Start a new game with 'new <scenario> <name> <class>'.")
	}

	return resp.Quit
}
