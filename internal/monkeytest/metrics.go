// Package monkeytest provides a monkey-testing harness for game balance evaluation.
// It runs parallel game sessions with weighted-random action selection and
// collects metrics for survival rate, XP progression, combat outcomes, etc.
package monkeytest

import (
	"math"
	"sort"
)

// SessionMetrics tracks observables from a single game session.
type SessionMetrics struct {
	Seed          int64  `json:"seed"`
	Class         string `json:"class"`
	TotalMoves    int    `json:"total_moves"`
	RoomsVisited  int    `json:"rooms_visited"`
	EnemiesKilled int    `json:"enemies_killed"`
	XPGained      int    `json:"xp_gained"`
	FinalLevel    int    `json:"final_level"`
	FinalHP       int    `json:"final_hp"`
	FinalMaxHP    int    `json:"final_max_hp"`
	ItemsPickedUp int    `json:"items_picked_up"`
	ItemsEquipped int    `json:"items_equipped"`
	CombatRounds  int    `json:"combat_rounds"`
	DamageDealt   int    `json:"damage_dealt"`
	DamageTaken   int    `json:"damage_taken"`
	Survived      bool   `json:"survived"`
	FleeAttempts  int    `json:"flee_attempts"`
	FleeSuccesses int    `json:"flee_successes"`
	SpellsCast    int    `json:"spells_cast"`
	HealsCast     int    `json:"heals_cast"`
}

// RunConfig configures a batch of monkey test sessions.
type RunConfig struct {
	Scenario string `json:"scenario"`
	Class    string `json:"class"` // fighter|mage|thief|priest|all
	Players  int    `json:"players"`
	MaxMoves int    `json:"max_moves"`
	Workers  int    `json:"workers"`
	Seed     int64  `json:"seed"`
}

// AggregateReport is the top-level JSON output.
type AggregateReport struct {
	Config    RunConfig                    `json:"config"`
	Aggregate AggregateMetrics             `json:"aggregate"`
	PerClass  map[string]AggregateMetrics  `json:"per_class,omitempty"`
	Sessions  []SessionMetrics             `json:"sessions,omitempty"`
}

// AggregateMetrics summarizes a batch of sessions.
type AggregateMetrics struct {
	Count           int     `json:"count"`
	SurvivalRate    float64 `json:"survival_rate"`
	MeanMoves       float64 `json:"mean_moves"`
	MeanXP          float64 `json:"mean_xp"`
	MeanRooms       float64 `json:"mean_rooms_visited"`
	MeanEnemies     float64 `json:"mean_enemies_killed"`
	MeanLevel       float64 `json:"mean_level"`
	MeanDmgDealt    float64 `json:"mean_damage_dealt"`
	MeanDmgTaken    float64 `json:"mean_damage_taken"`
	MeanCombatRnds  float64 `json:"mean_combat_rounds"`
	MeanFleeRate    float64 `json:"mean_flee_success_rate"`
	P50Moves        int     `json:"p50_moves"`
	P95Moves        int     `json:"p95_moves"`
	P50XP           int     `json:"p50_xp"`
	P95XP           int     `json:"p95_xp"`
	P50Enemies      int     `json:"p50_enemies_killed"`
	P95Enemies      int     `json:"p95_enemies_killed"`
}

// Compute calculates aggregate metrics from a slice of session results.
func Compute(sessions []SessionMetrics) AggregateMetrics {
	n := len(sessions)
	if n == 0 {
		return AggregateMetrics{}
	}

	var (
		survived    int
		totalMoves  int
		totalXP     int
		totalRooms  int
		totalKills  int
		totalLevel  int
		totalDDealt int
		totalDTaken int
		totalRounds int
		totalFAttempt int
		totalFSuccess int
	)

	moves := make([]int, n)
	xps := make([]int, n)
	kills := make([]int, n)

	for i, s := range sessions {
		if s.Survived {
			survived++
		}
		totalMoves += s.TotalMoves
		totalXP += s.XPGained
		totalRooms += s.RoomsVisited
		totalKills += s.EnemiesKilled
		totalLevel += s.FinalLevel
		totalDDealt += s.DamageDealt
		totalDTaken += s.DamageTaken
		totalRounds += s.CombatRounds
		totalFAttempt += s.FleeAttempts
		totalFSuccess += s.FleeSuccesses

		moves[i] = s.TotalMoves
		xps[i] = s.XPGained
		kills[i] = s.EnemiesKilled
	}

	fn := float64(n)
	fleeRate := 0.0
	if totalFAttempt > 0 {
		fleeRate = float64(totalFSuccess) / float64(totalFAttempt)
	}

	return AggregateMetrics{
		Count:          n,
		SurvivalRate:   float64(survived) / fn,
		MeanMoves:      float64(totalMoves) / fn,
		MeanXP:         float64(totalXP) / fn,
		MeanRooms:      float64(totalRooms) / fn,
		MeanEnemies:    float64(totalKills) / fn,
		MeanLevel:      float64(totalLevel) / fn,
		MeanDmgDealt:   float64(totalDDealt) / fn,
		MeanDmgTaken:   float64(totalDTaken) / fn,
		MeanCombatRnds: float64(totalRounds) / fn,
		MeanFleeRate:   math.Round(fleeRate*1000) / 1000,
		P50Moves:       percentile(moves, 50),
		P95Moves:       percentile(moves, 95),
		P50XP:          percentile(xps, 50),
		P95XP:          percentile(xps, 95),
		P50Enemies:     percentile(kills, 50),
		P95Enemies:     percentile(kills, 95),
	}
}

// percentile returns the p-th percentile of a slice of ints (nearest-rank method).
func percentile(data []int, p int) int {
	if len(data) == 0 {
		return 0
	}
	sorted := make([]int, len(data))
	copy(sorted, data)
	sort.Ints(sorted)

	rank := int(math.Ceil(float64(p)/100.0*float64(len(sorted)))) - 1
	if rank < 0 {
		rank = 0
	}
	if rank >= len(sorted) {
		rank = len(sorted) - 1
	}
	return sorted[rank]
}
