package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/punt-labs/cryptd/internal/monkeytest"
	"github.com/punt-labs/cryptd/internal/scenariodir"
)

func main() {
	scenarioID := flag.String("scenario", "minimal", "scenario ID")
	players := flag.Int("players", 1000, "number of sessions to run")
	maxMoves := flag.Int("max-moves", 200, "max moves per session before forced quit")
	class := flag.String("class", "fighter", "character class (fighter|mage|thief|priest|all)")
	workers := flag.Int("workers", 8, "parallel workers")
	seed := flag.Int64("seed", 42, "master RNG seed")
	verbose := flag.Bool("verbose", false, "include per-session data in output")
	flag.Parse()

	cfg := monkeytest.RunConfig{
		Scenario: *scenarioID,
		Class:    *class,
		Players:  *players,
		MaxMoves: *maxMoves,
		Workers:  *workers,
		Seed:     *seed,
	}

	fmt.Fprintf(os.Stderr, "eval-balance: %d %s sessions on %q (max %d moves, %d workers, seed %d)\n",
		*players, *class, *scenarioID, *maxMoves, *workers, *seed)

	report, err := monkeytest.Run(cfg, scenariodir.Dir(), *scenarioID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if !*verbose {
		report.Sessions = nil
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		fmt.Fprintf(os.Stderr, "error encoding report: %v\n", err)
		os.Exit(1)
	}

	// Print summary to stderr.
	a := report.Aggregate
	fmt.Fprintf(os.Stderr, "\nsurvival: %.0f%% | mean moves: %.1f | mean XP: %.1f | mean kills: %.1f | mean level: %.1f\n",
		a.SurvivalRate*100, a.MeanMoves, a.MeanXP, a.MeanEnemies, a.MeanLevel)
}
