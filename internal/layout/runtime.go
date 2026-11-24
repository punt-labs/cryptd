package layout

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	Tool              = "cryptd"
	runDir            = ".punt-labs/" + Tool + "/run"
	socketName        = "daemon.sock"
	DefaultSocketHelp = "Unix socket path (default ~/" + runDir + "/" + socketName + ")"
)

func RunDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(home) {
		return "", fmt.Errorf("home directory is not absolute")
	}
	return filepath.Join(home, runDir), nil
}

func DefaultSocketPath() (string, error) {
	dir, err := RunDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, socketName), nil
}
