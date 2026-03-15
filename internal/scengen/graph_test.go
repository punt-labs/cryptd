package scengen

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDirection_Opposite(t *testing.T) {
	tests := []struct {
		dir  Direction
		want Direction
	}{
		{North, South},
		{South, North},
		{East, West},
		{West, East},
		{Up, Down},
		{Down, Up},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, tt.dir.Opposite(), "Opposite of %s", tt.dir)
	}
}

func TestDirection_Valid(t *testing.T) {
	for _, d := range AllDirections {
		assert.True(t, d.Valid(), "%s should be valid", d)
	}
	assert.False(t, Direction("northeast").Valid())
	assert.False(t, Direction("").Valid())
}

func TestGraph_AddNode_Duplicate(t *testing.T) {
	g := NewGraph("a")
	require.NoError(t, g.AddNode("a", nil))
	err := g.AddNode("a", nil)
	require.Error(t, err)
	var dup *DuplicateNodeError
	assert.True(t, errors.As(err, &dup))
	assert.Equal(t, "a", dup.ID)
}

func TestGraph_AddEdge_Bidirectional(t *testing.T) {
	g := NewGraph("a")
	require.NoError(t, g.AddNode("a", nil))
	require.NoError(t, g.AddNode("b", nil))
	require.NoError(t, g.AddEdge("a", "b", North, "open"))

	require.Len(t, g.Edges, 1)
	e := g.Edges[0]
	assert.Equal(t, North, e.FromDir)
	assert.Equal(t, South, e.ToDir)
	assert.Equal(t, "a", e.From)
	assert.Equal(t, "b", e.To)
}

func TestGraph_AddEdge_DirectionOccupied(t *testing.T) {
	g := NewGraph("a")
	require.NoError(t, g.AddNode("a", nil))
	require.NoError(t, g.AddNode("b", nil))
	require.NoError(t, g.AddNode("c", nil))
	require.NoError(t, g.AddEdge("a", "b", North, "open"))

	// "a" already has a north edge.
	err := g.AddEdge("a", "c", North, "open")
	require.Error(t, err)
	var occ *DirectionOccupiedError
	assert.True(t, errors.As(err, &occ))
	assert.Equal(t, "a", occ.Node)
	assert.Equal(t, North, occ.Dir)
}

func TestGraph_AddEdge_ReverseDirectionOccupied(t *testing.T) {
	g := NewGraph("a")
	require.NoError(t, g.AddNode("a", nil))
	require.NoError(t, g.AddNode("b", nil))
	require.NoError(t, g.AddNode("c", nil))
	require.NoError(t, g.AddEdge("a", "b", North, "open"))

	// "b" already has a south edge (reverse of a→b north).
	err := g.AddEdge("c", "b", North, "open")
	require.Error(t, err)
	var occ *DirectionOccupiedError
	assert.True(t, errors.As(err, &occ))
	assert.Equal(t, "b", occ.Node)
	assert.Equal(t, South, occ.Dir)
}

func TestGraph_AddEdge_InvalidDirection(t *testing.T) {
	g := NewGraph("a")
	require.NoError(t, g.AddNode("a", nil))
	require.NoError(t, g.AddNode("b", nil))
	err := g.AddEdge("a", "b", "northeast", "open")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid direction")
}

func TestGraph_Degree(t *testing.T) {
	g := makeLinearGraph(t, 3) // a--b--c
	assert.Equal(t, 1, g.Degree("a"))
	assert.Equal(t, 2, g.Degree("b"))
	assert.Equal(t, 1, g.Degree("c"))
}

func TestGraph_Distance(t *testing.T) {
	g := makeLinearGraph(t, 4) // a--b--c--d, start=a
	assert.Equal(t, 0, g.Distance("a"))
	assert.Equal(t, 1, g.Distance("b"))
	assert.Equal(t, 2, g.Distance("c"))
	assert.Equal(t, 3, g.Distance("d"))
}

func TestGraph_Distance_Unreachable(t *testing.T) {
	g := NewGraph("a")
	require.NoError(t, g.AddNode("a", nil))
	require.NoError(t, g.AddNode("island", nil))
	assert.Equal(t, -1, g.Distance("island"))
}

func TestGraph_Validate_Valid(t *testing.T) {
	g := makeLinearGraph(t, 3)
	require.NoError(t, g.Validate())
}

func TestGraph_Validate_StartNotInGraph(t *testing.T) {
	g := &Graph{
		Nodes: map[string]*Node{"a": {ID: "a"}},
		Start: "missing",
	}
	err := g.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "start node")
}

func TestGraph_Validate_EdgeReferencesUnknownNode(t *testing.T) {
	g := NewGraph("a")
	require.NoError(t, g.AddNode("a", nil))
	g.Edges = append(g.Edges, Edge{From: "a", To: "ghost", FromDir: North, ToDir: South, Type: "open"})
	err := g.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ghost")
}

func TestGraph_Validate_BrokenBidirectionality(t *testing.T) {
	g := NewGraph("a")
	require.NoError(t, g.AddNode("a", nil))
	require.NoError(t, g.AddNode("b", nil))
	// Manually insert a bad edge where ToDir != FromDir.Opposite().
	g.Edges = append(g.Edges, Edge{From: "a", To: "b", FromDir: North, ToDir: North, Type: "open"})
	err := g.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "opposite")
}

func TestGraph_Validate_Unreachable(t *testing.T) {
	g := NewGraph("a")
	require.NoError(t, g.AddNode("a", nil))
	require.NoError(t, g.AddNode("island", nil))
	err := g.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unreachable")
}

func TestGraph_Validate_MaxDegree(t *testing.T) {
	// Create a hub node with 7 connections (exceeds max 6).
	g := NewGraph("hub")
	require.NoError(t, g.AddNode("hub", nil))
	dirs := AllDirections
	for i := 0; i < 7; i++ {
		id := string(rune('a' + i))
		require.NoError(t, g.AddNode(id, nil))
		if i < len(dirs) {
			require.NoError(t, g.AddEdge("hub", id, dirs[i], "open"))
		} else {
			// Force a 7th edge by directly appending (bypasses slot check).
			g.Edges = append(g.Edges, Edge{
				From: "hub", To: id, FromDir: North, ToDir: South, Type: "open",
			})
		}
	}
	err := g.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "degree")
}

// makeLinearGraph builds a→b→c→... chain using alternating N/S and E/W.
func makeLinearGraph(t *testing.T, n int) *Graph {
	t.Helper()
	if n < 1 {
		t.Fatal("need at least 1 node")
	}
	g := NewGraph("a")
	dirs := []Direction{North, East, Up}
	for i := 0; i < n; i++ {
		id := string(rune('a' + i))
		require.NoError(t, g.AddNode(id, nil))
	}
	for i := 0; i < n-1; i++ {
		from := string(rune('a' + i))
		to := string(rune('a' + i + 1))
		require.NoError(t, g.AddEdge(from, to, dirs[i%len(dirs)], "open"))
	}
	return g
}
