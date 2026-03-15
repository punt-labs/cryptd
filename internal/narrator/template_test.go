package narrator_test

import (
	"context"
	"testing"

	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/narrator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateNarrator_AllEventTypes(t *testing.T) {
	n := narrator.NewTemplate()
	state := model.GameState{}

	events := []string{
		"moved", "looked", "unknown_action", "quit", "locked_door", "no_exit",
		"picked_up", "dropped", "equipped", "unequipped", "examined",
		"inventory_listed", "item_not_found", "not_in_inventory",
		"too_heavy", "slot_occupied", "slot_empty", "not_equippable",
		"help",
	}
	for _, evType := range events {
		t.Run(evType, func(t *testing.T) {
			text, err := n.Narrate(context.Background(), model.EngineEvent{Type: evType}, state)
			require.NoError(t, err)
			assert.NotEmpty(t, text, "expected non-empty narration for event %q", evType)
		})
	}
}

func TestTemplateNarrator_MovedContainsRoom(t *testing.T) {
	n := narrator.NewTemplate()
	text, err := n.Narrate(context.Background(), model.EngineEvent{
		Type: "moved",
		Room: "goblin_lair",
	}, model.GameState{})
	require.NoError(t, err)
	assert.Contains(t, text, "goblin_lair")
}

func TestTemplateNarrator_LookedWithRoomName(t *testing.T) {
	n := narrator.NewTemplate()
	text, err := n.Narrate(context.Background(), model.EngineEvent{
		Type: "looked",
		Room: "entrance",
	}, model.GameState{})
	require.NoError(t, err)
	assert.Contains(t, text, "entrance")
}

func TestTemplateNarrator_MovedWithDescriptionExitsItems(t *testing.T) {
	n := narrator.NewTemplate()
	text, err := n.Narrate(context.Background(), model.EngineEvent{
		Type: "moved",
		Room: "The Cave",
		Details: map[string]any{
			"description": "A dark cave with dripping water.",
			"exits":       []string{"north", "south"},
			"items":       []string{"Rusty Sword", "Old Shield"},
		},
	}, model.GameState{})
	require.NoError(t, err)
	assert.Contains(t, text, "The Cave")
	assert.Contains(t, text, "A dark cave with dripping water.")
	assert.Contains(t, text, "Exits: north, south.")
	assert.Contains(t, text, "You see: Rusty Sword, Old Shield.")
}

func TestTemplateNarrator_LookedWithDescriptionExitsItems(t *testing.T) {
	n := narrator.NewTemplate()
	text, err := n.Narrate(context.Background(), model.EngineEvent{
		Type: "looked",
		Room: "Entrance Hall",
		Details: map[string]any{
			"description": "A grand entrance with marble columns.",
			"exits":       []string{"east"},
			"items":       []string{"Torch"},
		},
	}, model.GameState{})
	require.NoError(t, err)
	assert.Contains(t, text, "Entrance Hall")
	assert.Contains(t, text, "A grand entrance with marble columns.")
	assert.Contains(t, text, "Exits: east.")
	assert.Contains(t, text, "You see: Torch.")
}

func TestTemplateNarrator_MovedWithoutDetails(t *testing.T) {
	n := narrator.NewTemplate()
	text, err := n.Narrate(context.Background(), model.EngineEvent{
		Type: "moved",
		Room: "Empty Room",
	}, model.GameState{})
	require.NoError(t, err)
	assert.Contains(t, text, "Empty Room")
	assert.NotContains(t, text, "Exits:")
	assert.NotContains(t, text, "You see:")
}

func TestTemplateNarrator_UnknownEventFallback(t *testing.T) {
	n := narrator.NewTemplate()
	text, err := n.Narrate(context.Background(), model.EngineEvent{Type: "some_future_event"}, model.GameState{})
	require.NoError(t, err)
	assert.NotEmpty(t, text)
}

func TestTemplateNarrator_PickedUpWithName(t *testing.T) {
	n := narrator.NewTemplate()
	text, err := n.Narrate(context.Background(), model.EngineEvent{
		Type:    "picked_up",
		Details: map[string]any{"item_name": "Rusty Key"},
	}, model.GameState{})
	require.NoError(t, err)
	assert.Contains(t, text, "Rusty Key")
}

func TestTemplateNarrator_ExaminedWithDescription(t *testing.T) {
	n := narrator.NewTemplate()
	text, err := n.Narrate(context.Background(), model.EngineEvent{
		Type:    "examined",
		Details: map[string]any{"description": "A blade of fine steel.", "item_name": "Sword"},
	}, model.GameState{})
	require.NoError(t, err)
	assert.Equal(t, "A blade of fine steel.", text)
}

func TestTemplateNarrator_CombatEvents(t *testing.T) {
	n := narrator.NewTemplate()
	state := model.GameState{}

	events := []string{
		"combat_started", "attack_hit", "attack_kill",
		"enemy_attacks", "enemy_flees", "defend",
		"flee_success", "flee_fail", "combat_won",
		"hero_died", "not_in_combat", "in_combat",
		"not_hero_turn", "invalid_target",
		"spell_damage", "spell_heal",
		"unknown_spell", "not_caster", "insufficient_mp",
		"level_up",
		"game_saved", "game_loaded", "save_error", "load_error",
	}
	for _, evType := range events {
		t.Run(evType, func(t *testing.T) {
			text, err := n.Narrate(context.Background(), model.EngineEvent{
				Type: evType,
				Details: map[string]any{
					"target":      "Goblin",
					"damage":      5,
					"xp":          8,
					"enemy":       "Goblin",
					"enemy_names": "Goblin",
					"level":       2,
					"hp_gain":     8,
				},
			}, state)
			require.NoError(t, err)
			assert.NotEmpty(t, text, "expected non-empty narration for event %q", evType)
		})
	}
}

func TestTemplateNarrator_AttackHitContainsDetails(t *testing.T) {
	n := narrator.NewTemplate()
	text, err := n.Narrate(context.Background(), model.EngineEvent{
		Type:    "attack_hit",
		Details: map[string]any{"target": "Goblin", "damage": 5},
	}, model.GameState{})
	require.NoError(t, err)
	assert.Contains(t, text, "Goblin")
	assert.Contains(t, text, "5")
}

func TestTemplateNarrator_AttackKillContainsXP(t *testing.T) {
	n := narrator.NewTemplate()
	text, err := n.Narrate(context.Background(), model.EngineEvent{
		Type:    "attack_kill",
		Details: map[string]any{"target": "Goblin", "xp": 8},
	}, model.GameState{})
	require.NoError(t, err)
	assert.Contains(t, text, "Goblin")
	assert.Contains(t, text, "8")
}

func TestTemplateNarrator_Inventory(t *testing.T) {
	n := narrator.NewTemplate()

	tests := []struct {
		name     string
		details  map[string]any
		contains []string
		excludes []string
	}{
		{
			name: "typed items with equipped weapon",
			details: map[string]any{
				"items": []model.Item{
					{ID: "man_page", Name: "Man Page", Type: "misc", Weight: 1.0},
					{ID: "short_sword", Name: "Short Sword", Type: "weapon", Weight: 3.0},
				},
				"equipped": model.Equipment{Weapon: "short_sword"},
				"weight":   4.0,
				"capacity": 50.0,
			},
			contains: []string{"You are carrying:", "Man Page", "Short Sword (equipped)", "4.0/50.0"},
			excludes: []string{"Man Page (equipped)"},
		},
		{
			name: "empty inventory",
			details: map[string]any{
				"items":    []model.Item{},
				"equipped": model.Equipment{},
				"weight":   0.0,
				"capacity": 50.0,
			},
			contains: []string{"empty-handed"},
		},
		{
			name: "JSON-unmarshalled items and equipment",
			details: map[string]any{
				"items": []any{
					map[string]any{"id": "torch", "name": "Torch"},
					map[string]any{"id": "rope", "name": "Rope"},
				},
				"equipped": map[string]any{"weapon": "torch"},
				"weight":   2.0,
				"capacity": 50.0,
			},
			contains: []string{"You are carrying:", "Torch (equipped)", "Rope", "2.0/50.0"},
			excludes: []string{"Rope (equipped)"},
		},
		{
			name: "JSON items with missing name uses id",
			details: map[string]any{
				"items": []any{
					map[string]any{"id": "mystery_item"},
				},
				"equipped": map[string]any{},
				"weight":   1.0,
				"capacity": 50.0,
			},
			contains: []string{"mystery_item"},
		},
		{
			name: "JSON items with empty id and name are skipped",
			details: map[string]any{
				"items": []any{
					map[string]any{"id": "sword", "name": "Sword"},
					map[string]any{},
				},
				"equipped": map[string]any{},
				"weight":   1.0,
				"capacity": 50.0,
			},
			contains: []string{"Sword"},
		},
		{
			name: "nil items and equipment",
			details: map[string]any{},
			contains: []string{"empty-handed"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text, err := n.Narrate(context.Background(), model.EngineEvent{
				Type:    "inventory_listed",
				Details: tt.details,
			}, model.GameState{})
			require.NoError(t, err)
			for _, s := range tt.contains {
				assert.Contains(t, text, s)
			}
			for _, s := range tt.excludes {
				assert.NotContains(t, text, s)
			}
		})
	}
}

func TestTemplateNarrator_ExaminedNoDescription(t *testing.T) {
	n := narrator.NewTemplate()
	text, err := n.Narrate(context.Background(), model.EngineEvent{
		Type:    "examined",
		Details: map[string]any{"item_name": "Rock"},
	}, model.GameState{})
	require.NoError(t, err)
	assert.Contains(t, text, "Rock")
}
