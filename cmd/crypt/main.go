package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/punt-labs/cryptd/internal/observe"
	"golang.org/x/term"
)

func main() {
	// Check for subcommands before flag parsing.
	if len(os.Args) > 1 && os.Args[1] == "mcp" {
		mcpFS := flag.NewFlagSet("mcp", flag.ContinueOnError)
		mcpSocket := mcpFS.String("socket", "", "Unix socket path (default ~/.crypt/daemon.sock)")
		mcpAddr := mcpFS.String("addr", "", "TCP address (e.g. host:9000)")
		if err := mcpFS.Parse(os.Args[2:]); err != nil {
			if err == flag.ErrHelp {
				os.Exit(1)
			}
			os.Exit(rejectArgs("mcp"))
		}
		if *mcpSocket != "" && *mcpAddr != "" {
			fmt.Fprintln(os.Stderr, "error: --socket and --addr are mutually exclusive")
			os.Exit(rejectArgs("mcp"))
		}
		os.Exit(observe.Run("mcp", observe.ServeMCP, func(obs *observe.Runtime) int {
			return runMCP(obs.Log, *mcpSocket, *mcpAddr)
		}))
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
		if err == flag.ErrHelp {
			os.Exit(1)
		}
		os.Exit(rejectArgs("cli"))
	}

	if *socketPath != "" && *addr != "" {
		fmt.Fprintln(os.Stderr, "error: --socket and --addr are mutually exclusive")
		os.Exit(rejectArgs("cli"))
	}

	useTUI := !*plain && term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
	os.Exit(observe.Run("cli", observe.Play, func(obs *observe.Runtime) int {
		if useTUI {
			return runTUI(obs.Log, *socketPath, *addr, *scenario, *name, *class, *sessionID)
		}
		return run(obs.Log, *socketPath, *addr, *scenario, *name, *class, *sessionID)
	}))
}

func rejectArgs(role string) int {
	return observe.Run(role, "", func(obs *observe.Runtime) int {
		obs.Log.Warn("Rejected client arguments")
		return 1
	})
}
