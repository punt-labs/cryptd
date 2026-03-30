package daemon

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/punt-labs/cryptd/internal/protocol"
	"github.com/punt-labs/cryptd/internal/scenariodir"
)

// handleConnection reads NDJSON requests from r, processes them, and writes
// NDJSON responses to w. It runs until r is closed or an I/O error occurs.
//
// Session mode is determined per-session during session.init: normal-mode
// sessions run a game loop after game.new (text play via interpreter + narrator),
// passthrough sessions dispatch game.* methods directly as request-response.
func (s *Server) handleConnection(r io.Reader, w io.Writer) {
	scanner := bufio.NewScanner(r)
	// Allow up to 1 MB per line (generous for JSON-RPC).
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var sess *Session

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			s.writeResponse(w, Response{
				JSONRPC: "2.0",
				ID:      nil,
				Error:   &RPCError{Code: CodeParseError, Message: "parse error: " + err.Error()},
			})
			continue
		}

		if req.JSONRPC != "2.0" {
			s.writeResponse(w, Response{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &RPCError{Code: CodeInvalidRequest, Message: "jsonrpc must be \"2.0\""},
			})
			continue
		}

		resp := s.safeHandleRequest(req, &sess)

		// JSON-RPC 2.0: requests without an ID are notifications — no response.
		if req.ID == nil {
			continue
		}
		s.writeResponse(w, resp)

		// In normal mode, after a successful game.new OR a resumed session
		// with an active game, hand the connection to the game loop via the
		// game goroutine's RunLoop. The loop blocks until quit/death/EOF.
		s.mu.RLock()
		isPassthrough := sess != nil && sess.passthrough
		s.mu.RUnlock()
		if sess != nil && !isPassthrough {
			if s.isNewGameSuccess(req, resp) {
				s.runGameLoop(scanner, w, sess)
				return
			}
			// Session resume: if session.init found an existing game, send the
			// current room and enter the game loop.
			s.mu.RLock()
			hasGame := sess.gameID != ""
			s.mu.RUnlock()
			if req.Method == "session.init" && resp.Error == nil && hasGame {
				s.resumeGameLoop(scanner, w, sess)
				return
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("daemon: connection read error: %v", err)
	}
}

// safeHandleRequest wraps handleRequest with panic recovery so one bad request
// does not crash the daemon or drop the connection.
func (s *Server) safeHandleRequest(req Request, sess **Session) (resp Response) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("daemon: panic handling %s: %v", req.Method, r)
			resp = Response{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &RPCError{Code: CodeInternalError, Message: "internal server error"},
			}
		}
	}()
	return s.handleRequest(req, sess)
}

// handleRequest routes a single JSON-RPC request to the appropriate handler.
func (s *Server) handleRequest(req Request, sess **Session) Response {
	switch req.Method {
	case "session.init":
		var params protocol.InitializeParams
		if len(req.Params) > 0 {
			// Best-effort parse; missing or malformed params are fine.
			_ = json.Unmarshal(req.Params, &params)
		}
		var sid string
		if id := sanitizeSessionID(params.SessionID); id != "" {
			sid = id
		} else {
			var err error
			sid, err = generateID()
			if err != nil {
				return Response{
					JSONRPC: "2.0",
					ID:      req.ID,
					Error:   &RPCError{Code: CodeInternalError, Message: err.Error()},
				}
			}
		}

		*sess = s.getOrCreateSession(sid)

		// Set session mode from params. A session that requests passthrough
		// stays in passthrough for its lifetime. Sessions without a mode
		// default to normal when the server has an interpreter+narrator,
		// or passthrough otherwise.
		s.mu.Lock()
		(*sess).passthrough = params.Mode == "passthrough" || !s.hasNormalMode()
		s.mu.Unlock()

		s.mu.RLock()
		hasGame := (*sess).gameID != ""
		s.mu.RUnlock()

		s.mu.RLock()
		pt := (*sess).passthrough
		s.mu.RUnlock()
		log.Printf("daemon: session %s established (has_game=%v, passthrough=%v)", sid, hasGame, pt)

		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: InitializeResult{
				ProtocolVersion: "2024-11-05",
				ServerInfo:      ServerInfo{Name: "cryptd", Version: "0.1.0"},
				Capabilities:    map[string]any{"tools": map[string]any{}},
				SessionID:       sid,
				HasGame:         hasGame,
			},
		}

	case "game.list_scenarios":
		return s.handleListScenarios(req)

	case "game.list_sessions":
		return s.handleListSessions(req)

	case "game.new":
		s.mu.RLock()
		pt := *sess != nil && (*sess).passthrough
		s.mu.RUnlock()
		if *sess != nil && !pt {
			return s.handleNewGamePlay(req, sess)
		}
		return s.handleNewGamePassthrough(req, *sess)

	default:
		// Route game.* methods to the game goroutine in passthrough mode.
		s.mu.RLock()
		isPT := *sess != nil && (*sess).passthrough
		s.mu.RUnlock()
		if isPT && strings.HasPrefix(req.Method, "game.") {
			if req.Method == "game.play" {
				return Response{
					JSONRPC: "2.0",
					ID:      req.ID,
					Error:   &RPCError{Code: CodeMethodNotFound, Message: fmt.Sprintf("unknown method %q", req.Method)},
				}
			}
			return s.handleGameCommand(req, *sess)
		}
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: CodeMethodNotFound, Message: fmt.Sprintf("unknown method %q", req.Method)},
		}
	}
}

// isNewGameSuccess checks if a request/response pair is a successful game.new.
func (s *Server) isNewGameSuccess(req Request, resp Response) bool {
	return req.Method == "game.new" && resp.Error == nil
}

// runGameLoop sends a RunLoop command to the game goroutine, which runs the
// game loop internally. The loop drives the RPCRenderer: Render() writes
// PlayResponse NDJSON, Events() reads play requests. Blocks until quit/death/EOF.
func (s *Server) runGameLoop(scanner *bufio.Scanner, w io.Writer, sess *Session) {
	g, rpcErr := s.gameForSession(sess)
	if rpcErr != nil {
		log.Printf("daemon: runGameLoop: %s", rpcErr.Message)
		return
	}

	if rpcErr = g.RunLoop(s.ctx, &RunLoopRequest{
		Scanner:           scanner,
		Writer:            w,
		Interp:            s.interp,
		Narr:              s.narr,
		SkipInitialRender: true, // handleNewGamePlay already sent the initial room
	}); rpcErr != nil {
		log.Printf("daemon: game loop error: %s", rpcErr.Message)
	}
}

// resumeGameLoop resumes a normal-mode game loop for a reconnected session
// that already has an active game. Enters RunLoop without sending game.new —
// the game state is already initialized from the previous connection.
func (s *Server) resumeGameLoop(scanner *bufio.Scanner, w io.Writer, sess *Session) {
	g, rpcErr := s.gameForSession(sess)
	if rpcErr != nil {
		log.Printf("daemon: resumeGameLoop: %s", rpcErr.Message)
		return
	}

	log.Printf("daemon: resuming game loop for session %s", sess.id)

	if rpcErr = g.RunLoop(s.ctx, &RunLoopRequest{
		Scanner: scanner,
		Writer:  w,
		Interp:  s.interp,
		Narr:    s.narr,
	}); rpcErr != nil {
		log.Printf("daemon: resume game loop error: %s", rpcErr.Message)
	}
}

// handleGameCommand dispatches a game.* method to the game goroutine (passthrough mode).
func (s *Server) handleGameCommand(req Request, sess *Session) Response {
	if sess == nil {
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: CodeInvalidRequest, Message: "call session.init first"},
		}
	}

	g, rpcErr := s.gameForSession(sess)
	if rpcErr != nil {
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   rpcErr,
		}
	}

	// Strip "game." prefix to get the command name for the engine dispatcher.
	cmdName := strings.TrimPrefix(req.Method, "game.")

	result, rpcErr := g.Send(s.ctx, cmdName, req.Params)
	if rpcErr != nil {
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   rpcErr,
		}
	}

	return Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

// handleNewGamePassthrough creates a game (if needed) and sends new_game to it.
func (s *Server) handleNewGamePassthrough(req Request, sess *Session) Response {
	if sess == nil {
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: CodeInvalidRequest, Message: "call session.init first"},
		}
	}

	// Clean up any existing game for this session.
	var oldGame *Game
	s.mu.Lock()
	if sess.gameID != "" {
		if old, ok := s.games[sess.gameID]; ok {
			delete(s.games, sess.gameID)
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

	result, rpcErr := g.Send(s.ctx, "new_game", req.Params)
	if rpcErr != nil {
		s.removeGame(g.id)
		g.Stop()
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   rpcErr,
		}
	}

	// Bind the session to this game.
	s.mu.Lock()
	sess.gameID = g.id
	s.mu.Unlock()

	s.snapshotSessionMeta(sess, g)

	return Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

// handleListScenarios returns the available scenarios from the scenario directory.
func (s *Server) handleListScenarios(req Request) Response {
	entries, err := scenariodir.ListScenarios(s.scenarioDir)
	if err != nil {
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: CodeInternalError, Message: err.Error()},
		}
	}

	scenarios := make([]protocol.ScenarioInfo, len(entries))
	for i, e := range entries {
		scenarios[i] = protocol.ScenarioInfo{
			ID:          e.ID,
			Title:       e.Title,
			Description: e.Description,
		}
	}

	return Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  ListScenariosResult{Scenarios: scenarios},
	}
}

// writeResponse serialises a Response as a single NDJSON line.
func (s *Server) writeResponse(w io.Writer, resp Response) {
	data, err := json.Marshal(resp)
	if err != nil {
		log.Printf("daemon: marshal response: %v", err)
		return
	}
	data = append(data, '\n')
	if _, err := w.Write(data); err != nil {
		log.Printf("daemon: write response: %v", err)
	}
}
