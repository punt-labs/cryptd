package layout

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultSocketPath_UsesHomeWithoutCreatingDirectories(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Chdir(t.TempDir())

	socket, err := DefaultSocketPath()
	require.NoError(t, err)
	require.Equal(t, filepath.Join(home, ".punt-labs", "cryptd", "run", "daemon.sock"), socket)
	require.NoDirExists(t, filepath.Dir(socket))
}

func TestDefaultSocketPath_RejectsUnavailableHome(t *testing.T) {
	for _, home := range []string{"", "relative-home"} {
		t.Setenv("HOME", home)
		socket, err := DefaultSocketPath()
		require.Error(t, err)
		require.Empty(t, socket)
	}
}
