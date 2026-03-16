package scengen

import (
	"math/rand"
	"testing"

	"github.com/punt-labs/cryptd/internal/scenario"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnemyVisitor_PlacesEnemiesByDistance(t *testing.T) {
	// Linear graph: a → b → c → d → e (distances 0-4).
	g := NewGraph("a")
	for _, id := range []string{"a", "b", "c", "d", "e"} {
		require.NoError(t, g.AddNode(id, nil))
	}
	require.NoError(t, g.AddEdge("a", "b", North, "open"))
	require.NoError(t, g.AddEdge("b", "c", North, "open"))
	require.NoError(t, g.AddEdge("c", "d", North, "open"))
	require.NoError(t, g.AddEdge("d", "e", North, "open"))

	content := NewScenarioContent()
	v := &EnemyVisitor{
		Rng:       rand.New(rand.NewSource(42)),
		SpawnRate: 1.0, // 100% spawn for testing
	}
	require.NoError(t, v.Visit(g, content))

	// Start room (a, distance 0) should have no enemies.
	assert.Empty(t, content.Rooms["a"], "start room should have no enemies")

	// At least some rooms should have enemies.
	totalEnemies := 0
	for _, rc := range content.Rooms {
		totalEnemies += len(rc.Enemies)
	}
	assert.Greater(t, totalEnemies, 0, "should place at least one enemy")

	// Enemy templates should be registered in catalog.
	assert.NotEmpty(t, content.Enemies, "enemy templates should be in catalog")
}

func TestEnemyVisitor_SkipsHubNodes(t *testing.T) {
	g := NewGraph("a")
	require.NoError(t, g.AddNode("a", nil))
	require.NoError(t, g.AddNode("hub", map[string]string{"hub": "true"}))
	require.NoError(t, g.AddNode("b", nil))
	require.NoError(t, g.AddEdge("a", "hub", North, "open"))
	require.NoError(t, g.AddEdge("hub", "b", North, "open"))

	content := NewScenarioContent()
	v := &EnemyVisitor{
		Rng:       rand.New(rand.NewSource(42)),
		SpawnRate: 1.0,
	}
	require.NoError(t, v.Visit(g, content))

	if rc, ok := content.Rooms["hub"]; ok {
		assert.Empty(t, rc.Enemies, "hub nodes should not get enemies")
	}
}

func TestEnemyVisitor_DifficultyScales(t *testing.T) {
	// Long chain: 10 nodes. Deeper rooms should get harder enemies.
	g := makeLinearGraph(t, 10)

	content := NewScenarioContent()
	tiers := DefaultEnemyTiers()
	v := &EnemyVisitor{
		Tiers:     tiers,
		Rng:       rand.New(rand.NewSource(42)),
		SpawnRate: 1.0,
	}
	require.NoError(t, v.Visit(g, content))

	// The last node (distance 9) should have a harder enemy than the first (distance 1).
	// makeLinearGraph uses IDs 'a'+i, so node 0='a', node 9='j'.
	lastRC := content.Rooms["j"]
	firstRC := content.Rooms["b"]
	require.NotNil(t, lastRC)
	require.NotNil(t, firstRC)

	if len(lastRC.Enemies) > 0 && len(firstRC.Enemies) > 0 {
		lastEnemy := content.Enemies[lastRC.Enemies[0]]
		firstEnemy := content.Enemies[firstRC.Enemies[0]]
		assert.GreaterOrEqual(t, lastEnemy.HP, firstEnemy.HP,
			"deeper rooms should have tougher enemies: %s (%d HP) vs %s (%d HP)",
			lastEnemy.Name, lastEnemy.HP, firstEnemy.Name, firstEnemy.HP)
	}
}

func TestEnemyVisitor_UnreachableNode(t *testing.T) {
	// Graph with an unreachable node — visitor must not panic.
	g := NewGraph("a")
	require.NoError(t, g.AddNode("a", nil))
	require.NoError(t, g.AddNode("b", nil))
	require.NoError(t, g.AddNode("island", nil)) // unreachable
	require.NoError(t, g.AddEdge("a", "b", North, "open"))

	content := NewScenarioContent()
	v := &EnemyVisitor{
		Rng:       rand.New(rand.NewSource(42)),
		SpawnRate: 1.0,
	}
	// Should not panic on island node with Distance() == -1.
	require.NoError(t, v.Visit(g, content))

	// Island should have no enemies.
	if rc, ok := content.Rooms["island"]; ok {
		assert.Empty(t, rc.Enemies, "unreachable node should have no enemies")
	}
}

func TestEnemyVisitor_SpawnRateZero(t *testing.T) {
	g := makeLinearGraph(t, 5)

	content := NewScenarioContent()
	v := &EnemyVisitor{SpawnRate: 0.0001} // near-zero
	require.NoError(t, v.Visit(g, content))

	// Templates should still be registered even if no enemies placed.
	assert.NotEmpty(t, content.Enemies, "enemy catalog should be populated regardless of spawn rate")
}

func TestTotalXP(t *testing.T) {
	content := NewScenarioContent()
	content.Enemies["goblin"] = &scenario.EnemyTemplate{Name: "Goblin", HP: 8}
	content.Enemies["troll"] = &scenario.EnemyTemplate{Name: "Troll", HP: 18}
	content.Rooms["r1"] = &RoomContent{Enemies: []string{"goblin"}}
	content.Rooms["r2"] = &RoomContent{Enemies: []string{"troll", "goblin"}}

	assert.Equal(t, 34, TotalXP(content)) // 8 + 18 + 8
}

func TestEnemyCount(t *testing.T) {
	content := NewScenarioContent()
	content.Rooms["r1"] = &RoomContent{Enemies: []string{"goblin"}}
	content.Rooms["r2"] = &RoomContent{}
	content.Rooms["r3"] = &RoomContent{Enemies: []string{"troll"}}

	assert.Equal(t, 2, EnemyCount(content))
}
