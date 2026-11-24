package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/punt-labs/cryptd/internal/layout"
	"github.com/punt-labs/cryptd/internal/observe"
)

const daemonEnv = "_CRYPTD_DAEMON"

func isDaemonChild() bool {
	return os.Getenv(daemonEnv) == "1"
}

// daemonize starts a new process with no attached standard streams.
// The parent confirms only spawn, not that the child is listening.
func daemonize(log *slog.Logger) int {
	id := observe.NewLaunchID()
	log = log.With("launch_id", id)
	cmd := exec.Command(os.Args[0], os.Args[1:]...)
	cmd.Env = append(os.Environ(), daemonEnv+"=1", observe.LaunchIDEnv+"="+id)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	log.Info("Spawning detached daemon")
	if err := cmd.Start(); err != nil {
		log.Error("Could not spawn detached daemon", "error_kind", observe.ErrorKind(err))
		fmt.Fprintf(os.Stderr, "error: failed to daemonize: %v\n", err)
		return 1
	}
	pid := cmd.Process.Pid
	if err := cmd.Process.Release(); err != nil {
		log.Warn("Could not release detached daemon process handle")
	}
	fmt.Fprintf(os.Stderr, "cryptd: spawned daemon (PID %d); see ~/.punt-labs/cryptd/logs/cryptd.log for readiness\n", pid)
	return 0
}

func daemonPIDPath() (string, error) {
	dir, err := layout.RunDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create runtime directory: %w", err)
	}
	return filepath.Join(dir, "cryptd.pid"), nil
}

func writePIDFile(path string, pid int) error {
	return os.WriteFile(path, []byte(strconv.Itoa(pid)+"\n"), 0o600)
}

func removePIDFile(path string, log *slog.Logger) {
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Warn("Could not read daemon PID file for cleanup")
		}
		return
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err == nil && pid == os.Getpid() {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			log.Warn("Could not remove daemon PID file")
		}
	}
}
