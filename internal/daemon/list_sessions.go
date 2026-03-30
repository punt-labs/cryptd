package daemon

import (
	"github.com/punt-labs/cryptd/internal/engine"
	"github.com/punt-labs/cryptd/internal/model"
)

// handleListSessions returns metadata for all sessions that have an active game.
func (s *Server) handleListSessions(req Request) Response {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var infos []SessionInfo
	for _, sess := range s.sessions {
		if sess.gameID == "" {
			continue
		}
		infos = append(infos, SessionInfo{
			ID:             sess.id,
			ScenarioID:     sess.scenarioID,
			CharacterName:  sess.characterName,
			CharacterClass: sess.characterClass,
			Level:          sess.level,
			RoomName:       sess.roomName,
		})
	}

	// Return empty slice, not nil, so JSON encodes as [].
	if infos == nil {
		infos = []SessionInfo{}
	}

	return Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  ListSessionsResult{Sessions: infos},
	}
}

// snapshotSessionMeta reads game state via Inspect and caches key metadata
// on the session. Called after a successful game.new, before the game loop starts.
func (s *Server) snapshotSessionMeta(sess *Session, g *Game) {
	_ = g.Inspect(s.ctx, func(_ *engine.Engine, state *model.GameState) {
		if state == nil {
			return
		}
		var charName, charClass string
		var level int
		if len(state.Party) > 0 {
			charName = state.Party[0].Name
			charClass = state.Party[0].Class
			level = state.Party[0].Level
		}
		s.mu.Lock()
		sess.updateMeta(state.Scenario, charName, charClass, state.Dungeon.CurrentRoom, level)
		s.mu.Unlock()
	})
}
