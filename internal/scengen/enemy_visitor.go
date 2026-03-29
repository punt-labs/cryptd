package scengen

import (
	"math/rand"

	"github.com/punt-labs/cryptd/internal/scenario"
)

// EnemyTier defines an enemy template at a difficulty level.
type EnemyTier struct {
	ID     string
	Name   string
	HP     int
	Attack string // dice notation
	AI     string // aggressive|cautious
}

// DefaultEnemyTiers returns a 5-tier enemy progression from trivial to boss.
func DefaultEnemyTiers() []EnemyTier {
	return []EnemyTier{
		{ID: "rat", Name: "Giant Rat", HP: 3, Attack: "1d2", AI: "cautious"},
		{ID: "goblin", Name: "Goblin", HP: 6, Attack: "1d3", AI: "aggressive"},
		{ID: "skeleton", Name: "Skeleton Warrior", HP: 10, Attack: "1d4", AI: "aggressive"},
		{ID: "troll", Name: "Cave Troll", HP: 15, Attack: "1d6", AI: "aggressive"},
		{ID: "dragon", Name: "Elder Dragon", HP: 25, Attack: "1d8+2", AI: "aggressive"},
	}
}

// EnemyVisitor places enemies in rooms based on distance from start.
// Rooms at distance 0 (start) get no enemies. Deeper rooms get harder
// enemies. Hub nodes are skipped (corridors, not combat arenas).
type EnemyVisitor struct {
	Tiers []EnemyTier
	Rng   *rand.Rand // if nil, uses default source
	// SpawnRate is the probability (0.0-1.0) that a non-start, non-hub
	// room gets an enemy. Negative means use default (0.4). Zero means
	// no enemies (catalog still registered).
	SpawnRate float64
}

func (v *EnemyVisitor) Name() string { return "enemy" }

func (v *EnemyVisitor) Visit(g *Graph, content *ScenarioContent) error {
	tiers := v.Tiers
	if len(tiers) == 0 {
		tiers = DefaultEnemyTiers()
	}
	spawnRate := v.SpawnRate
	if spawnRate < 0 {
		spawnRate = 0.4 // default: 40% of non-start, non-hub rooms get enemies
	}

	// Register all enemy templates in the content catalog.
	for _, tier := range tiers {
		content.Enemies[tier.ID] = &scenario.EnemyTemplate{
			Name:   tier.Name,
			HP:     tier.HP,
			Attack: tier.Attack,
			AI:     tier.AI,
		}
	}

	maxDist := g.MaxDistance()
	if maxDist == 0 {
		return nil // single-node graph, no enemies
	}

	for id, node := range g.Nodes {
		dist := g.Distance(id)

		// Skip start room, unreachable nodes, and hub corridors.
		if dist <= 0 {
			continue
		}
		if node.Meta != nil && node.Meta["hub"] == "true" {
			continue
		}

		// Roll for spawn.
		if v.randFloat() > spawnRate {
			continue
		}

		// Map distance to tier index: normalize to 0..len(tiers)-1.
		tierIdx := (dist * len(tiers)) / (maxDist + 1)
		if tierIdx >= len(tiers) {
			tierIdx = len(tiers) - 1
		}

		// Deepest rooms get the highest tier (boss).
		// Occasionally bump ±1 tier for variety.
		tierIdx = v.jitter(tierIdx, len(tiers))

		enemy := tiers[tierIdx]

		rc := ensureRoom(content, id)
		rc.Enemies = append(rc.Enemies, enemy.ID)
	}

	return nil
}

func (v *EnemyVisitor) randFloat() float64 {
	if v.Rng != nil {
		return v.Rng.Float64()
	}
	return rand.Float64()
}

func (v *EnemyVisitor) randIntn(n int) int {
	if v.Rng != nil {
		return v.Rng.Intn(n)
	}
	return rand.Intn(n)
}

// jitter randomly adjusts tierIdx by -1, 0, or +1 (20% chance each direction).
func (v *EnemyVisitor) jitter(tierIdx, maxTiers int) int {
	r := v.randIntn(10)
	switch {
	case r < 2 && tierIdx > 0: // 20% chance downgrade
		return tierIdx - 1
	case r >= 8 && tierIdx < maxTiers-1: // 20% chance upgrade
		return tierIdx + 1
	default:
		return tierIdx
	}
}

// ensureRoom returns the RoomContent for a node, creating it if needed.
func ensureRoom(content *ScenarioContent, id string) *RoomContent {
	rc, ok := content.Rooms[id]
	if !ok {
		rc = &RoomContent{}
		content.Rooms[id] = rc
	}
	return rc
}

// TotalXP returns the sum of MaxHP for all enemies placed in the scenario.
// Useful for verifying enough XP exists for leveling.
func TotalXP(content *ScenarioContent) int {
	total := 0
	for _, rc := range content.Rooms {
		for _, enemyID := range rc.Enemies {
			if tmpl, ok := content.Enemies[enemyID]; ok {
				total += tmpl.HP
			}
		}
	}
	return total
}

// EnemyCount returns the number of rooms with enemies.
func EnemyCount(content *ScenarioContent) int {
	count := 0
	for _, rc := range content.Rooms {
		if len(rc.Enemies) > 0 {
			count++
		}
	}
	return count
}
