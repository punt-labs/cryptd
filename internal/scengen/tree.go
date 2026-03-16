package scengen

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// maxChildrenPerNode is the maximum number of children a single node can have
// before hub insertion is required. 5 (not 6) because one direction is used
// for the parent link.
const maxChildrenPerNode = 5

// TreeSource generates a graph from a filesystem directory tree.
// Each directory becomes a node; parent-child relationships become edges.
type TreeSource struct {
	Root string // filesystem path to walk
}

// Generate walks the filesystem tree rooted at Root and returns raw nodes
// and edges. Directories with >5 children get synthetic hub nodes inserted.
func (ts *TreeSource) Generate() ([]RawNode, []RawEdge, string, error) {
	root := filepath.Clean(ts.Root)
	info, err := os.Stat(root)
	if err != nil {
		return nil, nil, "", fmt.Errorf("stat source root: %w", err)
	}
	if !info.IsDir() {
		return nil, nil, "", fmt.Errorf("source root %q is not a directory", root)
	}

	var nodes []RawNode
	var edges []RawEdge
	idPaths := make(map[string]string) // node ID → original path (collision detection)

	startID, err := pathToID(root, root)
	if err != nil {
		return nil, nil, "", fmt.Errorf("id for root: %w", err)
	}
	nodes = append(nodes, RawNode{
		ID:   startID,
		Meta: map[string]string{"name": filepath.Base(root), "path": root},
	})
	idPaths[startID] = root

	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil // skip files
		}
		if path == root {
			return nil // already added
		}

		id, idErr := pathToID(path, root)
		if idErr != nil {
			return fmt.Errorf("id for %s: %w", path, idErr)
		}
		if existing, collision := idPaths[id]; collision {
			return fmt.Errorf("ID collision: %q and %q both produce ID %q", existing, path, id)
		}
		idPaths[id] = path

		nodes = append(nodes, RawNode{
			ID:   id,
			Meta: map[string]string{"name": filepath.Base(path), "path": path},
		})

		parentID, idErr := pathToID(filepath.Dir(path), root)
		if idErr != nil {
			return fmt.Errorf("id for parent of %s: %w", path, idErr)
		}
		edges = append(edges, RawEdge{From: parentID, To: id, EdgeType: "open"})

		return nil
	})
	if err != nil {
		return nil, nil, "", fmt.Errorf("walk source tree: %w", err)
	}

	// Insert hubs for nodes with >maxChildrenPerNode children.
	nodes, edges = insertHubs(nodes, edges)

	return nodes, edges, startID, nil
}

// insertHubs finds nodes with too many children and splits them using
// synthetic hub nodes chained together.
func insertHubs(nodes []RawNode, edges []RawEdge) ([]RawNode, []RawEdge) {
	// Build parent→children map.
	children := make(map[string][]int) // parent ID → edge indices
	for i, e := range edges {
		children[e.From] = append(children[e.From], i)
	}

	var newNodes []RawNode
	var newEdges []RawEdge

	// Copy all original edges, then replace those that need hub insertion.
	replaced := make(map[int]bool)

	for parentID, childIndices := range children {
		if len(childIndices) <= maxChildrenPerNode {
			continue
		}

		// Sort child indices for deterministic output.
		sort.Ints(childIndices)

		// Mark all child edges of this parent as replaced.
		for _, idx := range childIndices {
			replaced[idx] = true
		}

		// Chain hubs: the first batch connects directly to the parent but
		// reserves one direction slot for the hub link to the next batch.
		// Subsequent hubs get the full maxChildrenPerNode capacity (one
		// slot is used for the prev-hub link, leaving maxChildrenPerNode
		// for children since they have no other parent edge).
		hubNum := 0
		firstBatch := maxChildrenPerNode - 1 // reserve 1 slot for hub→next link
		pos := 0
		for pos < len(childIndices) {
			batchSize := maxChildrenPerNode
			if pos == 0 {
				batchSize = firstBatch
			}
			end := pos + batchSize
			if end > len(childIndices) {
				end = len(childIndices)
			}
			batch := childIndices[pos:end]

			var hubID string
			if pos == 0 {
				// First batch connects directly to the parent.
				hubID = parentID
			} else {
				hubNum++
				hubID = fmt.Sprintf("%s_hub_%d", parentID, hubNum)
				// Find parent name for the hub.
				parentName := parentID
				for _, n := range nodes {
					if n.ID == parentID {
						if name, ok := n.Meta["name"]; ok {
							parentName = name
						}
						break
					}
				}
				newNodes = append(newNodes, RawNode{
					ID: hubID,
					Meta: map[string]string{
						"name": fmt.Sprintf("%s Corridor %d", parentName, hubNum),
						"hub":  "true",
					},
				})
				// Link previous hub/parent to this hub.
				var prevHub string
				if hubNum == 1 {
					prevHub = parentID
				} else {
					prevHub = fmt.Sprintf("%s_hub_%d", parentID, hubNum-1)
				}
				newEdges = append(newEdges, RawEdge{From: prevHub, To: hubID, EdgeType: "open"})
			}

			// Connect batch children to this hub.
			for _, idx := range batch {
				newEdges = append(newEdges, RawEdge{From: hubID, To: edges[idx].To, EdgeType: edges[idx].EdgeType})
			}
			pos = end
		}
	}

	// Collect non-replaced original edges.
	var finalEdges []RawEdge
	for i, e := range edges {
		if !replaced[i] {
			finalEdges = append(finalEdges, e)
		}
	}
	finalEdges = append(finalEdges, newEdges...)

	allNodes := append(nodes, newNodes...)
	return allNodes, finalEdges
}

// pathToID converts a filesystem path to a graph node ID.
// Uses the relative path from root, replacing separators with underscores.
// Returns an error if the relative path cannot be computed.
func pathToID(path, root string) (string, error) {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return "", fmt.Errorf("relative path from %q to %q: %w", root, path, err)
	}
	if rel == "." {
		return sanitizeID(filepath.Base(path)), nil
	}
	return sanitizeID(rel), nil
}

// sanitizeID replaces path separators and special characters with underscores.
func sanitizeID(s string) string {
	s = strings.ReplaceAll(s, string(filepath.Separator), "_")
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, ".", "_")
	return strings.ToLower(s)
}
