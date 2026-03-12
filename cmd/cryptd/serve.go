package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/punt-labs/cryptd/internal/daemon"
	"github.com/punt-labs/cryptd/internal/scenariodir"
)

func runServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	socketPath := fs.String("socket", "", "Unix socket path (default ~/.crypt/daemon.sock)")
	listenAddr := fs.String("listen", "", "TCP listen address (e.g. :9000)")
	if err := fs.Parse(args); err != nil {
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

	absScenarioDir, err := filepath.Abs(scenariodir.Dir())
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
			var err error
			sock, err = daemon.DefaultSocketPath()
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: cannot determine home directory: %v\nUse --socket or --listen to specify an address explicitly.\n", err)
				os.Exit(1)
			}
		}
		srv = daemon.NewServer(sock, absScenarioDir)
	}
	if err := srv.ListenAndServe(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

