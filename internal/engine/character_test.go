package engine_test

import (
	"errors"
	"testing"

	"github.com/punt-labs/cryptd/internal/engine"
	"github.com/punt-labs/cryptd/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCharacter_DefaultStats(t *testing.T) {
	hero, err := engine.NewCharacter("Conan", "fighter", nil)
	require.NoError(t, err)
	assert.Equal(t, "Conan", hero.Name)
	assert.Equal(t, "fighter", hero.Class)
	assert.Equal(t, 1, hero.Level)
	assert.Equal(t, 21, hero.HP) // base 20 + CON 12 modifier (+1)
	assert.Equal(t, 0, hero.MP) // fighter has no MP
	assert.Equal(t, 14, hero.Stats.STR)
	assert.Equal(t, 12, hero.Stats.DEX)
	assert.Equal(t, 12, hero.Stats.CON)
}

func TestNewCharacter_MageGetsMP(t *testing.T) {
	hero, err := engine.NewCharacter("Merlin", "mage", nil)
	require.NoError(t, err)
	assert.Equal(t, 10, hero.MP)
	assert.Equal(t, 10, hero.MaxMP)
}

func TestNewCharacter_PriestGetsMP(t *testing.T) {
	hero, err := engine.NewCharacter("Cleric", "priest", nil)
	require.NoError(t, err)
	assert.Equal(t, 10, hero.MP)
}

func TestNewCharacter_ThiefNoMP(t *testing.T) {
	hero, err := engine.NewCharacter("Shadow", "thief", nil)
	require.NoError(t, err)
	assert.Equal(t, 0, hero.MP)
}

func TestNewCharacter_CustomStats(t *testing.T) {
	stats := &model.Stats{STR: 10, DEX: 18, CON: 10, INT: 10, WIS: 10, CHA: 10}
	hero, err := engine.NewCharacter("Speedy", "thief", stats)
	require.NoError(t, err)
	assert.Equal(t, 18, hero.Stats.DEX)
	assert.Equal(t, 10, hero.Stats.STR)
}

func TestNewCharacter_InvalidClass(t *testing.T) {
	_, err := engine.NewCharacter("Bard", "bard", nil)
	require.Error(t, err)
	var invalid *engine.InvalidStatsError
	assert.True(t, errors.As(err, &invalid))
	assert.Contains(t, invalid.Error(), "bard")
}

func TestValidateStats_Valid(t *testing.T) {
	err := engine.ValidateStats(model.Stats{STR: 14, DEX: 12, CON: 12, INT: 10, WIS: 10, CHA: 10})
	assert.NoError(t, err)
}

func TestValidateStats_TooManyPoints(t *testing.T) {
	err := engine.ValidateStats(model.Stats{STR: 18, DEX: 12, CON: 12, INT: 10, WIS: 10, CHA: 10})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "12")  // allocated 12
	assert.Contains(t, err.Error(), "8")   // must be 8
}

func TestValidateStats_TooFewPoints(t *testing.T) {
	err := engine.ValidateStats(model.Stats{STR: 12, DEX: 10, CON: 10, INT: 10, WIS: 10, CHA: 10})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "2")   // allocated 2
}

func TestValidateStats_BelowMinimum(t *testing.T) {
	err := engine.ValidateStats(model.Stats{STR: 8, DEX: 14, CON: 14, INT: 10, WIS: 10, CHA: 12})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "STR")
	assert.Contains(t, err.Error(), "8")
}

func TestValidateStats_AllEqual(t *testing.T) {
	// 8 points spread: each stat gets +1 except two... wait, 8/6 doesn't divide evenly.
	// 6 stats at 11 = 6 points, need 8. So not possible with all equal above 10.
	// All at 10 = 0 points, fails.
	err := engine.ValidateStats(model.Stats{STR: 10, DEX: 10, CON: 10, INT: 10, WIS: 10, CHA: 10})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "0")
}

func TestDefaultStats(t *testing.T) {
	s := engine.DefaultStats()
	total := (s.STR - 10) + (s.DEX - 10) + (s.CON - 10) + (s.INT - 10) + (s.WIS - 10) + (s.CHA - 10)
	assert.Equal(t, engine.PointBuyPool, total)
}
