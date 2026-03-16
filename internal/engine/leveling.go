package engine

import (
	"fmt"

	"github.com/punt-labs/cryptd/internal/model"
)

// BaseStatValue is the starting value for each attribute before point-buy.
const BaseStatValue = 10

// PointBuyPool is the total number of points a player can distribute.
const PointBuyPool = 8

// ValidClasses lists the four playable classes.
var ValidClasses = map[string]bool{
	"fighter": true, "mage": true, "priest": true, "thief": true,
}

// InvalidStatsError is returned when stat allocation violates constraints.
type InvalidStatsError struct {
	Reason string
}

func (e *InvalidStatsError) Error() string {
	return fmt.Sprintf("invalid stats: %s", e.Reason)
}

// NewCharacter creates a level-1 character with validated point-buy stats.
// If stats is nil, default allocation is used (STR 14, DEX 12, CON 12, others 10).
// Stats must each be >= BaseStatValue and the total points above base must equal PointBuyPool.
func NewCharacter(name, class string, stats *model.Stats) (model.Character, error) {
	if !ValidClasses[class] {
		return model.Character{}, &InvalidStatsError{
			Reason: fmt.Sprintf("unknown class %q; must be fighter, mage, priest, or thief", class),
		}
	}

	var s model.Stats
	if stats != nil {
		s = *stats
	} else {
		s = DefaultStats()
	}

	if err := ValidateStats(s); err != nil {
		return model.Character{}, err
	}

	baseHP := 20 + StatModifier(s.CON)
	if baseHP < 1 {
		baseHP = 1
	}
	hero := model.Character{
		ID: "hero", Name: name, Class: class,
		Level: 1, HP: baseHP, MaxHP: baseHP,
		Stats: s,
	}
	if class == "mage" || class == "priest" {
		hero.MP = 10
		hero.MaxMP = 10
	}
	return hero, nil
}

// DefaultStats returns the default stat allocation: STR 14, DEX 12, CON 12, rest 10.
func DefaultStats() model.Stats {
	return model.Stats{STR: 14, DEX: 12, CON: 12, INT: 10, WIS: 10, CHA: 10}
}

// ValidateStats checks that all stats are >= BaseStatValue and the total
// points allocated above base equals PointBuyPool.
func ValidateStats(s model.Stats) error {
	stats := []struct {
		name  string
		value int
	}{
		{"STR", s.STR}, {"DEX", s.DEX}, {"CON", s.CON},
		{"INT", s.INT}, {"WIS", s.WIS}, {"CHA", s.CHA},
	}

	total := 0
	for _, st := range stats {
		if st.value < BaseStatValue {
			return &InvalidStatsError{
				Reason: fmt.Sprintf("%s is %d, minimum is %d", st.name, st.value, BaseStatValue),
			}
		}
		total += st.value - BaseStatValue
	}

	if total != PointBuyPool {
		return &InvalidStatsError{
			Reason: fmt.Sprintf("allocated %d points, must be exactly %d", total, PointBuyPool),
		}
	}
	return nil
}

// LevelUpResult holds the outcome of a level-up check.
type LevelUpResult struct {
	Leveled  bool
	NewLevel int
	HPGain   int
	MPGain   int
	StatGain map[string]int // e.g. {"STR": 1, "CON": 1}
}

// xpThresholds defines cumulative XP required to reach each level, per class.
// Index 0 is unused (level 0 doesn't exist); index i is the XP needed for level i.
// Wizardry-inspired: fighters level cheaply, mages expensively.
var xpThresholds = map[string][]int{
	"fighter": {0, 0, 20, 50, 100, 200, 400, 800, 1600, 3200, 6400},
	"mage":    {0, 0, 30, 75, 150, 300, 600, 1200, 2400, 4800, 9600},
	"priest":  {0, 0, 25, 60, 120, 250, 500, 1000, 2000, 4000, 8000},
	"thief":   {0, 0, 22, 55, 110, 220, 440, 880, 1760, 3520, 7040},
}

// hpPerLevel defines the base HP gained per level-up, per class.
var hpPerLevel = map[string]int{
	"fighter": 8,
	"mage":    4,
	"priest":  6,
	"thief":   6,
}

// mpPerLevel defines the base MP gained per level-up, per class.
var mpPerLevel = map[string]int{
	"fighter": 0,
	"mage":    4,
	"priest":  3,
	"thief":   0,
}

// primaryStats defines which stats gain +1 on each level-up, per class.
var primaryStats = map[string][]string{
	"fighter": {"STR", "CON"},
	"mage":    {"INT", "WIS"},
	"priest":  {"WIS", "CHA"},
	"thief":   {"DEX", "CHA"},
}

// MaxLevel is the highest level a character can reach.
const MaxLevel = 10

// NextLevelXP returns the cumulative XP required for the next level of the
// given class. Returns 0 if the class is unknown, the level is invalid (<= 0),
// or the character is at max level.
func NextLevelXP(class string, level int) int {
	if level <= 0 {
		return 0
	}
	thresholds, ok := xpThresholds[class]
	if !ok {
		return 0
	}
	next := level + 1
	if next >= len(thresholds) {
		return 0
	}
	return thresholds[next]
}

// CheckLevelUp checks if the hero has enough XP to level up, and if so,
// applies the level-up effects (HP, MP, stats). May apply multiple levels
// if XP is sufficient.
func (e *Engine) CheckLevelUp(state *model.GameState) LevelUpResult {
	h := hero(state)
	thresholds, ok := xpThresholds[h.Class]
	if !ok {
		return LevelUpResult{}
	}

	result := LevelUpResult{StatGain: make(map[string]int)}

	for h.Level < MaxLevel {
		nextLevel := h.Level + 1
		if nextLevel >= len(thresholds) {
			break
		}
		if h.XP < thresholds[nextLevel] {
			break
		}

		h.Level = nextLevel
		result.Leveled = true
		result.NewLevel = nextLevel

		// Stat gain: +1 to each primary stat for the class.
		// Applied before HP so the CON modifier reflects the new value.
		for _, stat := range primaryStats[h.Class] {
			addStat(&h.Stats, stat, 1)
			result.StatGain[stat]++
		}

		// HP gain: base class amount + CON modifier.
		hp := hpPerLevel[h.Class] + StatModifier(h.Stats.CON)
		if hp < 1 {
			hp = 1 // always gain at least 1 HP
		}
		h.MaxHP += hp
		h.HP += hp
		result.HPGain += hp

		// MP gain.
		mp := mpPerLevel[h.Class]
		if mp > 0 {
			h.MaxMP += mp
			h.MP += mp
			result.MPGain += mp
		}
	}

	return result
}

// StatModifier returns the ability modifier for a given stat value.
// Formula: (stat - 10) / 2, rounded down (integer division truncates toward zero).
// Examples: CON 10 → +0, CON 12 → +1, CON 14 → +2, CON 8 → -1.
func StatModifier(stat int) int {
	return (stat - 10) / 2
}

// addStat increments the named stat by delta.
func addStat(s *model.Stats, name string, delta int) {
	switch name {
	case "STR":
		s.STR += delta
	case "INT":
		s.INT += delta
	case "DEX":
		s.DEX += delta
	case "CON":
		s.CON += delta
	case "WIS":
		s.WIS += delta
	case "CHA":
		s.CHA += delta
	}
}
