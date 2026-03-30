package main

import (
	"flag"
	"fmt"
	"os"

	"golang.org/x/term"
)

func main() {
	// Check for subcommands before flag parsing.
	if len(os.Args) > 1 && os.Args[1] == "mcp" {
		mcpFS := flag.NewFlagSet("mcp", flag.ContinueOnError)
		mcpSocket := mcpFS.String("socket", "", "Unix socket path (default ~/.crypt/daemon.sock)")
		mcpAddr := mcpFS.String("addr", "", "TCP address (e.g. host:9000)")
		if err := mcpFS.Parse(os.Args[2:]); err != nil {
			os.Exit(1)
		}
		if *mcpSocket != "" && *mcpAddr != "" {
			fmt.Fprintln(os.Stderr, "error: --socket and --addr are mutually exclusive")
			os.Exit(1)
		}
		os.Exit(runMCP(*mcpSocket, *mcpAddr))
	}

	fs := flag.NewFlagSet("crypt", flag.ContinueOnError)
	socketPath := fs.String("socket", "", "Unix socket path (default ~/.crypt/daemon.sock)")
	addr := fs.String("addr", "", "TCP address (e.g. host:9000)")
	scenario := fs.String("scenario", "", "auto-start game with this scenario ID")
	name := fs.String("name", "Adventurer", "character name (used with --scenario)")
	class := fs.String("class", "fighter", "character class: fighter, mage, priest, thief (used with --scenario)")
	sessionID := fs.String("session", "", "resume a previous session by ID")
	plain := fs.Bool("plain", false, "use plain readline interface instead of TUI")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(1)
	}

	if *socketPath != "" && *addr != "" {
		fmt.Fprintln(os.Stderr, "error: --socket and --addr are mutually exclusive")
		os.Exit(1)
	}

	useTUI := !*plain && term.IsTerminal(int(os.Stdin.Fd()))
	if useTUI {
		os.Exit(runTUI(*socketPath, *addr, *scenario, *name, *class, *sessionID))
	} else {
		os.Exit(run(*socketPath, *addr, *scenario, *name, *class, *sessionID))
	}
}
