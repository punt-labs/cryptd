package scengen

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore_RoundTrip(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := OpenStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	require.NoError(t, store.CreateSchema())

	// Build a graph.
	g := NewGraph("entrance")
	g.Meta["title"] = "Test Dungeon"
	g.Meta["death"] = "permadeath"
	require.NoError(t, g.AddNode("entrance", map[string]string{"name": "Entrance", "region": "surface"}))
	require.NoError(t, g.AddNode("hall", map[string]string{"name": "Grand Hall", "region": "surface"}))
	require.NoError(t, g.AddNode("cellar", map[string]string{"name": "Wine Cellar", "region": "underground"}))
	require.NoError(t, g.AddEdge("entrance", "hall", North, "open"))
	require.NoError(t, g.AddEdge("entrance", "cellar", Down, "stairway"))
	require.NoError(t, g.Validate())

	// Save.
	require.NoError(t, store.SaveGraph(g))

	// Load.
	loaded, err := store.LoadGraph()
	require.NoError(t, err)

	assert.Equal(t, g.Start, loaded.Start)
	assert.Len(t, loaded.Nodes, 3)
	assert.Len(t, loaded.Edges, 2)

	// Verify graph-level metadata round-trips.
	assert.Equal(t, "Test Dungeon", loaded.Meta["title"])
	assert.Equal(t, "permadeath", loaded.Meta["death"])

	// Verify node metadata.
	assert.Equal(t, "Entrance", loaded.Nodes["entrance"].Meta["name"])
	assert.Equal(t, "surface", loaded.Nodes["entrance"].Meta["region"])
	assert.Equal(t, "underground", loaded.Nodes["cellar"].Meta["region"])

	// Loaded graph should pass validation.
	require.NoError(t, loaded.Validate())
}

func TestStore_SaveOverwrites(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := OpenStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	require.NoError(t, store.CreateSchema())

	// Save first graph.
	g1 := NewGraph("a")
	require.NoError(t, g1.AddNode("a", map[string]string{"name": "A"}))
	require.NoError(t, store.SaveGraph(g1))

	// Save second graph (should replace).
	g2 := NewGraph("x")
	require.NoError(t, g2.AddNode("x", map[string]string{"name": "X"}))
	require.NoError(t, g2.AddNode("y", map[string]string{"name": "Y"}))
	require.NoError(t, g2.AddEdge("x", "y", East, "open"))
	require.NoError(t, store.SaveGraph(g2))

	// Load should return g2.
	loaded, err := store.LoadGraph()
	require.NoError(t, err)
	assert.Equal(t, "x", loaded.Start)
	assert.Len(t, loaded.Nodes, 2)
}

func TestStore_ForeignKeyConstraint(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := OpenStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	require.NoError(t, store.CreateSchema())

	// Direct insert with non-existent node should fail.
	_, err = store.db.Exec(
		"INSERT INTO edges (from_node, to_node, from_direction, to_direction, type) VALUES ('ghost', 'phantom', 'north', 'south', 'open')",
	)
	require.Error(t, err, "foreign key constraint should prevent inserting edge with non-existent nodes")
}

func TestStore_DirectionCheckConstraint(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := OpenStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	require.NoError(t, store.CreateSchema())

	// Insert a node first.
	_, err = store.db.Exec("INSERT INTO nodes (id, name) VALUES ('a', 'A')")
	require.NoError(t, err)
	_, err = store.db.Exec("INSERT INTO nodes (id, name) VALUES ('b', 'B')")
	require.NoError(t, err)

	// Invalid direction should fail.
	_, err = store.db.Exec(
		"INSERT INTO edges (from_node, to_node, from_direction, to_direction, type) VALUES ('a', 'b', 'northeast', 'southwest', 'open')",
	)
	require.Error(t, err, "CHECK constraint should reject invalid direction")
}

func TestStore_CreateSchemaIdempotent(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := OpenStore(dbPath)
	require.NoError(t, err)
	defer store.Close()

	require.NoError(t, store.CreateSchema())
	require.NoError(t, store.CreateSchema(), "calling CreateSchema twice should not error")
}
