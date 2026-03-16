package engine_test

import (
	"errors"
	"testing"

	"github.com/punt-labs/cryptd/internal/engine"
	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/scenario"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func consumableScenario() *scenario.Scenario {
	return &scenario.Scenario{
		ID:           "consumable-test",
		StartingRoom: "room",
		Rooms: map[string]*scenario.Room{
			"room": {Name: "Room", Items: []string{"health_potion", "sword"}},
		},
		Items: map[string]*scenario.ScenarioItem{
			"health_potion": {
				Name: "Health Potion", Type: "consumable",
				Effect: "heal", Power: "2d6",
				Weight: 0.5, Value: 10,
			},
			"sword": {
				Name: "Sword", Type: "weapon",
				Damage: "1d6", Weight: 3.0, Value: 10,
			},
		},
		Enemies: map[string]*scenario.EnemyTemplate{},
		Spells:  map[string]*scenario.SpellTemplate{},
	}
}

func TestUseItem_HealPotion(t *testing.T) {
	s := consumableScenario()
	e := engine.New(s)
	char := model.Character{
		ID: "hero", Name: "Test", Class: "fighter", Level: 1,
		HP: 10, MaxHP: 20,
		Stats: model.Stats{STR: 14, DEX: 12, CON: 12, INT: 10, WIS: 10, CHA: 10},
	}
	state, err := e.NewGame(char)
	require.NoError(t, err)

	// Pick up the potion first.
	_, err = e.PickUp(&state, "health_potion")
	require.NoError(t, err)
	require.Len(t, state.Party[0].Inventory, 1)

	result, err := e.UseItem(&state, "health_potion")
	require.NoError(t, err)
	assert.Equal(t, "heal", result.Effect)
	assert.Greater(t, result.Power, 0)
	assert.Greater(t, state.Party[0].HP, 10)
	assert.LessOrEqual(t, state.Party[0].HP, 20)
	// Potion consumed — removed from inventory.
	assert.Empty(t, state.Party[0].Inventory)
}

func TestUseItem_HealCapsAtMaxHP(t *testing.T) {
	s := consumableScenario()
	e := engine.New(s)
	char := model.Character{
		ID: "hero", Name: "Test", Class: "fighter", Level: 1,
		HP: 20, MaxHP: 20,
		Stats: model.Stats{STR: 14, DEX: 12, CON: 12, INT: 10, WIS: 10, CHA: 10},
	}
	state, err := e.NewGame(char)
	require.NoError(t, err)

	_, err = e.PickUp(&state, "health_potion")
	require.NoError(t, err)

	result, err := e.UseItem(&state, "health_potion")
	require.NoError(t, err)
	assert.Equal(t, 20, state.Party[0].HP) // capped at MaxHP
	assert.Greater(t, result.Power, 0)
}

func TestUseItem_NotConsumable(t *testing.T) {
	s := consumableScenario()
	e := engine.New(s)
	char := model.Character{
		ID: "hero", Name: "Test", Class: "fighter", Level: 1,
		HP: 20, MaxHP: 20,
		Stats: model.Stats{STR: 14, DEX: 12, CON: 12, INT: 10, WIS: 10, CHA: 10},
	}
	state, err := e.NewGame(char)
	require.NoError(t, err)

	_, err = e.PickUp(&state, "sword")
	require.NoError(t, err)

	_, err = e.UseItem(&state, "sword")
	require.Error(t, err)
	var notConsumable *engine.NotConsumableError
	assert.True(t, errors.As(err, &notConsumable))
	assert.Contains(t, notConsumable.Error(), "weapon")
	// Sword should still be in inventory (not consumed).
	assert.Len(t, state.Party[0].Inventory, 1)
}

func TestUseItem_NotInInventory(t *testing.T) {
	s := consumableScenario()
	e := engine.New(s)
	char := model.Character{
		ID: "hero", Name: "Test", Class: "fighter", Level: 1,
		HP: 20, MaxHP: 20,
		Stats: model.Stats{STR: 14, DEX: 12, CON: 12, INT: 10, WIS: 10, CHA: 10},
	}
	state, err := e.NewGame(char)
	require.NoError(t, err)

	_, err = e.UseItem(&state, "health_potion")
	require.Error(t, err)
	var notInInv *engine.ItemNotInInventoryError
	assert.True(t, errors.As(err, &notInInv))
}
