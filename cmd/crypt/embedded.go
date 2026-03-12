package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/punt-labs/cryptd/internal/engine"
	"github.com/punt-labs/cryptd/internal/game"
	"github.com/punt-labs/cryptd/internal/inference"
	"github.com/punt-labs/cryptd/internal/interpreter"
	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/narrator"
	"github.com/punt-labs/cryptd/internal/renderer"
	"github.com/punt-labs/cryptd/internal/scenariodir"
)

// defaultHero returns the default single-player character.
func defaultHero() model.Character {
	return model.Character{
		ID: "hero", Name: "Adventurer", Class: "fighter",
		Level: 1, HP: 20, MaxHP: 20,
		Stats: model.Stats{STR: 14, DEX: 12, CON: 12, INT: 10, WIS: 10, CHA: 10},
	}
}

func runSolo(args []string) {
	fs := flag.NewFlagSet("solo", flag.ExitOnError)
	scenarioID := fs.String("scenario", "", "scenario ID (filename without .yaml)")
	modelFlag := fs.String("model", "", "model name (auto-detected if omitted)")
	serverURL := fs.String("server", "", "inference server URL (auto-detected if omitted)")
	timeoutFlag := fs.Duration("timeout", 5*time.Second, "inference request timeout")
	luxMode := fs.Bool("lux", false, "use Lux JSON-lines display protocol on stdout/stdin")
	_ = fs.Parse(args)

	if *scenarioID == "" {
		fmt.Fprintln(os.Stderr, "usage: crypt solo --scenario <id> [--lux] [--model <name>] [--server <url>] [--timeout <duration>]")
		os.Exit(1)
	}

	s, err := scenariodir.Load(scenariodir.Dir(), *scenarioID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	eng := engine.New(s)
	state, err := eng.NewGame(defaultHero())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error starting game: %v\n", err)
		os.Exit(1)
	}
	state.PlayMode = "solo"

	var interp model.CommandInterpreter
	var narr model.Narrator
	rules := interpreter.NewRules()
	tmpl := narrator.NewTemplate()

	baseURL, modelName := resolveRuntime(*serverURL, *modelFlag)
	if baseURL != "" && modelName != "" {
		client := inference.NewClient(baseURL, modelName, *timeoutFlag)
		interp = interpreter.NewSLM(client, rules)
		narr = narrator.NewSLM(client, tmpl)
		fmt.Fprintf(os.Stderr, "Using %s (model: %s)\n", baseURL, modelName)
	} else {
		interp = rules
		narr = tmpl
		fmt.Fprintln(os.Stderr, "No inference server found — using rules+templates")
	}

	var rend model.Renderer
	if *luxMode {
		transport := renderer.NewJSONTransport(os.Stdout, os.Stdin)
		rend = renderer.NewLux(transport)
	} else {
		rend = renderer.NewCLI(os.Stdout, os.Stdin)
	}

	loop := game.NewLoop(eng, interp, narr, rend)
	if err := loop.Run(context.Background(), &state); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// resolveRuntime determines the inference server URL and model name from
// explicit flags or auto-detection. Returns empty strings if no runtime found.
func resolveRuntime(serverURL, modelName string) (string, string) {
	if serverURL != "" && modelName != "" {
		return serverURL, modelName
	}

	var endpoints []inference.Endpoint
	if serverURL != "" {
		endpoints = []inference.Endpoint{
			{Name: "user-server", BaseURL: serverURL, HealthPath: "/api/tags", ModelLister: inference.OllamaModels, Preferred: inference.PreferredModels},
			{Name: "user-server", BaseURL: serverURL, HealthPath: "/v1/models", ModelLister: inference.OpenAIModels, Preferred: inference.PreferredModels},
		}
	} else {
		endpoints = inference.DefaultEndpoints()
	}

	rt := inference.Probe(context.Background(), endpoints, time.Second)
	if rt == nil {
		if serverURL != "" {
			fmt.Fprintf(os.Stderr, "warning: --server %q is not responding — no model could be determined\n", serverURL)
		}
		if modelName != "" {
			fmt.Fprintf(os.Stderr, "warning: ignoring --model %q — no inference server found\n", modelName)
		}
		return "", ""
	}

	if serverURL == "" {
		serverURL = rt.BaseURL
	}
	if modelName == "" {
		modelName = rt.Model
	}
	return serverURL, modelName
}

func runHeadless(args []string) {
	fs := flag.NewFlagSet("headless", flag.ExitOnError)
	scenarioID := fs.String("scenario", "", "scenario ID (filename without .yaml)")
	_ = fs.Parse(args)

	if *scenarioID == "" {
		fmt.Fprintln(os.Stderr, "usage: crypt headless --scenario <id>")
		os.Exit(1)
	}

	s, err := scenariodir.Load(scenariodir.Dir(), *scenarioID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	eng := engine.New(s)
	state, err := eng.NewGame(defaultHero())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error starting game: %v\n", err)
		os.Exit(1)
	}
	state.PlayMode = "headless"

	loop := game.NewLoop(eng, interpreter.NewRules(), narrator.NewTemplate(), renderer.NewCLI(os.Stdout, os.Stdin))
	if err := loop.Run(context.Background(), &state); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func runAutoplay(args []string) {
	fs := flag.NewFlagSet("autoplay", flag.ExitOnError)
	scenarioID := fs.String("scenario", "", "scenario ID (filename without .yaml)")
	scriptFile := fs.String("script", "", "file with one command per line")
	jsonMode := fs.Bool("json", false, "output transcript as JSON")
	_ = fs.Parse(args)

	if *scenarioID == "" || *scriptFile == "" {
		fmt.Fprintln(os.Stderr, "usage: crypt autoplay --scenario <id> --script <file> [--json]")
		os.Exit(1)
	}

	s, err := scenariodir.Load(scenariodir.Dir(), *scenarioID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	commands, err := readCommandFile(*scriptFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading script: %v\n", err)
		os.Exit(1)
	}

	eng := engine.New(s)
	state, err := eng.NewGame(defaultHero())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error starting game: %v\n", err)
		os.Exit(1)
	}
	state.PlayMode = "autoplay"

	ap := renderer.NewAutoplay(os.Stdout, commands, *jsonMode)
	loop := game.NewLoop(eng, interpreter.NewRules(), narrator.NewTemplate(), ap)
	if err := loop.Run(context.Background(), &state); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if *jsonMode {
		if err := ap.WriteJSON(); err != nil {
			fmt.Fprintf(os.Stderr, "error writing JSON: %v\n", err)
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
