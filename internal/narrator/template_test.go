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

func TestTemplateNarrator_ExaminedNoDescription(t *testing.T) {
	n := narrator.NewTemplate()
	text, err := n.Narrate(context.Background(), model.EngineEvent{
		Type:    "examined",
		Details: map[string]any{"item_name": "Rock"},
	}, model.GameState{})
	require.NoError(t, err)
	assert.Contains(t, text, "Rock")
}
