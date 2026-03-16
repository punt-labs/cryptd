// Package scengen generates game scenarios from graph-based topology sources.
// See DES-027 for the design rationale.
package scengen

import "fmt"

// Direction represents one of six compass directions in the game world.
type Direction string

const (
	North Direction = "north"
	South Direction = "south"
	East  Direction = "east"
	West  Direction = "west"
	Up    Direction = "up"
	Down  Direction = "down"
)

// AllDirections lists every valid direction.
var AllDirections = []Direction{North, South, East, West, Up, Down}

// opposites maps each direction to its reverse.
var opposites = map[Direction]Direction{
	North: South,
	South: North,
	East:  West,
	West:  East,
	Up:    Down,
	Down:  Up,
}

// Opposite returns the reverse direction.
func (d Direction) Opposite() Direction {
	return opposites[d]
}

// Valid reports whether d is one of the six recognized directions.
func (d Direction) Valid() bool {
	_, ok := opposites[d]
	return ok
}

// Node is a single location in the generated scenario graph.
type Node struct {
	ID   string
	Meta map[string]string // topology-source metadata (e.g. filesystem path)
}

// Edge is a bidirectional connection between two nodes.
// ToDir is always FromDir.Opposite().
type Edge struct {
	From    string
	To      string
	FromDir Direction
	ToDir   Direction
	Type    string // open|locked|hidden|stairway
}

// Graph is the core scenario graph: nodes, edges, and a designated start node.
type Graph struct {
	Nodes map[string]*Node
	Edges []Edge
	Start string
	Meta  map[string]string // scenario-level metadata (title, death, etc.)
}

// NewGraph creates an empty graph with the given start node.
func NewGraph(start string) *Graph {
	return &Graph{
		Nodes: make(map[string]*Node),
		Meta:  make(map[string]string),
		Start: start,
	}
}

// AddNode adds a node to the graph. Returns an error if the ID already exists.
func (g *Graph) AddNode(id string, meta map[string]string) error {
	if _, exists := g.Nodes[id]; exists {
		return &DuplicateNodeError{ID: id}
	}
	g.Nodes[id] = &Node{ID: id, Meta: meta}
	return nil
}

// AddEdge adds a bidirectional edge. Returns an error if direction slots are occupied.
func (g *Graph) AddEdge(from, to string, fromDir Direction, edgeType string) error {
	if !fromDir.Valid() {
		return fmt.Errorf("invalid direction: %s", fromDir)
	}
	toDir := fromDir.Opposite()

	// Check that both direction slots are free.
	for _, e := range g.Edges {
		if e.From == from && e.FromDir == fromDir {
			return &DirectionOccupiedError{Node: from, Dir: fromDir}
		}
		if e.To == from && e.ToDir == fromDir {
			return &DirectionOccupiedError{Node: from, Dir: fromDir}
		}
		if e.From == to && e.FromDir == toDir {
			return &DirectionOccupiedError{Node: to, Dir: toDir}
		}
		if e.To == to && e.ToDir == toDir {
			return &DirectionOccupiedError{Node: to, Dir: toDir}
		}
	}

	g.Edges = append(g.Edges, Edge{
		From:    from,
		To:      to,
		FromDir: fromDir,
		ToDir:   toDir,
		Type:    edgeType,
	})
	return nil
}

// Degree returns the number of edges connected to a node.
func (g *Graph) Degree(id string) int {
	count := 0
	for _, e := range g.Edges {
		if e.From == id || e.To == id {
			count++
		}
	}
	return count
}

// Distance returns the BFS depth from Start to the given node.
// Returns -1 if the node is unreachable.
func (g *Graph) Distance(id string) int {
	if g.Start == id {
		return 0
	}

	adj := g.adjacencyMap()
	visited := map[string]bool{g.Start: true}
	queue := []string{g.Start}
	depth := map[string]int{g.Start: 0}

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		for _, neighbor := range adj[curr] {
			if visited[neighbor] {
				continue
			}
			visited[neighbor] = true
			depth[neighbor] = depth[curr] + 1
			if neighbor == id {
				return depth[neighbor]
			}
			queue = append(queue, neighbor)
		}
	}
	return -1
}

// MaxDistance returns the maximum BFS depth from Start across all nodes.
// Returns 0 if the graph has only the start node.
func (g *Graph) MaxDistance() int {
	adj := g.adjacencyMap()
	visited := map[string]bool{g.Start: true}
	queue := []string{g.Start}
	depth := map[string]int{g.Start: 0}
	maxD := 0

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		for _, neighbor := range adj[curr] {
			if visited[neighbor] {
				continue
			}
			visited[neighbor] = true
			depth[neighbor] = depth[curr] + 1
			if depth[neighbor] > maxD {
				maxD = depth[neighbor]
			}
			queue = append(queue, neighbor)
		}
	}
	return maxD
}

// Validate checks graph invariants:
//   - Start node exists in Nodes
//   - Every edge references existing nodes
//   - ToDir == FromDir.Opposite() (bidirectionality)
//   - No node has more than 6 edges (one per direction)
//   - All nodes are reachable from Start
func (g *Graph) Validate() error {
	if _, ok := g.Nodes[g.Start]; !ok {
		return &ValidationError{Reason: fmt.Sprintf("start node %q not in graph", g.Start)}
	}

	for i, e := range g.Edges {
		if _, ok := g.Nodes[e.From]; !ok {
			return &ValidationError{Reason: fmt.Sprintf("edge %d: from node %q not in graph", i, e.From)}
		}
		if _, ok := g.Nodes[e.To]; !ok {
			return &ValidationError{Reason: fmt.Sprintf("edge %d: to node %q not in graph", i, e.To)}
		}
		if e.ToDir != e.FromDir.Opposite() {
			return &ValidationError{Reason: fmt.Sprintf("edge %d: to_dir %s != opposite of from_dir %s", i, e.ToDir, e.FromDir)}
		}
	}

	for id := range g.Nodes {
		if g.Degree(id) > 6 {
			return &ValidationError{Reason: fmt.Sprintf("node %q has degree %d > 6", id, g.Degree(id))}
		}
	}

	// BFS reachability from Start.
	adj := g.adjacencyMap()
	visited := map[string]bool{g.Start: true}
	queue := []string{g.Start}
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		for _, neighbor := range adj[curr] {
			if !visited[neighbor] {
				visited[neighbor] = true
				queue = append(queue, neighbor)
			}
		}
	}

	for id := range g.Nodes {
		if !visited[id] {
			return &ValidationError{Reason: fmt.Sprintf("node %q unreachable from start %q", id, g.Start)}
		}
	}

	return nil
}

// adjacencyMap builds a bidirectional adjacency list from the edge list.
func (g *Graph) adjacencyMap() map[string][]string {
	adj := make(map[string][]string)
	for _, e := range g.Edges {
		adj[e.From] = append(adj[e.From], e.To)
		adj[e.To] = append(adj[e.To], e.From)
	}
	return adj
}

// --- Typed errors ---

// DuplicateNodeError is returned when adding a node with an ID that already exists.
type DuplicateNodeError struct {
	ID string
}

func (e *DuplicateNodeError) Error() string {
	return fmt.Sprintf("duplicate node ID: %s", e.ID)
}

// DirectionOccupiedError is returned when a direction slot is already in use.
type DirectionOccupiedError struct {
	Node string
	Dir  Direction
}

func (e *DirectionOccupiedError) Error() string {
	return fmt.Sprintf("node %q already has an edge in direction %s", e.Node, e.Dir)
}

// ValidationError is returned by Graph.Validate() when an invariant is violated.
type ValidationError struct {
	Reason string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("graph validation failed: %s", e.Reason)
}
