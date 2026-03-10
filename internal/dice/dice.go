// Package dice parses and evaluates dice notation (e.g. "2d6+3").
package dice

import (
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
)

var notation = regexp.MustCompile(`^(\d+)d(\d+)([+-]\d+)?$`)

// Dice holds the components of a parsed dice notation string.
type Dice struct {
	Count    int
	Sides    int
	Modifier int
}

// ParseError is returned when a notation string cannot be parsed.
type ParseError struct {
	Notation string
	Reason   string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("invalid dice notation %q: %s", e.Notation, e.Reason)
}

// Parse parses a dice notation string into a Dice value.
// Valid forms: "1d6", "2d6+3", "1d20-1".
// Returns *ParseError for any invalid input.
func Parse(s string) (Dice, error) {
	m := notation.FindStringSubmatch(s)
	if m == nil {
		return Dice{}, &ParseError{Notation: s, Reason: "must match NdS or NdS±M"}
	}
	count, _ := strconv.Atoi(m[1])
	sides, _ := strconv.Atoi(m[2])
	if count < 1 {
		return Dice{}, &ParseError{Notation: s, Reason: "count must be ≥ 1"}
	}
	if sides < 1 {
		return Dice{}, &ParseError{Notation: s, Reason: "sides must be ≥ 1"}
	}
	var mod int
	if m[3] != "" {
		mod, _ = strconv.Atoi(m[3])
	}
	return Dice{Count: count, Sides: sides, Modifier: mod}, nil
}

// Roll returns a random result in [Min(), Max()].
func (d Dice) Roll() int {
	total := d.Modifier
	for i := 0; i < d.Count; i++ {
		total += rand.Intn(d.Sides) + 1
	}
	return total
}

// Min returns the lowest possible roll result.
func (d Dice) Min() int { return d.Count*1 + d.Modifier }

// Max returns the highest possible roll result.
func (d Dice) Max() int { return d.Count*d.Sides + d.Modifier }
