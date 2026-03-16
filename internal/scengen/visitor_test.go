package scengen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDescriptionVisitor_Name(t *testing.T) {
	v := &DescriptionVisitor{}
	assert.Equal(t, "description", v.Name())
}

func TestDescriptionVisitor_UsesMetadataName(t *testing.T) {
	g := NewGraph("root")
	require.NoError(t, g.AddNode("root", map[string]string{"name": "Root Chamber", "path": "/"}))

	content := NewScenarioContent()
	v := &DescriptionVisitor{}
	require.NoError(t, v.Visit(g, content))

	rc := content.Rooms["root"]
	require.NotNil(t, rc)
	assert.Equal(t, "Root Chamber", rc.Name)
	assert.Contains(t, rc.DescriptionSeed, "/")
}

func TestDescriptionVisitor_FallsBackToID(t *testing.T) {
	g := NewGraph("room_42")
	require.NoError(t, g.AddNode("room_42", nil))

	content := NewScenarioContent()
	v := &DescriptionVisitor{}
	require.NoError(t, v.Visit(g, content))

	rc := content.Rooms["room_42"]
	require.NotNil(t, rc)
	assert.Equal(t, "room_42", rc.Name)
	assert.Contains(t, rc.DescriptionSeed, "room_42")
}

func TestDescriptionVisitor_HubNode(t *testing.T) {
	g := NewGraph("hub_1")
	require.NoError(t, g.AddNode("hub_1", map[string]string{"name": "Junction", "hub": "true"}))

	content := NewScenarioContent()
	v := &DescriptionVisitor{}
	require.NoError(t, v.Visit(g, content))

	rc := content.Rooms["hub_1"]
	require.NotNil(t, rc)
	assert.Equal(t, "Junction", rc.Name)
	assert.Contains(t, rc.DescriptionSeed, "corridor")
}

func TestDescriptionVisitor_DoesNotOverwrite(t *testing.T) {
	g := NewGraph("a")
	require.NoError(t, g.AddNode("a", map[string]string{"name": "Should Not Replace"}))

	content := NewScenarioContent()
	content.Rooms["a"] = &RoomContent{Name: "Already Set", DescriptionSeed: "Already described."}

	v := &DescriptionVisitor{}
	require.NoError(t, v.Visit(g, content))

	assert.Equal(t, "Already Set", content.Rooms["a"].Name)
}

func TestDescriptionVisitor_MultipleNodes(t *testing.T) {
	g := makeLinearGraph(t, 3) // a, b, c
	g.Nodes["a"].Meta = map[string]string{"name": "Entrance", "path": "/entrance"}
	g.Nodes["b"].Meta = map[string]string{"name": "Hall", "path": "/hall"}
	g.Nodes["c"].Meta = map[string]string{"name": "Treasury", "path": "/treasury"}

	content := NewScenarioContent()
	v := &DescriptionVisitor{}
	require.NoError(t, v.Visit(g, content))

	assert.Len(t, content.Rooms, 3)
	assert.Equal(t, "Entrance", content.Rooms["a"].Name)
	assert.Equal(t, "Hall", content.Rooms["b"].Name)
	assert.Equal(t, "Treasury", content.Rooms["c"].Name)
}
