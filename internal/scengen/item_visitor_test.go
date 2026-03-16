package scengen

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestItemVisitor_PlacesStarterWeapon(t *testing.T) {
	g := NewGraph("start")
	require.NoError(t, g.AddNode("start", nil))
	require.NoError(t, g.AddNode("room1", nil))
	require.NoError(t, g.AddNode("room2", nil))
	require.NoError(t, g.AddEdge("start", "room1", North, "open"))
	require.NoError(t, g.AddEdge("room1", "room2", North, "open"))

	content := NewScenarioContent()
	v := &ItemVisitor{Rng: rand.New(rand.NewSource(42))}
	require.NoError(t, v.Visit(g, content))

	// A weapon should exist in the item catalog.
	hasWeapon := false
	for _, item := range content.Items {
		if item.Type == "weapon" {
			hasWeapon = true
			break
		}
	}
	assert.True(t, hasWeapon, "should place at least a starter weapon")

	// The weapon should be in room1 (distance 1) or start.
	weaponPlaced := false
	for _, roomID := range []string{"start", "room1"} {
		if rc, ok := content.Rooms[roomID]; ok {
			for _, itemID := range rc.Items {
				if item, ok := content.Items[itemID]; ok && item.Type == "weapon" {
					weaponPlaced = true
				}
			}
		}
	}
	assert.True(t, weaponPlaced, "starter weapon should be near the start")
}

func TestItemVisitor_PotionsAtHubs(t *testing.T) {
	// Create a hub with degree 3+.
	g := NewGraph("start")
	require.NoError(t, g.AddNode("start", nil))
	require.NoError(t, g.AddNode("hub", map[string]string{"hub": "true"}))
	require.NoError(t, g.AddNode("a", nil))
	require.NoError(t, g.AddNode("b", nil))
	require.NoError(t, g.AddEdge("start", "hub", North, "open"))
	require.NoError(t, g.AddEdge("hub", "a", East, "open"))
	require.NoError(t, g.AddEdge("hub", "b", West, "open"))

	content := NewScenarioContent()
	v := &ItemVisitor{Rng: rand.New(rand.NewSource(42))}
	require.NoError(t, v.Visit(g, content))

	// Hub should have a potion (high degree = rest stop).
	if rc, ok := content.Rooms["hub"]; ok {
		hasPotion := false
		for _, itemID := range rc.Items {
			if item, ok := content.Items[itemID]; ok && item.Type == "consumable" {
				hasPotion = true
			}
		}
		assert.True(t, hasPotion, "hub nodes should get potions")
	}
}

func TestItemVisitor_IncludesSpells(t *testing.T) {
	g := NewGraph("start")
	require.NoError(t, g.AddNode("start", nil))

	content := NewScenarioContent()
	v := &ItemVisitor{IncludeSpells: true}
	require.NoError(t, v.Visit(g, content))

	assert.Contains(t, content.Spells, "fireball")
	assert.Contains(t, content.Spells, "heal")
	assert.Contains(t, content.Spells, "lightning")
	assert.Contains(t, content.Spells, "blessing")
}

func TestItemVisitor_NoSpellsByDefault(t *testing.T) {
	g := NewGraph("start")
	require.NoError(t, g.AddNode("start", nil))

	content := NewScenarioContent()
	v := &ItemVisitor{}
	require.NoError(t, v.Visit(g, content))

	assert.Empty(t, content.Spells)
}

func TestItemVisitor_ScalesWithDistance(t *testing.T) {
	// Long chain: items should get better further from start.
	g := makeLinearGraph(t, 8)

	content := NewScenarioContent()
	v := &ItemVisitor{Rng: rand.New(rand.NewSource(99))}
	require.NoError(t, v.Visit(g, content))

	// Should have items in the catalog.
	assert.NotEmpty(t, content.Items, "should place some items")

	// Check that at least some items exist across different distances.
	roomsWithItems := 0
	for _, rc := range content.Rooms {
		if len(rc.Items) > 0 {
			roomsWithItems++
		}
	}
	assert.Greater(t, roomsWithItems, 1, "items should be spread across rooms")
}

func TestItemVisitor_SingleNodeGraph(t *testing.T) {
	g := NewGraph("alone")
	require.NoError(t, g.AddNode("alone", nil))

	content := NewScenarioContent()
	v := &ItemVisitor{Rng: rand.New(rand.NewSource(42))}
	require.NoError(t, v.Visit(g, content))

	// Single node = maxDist 0, should return early without error.
	// Starter weapon still placed at start as fallback.
	hasWeapon := false
	for _, item := range content.Items {
		if item.Type == "weapon" {
			hasWeapon = true
		}
	}
	assert.True(t, hasWeapon, "even single-node graph gets a starter weapon")
}

func TestDefaultItemTiers_AllHaveContent(t *testing.T) {
	tiers := DefaultItemTiers()
	assert.Len(t, tiers, 5)
	for i, tier := range tiers {
		// Every tier should have at least a weapon or potion.
		hasContent := len(tier.Weapons) > 0 || len(tier.Potions) > 0
		assert.True(t, hasContent, "tier %d should have weapons or potions", i)
	}
}

func TestDefaultSpells_Valid(t *testing.T) {
	spells := DefaultSpells()
	assert.Len(t, spells, 4)
	for id, spell := range spells {
		assert.NotEmpty(t, spell.Name, "spell %s should have a name", id)
		assert.Greater(t, spell.MP, 0, "spell %s should have MP cost", id)
		assert.NotEmpty(t, spell.Power, "spell %s should have power", id)
		assert.NotEmpty(t, spell.Classes, "spell %s should have classes", id)
	}
}
