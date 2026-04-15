package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/punt-labs/cryptd/internal/daemon"
	"github.com/punt-labs/cryptd/internal/engine"
	"github.com/punt-labs/cryptd/internal/game"
	"github.com/punt-labs/cryptd/internal/inference"
	"github.com/punt-labs/cryptd/internal/interpreter"
	"github.com/punt-labs/cryptd/internal/lux"
	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/narrator"
	"github.com/punt-labs/cryptd/internal/renderer"
	"github.com/punt-labs/cryptd/internal/scenariodir"
)

func runServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)

	// Network flags.
	socketPath := fs.String("socket", "", "Unix socket path (default ~/.crypt/daemon.sock)")
	listenAddr := fs.String("listen", "", "TCP listen address (e.g. :9000)")

	// Daemon flags.
	foreground := fs.Bool("f", false, "run in foreground (don't daemonize)")
	testing := fs.Bool("t", false, "testing mode: stdin/stdout, no network (implies -f)")

	// Inference flags.
	apiKey := fs.String("api-key", "", "Anthropic API key for Claude LLM tier (or set CRYPTD_API_KEY)")
	modelFlag := fs.String("model", "claude-sonnet-4-20250514", "model name for Claude LLM tier")

	// Scenario flag — used for both -t mode and daemon default scenario.
	scenarioID := fs.String("scenario", "", "scenario ID (-t mode: required; daemon: default for new_game)")
	charName := fs.String("name", "Adventurer", "character name for -t mode")
	charClass := fs.String("class", "fighter", "character class for -t mode")
	scriptFile := fs.String("script", "", "read commands from file instead of stdin (-t mode)")
	jsonMode := fs.Bool("json", false, "output JSON transcript (-t mode, requires --script)")
	luxMode := fs.Bool("lux", false, "render to Lux ImGui display instead of terminal (-t mode)")

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if *testing {
		*foreground = true
	}

	// Resolve API key: flag takes precedence, then env var.
	resolvedKey := *apiKey
	if resolvedKey == "" {
		resolvedKey = os.Getenv("CRYPTD_API_KEY")
	}

	if *apiKey != "" {
		log.Println("cryptd: warning: --api-key is visible in process listings; prefer CRYPTD_API_KEY env var")
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
		runTestingMode(*scenarioID, *charName, *charClass, *scriptFile, *jsonMode, *luxMode, resolvedKey, *modelFlag)
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

	// Always configure interpreter+narrator when available. Per-session mode
	// selection (passthrough vs normal) happens at session.init time.
	interp, narr := resolveInterpreterNarrator(resolvedKey, *modelFlag)
	opts = append(opts, daemon.WithInterpreter(interp), daemon.WithNarrator(narr))

	if *scenarioID != "" {
		opts = append(opts, daemon.WithDefaultScenario(*scenarioID))
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

// runTestingMode runs the engine on stdin/stdout. Scripted mode uses
// Rules+Template (deterministic, no inference). Interactive and Lux modes
// probe for available inference (Claude API → SLM → rules+templates).
func runTestingMode(scenarioID, charName, charClass, scriptFile string, jsonMode, luxMode bool, anthropicKey, modelName string) {
	s, err := scenariodir.Load(scenariodir.Dir(), scenarioID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	var hero model.Character
	var stdinBuf *bufio.Reader // shared buffer for interactive mode

	if scriptFile != "" || luxMode {
		// Scripted or Lux mode: use defaults (no interactive prompt).
		hero, err = engine.NewCharacter(charName, charClass, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error creating character: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Interactive mode: wrap stdin in a shared bufio.Reader so
		// character creation and the CLI renderer read from the same
		// buffer. Without this, promptCharacterCreation's internal
		// bufio.Scanner consumes stdin ahead of what it needs, and the
		// renderer's scanner sees EOF.
		stdinBuf = bufio.NewReader(os.Stdin)
		hero, err = promptCharacterCreation(os.Stdout, stdinBuf)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	}

	eng := engine.New(s)
	state, err := eng.NewGame(hero)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error starting game: %v\n", err)
		os.Exit(1)
	}

	if scriptFile != "" {
		// Scripted mode: deterministic rules+templates, no SLM.
		interp := interpreter.NewRules()
		narr := narrator.NewTemplate()
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
	} else if luxMode {
		// Lux mode: render to Lux ImGui display, input via button clicks.
		interp, narr := resolveInterpreterNarrator(anthropicKey, modelName)
		rend, cleanup, err := createLuxRenderer()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			fmt.Fprintln(os.Stderr, "hint: start Lux first, or omit --lux for terminal mode")
			os.Exit(1)
		}
		defer cleanup()
		loop := game.NewLoop(eng, interp, narr, rend)
		if err := loop.Run(context.Background(), &state); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Interactive mode: use the shared buffered reader so the
		// renderer picks up where character creation left off.
		interp, narr := resolveInterpreterNarrator(anthropicKey, modelName)
		rend := renderer.NewCLI(os.Stdout, stdinBuf)
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

// resolveInterpreterNarrator selects the best available inference tier:
// Large (Claude API) > Medium/Small (local SLM) > Tiny (rules+templates).
func resolveInterpreterNarrator(anthropicKey, modelName string) (model.CommandInterpreter, model.Narrator) {
	rules := interpreter.NewRules()
	tmpl := narrator.NewTemplate()

	// Large tier: Claude API (if API key provided).
	// Anthropic exposes an OpenAI-compatible /v1/chat/completions endpoint
	// that accepts Authorization: Bearer <key> — same wire format as ollama.
	if anthropicKey != "" {
		client := inference.NewClientWithOpts(
			"https://api.anthropic.com",
			modelName,
			inference.WithAPIKey(anthropicKey),
			inference.WithTimeout(15*time.Second),
		)

		// Validate API key before committing to LLM tier.
		probeCtx, probeCancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, probeErr := client.ChatCompletion(probeCtx, []inference.Message{
			{Role: inference.RoleUser, Content: "ping"},
		}, &inference.Options{MaxTokens: 1})
		probeCancel()
		if probeErr != nil {
			log.Printf("cryptd: Claude API key validation failed: %v", probeErr)
			log.Println("cryptd: falling back to local inference")
			// Fall through to SLM probe below.
		} else {
			log.Printf("cryptd: using Claude API (model: %s)", modelName)
			return interpreter.NewLLM(client, rules), narrator.NewLLM(client, tmpl)
		}
	}

	// Medium/Small tier: local SLM (ollama).
	rt := inference.Probe(context.Background(), inference.DefaultEndpoints(), time.Second)
	if rt != nil {
		client := inference.NewClient(rt.BaseURL, rt.Model, 5*time.Second)
		log.Printf("cryptd: using %s (model: %s)", rt.BaseURL, rt.Model)
		return interpreter.NewSLM(client, rules), narrator.NewSLM(client, tmpl)
	}

	log.Println("cryptd: no inference server found — using rules+templates")
	return rules, tmpl
}

// promptCharacterCreation runs an interactive character creation flow,
// asking for name, class, and stat allocation. It reads from br using
// ReadString so it consumes only the bytes it needs; the caller can
// continue reading from the same *bufio.Reader afterward.
func promptCharacterCreation(out io.Writer, br *bufio.Reader) (model.Character, error) {
	prompt := func(msg string) string {
		fmt.Fprint(out, msg)
		line, err := br.ReadString('\n')
		if err != nil && line == "" {
			return ""
		}
		return strings.TrimSpace(line)
	}

	fmt.Fprintln(out, "=== Character Creation ===")
	fmt.Fprintln(out)

	// Name.
	name := prompt("Name: ")
	if name == "" {
		name = "Adventurer"
	}

	// Class.
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Classes: fighter, mage, priest, thief")
	class := strings.ToLower(prompt("Class: "))
	if !engine.ValidClasses[class] {
		return model.Character{}, fmt.Errorf("unknown class %q", class)
	}

	// Stat allocation.
	fmt.Fprintln(out)
	fmt.Fprintf(out, "Distribute %d points across 6 stats (base %d each).\n",
		engine.PointBuyPool, engine.BaseStatValue)
	fmt.Fprintln(out, "Enter points to ADD to each stat (e.g. 4 0 2 0 2 0).")
	fmt.Fprintln(out, "Press Enter for defaults (STR +4, DEX +2, CON +2).")
	fmt.Fprintln(out)

	statNames := []string{"STR", "DEX", "CON", "INT", "WIS", "CHA"}
	defaults := engine.DefaultStats()
	defaultBonuses := []int{
		defaults.STR - engine.BaseStatValue,
		defaults.DEX - engine.BaseStatValue,
		defaults.CON - engine.BaseStatValue,
		defaults.INT - engine.BaseStatValue,
		defaults.WIS - engine.BaseStatValue,
		defaults.CHA - engine.BaseStatValue,
	}

	// Show current defaults.
	for i, n := range statNames {
		fmt.Fprintf(out, "  %s: %d (+%d)\n", n, engine.BaseStatValue+defaultBonuses[i], defaultBonuses[i])
	}
	fmt.Fprintln(out)

	line := prompt("Allocation (6 numbers, or Enter for defaults): ")

	var stats model.Stats
	if line == "" {
		stats = defaults
	} else {
		parts := strings.Fields(line)
		if len(parts) != 6 {
			return model.Character{}, fmt.Errorf("expected 6 numbers, got %d", len(parts))
		}
		bonuses := make([]int, 6)
		for i, p := range parts {
			v, err := strconv.Atoi(p)
			if err != nil {
				return model.Character{}, fmt.Errorf("invalid number %q for %s", p, statNames[i])
			}
			if v < 0 {
				return model.Character{}, fmt.Errorf("%s bonus cannot be negative", statNames[i])
			}
			bonuses[i] = v
		}
		stats = model.Stats{
			STR: engine.BaseStatValue + bonuses[0],
			DEX: engine.BaseStatValue + bonuses[1],
			CON: engine.BaseStatValue + bonuses[2],
			INT: engine.BaseStatValue + bonuses[3],
			WIS: engine.BaseStatValue + bonuses[4],
			CHA: engine.BaseStatValue + bonuses[5],
		}
	}

	hero, err := engine.NewCharacter(name, class, &stats)
	if err != nil {
		return model.Character{}, err
	}

	fmt.Fprintln(out)
	fmt.Fprintf(out, "Created %s the %s (STR %d, DEX %d, CON %d, INT %d, WIS %d, CHA %d)\n",
		hero.Name, hero.Class,
		hero.Stats.STR, hero.Stats.DEX, hero.Stats.CON,
		hero.Stats.INT, hero.Stats.WIS, hero.Stats.CHA)
	fmt.Fprintln(out)

	return hero, nil
}

// createLuxRenderer connects to a running Lux display and returns a
// renderer that draws into a "cryptd" frame. The returned cleanup
// function closes the display and client.
func createLuxRenderer() (model.Renderer, func(), error) {
	client := lux.NewClient(
		lux.WithName("cryptd"),
		lux.WithConnectTimeout(5*time.Second),
		lux.WithRecvTimeout(30*time.Second),
	)
	if err := client.Connect(); err != nil {
		_ = client.Close() // release resources allocated by NewClient
		return nil, nil, fmt.Errorf("cannot connect to Lux display: %w", err)
	}
	fmt.Fprintln(os.Stderr, "cryptd: connected to Lux display")

	display := lux.NewDisplay(client, "cryptd-game", &lux.ShowOpts{
		FrameID:    "cryptd",
		FrameTitle: "Crypt",
		FrameSize:  [2]int{450, 650},
	})

	rend := renderer.NewLux(display)
	cleanup := func() {
		if err := display.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: lux display close: %v\n", err)
		}
		if err := client.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: lux client close: %v\n", err)
		}
	}
	return rend, cleanup, nil
}
