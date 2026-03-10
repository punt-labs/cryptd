package model

import "context"

// EngineAction is the result of interpreting a player's command.
// Fields are populated in Milestone 1; the struct exists here so testutil
// fakes can compile.
type EngineAction struct {
	Action    string
	Direction string
	Target    string
	ItemID    string
}

// EngineEvent is emitted by the engine after resolving an action.
type EngineEvent struct {
	Type    string
	Details map[string]any
}

// GameState represents the full, serialisable state of a game session.
// Fields are added in Milestone 1.
type GameState struct{}

// InputEvent is an event received from the renderer (e.g. a keypress or
// button click in Lux).
type InputEvent struct {
	Type    string
	Payload string
}

// CommandInterpreter parses free-form player input into a deterministic
// EngineAction. The engine calls this interface; it never implements it.
type CommandInterpreter interface {
	Interpret(ctx context.Context, input string, state GameState) (EngineAction, error)
}

// Narrator produces natural-language text for an EngineEvent.
// The engine calls this interface; it never implements it.
type Narrator interface {
	Narrate(ctx context.Context, event EngineEvent, state GameState) (string, error)
}

// Renderer presents the current game state and narration to the player and
// returns a channel of InputEvents from the player.
// The engine calls this interface; it never implements it.
type Renderer interface {
	Render(ctx context.Context, state GameState, narration string) error
	Events() <-chan InputEvent
}
