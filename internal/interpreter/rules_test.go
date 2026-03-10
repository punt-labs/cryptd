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
	for _, input := range []string{"frobnicate", "attack goblin", "", "   "} {
		t.Run(input, func(t *testing.T) {
			a := interpret(t, input)
			assert.Equal(t, "unknown", a.Type)
		})
	}
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
