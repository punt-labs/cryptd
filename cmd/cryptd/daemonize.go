package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
)

const daemonEnv = "_CRYPTD_DAEMON"

// isDaemonChild returns true if this process was spawned by daemonize.
func isDaemonChild() bool {
	return os.Getenv(daemonEnv) == "1"
}

// daemonize re-executes the current process in the background, detached from
// the terminal. The parent prints the child PID and exits. The child continues
// with isDaemonChild() == true.
//
// This follows the sshd/nginx pattern: the default is to daemonize; -f keeps
// the process in the foreground.
func daemonize() {
	logPath, pidPath, err := daemonPaths()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot open log %s: %v\n", logPath, err)
		os.Exit(1)
	}

	cmd := exec.Command(os.Args[0], os.Args[1:]...)
	cmd.Env = append(os.Environ(), daemonEnv+"=1")
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		logFile.Close()
		fmt.Fprintf(os.Stderr, "error: failed to daemonize: %v\n", err)
		os.Exit(1)
	}
	logFile.Close()

	if err := writePIDFile(pidPath, cmd.Process.Pid); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not write PID file: %v\n", err)
	}

	fmt.Fprintf(os.Stderr, "cryptd: daemon started (PID %d, log %s)\n", cmd.Process.Pid, logPath)
	os.Exit(0)
}

// daemonPaths returns the log and PID file paths under ~/.crypt/.
func daemonPaths() (logPath, pidPath string, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	dir := filepath.Join(home, ".crypt")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", "", fmt.Errorf("create ~/.crypt: %w", err)
	}
	return filepath.Join(dir, "cryptd.log"), filepath.Join(dir, "cryptd.pid"), nil
}

// writePIDFile writes the given PID to the file.
func writePIDFile(path string, pid int) error {
	return os.WriteFile(path, []byte(strconv.Itoa(pid)+"\n"), 0o644)
}

// removePIDFile removes the PID file if it contains our PID.
func removePIDFile() {
	_, pidPath, err := daemonPaths()
	if err != nil {
		return
	}
	data, err := os.ReadFile(pidPath)
	if err != nil {
		return
	}
	pid, err := strconv.Atoi(string(data[:len(data)-1])) // trim newline
	if err != nil {
		return
	}
	if pid == os.Getpid() {
		os.Remove(pidPath)
	}
}
