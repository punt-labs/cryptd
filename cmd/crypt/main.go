package main

import (
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
		fmt.Fprintln(os.Stderr, "commands: headless, validate")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "headless":
		runHeadless(os.Args[2:])
	case "validate":
		runValidate(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
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
	if strings.ContainsAny(*scenarioID, `/\`) || strings.Contains(*scenarioID, "..") || filepath.VolumeName(*scenarioID) != "" {
		fmt.Fprintln(os.Stderr, "error: invalid scenario ID")
		os.Exit(1)
	}

	scenarioDir := os.Getenv("CRYPT_SCENARIO_DIR")
	if scenarioDir == "" {
		scenarioDir = "scenarios"
	}
	absScenarioDir, err := filepath.Abs(scenarioDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error resolving scenario directory: %v\n", err)
		os.Exit(1)
	}
	absPath, err := filepath.Abs(filepath.Join(scenarioDir, *scenarioID+".yaml"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error resolving scenario path: %v\n", err)
		os.Exit(1)
	}
	if absPath != absScenarioDir && !strings.HasPrefix(absPath, absScenarioDir+string(os.PathSeparator)) {
		fmt.Fprintln(os.Stderr, "error: invalid scenario ID")
		os.Exit(1)
	}
	path := absPath
	s, err := scenario.Load(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading scenario: %v\n", err)
		os.Exit(1)
	}

	char := model.Character{
		ID: "hero", Name: "Adventurer", Class: "fighter",
		Level: 1, HP: 10, MaxHP: 10,
	}

	eng := engine.New(s)
	state, err := eng.NewGame(char)
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

