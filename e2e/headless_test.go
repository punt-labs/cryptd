//go:build e2e

package e2e

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeadless_MinimalRunCommands(t *testing.T) {
	bin := serverBinary(t)
	root := repoRoot(t)

	// Provide headless commands via stdin.
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
