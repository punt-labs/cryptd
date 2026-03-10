//go:build e2e

package e2e

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// binary returns the path to the compiled cryptd binary, building it if needed.
func binary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "cryptd")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/crypt")
	cmd.Dir = repoRoot(t)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "build failed: %s", out)
	return bin
}

func repoRoot(t *testing.T) string {
	t.Helper()
	// Navigate up from e2e/ to repo root.
	wd, err := os.Getwd()
	require.NoError(t, err)
	return filepath.Dir(wd)
}

func TestHeadless_MinimalRunCommands(t *testing.T) {
	bin := binary(t)
	root := repoRoot(t)

	// Build stdin from script steps.
	stdin := "go south\nlook\ngo north\nquit\n"

	cmd := exec.Command(bin, "headless", "--scenario", "minimal")
	cmd.Dir = root
	cmd.Stdin = strings.NewReader(stdin)
	// Point scenario lookup at testdata.
	cmd.Env = append(os.Environ(), "CRYPT_SCENARIO_DIR=testdata/scenarios")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	require.NoError(t, err, "cryptd headless exited non-zero; stderr: %s", stderr.String())

	out := stdout.String()
	assert.Contains(t, out, "goblin_lair", "expected goblin_lair in output after moving south")
	assert.Contains(t, out, "entrance", "expected entrance in output after moving north")
}
