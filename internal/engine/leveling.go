package engine

import "github.com/punt-labs/cryptd/internal/model"

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

		// HP gain.
		hp := hpPerLevel[h.Class]
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

		// Stat gain: +1 to each primary stat for the class.
		for _, stat := range primaryStats[h.Class] {
			addStat(&h.Stats, stat, 1)
			result.StatGain[stat]++
		}
	}

	return result
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
