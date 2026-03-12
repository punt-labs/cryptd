package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/punt-labs/cryptd/internal/daemon"
)

func runServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	socketPath := fs.String("socket", "", "Unix socket path (default ~/.crypt/daemon.sock)")
	listenAddr := fs.String("listen", "", "TCP listen address (e.g. :9000)")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	socketExplicit := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "socket" {
			socketExplicit = true
		}
	})
	if socketExplicit && *listenAddr != "" {
		fmt.Fprintln(os.Stderr, "error: --socket and --listen are mutually exclusive")
		os.Exit(1)
	}

	scenarioDir := os.Getenv("CRYPT_SCENARIO_DIR")
	if scenarioDir == "" {
		scenarioDir = "scenarios"
	}
	absScenarioDir, err := filepath.Abs(scenarioDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	var srv *daemon.Server
	if *listenAddr != "" {
		srv = daemon.NewTCPServer(*listenAddr, absScenarioDir)
	} else {
		sock := *socketPath
		if sock == "" {
			sock = defaultSocketPath()
		}
		srv = daemon.NewServer(sock, absScenarioDir)
	}
	if err := srv.ListenAndServe(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// defaultSocketPath returns ~/.crypt/daemon.sock.
// Only called when --listen is not set, so $HOME must be available.
func defaultSocketPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot determine home directory: %v\nUse --socket or --listen to specify an address explicitly.\n", err)
		os.Exit(1)
	}
	return filepath.Join(home, ".crypt", "daemon.sock")
}
