package scengen

import (
	"fmt"

	"github.com/punt-labs/cryptd/internal/scenario"
)

// GenerateOptions configures the generation pipeline.
type GenerateOptions struct {
	Title    string
	Death    string // permadeath|respawn; defaults to "respawn"
	Visitors []Visitor
}

// Generate runs the full pipeline: topology → direction assignment → visitors → Scenario.
func Generate(source TopologySource, opts GenerateOptions) (*scenario.Scenario, error) {
	rawNodes, rawEdges, start, err := source.Generate()
	if err != nil {
		return nil, fmt.Errorf("topology source: %w", err)
	}

	g, err := assignDirections(rawNodes, rawEdges, start)
	if err != nil {
		return nil, fmt.Errorf("direction assignment: %w", err)
	}

	if err := g.Validate(); err != nil {
		return nil, fmt.Errorf("post-assignment validation: %w", err)
	}

	content := NewScenarioContent()
	content.Title = opts.Title
	content.Death = opts.Death
	if content.Death == "" {
		content.Death = "respawn"
	}

	for _, v := range opts.Visitors {
		if err := v.Visit(g, content); err != nil {
			return nil, fmt.Errorf("visitor %s: %w", v.Name(), err)
		}
	}

	return toScenario(g, content), nil
}

// GenerateGraph runs the pipeline up to graph construction without visitors,
// returning the validated graph for further processing (e.g., export, SQLite storage).
func GenerateGraph(source TopologySource) (*Graph, error) {
	rawNodes, rawEdges, start, err := source.Generate()
	if err != nil {
		return nil, fmt.Errorf("topology source: %w", err)
	}

	g, err := assignDirections(rawNodes, rawEdges, start)
	if err != nil {
		return nil, fmt.Errorf("direction assignment: %w", err)
	}

	if err := g.Validate(); err != nil {
		return nil, fmt.Errorf("post-assignment validation: %w", err)
	}

	return g, nil
}

// assignDirections converts raw topology into a directed Graph.
// Uses BFS from start to assign parent→child directions greedily.
func assignDirections(rawNodes []RawNode, rawEdges []RawEdge, start string) (*Graph, error) {
	g := NewGraph(start)

	for _, rn := range rawNodes {
		if err := g.AddNode(rn.ID, rn.Meta); err != nil {
			return nil, err
		}
	}

	if _, ok := g.Nodes[start]; !ok {
		return nil, fmt.Errorf("start node %q not found in topology", start)
	}

	// Build adjacency list from raw edges.
	type adjEntry struct {
		to       string
		edgeType string
	}
	adj := make(map[string][]adjEntry)
	for _, re := range rawEdges {
		adj[re.From] = append(adj[re.From], adjEntry{to: re.To, edgeType: re.EdgeType})
		adj[re.To] = append(adj[re.To], adjEntry{to: re.From, edgeType: re.EdgeType})
	}

	// BFS from start, assigning directions as we go.
	visited := map[string]bool{start: true}
	queue := []string{start}

	// Track which directions are used at each node.
	usedDirs := make(map[string]map[Direction]bool)
	for _, rn := range rawNodes {
		usedDirs[rn.ID] = make(map[Direction]bool)
	}

	// Preferred direction order for children (horizontal first, then vertical).
	preferredDirs := []Direction{North, East, South, West, Up, Down}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		for _, neighbor := range adj[curr] {
			if visited[neighbor.to] {
				continue
			}
			visited[neighbor.to] = true

			// Find an available direction pair.
			assigned := false
			for _, dir := range preferredDirs {
				opp := dir.Opposite()
				if usedDirs[curr][dir] || usedDirs[neighbor.to][opp] {
					continue
				}
				usedDirs[curr][dir] = true
				usedDirs[neighbor.to][opp] = true
				if err := g.AddEdge(curr, neighbor.to, dir, neighbor.edgeType); err != nil {
					return nil, fmt.Errorf("adding edge %s→%s: %w", curr, neighbor.to, err)
				}
				assigned = true
				break
			}
			if !assigned {
				return nil, fmt.Errorf("no available direction for edge %s→%s (both nodes fully connected)", curr, neighbor.to)
			}

			queue = append(queue, neighbor.to)
		}
	}

	return g, nil
}

// toScenario converts a Graph + ScenarioContent into a scenario.Scenario.
func toScenario(g *Graph, content *ScenarioContent) *scenario.Scenario {
	rooms := make(map[string]*scenario.Room, len(g.Nodes))

	// Initialize all rooms from graph nodes.
	for id := range g.Nodes {
		rc := content.Rooms[id]
		room := &scenario.Room{
			Connections: make(map[string]*scenario.Connection),
		}
		if rc != nil {
			room.Name = rc.Name
			room.DescriptionSeed = rc.DescriptionSeed
			room.Items = rc.Items
			room.Enemies = rc.Enemies
		}
		if room.Name == "" {
			room.Name = id
		}
		rooms[id] = room
	}

	// Populate connections from edges.
	for _, e := range g.Edges {
		rooms[e.From].Connections[string(e.FromDir)] = &scenario.Connection{
			Room: e.To,
			Type: e.Type,
		}
		rooms[e.To].Connections[string(e.ToDir)] = &scenario.Connection{
			Room: e.From,
			Type: e.Type,
		}
	}

	s := &scenario.Scenario{
		Title:        content.Title,
		StartingRoom: g.Start,
		Death:        content.Death,
		Rooms:        rooms,
		Items:        content.Items,
		Enemies:      content.Enemies,
		Spells:       content.Spells,
	}

	if s.Items == nil {
		s.Items = make(map[string]*scenario.ScenarioItem)
	}
	if s.Enemies == nil {
		s.Enemies = make(map[string]*scenario.EnemyTemplate)
	}
	if s.Spells == nil {
		s.Spells = make(map[string]*scenario.SpellTemplate)
	}

	return s
}
