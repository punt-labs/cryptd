package scengen

import (
	"fmt"

	"github.com/punt-labs/cryptd/internal/scenario"
)

// Visitor populates ScenarioContent by walking a Graph.
type Visitor interface {
	Name() string
	Visit(g *Graph, content *ScenarioContent) error
}

// ScenarioContent holds the generated content for all rooms.
type ScenarioContent struct {
	Title   string
	Death   string // permadeath|respawn
	Rooms   map[string]*RoomContent
	Items   map[string]*scenario.ScenarioItem
	Enemies map[string]*scenario.EnemyTemplate
	Spells  map[string]*scenario.SpellTemplate
}

// NewScenarioContent creates an empty content container.
func NewScenarioContent() *ScenarioContent {
	return &ScenarioContent{
		Rooms:   make(map[string]*RoomContent),
		Items:   make(map[string]*scenario.ScenarioItem),
		Enemies: make(map[string]*scenario.EnemyTemplate),
		Spells:  make(map[string]*scenario.SpellTemplate),
	}
}

// RoomContent holds generated content for a single room.
type RoomContent struct {
	Name            string
	DescriptionSeed string
	Items           []string
	Enemies         []string
}

// DescriptionVisitor populates room names and description seeds from node metadata.
// For regular nodes, it uses the "name" and "path" metadata fields.
// For synthetic hub nodes (inserted when a directory exceeds maxChildrenPerNode children),
// it generates generic corridor descriptions.
type DescriptionVisitor struct{}

func (v *DescriptionVisitor) Name() string { return "description" }

func (v *DescriptionVisitor) Visit(g *Graph, content *ScenarioContent) error {
	for id, node := range g.Nodes {
		rc, ok := content.Rooms[id]
		if !ok {
			rc = &RoomContent{}
			content.Rooms[id] = rc
		}

		if rc.Name != "" {
			continue // already populated by a prior visitor
		}

		if name, ok := node.Meta["name"]; ok && name != "" {
			rc.Name = name
		} else {
			rc.Name = id
		}

		if node.Meta["hub"] == "true" {
			rc.DescriptionSeed = fmt.Sprintf("A corridor connecting several passages near %s.", rc.Name)
		} else if path, ok := node.Meta["path"]; ok && path != "" {
			rc.DescriptionSeed = fmt.Sprintf("You find yourself in %s.", path)
		} else {
			rc.DescriptionSeed = fmt.Sprintf("A room called %s.", rc.Name)
		}
	}
	return nil
}
