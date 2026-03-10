package dice_test

import (
	"testing"

	"github.com/punt-labs/cryptd/internal/dice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	tests := []struct {
		notation string
		want     dice.Dice
		wantErr  bool
	}{
		{"1d6", dice.Dice{Count: 1, Sides: 6, Modifier: 0}, false},
		{"2d6+3", dice.Dice{Count: 2, Sides: 6, Modifier: 3}, false},
		{"1d20-1", dice.Dice{Count: 1, Sides: 20, Modifier: -1}, false},
		{"3d8+0", dice.Dice{Count: 3, Sides: 8, Modifier: 0}, false},
		{"10d10+10", dice.Dice{Count: 10, Sides: 10, Modifier: 10}, false},
		{"1d4", dice.Dice{Count: 1, Sides: 4, Modifier: 0}, false},
		// Invalid notations
		{"0d6", dice.Dice{}, true},
		{"1d0", dice.Dice{}, true},
		{"abc", dice.Dice{}, true},
		{"", dice.Dice{}, true},
		{"2x6", dice.Dice{}, true},
		{"d6", dice.Dice{}, true},
		{"1d", dice.Dice{}, true},
		{"-1d6", dice.Dice{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.notation, func(t *testing.T) {
			got, err := dice.Parse(tt.notation)
			if tt.wantErr {
				require.Error(t, err)
				var pe *dice.ParseError
				assert.ErrorAs(t, err, &pe, "error should be *dice.ParseError")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDiceMinMax(t *testing.T) {
	tests := []struct {
		notation string
		min, max int
	}{
		{"1d6", 1, 6},
		{"2d6+3", 5, 15},
		{"1d20-1", 0, 19},
		{"3d4", 3, 12},
		{"1d1", 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.notation, func(t *testing.T) {
			d, err := dice.Parse(tt.notation)
			require.NoError(t, err)
			assert.Equal(t, tt.min, d.Min())
			assert.Equal(t, tt.max, d.Max())
		})
	}
}

func TestDiceRollInBounds(t *testing.T) {
	tests := []string{"1d6", "2d6+3", "1d20-1", "3d8", "1d4"}
	for _, notation := range tests {
		t.Run(notation, func(t *testing.T) {
			d, err := dice.Parse(notation)
			require.NoError(t, err)
			for i := 0; i < 100; i++ {
				r := d.Roll()
				assert.GreaterOrEqual(t, r, d.Min(), "roll %d < min %d", r, d.Min())
				assert.LessOrEqual(t, r, d.Max(), "roll %d > max %d", r, d.Max())
			}
		})
	}
}

func TestParseErrorContainsNotation(t *testing.T) {
	_, err := dice.Parse("bad_notation")
	require.Error(t, err)
	var pe *dice.ParseError
	require.ErrorAs(t, err, &pe)
	assert.Equal(t, "bad_notation", pe.Notation)
	assert.NotEmpty(t, pe.Reason)
	assert.Contains(t, pe.Error(), "bad_notation")
}
