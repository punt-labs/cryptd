// Package game implements the main game loop that wires together the engine,
// interpreter, narrator, and renderer.
package game

import (
	"context"
	"errors"
	"fmt"

	"github.com/punt-labs/cryptd/internal/engine"
	"github.com/punt-labs/cryptd/internal/model"
)

// Loop runs the game by pulling input from the renderer, routing it through
// the interpreter and engine, and pushing narrated output back to the renderer.
type Loop struct {
	eng    *engine.Engine
	interp model.CommandInterpreter
	narr   model.Narrator
	rend   model.Renderer
}

// NewLoop creates a game Loop.
func NewLoop(eng *engine.Engine, interp model.CommandInterpreter, narr model.Narrator, rend model.Renderer) *Loop {
	return &Loop{eng: eng, interp: interp, narr: narr, rend: rend}
}

// Run drives the game loop until the player quits or the context is cancelled.
func (l *Loop) Run(ctx context.Context, state *model.GameState) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	// Render initial state.
	look := l.eng.Look(state)
	narration, err := l.narr.Narrate(ctx, model.EngineEvent{Type: "looked", Room: look.Room}, *state)
	if err != nil {
		return err
	}
	if err := l.rend.Render(ctx, *state, narration); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-l.rend.Events():
			if !ok {
				return nil
			}

			// Route all events — including renderer-emitted quits — through
			// dispatch so the quit narration is rendered before returning.
			var action model.EngineAction
			switch ev.Type {
			case "quit":
				action = model.EngineAction{Type: "quit"}
			case "error":
				return fmt.Errorf("renderer error: %s", ev.Payload)
			default:
				action, err = l.interp.Interpret(ctx, ev.Payload, *state)
				if err != nil {
					return err
				}
			}

			event, narration, err := l.dispatch(ctx, state, action)
			if err != nil {
				return err
			}

			if err := l.rend.Render(ctx, *state, narration); err != nil {
				return err
			}

			if event.Type == "quit" {
				return nil
			}
		}
	}
}

func (l *Loop) dispatch(ctx context.Context, state *model.GameState, action model.EngineAction) (model.EngineEvent, string, error) {
	var event model.EngineEvent

	switch action.Type {
	case "move":
		result, err := l.eng.Move(state, action.Direction)
		if err != nil {
			var locked *engine.LockedError
			var noExit *engine.NoExitError
			switch {
			case errors.As(err, &locked):
				event = model.EngineEvent{Type: "locked_door", Details: map[string]any{"direction": locked.Direction}}
			case errors.As(err, &noExit):
				event = model.EngineEvent{Type: "no_exit"}
			default:
				return model.EngineEvent{}, "", err
			}
		} else {
			event = model.EngineEvent{Type: "moved", Room: result.NewRoom}
		}

	case "look":
		look := l.eng.Look(state)
		event = model.EngineEvent{Type: "looked", Room: look.Room}

	case "quit":
		event = model.EngineEvent{Type: "quit"}

	default:
		event = model.EngineEvent{Type: "unknown_action"}
	}

	narration, err := l.narr.Narrate(ctx, event, *state)
	if err != nil {
		return event, "", err
	}
	return event, narration, nil
}
