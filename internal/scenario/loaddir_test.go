package scenario_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/punt-labs/cryptd/internal/scenario"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDir_Valid(t *testing.T) {
	dir := setupDirScenario(t, `title: "Test"
starting_room: hall
death: respawn
regions:
  - regions/default.yaml
`, map[string]string{
		"default": `rooms:
  hall:
    name: "Hall"
`,
	})

	s, err := scenario.LoadDir(dir)
	require.NoError(t, err)
	assert.Equal(t, "Test", s.Title)
	assert.Contains(t, s.Rooms, "hall")
}

func TestLoadDir_PathTraversal(t *testing.T) {
	dir := setupDirScenario(t, `title: "Evil"
starting_room: hall
death: respawn
regions:
  - ../../../etc/passwd
`, nil)

	_, err := scenario.LoadDir(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "escapes scenario directory")
}

func TestLoadDir_DuplicateRoom(t *testing.T) {
	dir := setupDirScenario(t, `title: "Dup"
starting_room: hall
death: respawn
regions:
  - regions/a.yaml
  - regions/b.yaml
`, map[string]string{
		"a": `rooms:
  hall:
    name: "Hall A"
`,
		"b": `rooms:
  hall:
    name: "Hall B"
`,
	})

	_, err := scenario.LoadDir(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate room ID")
}

func setupDirScenario(t *testing.T, manifest string, regions map[string]string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "test-scenario")
	regionsDir := filepath.Join(dir, "regions")
	require.NoError(t, os.MkdirAll(regionsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "scenario.yaml"), []byte(manifest), 0o644))
	for name, content := range regions {
		require.NoError(t, os.WriteFile(filepath.Join(regionsDir, name+".yaml"), []byte(content), 0o644))
	}
	return dir
}
