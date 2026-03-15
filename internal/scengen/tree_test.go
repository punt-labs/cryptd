package scengen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTreeSource_SimpleTree(t *testing.T) {
	// Create:  root/ → {a/, b/}
	root := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(root, "a"), 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(root, "b"), 0o755))

	ts := &TreeSource{Root: root}
	nodes, edges, start, err := ts.Generate()
	require.NoError(t, err)

	assert.Len(t, nodes, 3)
	assert.Len(t, edges, 2)
	assert.Equal(t, sanitizeID(filepath.Base(root)), start)

	nodeIDs := nodeIDSet(nodes)
	assert.Contains(t, nodeIDs, "a")
	assert.Contains(t, nodeIDs, "b")
}

func TestTreeSource_NestedTree(t *testing.T) {
	// Create:  root/ → a/ → b/
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "a", "b"), 0o755))

	ts := &TreeSource{Root: root}
	nodes, edges, _, err := ts.Generate()
	require.NoError(t, err)

	assert.Len(t, nodes, 3) // root, a, a/b
	assert.Len(t, edges, 2) // root→a, a→a/b
}

func TestTreeSource_SkipsFiles(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(root, "dir"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "file.txt"), []byte("hi"), 0o644))

	ts := &TreeSource{Root: root}
	nodes, _, _, err := ts.Generate()
	require.NoError(t, err)

	// Only root and dir — file.txt is skipped.
	assert.Len(t, nodes, 2)
}

func TestTreeSource_HubInsertion(t *testing.T) {
	// Create root with 8 children (exceeds maxChildrenPerNode=5).
	root := t.TempDir()
	for i := 0; i < 8; i++ {
		name := string(rune('a' + i))
		require.NoError(t, os.Mkdir(filepath.Join(root, name), 0o755))
	}

	ts := &TreeSource{Root: root}
	nodes, edges, start, err := ts.Generate()
	require.NoError(t, err)

	// Should have 8 original dirs + root + hub nodes.
	assert.Greater(t, len(nodes), 9, "hubs should be inserted")

	// Verify no parent has >maxChildrenPerNode children in the edge list.
	childCount := make(map[string]int)
	for _, e := range edges {
		childCount[e.From]++
	}
	for id, count := range childCount {
		assert.LessOrEqual(t, count, maxChildrenPerNode+1,
			"node %s has %d children after hub insertion", id, count)
	}

	// All original child IDs should still be reachable.
	nodeIDs := nodeIDSet(nodes)
	for i := 0; i < 8; i++ {
		name := string(rune('a' + i))
		assert.Contains(t, nodeIDs, name, "child %s missing after hub insertion", name)
	}

	// Hub nodes should be flagged.
	for _, n := range nodes {
		if n.Meta != nil && n.Meta["hub"] == "true" {
			assert.Contains(t, n.ID, "_hub_")
		}
	}

	assert.NotEmpty(t, start)
}

func TestTreeSource_EmptyRoot(t *testing.T) {
	root := t.TempDir()

	ts := &TreeSource{Root: root}
	nodes, edges, _, err := ts.Generate()
	require.NoError(t, err)

	assert.Len(t, nodes, 1) // just root
	assert.Empty(t, edges)
}

func TestTreeSource_NotADirectory(t *testing.T) {
	f := filepath.Join(t.TempDir(), "file.txt")
	require.NoError(t, os.WriteFile(f, []byte("hi"), 0o644))

	ts := &TreeSource{Root: f}
	_, _, _, err := ts.Generate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a directory")
}

func TestTreeSource_NonexistentRoot(t *testing.T) {
	ts := &TreeSource{Root: "/nonexistent/path/abcdef"}
	_, _, _, err := ts.Generate()
	require.Error(t, err)
}

func TestPathToID(t *testing.T) {
	tests := []struct {
		path string
		root string
		want string
	}{
		{"/usr/local/bin", "/usr/local", "bin"},
		{"/usr/local", "/usr/local", "local"},
		{"/usr/local/share/man", "/usr/local", "share_man"},
	}
	for _, tt := range tests {
		got, err := pathToID(tt.path, tt.root)
		require.NoError(t, err, "pathToID(%q, %q)", tt.path, tt.root)
		assert.Equal(t, tt.want, got, "pathToID(%q, %q)", tt.path, tt.root)
	}
}

func TestTreeSource_IDCollision(t *testing.T) {
	// Two directories whose names collide after sanitization: "my-dir" and "my.dir"
	root := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(root, "my-dir"), 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(root, "my.dir"), 0o755))

	ts := &TreeSource{Root: root}
	_, _, _, err := ts.Generate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ID collision")
}

func TestSanitizeID(t *testing.T) {
	assert.Equal(t, "hello_world", sanitizeID("hello-world"))
	assert.Equal(t, "foo_bar", sanitizeID("foo.bar"))
	assert.Equal(t, "a_b", sanitizeID("a/b"))
}

func nodeIDSet(nodes []RawNode) map[string]bool {
	ids := make(map[string]bool, len(nodes))
	for _, n := range nodes {
		ids[n.ID] = true
	}
	return ids
}
