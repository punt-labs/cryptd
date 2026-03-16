package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/punt-labs/cryptd/internal/daemon"
	"github.com/punt-labs/cryptd/internal/engine"
	"github.com/punt-labs/cryptd/internal/game"
	"github.com/punt-labs/cryptd/internal/inference"
	"github.com/punt-labs/cryptd/internal/interpreter"
	"github.com/punt-labs/cryptd/internal/narrator"
	"github.com/punt-labs/cryptd/internal/renderer"
	"github.com/punt-labs/cryptd/internal/scenariodir"
)

func runServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)

	// Network flags.
	socketPath := fs.String("socket", "", "Unix socket path (default ~/.crypt/daemon.sock)")
	listenAddr := fs.String("listen", "", "TCP listen address (e.g. :9000)")
	passthrough := fs.Bool("passthrough", false, "MCP passthrough mode (structured JSON, no interpreter/narrator)")

	// Daemon flags.
	foreground := fs.Bool("f", false, "run in foreground (don't daemonize)")
	testing := fs.Bool("t", false, "testing mode: stdin/stdout, no network (implies -f)")

	// Testing mode flags.
	scenarioID := fs.String("scenario", "", "scenario ID for -t mode")
	charName := fs.String("name", "Adventurer", "character name for -t mode")
	charClass := fs.String("class", "fighter", "character class for -t mode")
	scriptFile := fs.String("script", "", "read commands from file instead of stdin (-t mode)")
	jsonMode := fs.Bool("json", false, "output JSON transcript (-t mode, requires --script)")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if *testing {
		*foreground = true
	}

	// Validate flag combinations.
	if *testing {
		if *scenarioID == "" {
			fmt.Fprintln(os.Stderr, "error: -t requires --scenario")
			os.Exit(1)
		}
		if *jsonMode && *scriptFile == "" {
			fmt.Fprintln(os.Stderr, "error: --json requires --script")
			os.Exit(1)
		}
		runTestingMode(*scenarioID, *charName, *charClass, *scriptFile, *jsonMode)
		return
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

	// Daemonize unless -f or already the daemon child.
	if !*foreground && !isDaemonChild() {
		daemonize()
		// daemonize calls os.Exit — parent never reaches here.
	}

	absScenarioDir, err := filepath.Abs(scenariodir.Dir())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	var opts []daemon.ServerOption

	if *passthrough {
		opts = append(opts, daemon.WithPassthrough())
	} else {
		interp, narr := resolveInterpreterNarrator()
		opts = append(opts, daemon.WithInterpreter(interp), daemon.WithNarrator(narr))
	}

	var srv *daemon.Server
	if *listenAddr != "" {
		srv = daemon.NewTCPServer(*listenAddr, absScenarioDir, opts...)
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
		srv = daemon.NewServer(sock, absScenarioDir, opts...)
	}

	// Clean up PID file on exit (only matters for daemon child).
	defer removePIDFile()

	if err := srv.ListenAndServe(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// runTestingMode runs the engine on stdin/stdout with Rules+Template.
// No network, no SLM — deterministic and fast.
func runTestingMode(scenarioID, charName, charClass, scriptFile string, jsonMode bool) {
	s, err := scenariodir.Load(scenariodir.Dir(), scenarioID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	hero, err := engine.NewCharacter(charName, charClass, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating character: %v\n", err)
		os.Exit(1)
	}

	eng := engine.New(s)
	state, err := eng.NewGame(hero)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error starting game: %v\n", err)
		os.Exit(1)
	}

	interp := interpreter.NewRules()
	narr := narrator.NewTemplate()

	if scriptFile != "" {
		commands, err := readCommandFile(scriptFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading script: %v\n", err)
			os.Exit(1)
		}
		ap := renderer.NewAutoplay(os.Stdout, commands, jsonMode)
		loop := game.NewLoop(eng, interp, narr, ap)
		if err := loop.Run(context.Background(), &state); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if jsonMode {
			if err := ap.WriteJSON(); err != nil {
				fmt.Fprintf(os.Stderr, "error writing JSON: %v\n", err)
				os.Exit(1)
			}
		}
	} else {
		rend := renderer.NewCLI(os.Stdout, os.Stdin)
		loop := game.NewLoop(eng, interp, narr, rend)
		if err := loop.Run(context.Background(), &state); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	}
}

// readCommandFile reads a text file with one command per line.
// Blank lines and lines starting with # are skipped.
func readCommandFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var commands []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		commands = append(commands, line)
	}
	return commands, scanner.Err()
}

// resolveInterpreterNarrator probes for an SLM (ollama) and returns the
// appropriate interpreter and narrator. Falls back to Rules + Template
// if no inference server is available.
func resolveInterpreterNarrator() (*interpreter.SLM, *narrator.SLM) {
	rules := interpreter.NewRules()
	tmpl := narrator.NewTemplate()

	rt := inference.Probe(context.Background(), inference.DefaultEndpoints(), time.Second)
	if rt != nil {
		client := inference.NewClient(rt.BaseURL, rt.Model, 5*time.Second)
		fmt.Fprintf(os.Stderr, "cryptd: using %s (model: %s)\n", rt.BaseURL, rt.Model)
		return interpreter.NewSLM(client, rules), narrator.NewSLM(client, tmpl)
	}

	fmt.Fprintln(os.Stderr, "cryptd: no inference server found — using rules+templates")
	return interpreter.NewSLM(nil, rules), narrator.NewSLM(nil, tmpl)
}
