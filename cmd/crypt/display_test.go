package main

import (
	"bytes"
	"testing"

	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/protocol"
	"github.com/stretchr/testify/assert"
)

func TestFormatBar(t *testing.T) {
	tests := []struct {
		name     string
		label    string
		cur, max int
		want     string
	}{
		{
			name:  "full",
			label: "HP",
			cur:   20, max: 20,
			want: "HP 20/20 [██████████]",
		},
		{
			name:  "half",
			label: "HP",
			cur:   10, max: 20,
			want: "HP 10/20 [█████░░░░░]",
		},
		{
			name:  "empty",
			label: "HP",
			cur:   0, max: 20,
			want: "HP 0/20 [░░░░░░░░░░]",
		},
		{
			name:  "over max clamps",
			label: "MP",
			cur:   25, max: 20,
			want: "MP 25/20 [██████████]",
		},
		{
			name:  "zero max",
			label: "MP",
			cur:   0, max: 0,
			want: "MP 0/0 [░░░░░░░░░░]",
		},
		{
			name:  "one tenth",
			label: "HP",
			cur:   2, max: 20,
			want: "HP 2/20 [█░░░░░░░░░]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatBar(tt.label, tt.cur, tt.max)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatHUD(t *testing.T) {
	t.Run("hp only", func(t *testing.T) {
		char := model.Character{HP: 15, MaxHP: 20, MP: 0, MaxMP: 0}
		got := formatHUD(char)
		assert.Equal(t, "HP 15/20 [███████░░░]", got)
		assert.NotContains(t, got, "MP")
	})

	t.Run("hp and mp", func(t *testing.T) {
		char := model.Character{HP: 15, MaxHP: 20, MP: 3, MaxMP: 5}
		got := formatHUD(char)
		assert.Contains(t, got, "HP 15/20 [███████░░░]")
		assert.Contains(t, got, "MP 3/5 [██████░░░░]")
	})
}

func TestFormatEnemyLine(t *testing.T) {
	enemy := model.EnemyInstance{Name: "Goblin", HP: 8, MaxHP: 8}
	got := formatEnemyLine(enemy)
	assert.Equal(t, "  Goblin 8/8 [██████████]", got)
}

func TestDisplayPlayResponse(t *testing.T) {
	t.Run("full response with combat", func(t *testing.T) {
		var buf bytes.Buffer
		resp := protocol.PlayResponse{
			Text: "You swing your sword!",
			State: &model.GameState{
				Party: []model.Character{
					{Name: "Hero", HP: 15, MaxHP: 20, MP: 3, MaxMP: 5},
				},
				Dungeon: model.DungeonState{
					CurrentRoom: "dark_cave",
					Combat: model.CombatState{
						Active: true,
						Enemies: []model.EnemyInstance{
							{Name: "Goblin", HP: 5, MaxHP: 8},
							{Name: "Dead Rat", HP: 0, MaxHP: 4},
						},
					},
				},
			},
		}

		quit := displayPlayResponse(&buf, resp)
		assert.False(t, quit)

		out := buf.String()
		assert.Contains(t, out, "[dark_cave]")
		assert.Contains(t, out, "HP 15/20")
		assert.Contains(t, out, "MP 3/5")
		assert.Contains(t, out, "Goblin 5/8")
		assert.NotContains(t, out, "Dead Rat")
		assert.Contains(t, out, "You swing your sword!")
	})

	t.Run("dead response", func(t *testing.T) {
		var buf bytes.Buffer
		resp := protocol.PlayResponse{
			Text: "The goblin strikes you down.",
			Dead: true,
			State: &model.GameState{
				Party: []model.Character{
					{Name: "Hero", HP: 0, MaxHP: 20},
				},
				Dungeon: model.DungeonState{CurrentRoom: "cave"},
			},
		}

		quit := displayPlayResponse(&buf, resp)
		assert.False(t, quit)
		assert.Contains(t, buf.String(), "You have been slain")
	})

	t.Run("quit response", func(t *testing.T) {
		var buf bytes.Buffer
		resp := protocol.PlayResponse{Quit: true}
		quit := displayPlayResponse(&buf, resp)
		assert.True(t, quit)
	})

	t.Run("text only no state", func(t *testing.T) {
		var buf bytes.Buffer
		resp := protocol.PlayResponse{Text: "Welcome!"}
		quit := displayPlayResponse(&buf, resp)
		assert.False(t, quit)
		assert.Contains(t, buf.String(), "Welcome!")
		assert.NotContains(t, buf.String(), "[")
	})
}
