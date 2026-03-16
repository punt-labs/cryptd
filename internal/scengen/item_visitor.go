package scengen

import (
	"fmt"
	"math/rand"
	"sort"

	"github.com/punt-labs/cryptd/internal/scenario"
)

// ItemTier defines items available at a difficulty level.
type ItemTier struct {
	Weapons []scenario.ScenarioItem
	Armor   []scenario.ScenarioItem
	Potions []scenario.ScenarioItem
}

// DefaultItemTiers returns a 5-tier item progression.
func DefaultItemTiers() []ItemTier {
	return []ItemTier{
		// Tier 0: starter — placed near start
		{
			Weapons: []scenario.ScenarioItem{
				{Name: "Rusty Dagger", Type: "weapon", Damage: "1d4", Weight: 1.5, Value: 5, Description: "A pitted blade, but better than bare fists."},
			},
			Potions: []scenario.ScenarioItem{
				{Name: "Minor Healing Potion", Type: "consumable", Effect: "heal", Power: "1d6", Weight: 0.3, Value: 5, Description: "A small vial of red liquid."},
			},
		},
		// Tier 1: early
		{
			Weapons: []scenario.ScenarioItem{
				{Name: "Short Sword", Type: "weapon", Damage: "1d6", Weight: 3.0, Value: 15, Description: "A reliable blade."},
			},
			Armor: []scenario.ScenarioItem{
				{Name: "Leather Armor", Type: "armor", Defense: 1, Weight: 5.0, Value: 20, Description: "Tough hide, lightly protective."},
			},
			Potions: []scenario.ScenarioItem{
				{Name: "Healing Potion", Type: "consumable", Effect: "heal", Power: "2d6", Weight: 0.5, Value: 15, Description: "A shimmering red vial."},
			},
		},
		// Tier 2: mid
		{
			Weapons: []scenario.ScenarioItem{
				{Name: "Longsword", Type: "weapon", Damage: "1d8", Weight: 4.0, Value: 30, Description: "A well-balanced two-handed blade."},
			},
			Armor: []scenario.ScenarioItem{
				{Name: "Chain Mail", Type: "armor", Defense: 2, Weight: 8.0, Value: 40, Description: "Interlocking rings of steel."},
			},
			Potions: []scenario.ScenarioItem{
				{Name: "Greater Healing Potion", Type: "consumable", Effect: "heal", Power: "3d6", Weight: 0.5, Value: 25, Description: "A glowing crimson vial."},
			},
		},
		// Tier 3: late
		{
			Weapons: []scenario.ScenarioItem{
				{Name: "War Axe", Type: "weapon", Damage: "1d10", Weight: 5.0, Value: 50, Description: "A heavy, brutal weapon."},
			},
			Armor: []scenario.ScenarioItem{
				{Name: "Plate Armor", Type: "armor", Defense: 3, Weight: 12.0, Value: 60, Description: "Full plate, forged in fire."},
			},
			Potions: []scenario.ScenarioItem{
				{Name: "Superior Healing Potion", Type: "consumable", Effect: "heal", Power: "4d6", Weight: 0.5, Value: 40, Description: "Liquid gold with a crimson swirl."},
			},
		},
		// Tier 4: endgame / boss loot
		{
			Weapons: []scenario.ScenarioItem{
				{Name: "Vorpal Blade", Type: "weapon", Damage: "1d12+2", Weight: 4.0, Value: 100, Description: "The edge hums with lethal precision."},
			},
			Armor: []scenario.ScenarioItem{
				{Name: "Dragon Scale Mail", Type: "armor", Defense: 4, Weight: 10.0, Value: 100, Description: "Scales shimmer with ancient power."},
			},
			Potions: []scenario.ScenarioItem{
				{Name: "Elixir of Life", Type: "consumable", Effect: "heal", Power: "6d6", Weight: 0.5, Value: 75, Description: "Restores body and spirit alike."},
			},
		},
	}
}

// DefaultSpells returns a standard spell catalog for generated scenarios.
func DefaultSpells() map[string]*scenario.SpellTemplate {
	return map[string]*scenario.SpellTemplate{
		"fireball": {Name: "Fireball", MP: 3, Effect: "damage", Power: "2d6", Classes: []string{"mage", "priest"}},
		"heal":     {Name: "Heal", MP: 2, Effect: "heal", Power: "1d6+2", Classes: []string{"priest", "mage"}},
		"lightning": {Name: "Lightning Bolt", MP: 5, Effect: "damage", Power: "3d6", Classes: []string{"mage"}},
		"blessing":  {Name: "Blessing", MP: 3, Effect: "heal", Power: "2d6+3", Classes: []string{"priest"}},
	}
}

// ItemVisitor places weapons, armor, and potions based on distance from start.
// Starter weapon always placed at distance 1 (first room after start).
// Potions placed at high-degree hubs (rest stops) and before boss rooms.
// Weapon/armor upgrades placed at tier-appropriate distances.
type ItemVisitor struct {
	Tiers []ItemTier
	Rng   *rand.Rand
	// IncludeSpells adds a default spell catalog to the scenario.
	IncludeSpells bool
}

func (v *ItemVisitor) Name() string { return "item" }

func (v *ItemVisitor) Visit(g *Graph, content *ScenarioContent) error {
	tiers := v.Tiers
	if len(tiers) == 0 {
		tiers = DefaultItemTiers()
	}

	if v.IncludeSpells {
		for id, spell := range DefaultSpells() {
			content.Spells[id] = spell
		}
	}

	// Track placed item IDs to avoid duplicates.
	itemCounter := 0
	placeItem := func(roomID string, item scenario.ScenarioItem) {
		itemCounter++
		itemID := fmt.Sprintf("%s_%d", sanitizeItemID(item.Name), itemCounter)
		content.Items[itemID] = &item
		rc := ensureRoom(content, roomID)
		rc.Items = append(rc.Items, itemID)
	}

	maxDist := g.MaxDistance()

	// Classify nodes by distance tier, sorted for deterministic output.
	type nodeInfo struct {
		id       string
		distance int
		degree   int
		isHub    bool
	}
	var nodes []nodeInfo
	for id, node := range g.Nodes {
		dist := g.Distance(id)
		if dist < 0 {
			continue // skip unreachable nodes
		}
		isHub := node.Meta != nil && node.Meta["hub"] == "true"
		nodes = append(nodes, nodeInfo{
			id:       id,
			distance: dist,
			degree:   g.Degree(id),
			isHub:    isHub,
		})
	}
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].distance != nodes[j].distance {
			return nodes[i].distance < nodes[j].distance
		}
		return nodes[i].id < nodes[j].id
	})

	// Place starter weapon in the first non-start room (distance 1).
	starterPlaced := false
	for _, n := range nodes {
		if n.distance == 1 && !n.isHub && !starterPlaced {
			if len(tiers) > 0 && len(tiers[0].Weapons) > 0 {
				placeItem(n.id, tiers[0].Weapons[0])
				starterPlaced = true
			}
			break
		}
	}

	// If no distance-1 room, place at start.
	if !starterPlaced && len(tiers) > 0 && len(tiers[0].Weapons) > 0 {
		placeItem(g.Start, tiers[0].Weapons[0])
	}

	// Always place a starter potion at the start room.
	if len(tiers) > 0 && len(tiers[0].Potions) > 0 {
		placeItem(g.Start, tiers[0].Potions[0])
	}

	if maxDist == 0 {
		return nil // single-node graph: starter weapon placed, nothing else to do
	}

	// Place items by tier at appropriate distances.
	for _, n := range nodes {
		if n.distance == 0 {
			continue // start room: starter weapon only
		}

		tierIdx := (n.distance * len(tiers)) / (maxDist + 1)
		if tierIdx >= len(tiers) {
			tierIdx = len(tiers) - 1
		}
		tier := tiers[tierIdx]

		// High-degree hubs and rooms before enemies get potions.
		if n.degree >= 3 || n.isHub {
			if len(tier.Potions) > 0 {
				potion := tier.Potions[v.randIntn(len(tier.Potions))]
				placeItem(n.id, potion)
			}
			continue // hubs get potions, not weapons/armor
		}

		// ~30% chance of weapon upgrade at this tier.
		if len(tier.Weapons) > 0 && v.randFloat() < 0.3 {
			weapon := tier.Weapons[v.randIntn(len(tier.Weapons))]
			placeItem(n.id, weapon)
		}

		// ~20% chance of armor at this tier.
		if len(tier.Armor) > 0 && v.randFloat() < 0.2 {
			armor := tier.Armor[v.randIntn(len(tier.Armor))]
			placeItem(n.id, armor)
		}

		// ~40% chance of potion in rooms with enemies.
		rc := ensureRoom(content, n.id)
		if len(rc.Enemies) > 0 && len(tier.Potions) > 0 && v.randFloat() < 0.4 {
			potion := tier.Potions[v.randIntn(len(tier.Potions))]
			placeItem(n.id, potion)
		}
	}

	return nil
}

func (v *ItemVisitor) randFloat() float64 {
	if v.Rng != nil {
		return v.Rng.Float64()
	}
	return rand.Float64()
}

func (v *ItemVisitor) randIntn(n int) int {
	if n <= 0 {
		return 0
	}
	if v.Rng != nil {
		return v.Rng.Intn(n)
	}
	return rand.Intn(n)
}

// sanitizeItemID converts an item name to a snake_case ID.
func sanitizeItemID(name string) string {
	return sanitizeID(name)
}
