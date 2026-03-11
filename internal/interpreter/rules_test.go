package interpreter_test

import (
	"context"
	"testing"

	"github.com/punt-labs/cryptd/internal/interpreter"
	"github.com/punt-labs/cryptd/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func interpret(t *testing.T, input string) model.EngineAction {
	t.Helper()
	r := interpreter.NewRules()
	action, err := r.Interpret(context.Background(), input, model.GameState{})
	require.NoError(t, err)
	return action
}

var moveTests = []struct {
	input     string
	wantType  string
	wantParam string
}{
	{"go north", "move", "north"},
	{"go south", "move", "south"},
	{"go east", "move", "east"},
	{"go west", "move", "west"},
	{"go up", "move", "up"},
	{"go down", "move", "down"},
	{"n", "move", "north"},
	{"s", "move", "south"},
	{"e", "move", "east"},
	{"w", "move", "west"},
	{"north", "move", "north"},
	{"south", "move", "south"},
	{"east", "move", "east"},
	{"west", "move", "west"},
}

func TestRulesInterpreter_MoveVerbs(t *testing.T) {
	for _, tt := range moveTests {
		t.Run(tt.input, func(t *testing.T) {
			a := interpret(t, tt.input)
			assert.Equal(t, tt.wantType, a.Type)
			assert.Equal(t, tt.wantParam, a.Direction)
		})
	}
}

func TestRulesInterpreter_Look(t *testing.T) {
	for _, input := range []string{"look", "l", "look around", "examine room"} {
		t.Run(input, func(t *testing.T) {
			a := interpret(t, input)
			assert.Equal(t, "look", a.Type)
		})
	}
}

func TestRulesInterpreter_Quit(t *testing.T) {
	for _, input := range []string{"quit", "exit", "q"} {
		t.Run(input, func(t *testing.T) {
			a := interpret(t, input)
			assert.Equal(t, "quit", a.Type)
		})
	}
}

func TestRulesInterpreter_Unknown(t *testing.T) {
	for _, input := range []string{"frobnicate", "", "   "} {
		t.Run(input, func(t *testing.T) {
			a := interpret(t, input)
			assert.Equal(t, "unknown", a.Type)
		})
	}
}

var takeTests = []struct {
	input  string
	itemID string
}{
	{"take rusty_key", "rusty_key"},
	{"get rusty_key", "rusty_key"},
	{"grab rusty_key", "rusty_key"},
	{"pick up rusty_key", "rusty_key"},
	{"TAKE Rusty_Key", "rusty_key"},
}

func TestRulesInterpreter_TakeVerbs(t *testing.T) {
	for _, tt := range takeTests {
		t.Run(tt.input, func(t *testing.T) {
			a := interpret(t, tt.input)
			assert.Equal(t, "take", a.Type)
			assert.Equal(t, tt.itemID, a.ItemID)
		})
	}
}

func TestRulesInterpreter_Drop(t *testing.T) {
	a := interpret(t, "drop rusty_key")
	assert.Equal(t, "drop", a.Type)
	assert.Equal(t, "rusty_key", a.ItemID)
}

func TestRulesInterpreter_Equip(t *testing.T) {
	for _, input := range []string{"equip short_sword", "wear short_sword", "wield short_sword"} {
		t.Run(input, func(t *testing.T) {
			a := interpret(t, input)
			assert.Equal(t, "equip", a.Type)
			assert.Equal(t, "short_sword", a.ItemID)
		})
	}
}

func TestRulesInterpreter_Unequip(t *testing.T) {
	for _, input := range []string{"unequip weapon", "remove weapon"} {
		t.Run(input, func(t *testing.T) {
			a := interpret(t, input)
			assert.Equal(t, "unequip", a.Type)
			assert.Equal(t, "weapon", a.Target)
		})
	}
}

func TestRulesInterpreter_Inventory(t *testing.T) {
	for _, input := range []string{"inventory", "i"} {
		t.Run(input, func(t *testing.T) {
			a := interpret(t, input)
			assert.Equal(t, "inventory", a.Type)
		})
	}
}

func TestRulesInterpreter_Examine(t *testing.T) {
	a := interpret(t, "examine short_sword")
	assert.Equal(t, "examine", a.Type)
	assert.Equal(t, "short_sword", a.ItemID)

	// "examine room" is still a look
	a = interpret(t, "examine room")
	assert.Equal(t, "look", a.Type)

	// "x" alone is a look
	a = interpret(t, "x")
	assert.Equal(t, "look", a.Type)

	// "x short_sword" is examine
	a = interpret(t, "x short_sword")
	assert.Equal(t, "examine", a.Type)
	assert.Equal(t, "short_sword", a.ItemID)
}

func TestRulesInterpreter_Attack(t *testing.T) {
	for _, tt := range []struct {
		input  string
		target string
	}{
		{"attack goblin_0", "goblin_0"},
		{"a goblin_0", "goblin_0"},
		{"hit goblin_0", "goblin_0"},
		{"strike goblin_0", "goblin_0"},
		{"kill goblin_0", "goblin_0"},
		{"attack", ""},     // no target — default to first alive
		{"a", ""},           // shorthand, no target
	} {
		t.Run(tt.input, func(t *testing.T) {
			a := interpret(t, tt.input)
			assert.Equal(t, "attack", a.Type)
			assert.Equal(t, tt.target, a.Target)
		})
	}
}

func TestRulesInterpreter_Defend(t *testing.T) {
	for _, input := range []string{"defend", "block", "guard"} {
		t.Run(input, func(t *testing.T) {
			a := interpret(t, input)
			assert.Equal(t, "defend", a.Type)
		})
	}
}

func TestRulesInterpreter_Flee(t *testing.T) {
	for _, input := range []string{"flee", "run", "escape"} {
		t.Run(input, func(t *testing.T) {
			a := interpret(t, input)
			assert.Equal(t, "flee", a.Type)
		})
	}
}

func TestRulesInterpreter_Cast(t *testing.T) {
	// Simple cast.
	a := interpret(t, "cast fireball")
	assert.Equal(t, "cast", a.Type)
	assert.Equal(t, "fireball", a.SpellID)
	assert.Equal(t, "", a.Target)

	// Cast with target using "at".
	a = interpret(t, "cast fireball at goblin_0")
	assert.Equal(t, "cast", a.Type)
	assert.Equal(t, "fireball", a.SpellID)
	assert.Equal(t, "goblin_0", a.Target)

	// Cast alone — no spell specified.
	a = interpret(t, "cast")
	assert.Equal(t, "unknown", a.Type)
}

func TestRulesInterpreter_Save(t *testing.T) {
	// Default slot.
	a := interpret(t, "save")
	assert.Equal(t, "save", a.Type)
	assert.Equal(t, "", a.Target)

	// Named slot.
	a = interpret(t, "save slot1")
	assert.Equal(t, "save", a.Type)
	assert.Equal(t, "slot1", a.Target)
}

func TestRulesInterpreter_Load(t *testing.T) {
	a := interpret(t, "load")
	assert.Equal(t, "load", a.Type)
	assert.Equal(t, "", a.Target)

	a = interpret(t, "load slot1")
	assert.Equal(t, "load", a.Type)
	assert.Equal(t, "slot1", a.Target)
}

func TestRulesInterpreter_CaseInsensitive(t *testing.T) {
	a := interpret(t, "GO NORTH")
	assert.Equal(t, "move", a.Type)
	assert.Equal(t, "north", a.Direction)
}

func TestRulesInterpreter_ExtraWhitespace(t *testing.T) {
	a := interpret(t, "  go   south  ")
	assert.Equal(t, "move", a.Type)
	assert.Equal(t, "south", a.Direction)
}

func TestRulesInterpreter_MultiWordItemNormalized(t *testing.T) {
	// "take short sword" should produce ItemID "short_sword" (spaces → underscores)
	a := interpret(t, "take short sword")
	assert.Equal(t, "take", a.Type)
	assert.Equal(t, "short_sword", a.ItemID)

	// "pick up rusty key" too
	a = interpret(t, "pick up rusty key")
	assert.Equal(t, "take", a.Type)
	assert.Equal(t, "rusty_key", a.ItemID)

	// "examine short sword"
	a = interpret(t, "examine short sword")
	assert.Equal(t, "examine", a.Type)
	assert.Equal(t, "short_sword", a.ItemID)

	// "drop short sword"
	a = interpret(t, "drop short sword")
	assert.Equal(t, "drop", a.Type)
	assert.Equal(t, "short_sword", a.ItemID)
}
