package scenariodir

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_SingleFile(t *testing.T) {
	s, err := Load("../../testdata/scenarios", "minimal")
	require.NoError(t, err)
	assert.Equal(t, "minimal", s.ID)
	assert.Equal(t, "Minimal Dungeon", s.Title)
}

func TestLoad_DirectoryFormat(t *testing.T) {
	// Create a directory-format scenario in a temp dir.
	tmp := t.TempDir()
	scenDir := filepath.Join(tmp, "test-dir")
	regionsDir := filepath.Join(scenDir, "regions")
	require.NoError(t, os.MkdirAll(regionsDir, 0o755))

	manifest := `title: "Dir Test"
starting_room: hall
death: respawn
regions:
  - regions/default.yaml
`
	require.NoError(t, os.WriteFile(filepath.Join(scenDir, "scenario.yaml"), []byte(manifest), 0o644))

	region := `rooms:
  hall:
    name: "Grand Hall"
    description_seed: "A vast hall."
`
	require.NoError(t, os.WriteFile(filepath.Join(regionsDir, "default.yaml"), []byte(region), 0o644))

	s, err := Load(tmp, "test-dir")
	require.NoError(t, err)
	assert.Equal(t, "test-dir", s.ID)
	assert.Equal(t, "Dir Test", s.Title)
	assert.Contains(t, s.Rooms, "hall")
}

func TestLoad_DirectoryTakesPrecedence(t *testing.T) {
	// When both dir/id/scenario.yaml and dir/id.yaml exist,
	// directory format wins.
	tmp := t.TempDir()

	// Create single-file scenario.
	singleFile := `title: "Single File"
starting_room: room
rooms:
  room:
    name: "Room"
`
	require.NoError(t, os.WriteFile(filepath.Join(tmp, "ambiguous.yaml"), []byte(singleFile), 0o644))

	// Create directory-format scenario.
	scenDir := filepath.Join(tmp, "ambiguous")
	regionsDir := filepath.Join(scenDir, "regions")
	require.NoError(t, os.MkdirAll(regionsDir, 0o755))

	manifest := `title: "Directory Format"
starting_room: room
death: respawn
regions:
  - regions/default.yaml
`
	require.NoError(t, os.WriteFile(filepath.Join(scenDir, "scenario.yaml"), []byte(manifest), 0o644))
	region := `rooms:
  room:
    name: "Room"
`
	require.NoError(t, os.WriteFile(filepath.Join(regionsDir, "default.yaml"), []byte(region), 0o644))

	s, err := Load(tmp, "ambiguous")
	require.NoError(t, err)
	assert.Equal(t, "Directory Format", s.Title, "directory format should take precedence")
}

func TestLoad_PathTraversal(t *testing.T) {
	_, err := Load("/tmp", "../etc/passwd")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid scenario ID")
}

func TestDir_Default(t *testing.T) {
	orig := os.Getenv("CRYPT_SCENARIO_DIR")
	defer func() { os.Setenv("CRYPT_SCENARIO_DIR", orig) }()

	os.Unsetenv("CRYPT_SCENARIO_DIR")
	assert.Equal(t, "scenarios", Dir())

	os.Setenv("CRYPT_SCENARIO_DIR", "/custom/path")
	assert.Equal(t, "/custom/path", Dir())
}
