package daemon

import (
	"context"
	"encoding/json"

	"github.com/punt-labs/cryptd/internal/game"
	"github.com/punt-labs/cryptd/internal/model"
)

// deepCopyState returns a deep copy of the game state via JSON round-trip.
// This ensures no slice backing arrays are shared with the original, which
// would be unsafe after the mutex is released.
//
// Panics on marshal/unmarshal failure — GameState is always valid JSON, so
// a failure here indicates a programmer error (e.g. unexportable field added).
func deepCopyState(state *model.GameState) *model.GameState {
	data, err := json.Marshal(state)
	if err != nil {
		panic("deepCopyState: marshal: " + err.Error())
	}
	var cp model.GameState
	if err := json.Unmarshal(data, &cp); err != nil {
		panic("deepCopyState: unmarshal: " + err.Error())
	}
	return &cp
}

// handlePlay processes a "play" request: interpret text → engine → narrate → text.
// Only available in normal mode (not passthrough).
func (s *Server) handlePlay(req Request) Response {
	if s.passthrough {
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: CodeMethodNotFound, Message: "play is not available in passthrough mode — use tools/call"},
		}
	}

	var params PlayRequest
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: CodeInvalidParams, Message: "invalid params: " + err.Error()},
		}
	}
	if params.Text == "" {
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: CodeInvalidParams, Message: "text is required"},
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.eng == nil || s.state == nil {
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: CodeNoActiveGame, Message: "no active game — call new_game first"},
		}
	}

	// Rebuild the loop if needed (after new_game creates a new engine).
	if s.loop == nil {
		s.loop = game.NewLoop(s.eng, s.interp, s.narr, nil)
	}

	ctx := context.Background()

	// Interpret text → engine action.
	action, err := s.interp.Interpret(ctx, params.Text, *s.state)
	if err != nil {
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  PlayResponse{Text: "I don't understand that command."},
		}
	}

	if action.Type == "quit" {
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  PlayResponse{Text: "Farewell, adventurer.", Quit: true},
		}
	}

	// Dispatch through the game loop (engine + narration).
	events, narration, err := s.loop.Dispatch(ctx, s.state, action)
	if err != nil {
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: CodeInternalError, Message: err.Error()},
		}
	}

	state := deepCopyState(s.state)
	result := PlayResponse{
		Text:  narration,
		State: state,
	}

	// Check for terminal events.
	for _, ev := range events {
		if ev.Type == "hero_died" {
			result.Dead = true
		}
	}

	return Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

// handleNewGamePlay handles new_game in normal mode: starts the game and returns
// the initial room narration as text.
func (s *Server) handleNewGamePlay(req Request) Response {
	// Use the passthrough dispatcher to create the game.
	_, rpcErr := s.dispatchNewGame(req.Params)
	if rpcErr != nil {
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   rpcErr,
		}
	}

	// Now narrate the initial look.
	s.mu.Lock()
	s.loop = game.NewLoop(s.eng, s.interp, s.narr, nil)
	look := s.eng.Look(s.state)
	stateCopy := *s.state
	s.mu.Unlock()

	ctx := context.Background()
	event := model.EngineEvent{
		Type: "looked",
		Room: look.Name,
		Details: map[string]any{
			"description": look.Description,
			"exits":       look.Exits,
			"items":       look.Items,
		},
	}
	narration, err := s.narr.Narrate(ctx, event, stateCopy)
	if err != nil {
		// Fall back to the structured result if narration fails.
		data, _ := json.Marshal(stateCopy)
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  PlayResponse{Text: string(data)},
		}
	}

	s.mu.Lock()
	state := deepCopyState(s.state)
	s.mu.Unlock()

	return Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: PlayResponse{
			Text:  narration,
			State: state,
		},
	}
}
