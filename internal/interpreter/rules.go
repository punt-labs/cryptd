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
	// Strip leading articles (the, a, an) before joining.
	rest := ""
	if len(fields) >= 2 {
		args := fields[1:]
		if len(args) > 1 && (args[0] == "the" || args[0] == "a" || args[0] == "an") {
			args = args[1:]
		}
		rest = strings.Join(args, "_")
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
		// "look at <item>" → examine; plain "look" → look.
		if len(fields) >= 3 && fields[1] == "at" {
			target := fields[2:]
			if len(target) > 1 && (target[0] == "the" || target[0] == "a" || target[0] == "an") {
				target = target[1:]
			}
			return model.EngineAction{Type: "examine", ItemID: strings.Join(target, "_")}, nil
		}
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
			args := fields[2:]
			if len(args) > 1 && (args[0] == "the" || args[0] == "a" || args[0] == "an") {
				args = args[1:]
			}
			return model.EngineAction{Type: "take", ItemID: strings.Join(args, "_")}, nil
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

	case "attack", "a", "hit", "strike", "kill":
		if rest != "" {
			return model.EngineAction{Type: "attack", Target: rest}, nil
		}
		return model.EngineAction{Type: "attack"}, nil

	case "defend", "block", "guard":
		return model.EngineAction{Type: "defend"}, nil

	case "cast":
		if rest != "" {
			// "cast fireball" or "cast fireball at goblin_0"
			// The underscore-join turns "fireball at goblin_0" into "fireball_at_goblin_0".
			parts := strings.SplitN(rest, "_at_", 2)
			if len(parts) == 2 {
				return model.EngineAction{Type: "cast", SpellID: parts[0], Target: parts[1]}, nil
			}
			return model.EngineAction{Type: "cast", SpellID: rest}, nil
		}

	case "flee", "run", "escape":
		return model.EngineAction{Type: "flee"}, nil

	case "inventory", "i":
		return model.EngineAction{Type: "inventory"}, nil

	case "save":
		if rest != "" {
			return model.EngineAction{Type: "save", Target: rest}, nil
		}
		return model.EngineAction{Type: "save"}, nil

	case "load":
		if rest != "" {
			return model.EngineAction{Type: "load", Target: rest}, nil
		}
		return model.EngineAction{Type: "load"}, nil

	case "help", "?":
		return model.EngineAction{Type: "help"}, nil

	case "quit", "exit", "q":
		return model.EngineAction{Type: "quit"}, nil
	}

	return model.EngineAction{Type: "unknown"}, nil
}
