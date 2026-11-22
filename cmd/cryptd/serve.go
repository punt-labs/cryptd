package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
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
	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/narrator"
	"github.com/punt-labs/cryptd/internal/observe"
	"github.com/punt-labs/cryptd/internal/renderer"
	"github.com/punt-labs/cryptd/internal/scenariodir"
)

func runServe(args []string) int {
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

	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 1
		}
		return rejectServe("Invalid serve flags")
	}

	// Resolve API key: flag takes precedence, then env var.
	resolvedKey := *apiKey
	if resolvedKey == "" {
		resolvedKey = os.Getenv("CRYPTD_API_KEY")
	}

	warnAPIKeyFlag := func(log *slog.Logger) {
		if *apiKey != "" {
			log.Warn("API key flag is visible in process listings; use CRYPTD_API_KEY")
		}
	}

	if *testing {
		if *scenarioID == "" {
			fmt.Fprintln(os.Stderr, "error: -t requires --scenario")
			return rejectServe("Headless play requires a scenario")
		}
		if *jsonMode && *scriptFile == "" {
			fmt.Fprintln(os.Stderr, "error: --json requires --script")
			return rejectServe("JSON output requires a script")
		}
		return observe.Run("cli", observe.Headless, func(obs *observe.Runtime) int {
			warnAPIKeyFlag(obs.Log)
			code := runTestingMode(obs.Log, *scenarioID, *charName, *charClass, *scriptFile, *jsonMode, resolvedKey, *modelFlag)
			obs.Log.Info("Finished headless game", "ok", code == 0)
			return code
		})
	}

	socketExplicit := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "socket" {
			socketExplicit = true
		}
	})
	if socketExplicit && *listenAddr != "" {
		fmt.Fprintln(os.Stderr, "error: --socket and --listen are mutually exclusive")
		return rejectServe("Both socket and TCP address were specified")
	}

	if !*foreground && !isDaemonChild() {
		return observe.Run("launcher", observe.SpawnDaemon, func(obs *observe.Runtime) int {
			return daemonize(obs.Log)
		})
	}

	return observe.Run("daemon", observe.ServeDaemon, func(obs *observe.Runtime) int {
		log := observe.ChildLogger(obs.Log)
		warnAPIKeyFlag(log)
		return serveDaemon(log, obs, *socketPath, *listenAddr, *scenarioID, resolvedKey, *modelFlag)
	})
}

func rejectServe(reason string) int {
	return observe.Run("cli", "", func(obs *observe.Runtime) int {
		obs.Log.Warn("Rejected serve arguments", "reason", reason)
		return 1
	})
}

func serveDaemon(log *slog.Logger, obs *observe.Runtime, socketPath, listenAddr, scenarioID, resolvedKey, modelName string) int {
	log.Info("Starting game daemon")
	logger := log.With("component", "cryptd.inference")
	absScenarioDir, err := filepath.Abs(scenariodir.Dir())
	if err != nil {
		log.Error("Could not resolve scenario directory", "error_kind", observe.ErrorKind(err))
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	// Always configure interpreter+narrator when available. Per-session mode
	// selection (passthrough vs normal) happens at session.init time.
	interp, narr := resolveInterpreterNarrator(logger, resolvedKey, modelName)
	opts := []daemon.ServerOption{
		daemon.WithInterpreter(interp), daemon.WithNarrator(narr),
		daemon.WithDefaultScenario(scenarioID), daemon.WithLogger(log),
		daemon.WithGameObserver(func(started time.Time, ok bool) {
			obs.Record(observe.NewGame, started, ok)
		}),
	}

	var srv *daemon.Server
	if listenAddr != "" {
		srv = daemon.NewTCPServer(listenAddr, absScenarioDir, opts...)
	} else {
		if socketPath == "" {
			var err error
			socketPath, err = daemon.DefaultSocketPath()
			if err != nil {
				log.Error("Could not resolve default socket path", "error_kind", observe.ErrorKind(err))
				fmt.Fprintf(os.Stderr, "error: cannot determine home directory: %v\nUse --socket or --listen to specify an address explicitly.\n", err)
				return 1
			}
		}
		srv = daemon.NewServer(socketPath, absScenarioDir, opts...)
	}

	if isDaemonChild() {
		pidPath, err := daemonPIDPath()
		if err != nil {
			log.Warn("Could not prepare daemon PID file directory", "error_kind", observe.ErrorKind(err))
		} else if err := writePIDFile(pidPath, os.Getpid()); err != nil {
			log.Warn("Could not write daemon PID file", "error_kind", observe.ErrorKind(err))
		} else {
			defer removePIDFile(pidPath, log)
		}
	}

	if err := srv.ListenAndServe(); err != nil {
		if errors.Is(err, os.ErrPermission) {
			log.Warn("Game daemon could not access listener", "error_kind", observe.ErrorKind(err))
		} else {
			log.Error("Game daemon stopped after a listener failure", "error_kind", observe.ErrorKind(err))
		}
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	log.Info("Game daemon stopped")
	return 0
}

// runTestingMode runs the engine on stdin/stdout. Scripted mode uses
// Rules+Template (deterministic, no inference). Interactive mode probes
// for available inference (Claude API → SLM → rules+templates).
func runTestingMode(log *slog.Logger, scenarioID, charName, charClass, scriptFile string, jsonMode bool, anthropicKey, modelName string) int {
	s, err := scenariodir.Load(scenariodir.Dir(), scenarioID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}

	var hero model.Character
	var stdinBuf *bufio.Reader // shared buffer for interactive mode

	if scriptFile != "" {
		// Scripted mode: use defaults (no interactive prompt).
		hero, err = engine.NewCharacter(charName, charClass, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error creating character: %v\n", err)
			return 1
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
			return 1
		}
	}

	eng := engine.New(s)
	state, err := eng.NewGame(hero)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error starting game: %v\n", err)
		return 1
	}

	if scriptFile != "" {
		// Scripted mode: deterministic rules+templates, no SLM.
		interp := interpreter.NewRules()
		narr := narrator.NewTemplate(log.With("component", "cryptd.narrator"))
		commands, err := readCommandFile(scriptFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading script: %v\n", err)
			return 1
		}
		ap := renderer.NewAutoplay(os.Stdout, commands, jsonMode)
		loop := game.NewLoop(eng, interp, narr, ap)
		if err := loop.Run(context.Background(), &state); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 1
		}
		if jsonMode {
			if err := ap.WriteJSON(); err != nil {
				fmt.Fprintf(os.Stderr, "error writing JSON: %v\n", err)
				return 1
			}
		}
	} else {
		// Interactive mode: use the shared buffered reader so the
		// renderer picks up where character creation left off.
		interp, narr := resolveInterpreterNarrator(log.With("component", "cryptd.inference"), anthropicKey, modelName)
		rend := renderer.NewCLI(os.Stdout, stdinBuf)
		loop := game.NewLoop(eng, interp, narr, rend)
		if err := loop.Run(context.Background(), &state); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 1
		}
	}
	return 0
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
func resolveInterpreterNarrator(log *slog.Logger, anthropicKey, modelName string) (model.CommandInterpreter, model.Narrator) {
	rules := interpreter.NewRules()
	narrLog := log.With("component", "cryptd.narrator")
	tmpl := narrator.NewTemplate(narrLog)

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
			log.Warn("Claude API check failed; trying local inference", "error_kind", observe.ErrorKind(probeErr))
			// Fall through to SLM probe below.
		} else {
			log.Info("Selected Claude inference")
			return interpreter.NewLLM(client, rules, log.With("component", "cryptd.interpreter")), narrator.NewLLM(client, tmpl, narrLog)
		}
	}

	// Medium/Small tier: local SLM (ollama).
	rt := inference.Probe(context.Background(), inference.DefaultEndpoints(), time.Second)
	if rt != nil {
		client := inference.NewClient(rt.BaseURL, rt.Model, 5*time.Second)
		log.Info("Selected local inference")
		return interpreter.NewSLM(client, rules), narrator.NewSLM(client, tmpl)
	}

	log.Info("Selected rules and templates; no inference server was available")
	return rules, tmpl
}

// promptCharacterCreation runs an interactive character creation flow,
// asking for name, class, and stat allocation. It reads from br using
// ReadString so it consumes only the bytes it needs; the caller can
// continue reading from the same *bufio.Reader afterward.
func promptCharacterCreation(out io.Writer, br *bufio.Reader) (model.Character, error) {
	prompt := func(msg string) (string, error) {
		fmt.Fprint(out, msg)
		line, err := br.ReadString('\n')
		if err != nil && err != io.EOF {
			return "", err
		}
		return strings.TrimSpace(line), nil
	}

	fmt.Fprintln(out, "=== Character Creation ===")
	fmt.Fprintln(out)

	// Name.
	name, err := prompt("Name: ")
	if err != nil {
		return model.Character{}, fmt.Errorf("reading name: %w", err)
	}
	if name == "" {
		name = "Adventurer"
	}

	// Class.
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Classes: fighter, mage, priest, thief")
	classInput, err := prompt("Class: ")
	if err != nil {
		return model.Character{}, fmt.Errorf("reading class: %w", err)
	}
	class := strings.ToLower(classInput)
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

	line, err := prompt("Allocation (6 numbers, or Enter for defaults): ")
	if err != nil {
		return model.Character{}, fmt.Errorf("reading stat allocation: %w", err)
	}

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
