package protocol

import (
	"os"
	"path/filepath"
)

// DefaultSocketPath returns ~/.crypt/daemon.sock.
// Both the server (cryptd serve) and client (crypt) use this
// as the default socket address when no explicit path is given.
func DefaultSocketPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".crypt", "daemon.sock"), nil
}
