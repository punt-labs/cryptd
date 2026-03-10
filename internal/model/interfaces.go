package model

import "context"

// EngineAction is the result of interpreting a player's command.
type EngineAction struct {
	Type        string
	Direction   string
	Target      string
	ItemID      string
	CharacterID string // optional; for party-ready multi-character support (DES-021)
}

// EngineEvent is emitted by the engine after resolving an action.
type EngineEvent struct {
	Type    string
	Room    string         // destination room for "moved"; current room for "looked"
	Details map[string]any // additional context for narrators and renderers
}

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
