package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	fs := flag.NewFlagSet("crypt", flag.ContinueOnError)
	socketPath := fs.String("socket", "", "Unix socket path (default ~/.crypt/daemon.sock)")
	addr := fs.String("addr", "", "TCP address (e.g. host:9000)")
	scenario := fs.String("scenario", "", "auto-start game with this scenario ID")
	name := fs.String("name", "Adventurer", "character name (used with --scenario)")
	class := fs.String("class", "fighter", "character class: fighter, mage, priest, thief (used with --scenario)")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(1)
	}

	if *socketPath != "" && *addr != "" {
		fmt.Fprintln(os.Stderr, "error: --socket and --addr are mutually exclusive")
		os.Exit(1)
	}

	os.Exit(run(*socketPath, *addr, *scenario, *name, *class))
}
