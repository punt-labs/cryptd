// Package interpreter provides CommandInterpreter implementations.
package interpreter

import (
	"context"
	"strings"

	"github.com/punt-labs/cryptd/internal/model"
)

// Rules maps a fixed vocabulary of text commands to engine actions.
// Unknown input returns an action of type "unknown" — not an error.
type Rules struct{}

// NewRules returns a new RulesInterpreter.
func NewRules() *Rules { return &Rules{} }

var shortDirs = map[string]string{
	"n": "north", "s": "south", "e": "east", "w": "west",
	"u": "up", "d": "down",
}

var longDirs = map[string]bool{
	"north": true, "south": true, "east": true,
	"west": true, "up": true, "down": true,
}

// Interpret parses input and returns the corresponding EngineAction.
// It never returns an error; unrecognised input yields Type="unknown".
func (r *Rules) Interpret(_ context.Context, input string, _ model.GameState) (model.EngineAction, error) {
	fields := strings.Fields(strings.ToLower(input))
	if len(fields) == 0 {
		return model.EngineAction{Type: "unknown"}, nil
	}

	verb := fields[0]

	// Single-character direction shortcuts.
	if dir, ok := shortDirs[verb]; ok && len(fields) == 1 {
		return model.EngineAction{Type: "move", Direction: dir}, nil
	}

	// Long-form direction words used alone.
	if longDirs[verb] && len(fields) == 1 {
		return model.EngineAction{Type: "move", Direction: verb}, nil
	}

	// Join remaining fields for multi-word targets, using underscores to
	// match scenario item IDs (e.g. "short sword" → "short_sword").
	rest := ""
	if len(fields) >= 2 {
		rest = strings.Join(fields[1:], "_")
	}

	switch verb {
	case "go":
		if len(fields) >= 2 {
			dir := fields[1]
			if d, ok := shortDirs[dir]; ok {
				dir = d
			}
			return model.EngineAction{Type: "move", Direction: dir}, nil
		}

	case "look", "l":
		return model.EngineAction{Type: "look"}, nil

	case "examine", "x":
		if rest == "" || rest == "room" {
			return model.EngineAction{Type: "look"}, nil
		}
		return model.EngineAction{Type: "examine", ItemID: rest}, nil

	case "take", "get", "grab":
		if rest != "" {
			return model.EngineAction{Type: "take", ItemID: rest}, nil
		}

	case "pick":
		// "pick up <item>"
		if len(fields) >= 3 && fields[1] == "up" {
			target := strings.Join(fields[2:], "_")
			return model.EngineAction{Type: "take", ItemID: target}, nil
		}

	case "drop":
		if rest != "" {
			return model.EngineAction{Type: "drop", ItemID: rest}, nil
		}

	case "equip", "wear", "wield":
		if rest != "" {
			return model.EngineAction{Type: "equip", ItemID: rest}, nil
		}

	case "unequip", "remove":
		if rest != "" {
			return model.EngineAction{Type: "unequip", Target: rest}, nil
		}

	case "inventory", "i":
		return model.EngineAction{Type: "inventory"}, nil

	case "quit", "exit", "q":
		return model.EngineAction{Type: "quit"}, nil
	}

	return model.EngineAction{Type: "unknown"}, nil
}
