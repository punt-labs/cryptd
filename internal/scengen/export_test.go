package scengen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/punt-labs/cryptd/internal/scenario"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteYAMLDir_RoundTrip(t *testing.T) {
	// Generate a scenario from a fake source.
	src := &fakeSource{
		nodes: []RawNode{
			{ID: "entrance", Meta: map[string]string{"name": "Entrance Hall"}},
			{ID: "treasury", Meta: map[string]string{"name": "Treasury"}},
			{ID: "cellar", Meta: map[string]string{"name": "Wine Cellar"}},
		},
		edges: []RawEdge{
			{From: "entrance", To: "treasury", EdgeType: "open"},
			{From: "entrance", To: "cellar", EdgeType: "locked"},
		},
		start: "entrance",
	}

	g, err := GenerateGraph(src)
	require.NoError(t, err)

	content := NewScenarioContent()
	content.Title = "Round Trip Test"
	content.Death = "respawn"
	v := &DescriptionVisitor{}
	require.NoError(t, v.Visit(g, content))

	// Add an item and enemy to test catalog export.
	content.Items["sword"] = &scenario.ScenarioItem{
		Name: "Short Sword", Type: "weapon", Damage: "1d6", Weight: 3.0, Value: 10,
	}
	content.Enemies["rat"] = &scenario.EnemyTemplate{
		Name: "Giant Rat", HP: 5, Attack: "1d3", AI: "aggressive",
	}
	content.Rooms["entrance"].Items = []string{"sword"}
	content.Rooms["treasury"].Enemies = []string{"rat"}

	// Export to YAML directory.
	outputDir := filepath.Join(t.TempDir(), "test-scenario")
	require.NoError(t, WriteYAMLDir(g, content, outputDir))

	// Verify file structure.
	_, err = os.Stat(filepath.Join(outputDir, "scenario.yaml"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(outputDir, "regions"))
	require.NoError(t, err)

	// Round-trip: load it back via LoadDir.
	loaded, err := scenario.LoadDir(outputDir)
	require.NoError(t, err)

	assert.Equal(t, "Round Trip Test", loaded.Title)
	assert.Equal(t, "entrance", loaded.StartingRoom)
	assert.Equal(t, "respawn", loaded.Death)
	assert.Len(t, loaded.Rooms, 3)

	// Verify rooms loaded.
	assert.Equal(t, "Entrance Hall", loaded.Rooms["entrance"].Name)
	assert.Equal(t, "Treasury", loaded.Rooms["treasury"].Name)
	assert.Equal(t, "Wine Cellar", loaded.Rooms["cellar"].Name)

	// Verify connections are bidirectional.
	for roomID, room := range loaded.Rooms {
		for dir, conn := range room.Connections {
			target := loaded.Rooms[conn.Room]
			require.NotNil(t, target, "room %s→%s missing", roomID, conn.Room)
			reverseDir := string(Direction(dir).Opposite())
			_, hasReverse := target.Connections[reverseDir]
			assert.True(t, hasReverse, "missing reverse %s→%s via %s", conn.Room, roomID, reverseDir)
		}
	}

	// Verify catalogs.
	require.Contains(t, loaded.Items, "sword")
	assert.Equal(t, "Short Sword", loaded.Items["sword"].Name)
	require.Contains(t, loaded.Enemies, "rat")
	assert.Equal(t, "Giant Rat", loaded.Enemies["rat"].Name)

	// Verify room items/enemies.
	assert.Contains(t, loaded.Rooms["entrance"].Items, "sword")
	assert.Contains(t, loaded.Rooms["treasury"].Enemies, "rat")

	// Full validation.
	require.NoError(t, scenario.Validate(loaded))
}

func TestWriteYAMLDir_MultipleRegions(t *testing.T) {
	src := &fakeSource{
		nodes: []RawNode{
			{ID: "root", Meta: map[string]string{"name": "Root", "region": "surface"}},
			{ID: "cave", Meta: map[string]string{"name": "Cave", "region": "underground"}},
		},
		edges: []RawEdge{{From: "root", To: "cave", EdgeType: "stairway"}},
		start: "root",
	}

	g, err := GenerateGraph(src)
	require.NoError(t, err)

	content := NewScenarioContent()
	content.Title = "Multi-Region"
	content.Death = "respawn"
	v := &DescriptionVisitor{}
	require.NoError(t, v.Visit(g, content))

	outputDir := filepath.Join(t.TempDir(), "multi-region")
	require.NoError(t, WriteYAMLDir(g, content, outputDir))

	// Should have two region files.
	_, err = os.Stat(filepath.Join(outputDir, "regions", "surface.yaml"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(outputDir, "regions", "underground.yaml"))
	require.NoError(t, err)

	// Round-trip.
	loaded, err := scenario.LoadDir(outputDir)
	require.NoError(t, err)
	assert.Len(t, loaded.Rooms, 2)
	require.NoError(t, scenario.Validate(loaded))
}

func TestWriteYAMLDir_EmptyScenario(t *testing.T) {
	src := &fakeSource{
		nodes: []RawNode{{ID: "alone", Meta: map[string]string{"name": "Alone"}}},
		start: "alone",
	}

	g, err := GenerateGraph(src)
	require.NoError(t, err)

	content := NewScenarioContent()
	content.Title = "Solo"
	content.Death = "respawn"
	v := &DescriptionVisitor{}
	require.NoError(t, v.Visit(g, content))

	outputDir := filepath.Join(t.TempDir(), "solo")
	require.NoError(t, WriteYAMLDir(g, content, outputDir))

	loaded, err := scenario.LoadDir(outputDir)
	require.NoError(t, err)
	assert.Len(t, loaded.Rooms, 1)
	require.NoError(t, scenario.Validate(loaded))
}
