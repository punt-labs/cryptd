package scengen

// RawNode is a node emitted by a TopologySource before direction assignment.
type RawNode struct {
	ID   string
	Meta map[string]string
}

// RawEdge is an undirected connection emitted by a TopologySource.
// Directions are assigned later by the generator.
type RawEdge struct {
	From     string
	To       string
	EdgeType string // open|locked|hidden|stairway
}

// TopologySource generates raw nodes and edges from an external structure
// (filesystem tree, grid, cave algorithm, etc.). It does not assign directions.
type TopologySource interface {
	Generate() (nodes []RawNode, edges []RawEdge, start string, err error)
}
