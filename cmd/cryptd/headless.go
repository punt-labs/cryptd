package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/punt-labs/cryptd/internal/engine"
	"github.com/punt-labs/cryptd/internal/game"
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

func runHeadless(args []string) {
	fs := flag.NewFlagSet("headless", flag.ExitOnError)
	scenarioID := fs.String("scenario", "", "scenario ID (filename without .yaml)")
	_ = fs.Parse(args)

	if *scenarioID == "" {
		fmt.Fprintln(os.Stderr, "usage: cryptd headless --scenario <id>")
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
		fmt.Fprintln(os.Stderr, "usage: cryptd autoplay --scenario <id> --script <file> [--json]")
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
