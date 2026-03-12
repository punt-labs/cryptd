package daemon

import (
	"context"
	"encoding/json"

	"github.com/punt-labs/cryptd/internal/game"
	"github.com/punt-labs/cryptd/internal/model"
)

// playParams is the JSON-RPC params for the "play" method.
type playParams struct {
	Text string `json:"text"`
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

	var params playParams
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
			Result:  map[string]any{"text": "I don't understand that command."},
		}
	}

	if action.Type == "quit" {
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]any{"text": "Farewell, adventurer.", "quit": true},
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

	result := map[string]any{"text": narration}

	// Check for terminal events.
	for _, ev := range events {
		if ev.Type == "hero_died" {
			result["dead"] = true
		}
	}

	// Include hero status for the prompt.
	result["hero"] = heroSummary(s.state)

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
	result, rpcErr := s.dispatchNewGame(req.Params)
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
	narration, err := s.narr.Narrate(ctx, event, *s.state)
	if err != nil {
		// Fall back to the structured result if narration fails.
		data, _ := json.Marshal(result)
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]any{"text": string(data)},
		}
	}

	return Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]any{
			"text": narration,
			"hero": heroSummary(s.state),
		},
	}
}
