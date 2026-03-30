package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/punt-labs/cryptd/internal/model"
)

// CombatOverlay renders the combat panel as a centered overlay. Pure rendering.
type CombatOverlay struct{}

// NewCombatOverlay creates an overlay that centers within the given width.
func NewCombatOverlay(_ int) CombatOverlay {
	return CombatOverlay{}
}

// Render returns the overlay panel. Returns empty string if combat is not active.
func (c CombatOverlay) Render(combat model.CombatState, width, height int) string {
	if !combat.Active {
		return ""
	}

	var lines []string

	// Title
	lines = append(lines, StyleCombatTitle.Render(fmt.Sprintf("⚔ COMBAT — Round %d", combat.Round)))
	lines = append(lines, "")

	// Enemy HP bars (skip dead enemies)
	barWidth := 20
	for _, enemy := range combat.Enemies {
		if enemy.HP <= 0 {
			continue
		}
		bar := formatBar("HP", enemy.HP, enemy.MaxHP, barWidth+len(enemy.Name)+4,
			BarStyle(enemy.HP, enemy.MaxHP))
		lines = append(lines, enemy.Name+"  "+bar)
	}

	lines = append(lines, "")

	// Action hints: bracket letter in gold, rest in red
	hints := []struct {
		bracket string
		rest    string
	}{
		{"A", "ttack"},
		{"D", "efend"},
		{"F", "lee"},
		{"U", "se Item"},
	}
	var parts []string
	for _, h := range hints {
		parts = append(parts,
			"["+lipgloss.NewStyle().Foreground(ColorGold).Render(h.bracket)+"]"+
				lipgloss.NewStyle().Foreground(ColorRed).Render(h.rest))
	}
	lines = append(lines, strings.Join(parts, "  "))

	lines = append(lines, "")

	// Turn indicator
	if isHeroTurn(combat) {
		lines = append(lines, lipgloss.NewStyle().Foreground(ColorGold).Render("Your turn"))
	} else {
		lines = append(lines, StyleSystem.Render("Enemy turn..."))
	}

	content := strings.Join(lines, "\n")
	panel := StyleCombatOverlay.Render(content)

	// Constrain overlay width
	overlayWidth := 50
	if width-4 < overlayWidth {
		overlayWidth = width - 4
	}
	if overlayWidth < 20 {
		overlayWidth = 20
	}
	panel = lipgloss.NewStyle().Width(overlayWidth).Render(panel)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, panel)
}

func isHeroTurn(combat model.CombatState) bool {
	if len(combat.TurnOrder) == 0 {
		return false
	}
	idx := combat.CurrentTurn % len(combat.TurnOrder)
	return combat.TurnOrder[idx] == "hero"
}
