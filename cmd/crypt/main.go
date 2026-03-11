package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/punt-labs/cryptd/internal/engine"
	"github.com/punt-labs/cryptd/internal/game"
	"github.com/punt-labs/cryptd/internal/interpreter"
	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/narrator"
	"github.com/punt-labs/cryptd/internal/renderer"
	"github.com/punt-labs/cryptd/internal/scenario"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: cryptd <command> [args]")
		fmt.Fprintln(os.Stderr, "commands: headless, autoplay, validate")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "headless":
		runHeadless(os.Args[2:])
	case "autoplay":
		runAutoplay(os.Args[2:])
	case "validate":
		runValidate(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

// loadScenario resolves a scenario ID to a file path and loads it.
func loadScenario(id string) (*scenario.Scenario, error) {
	if strings.ContainsAny(id, `/\`) || strings.Contains(id, "..") || filepath.VolumeName(id) != "" {
		return nil, fmt.Errorf("invalid scenario ID")
	}

	scenarioDir := os.Getenv("CRYPT_SCENARIO_DIR")
	if scenarioDir == "" {
		scenarioDir = "scenarios"
	}
	absScenarioDir, err := filepath.Abs(scenarioDir)
	if err != nil {
		return nil, fmt.Errorf("resolving scenario directory: %w", err)
	}
	absPath, err := filepath.Abs(filepath.Join(scenarioDir, id+".yaml"))
	if err != nil {
		return nil, fmt.Errorf("resolving scenario path: %w", err)
	}
	rel, err := filepath.Rel(absScenarioDir, absPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
		return nil, fmt.Errorf("invalid scenario ID")
	}
	return scenario.Load(absPath)
}

// defaultHero returns the default single-player character.
func defaultHero() model.Character {
	return model.Character{
		ID: "hero", Name: "Adventurer", Class: "fighter",
		Level: 1, HP: 20, MaxHP: 20,
		Stats: model.Stats{STR: 14, DEX: 12, CON: 12, INT: 10, WIS: 10, CHA: 10},
	}
}

func runHeadless(args []string) {
	fs := flag.NewFlagSet("headless", flag.ExitOnError)
	scenarioID := fs.String("scenario", "", "scenario ID (filename without .yaml)")
	_ = fs.Parse(args)

	if *scenarioID == "" {
		fmt.Fprintln(os.Stderr, "usage: cryptd headless --scenario <id>")
		os.Exit(1)
	}

	s, err := loadScenario(*scenarioID)
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
		fmt.Fprintln(os.Stderr, "usage: cryptd autoplay --scenario <id> --script <file> [--json]")
		os.Exit(1)
	}

	s, err := loadScenario(*scenarioID)
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

func runValidate(args []string) {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: cryptd validate <scenario-file>")
		os.Exit(1)
	}
	path := args[0]
	if _, err := scenario.Load(path); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("OK")
}
