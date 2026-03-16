package scengen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/punt-labs/cryptd/internal/scenario"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeSource is a TopologySource that returns canned data.
type fakeSource struct {
	nodes []RawNode
	edges []RawEdge
	start string
	err   error
}

func (f *fakeSource) Generate() ([]RawNode, []RawEdge, string, error) {
	return f.nodes, f.edges, f.start, f.err
}

func TestGenerate_SimpleChain(t *testing.T) {
	src := &fakeSource{
		nodes: []RawNode{
			{ID: "a", Meta: map[string]string{"name": "Room A"}},
			{ID: "b", Meta: map[string]string{"name": "Room B"}},
			{ID: "c", Meta: map[string]string{"name": "Room C"}},
		},
		edges: []RawEdge{
			{From: "a", To: "b", EdgeType: "open"},
			{From: "b", To: "c", EdgeType: "locked"},
		},
		start: "a",
	}

	s, err := Generate(src, GenerateOptions{
		Title:    "Test Dungeon",
		Visitors: []Visitor{&DescriptionVisitor{}},
	})
	require.NoError(t, err)

	assert.Equal(t, "Test Dungeon", s.Title)
	assert.Equal(t, "a", s.StartingRoom)
	assert.Equal(t, "respawn", s.Death)
	assert.Len(t, s.Rooms, 3)

	// Verify bidirectional connections.
	for roomID, room := range s.Rooms {
		for dir, conn := range room.Connections {
			target := s.Rooms[conn.Room]
			require.NotNil(t, target, "room %s has connection to %s which doesn't exist", roomID, conn.Room)
			// Target should have a reverse connection.
			d := Direction(dir)
			reverseDir := string(d.Opposite())
			reverseConn, ok := target.Connections[reverseDir]
			assert.True(t, ok, "room %s→%s via %s has no reverse", roomID, conn.Room, dir)
			if ok {
				assert.Equal(t, roomID, reverseConn.Room)
			}
		}
	}

	// Validate the generated scenario.
	require.NoError(t, scenario.Validate(s))
}

func TestGenerate_DefaultDeath(t *testing.T) {
	src := &fakeSource{
		nodes: []RawNode{{ID: "start", Meta: map[string]string{"name": "Start"}}},
		start: "start",
	}
	s, err := Generate(src, GenerateOptions{Title: "T", Visitors: []Visitor{&DescriptionVisitor{}}})
	require.NoError(t, err)
	assert.Equal(t, "respawn", s.Death)
}

func TestGenerate_PermadeathOption(t *testing.T) {
	src := &fakeSource{
		nodes: []RawNode{{ID: "start", Meta: map[string]string{"name": "Start"}}},
		start: "start",
	}
	s, err := Generate(src, GenerateOptions{
		Title:    "Hardcore",
		Death:    "permadeath",
		Visitors: []Visitor{&DescriptionVisitor{}},
	})
	require.NoError(t, err)
	assert.Equal(t, "permadeath", s.Death)
}

func TestGenerate_FromTreeSource(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "dungeon", "hall"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "dungeon", "treasury"), 0o755))

	ts := &TreeSource{Root: filepath.Join(root, "dungeon")}
	s, err := Generate(ts, GenerateOptions{
		Title:    "Tree Dungeon",
		Visitors: []Visitor{&DescriptionVisitor{}},
	})
	require.NoError(t, err)

	assert.Equal(t, "Tree Dungeon", s.Title)
	assert.Len(t, s.Rooms, 3) // dungeon, hall, treasury
	require.NoError(t, scenario.Validate(s))
}

func TestGenerate_SourceError(t *testing.T) {
	src := &fakeSource{err: assert.AnError}
	_, err := Generate(src, GenerateOptions{Title: "T"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "topology source")
}

func TestGenerateGraph_ReturnsValidGraph(t *testing.T) {
	src := &fakeSource{
		nodes: []RawNode{
			{ID: "a", Meta: map[string]string{"name": "A"}},
			{ID: "b", Meta: map[string]string{"name": "B"}},
		},
		edges: []RawEdge{{From: "a", To: "b", EdgeType: "open"}},
		start: "a",
	}
	g, err := GenerateGraph(src)
	require.NoError(t, err)
	require.NoError(t, g.Validate())
	assert.Len(t, g.Nodes, 2)
	assert.Len(t, g.Edges, 1)
}

func TestAssignDirections_StarTopology(t *testing.T) {
	// Hub with 5 children — should use 5 of 6 available directions.
	nodes := []RawNode{{ID: "hub"}}
	var edges []RawEdge
	for i := 0; i < 5; i++ {
		id := string(rune('a' + i))
		nodes = append(nodes, RawNode{ID: id})
		edges = append(edges, RawEdge{From: "hub", To: id, EdgeType: "open"})
	}

	g, err := assignDirections(nodes, edges, "hub")
	require.NoError(t, err)
	require.NoError(t, g.Validate())
	assert.Equal(t, 5, g.Degree("hub"))
}

func TestAssignDirections_FullyConnectedHub(t *testing.T) {
	// Hub with 6 children — uses all 6 directions.
	nodes := []RawNode{{ID: "hub"}}
	var edges []RawEdge
	for i := 0; i < 6; i++ {
		id := string(rune('a' + i))
		nodes = append(nodes, RawNode{ID: id})
		edges = append(edges, RawEdge{From: "hub", To: id, EdgeType: "open"})
	}

	g, err := assignDirections(nodes, edges, "hub")
	require.NoError(t, err)
	require.NoError(t, g.Validate())
	assert.Equal(t, 6, g.Degree("hub"))
}

func TestAssignDirections_SevenChildrenFails(t *testing.T) {
	// 7 children with no hub insertion should fail.
	nodes := []RawNode{{ID: "hub"}}
	var edges []RawEdge
	for i := 0; i < 7; i++ {
		id := string(rune('a' + i))
		nodes = append(nodes, RawNode{ID: id})
		edges = append(edges, RawEdge{From: "hub", To: id, EdgeType: "open"})
	}

	_, err := assignDirections(nodes, edges, "hub")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no available direction")
}
