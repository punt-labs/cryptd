package daemon

import (
	"context"
	"encoding/json"

	"github.com/punt-labs/cryptd/internal/engine"
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

// handleNewGamePlay handles game.new in normal mode: creates a game, sends
// new_game to its goroutine, then narrates the initial room as a PlayResponse.
// After this returns, handleConnection starts the game loop via RunLoop.
func (s *Server) handleNewGamePlay(req Request, sess **Session) Response {
	// Ensure we have a session.
	if *sess == nil {
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: CodeInvalidRequest, Message: "call session.init first"},
		}
	}

	// Clean up any existing game for this session.
	var oldGame *Game
	s.mu.Lock()
	if (*sess).gameID != "" {
		if old, ok := s.games[(*sess).gameID]; ok {
			delete(s.games, (*sess).gameID)
			oldGame = old
		}
	}
	s.mu.Unlock()
	if oldGame != nil {
		go oldGame.Stop() // async: may block if RunLoop is active on another connection
	}

	g, err := s.createAndStartGame()
	if err != nil {
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: CodeInternalError, Message: err.Error()},
		}
	}

	_, rpcErr := g.Send(s.ctx, "new_game", req.Params)
	if rpcErr != nil {
		s.removeGame(g.id)
		g.Stop()
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   rpcErr,
		}
	}

	// Bind session to game.
	s.mu.Lock()
	(*sess).gameID = g.id
	s.mu.Unlock()

	// Get the look result and state copy from inside the game goroutine.
	var lookResult engine.LookResult
	var stateCopy model.GameState
	var stateForResp *model.GameState
	var nextLevelXP int

	if err := g.Inspect(s.ctx, func(eng *engine.Engine, state *model.GameState) {
		lookResult = eng.Look(state)
		stateForResp = deepCopyState(state)
		stateCopy = *stateForResp // value copy, no second JSON round-trip
		if len(state.Party) > 0 {
			nextLevelXP = engine.NextLevelXP(state.Party[0].Class, state.Party[0].Level)
		}
	}); err != nil {
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: CodeInternalError, Message: "game inspect failed: " + err.Error()},
		}
	}

	// Narrate the initial room.
	ctx := context.Background()
	event := model.EngineEvent{
		Type: "looked",
		Room: lookResult.Name,
		Details: map[string]any{
			"description": lookResult.Description,
			"exits":       lookResult.Exits,
			"items":       lookResult.Items,
		},
	}
	narration, narrErr := s.narr.Narrate(ctx, event, stateCopy)
	if narrErr != nil {
		fallback := lookResult.Description
		if fallback == "" {
			fallback = "You enter the dungeon."
		}
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: PlayResponse{
				Text:        fallback,
				State:       stateForResp,
				Exits:       lookResult.Exits,
				NextLevelXP: nextLevelXP,
			},
		}
	}

	return Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: PlayResponse{
			Text:        narration,
			State:       stateForResp,
			Exits:       lookResult.Exits,
			NextLevelXP: nextLevelXP,
		},
	}
}
