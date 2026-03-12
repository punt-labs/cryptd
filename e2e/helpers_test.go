//go:build e2e

package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// serverBinary builds and returns the path to the cryptd server binary.
func serverBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "cryptd")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/cryptd")
	cmd.Dir = repoRoot(t)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "build cryptd failed: %s", out)
	return bin
}

func repoRoot(t *testing.T) string {
	t.Helper()
	// Navigate up from e2e/ to repo root.
	wd, err := os.Getwd()
	require.NoError(t, err)
	return filepath.Dir(wd)
}
