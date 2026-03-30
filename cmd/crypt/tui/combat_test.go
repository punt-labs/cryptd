package tui

import (
	"testing"

	"github.com/punt-labs/cryptd/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testCombat() model.CombatState {
	return model.CombatState{
		Active: true,
		Round:  2,
		Enemies: []model.EnemyInstance{
			{ID: "goblin-1", Name: "Goblin", HP: 4, MaxHP: 6},
			{ID: "goblin-2", Name: "Goblin Archer", HP: 3, MaxHP: 5},
		},
		TurnOrder:   []string{"hero", "goblin-1", "goblin-2"},
		CurrentTurn: 0,
	}
}

func TestCombat_InactiveReturnsEmpty(t *testing.T) {
	c := NewCombatOverlay(80)
	combat := model.CombatState{Active: false}
	assert.Equal(t, "", c.Render(combat, 80, 24))
}

func TestCombat_ActiveRendersEnemyBars(t *testing.T) {
	c := NewCombatOverlay(80)
	combat := testCombat()
	out := c.Render(combat, 80, 24)
	require.NotEmpty(t, out)
	assert.Contains(t, out, "Goblin")
	assert.Contains(t, out, "Goblin Archer")
	assert.Contains(t, out, "HP 4/6")
	assert.Contains(t, out, "HP 3/5")
}

func TestCombat_DeadEnemiesExcluded(t *testing.T) {
	c := NewCombatOverlay(80)
	combat := testCombat()
	combat.Enemies[0].HP = 0 // Goblin is dead
	out := c.Render(combat, 80, 24)
	require.NotEmpty(t, out)
	assert.Contains(t, out, "Goblin Archer")
	// "Goblin" without "Archer" still appears in "Goblin Archer", so check HP
	assert.NotContains(t, out, "HP 0/6")
}

func TestCombat_ActionHints(t *testing.T) {
	c := NewCombatOverlay(80)
	combat := testCombat()
	out := c.Render(combat, 80, 24)
	require.NotEmpty(t, out)
	assert.Contains(t, out, "A")
	assert.Contains(t, out, "ttack")
	assert.Contains(t, out, "D")
	assert.Contains(t, out, "efend")
	assert.Contains(t, out, "F")
	assert.Contains(t, out, "lee")
	assert.Contains(t, out, "U")
	assert.Contains(t, out, "se Item")
}

func TestCombat_HeroTurn(t *testing.T) {
	c := NewCombatOverlay(80)
	combat := testCombat()
	combat.CurrentTurn = 0 // hero's turn
	out := c.Render(combat, 80, 24)
	require.NotEmpty(t, out)
	assert.Contains(t, out, "Your turn")
}

func TestCombat_EnemyTurn(t *testing.T) {
	c := NewCombatOverlay(80)
	combat := testCombat()
	combat.CurrentTurn = 1 // goblin-1's turn
	out := c.Render(combat, 80, 24)
	require.NotEmpty(t, out)
	assert.Contains(t, out, "Enemy turn...")
}

func TestCombat_RoundNumber(t *testing.T) {
	c := NewCombatOverlay(80)
	combat := testCombat()
	combat.Round = 5
	out := c.Render(combat, 80, 24)
	require.NotEmpty(t, out)
	assert.Contains(t, out, "Round 5")
}

func TestIsHeroTurn(t *testing.T) {
	tests := []struct {
		name     string
		combat   model.CombatState
		expected bool
	}{
		{
			name:     "empty turn order",
			combat:   model.CombatState{TurnOrder: nil, CurrentTurn: 0},
			expected: false,
		},
		{
			name:     "hero first",
			combat:   model.CombatState{TurnOrder: []string{"hero", "goblin"}, CurrentTurn: 0},
			expected: true,
		},
		{
			name:     "enemy turn",
			combat:   model.CombatState{TurnOrder: []string{"hero", "goblin"}, CurrentTurn: 1},
			expected: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isHeroTurn(tt.combat))
		})
	}
}
