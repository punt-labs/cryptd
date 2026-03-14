package daemon

import (
	"context"
	"encoding/json"

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

// handleNewGamePlay handles new_game in normal mode: starts the game via the
// dispatcher, then narrates the initial room as a PlayResponse. After this
// returns, handleConnection starts the game loop with RPCRenderer.
func (s *Server) handleNewGamePlay(req Request) Response {
	var params ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: CodeInvalidParams, Message: "invalid tool call params: " + err.Error()},
		}
	}

	_, rpcErr := s.dispatchNewGame(params.Arguments)
	if rpcErr != nil {
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   rpcErr,
		}
	}

	// Narrate the initial room.
	s.mu.Lock()
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
