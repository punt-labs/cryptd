package monkeytest

import (
	"context"
	"fmt"
	"math/rand"
	"sync"

	"github.com/punt-labs/cryptd/internal/engine"
	"github.com/punt-labs/cryptd/internal/game"
	"github.com/punt-labs/cryptd/internal/interpreter"
	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/narrator"
	"github.com/punt-labs/cryptd/internal/scenario"
)

// AllClasses is the list of playable character classes.
var AllClasses = []string{"fighter", "mage", "priest", "thief"}

// Run executes a batch of monkey test sessions and returns the aggregate report.
// scenarioDir is the path to the directory containing scenario files.
func Run(cfg RunConfig, scenarioDir string, scenarioID string) (AggregateReport, error) {
	s, err := loadScenario(scenarioDir, scenarioID)
	if err != nil {
		return AggregateReport{}, fmt.Errorf("load scenario: %w", err)
	}

	// Generate per-session seeds deterministically from the master seed.
	baseRng := rand.New(rand.NewSource(cfg.Seed))
	type sessionSpec struct {
		index int
		seed  int64
		class string
	}

	specs := make([]sessionSpec, cfg.Players)
	for i := range specs {
		specs[i].index = i
		specs[i].seed = baseRng.Int63()
		if cfg.Class == "all" {
			specs[i].class = AllClasses[i%len(AllClasses)]
		} else {
			specs[i].class = cfg.Class
		}
	}

	// Run sessions in parallel.
	results := make([]SessionMetrics, cfg.Players)
	var runErr error
	var errOnce sync.Once

	workers := cfg.Workers
	if workers < 1 {
		workers = 1
	}

	work := make(chan sessionSpec, len(specs))
	for _, sp := range specs {
		work <- sp
	}
	close(work)

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for sp := range work {
				m, err := runSession(s, sp.class, sp.seed, cfg.MaxMoves)
				if err != nil {
					errOnce.Do(func() { runErr = fmt.Errorf("session %d: %w", sp.index, err) })
					return
				}
				results[sp.index] = m
			}
		}()
	}
	wg.Wait()

	if runErr != nil {
		return AggregateReport{}, runErr
	}

	// Compute aggregates.
	report := AggregateReport{
		Config:    cfg,
		Aggregate: Compute(results),
		Sessions:  results,
	}

	// Per-class breakdown.
	if cfg.Class == "all" {
		report.PerClass = make(map[string]AggregateMetrics, len(AllClasses))
		for _, cls := range AllClasses {
			var classSessions []SessionMetrics
			for _, s := range results {
				if s.Class == cls {
					classSessions = append(classSessions, s)
				}
			}
			if len(classSessions) > 0 {
				report.PerClass[cls] = Compute(classSessions)
			}
		}
	}

	return report, nil
}

// runSession executes a single game session and returns its metrics.
func runSession(s *scenario.Scenario, class string, seed int64, maxMoves int) (SessionMetrics, error) {
	// The engine uses math/rand global functions for combat dice. With parallel
	// workers, combat outcomes are non-deterministic. The monkey's own action
	// selection uses a per-session *rand.Rand for reproducible strategy.
	// Use --workers 1 for fully deterministic sessions.

	hero := model.Character{
		ID: "hero", Name: "Monkey", Class: class,
		Level: 1, HP: 20, MaxHP: 20,
		Stats: model.Stats{STR: 14, DEX: 12, CON: 12, INT: 10, WIS: 10, CHA: 10},
	}
	if class == "mage" || class == "priest" {
		hero.MP = 10
		hero.MaxMP = 10
	}

	eng := engine.New(s)
	state, err := eng.NewGame(hero)
	if err != nil {
		return SessionMetrics{}, fmt.Errorf("new game: %w", err)
	}

	interp := interpreter.NewRules()
	narr := narrator.NewTemplate()
	rng := rand.New(rand.NewSource(seed))
	monkey := NewMonkey(rng, maxMoves, class, seed)

	loop := game.NewLoop(eng, interp, narr, monkey)
	if err := loop.Run(context.Background(), &state); err != nil {
		return SessionMetrics{}, fmt.Errorf("game loop: %w", err)
	}

	return monkey.Metrics(), nil
}

// loadScenario loads a scenario from a directory, trying directory format first.
func loadScenario(scenarioDir, id string) (*scenario.Scenario, error) {
	// Try the scenariodir loader which handles both formats.
	// Import it inline to avoid circular deps — but scenariodir imports scenario,
	// and we only need scenario.Load / scenario.LoadDir here.
	// Use scenario.Load directly with the resolved path.
	path := scenarioDir + "/" + id + ".yaml"
	s, err := scenario.Load(path)
	if err != nil {
		// Try directory format.
		dirPath := scenarioDir + "/" + id
		s, err = scenario.LoadDir(dirPath)
		if err != nil {
			return nil, fmt.Errorf("load %s: %w", id, err)
		}
	}
	return s, nil
}
