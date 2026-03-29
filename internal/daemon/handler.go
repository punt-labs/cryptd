package daemon

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"

	"github.com/punt-labs/cryptd/internal/protocol"
)

// toolRegistry holds the static list of MCP tools for tools/list responses.
var toolRegistry = []ToolInfo{
	{Name: "new_game", Description: "Start a new game with the given scenario and character.", InputSchema: objectSchema(
		prop("scenario_id", "string", "Scenario identifier", true),
		prop("character_name", "string", "Hero name", true),
		prop("character_class", "string", "Hero class: fighter, mage, priest, thief", true),
	)},
	{Name: "move", Description: "Move the hero in a direction.", InputSchema: objectSchema(
		prop("direction", "string", "Direction: north, south, east, west, up, down", true),
	)},
	{Name: "look", Description: "Describe the current room.", InputSchema: objectSchema()},
	{Name: "pick_up", Description: "Pick up an item from the current room.", InputSchema: objectSchema(
		prop("item_id", "string", "Item identifier", true),
	)},
	{Name: "drop", Description: "Drop an item from inventory into the current room.", InputSchema: objectSchema(
		prop("item_id", "string", "Item identifier", true),
	)},
	{Name: "equip", Description: "Equip an item from inventory.", InputSchema: objectSchema(
		prop("item_id", "string", "Item identifier", true),
	)},
	{Name: "unequip", Description: "Unequip an item from an equipment slot.", InputSchema: objectSchema(
		prop("slot", "string", "Equipment slot: weapon, armor, ring, amulet", true),
	)},
	{Name: "examine", Description: "Examine an item in inventory, equipped, or in the room.", InputSchema: objectSchema(
		prop("item_id", "string", "Item identifier", true),
	)},
	{Name: "inventory", Description: "List the hero's inventory and equipment.", InputSchema: objectSchema()},
	{Name: "attack", Description: "Attack an enemy in combat.", InputSchema: objectSchema(
		prop("target_id", "string", "Enemy instance ID (default: first alive)", false),
	)},
	{Name: "defend", Description: "Raise guard to halve incoming damage for one round.", InputSchema: objectSchema()},
	{Name: "flee", Description: "Attempt to flee from combat (DEX check).", InputSchema: objectSchema()},
	{Name: "cast_spell", Description: "Cast a spell. Damage spells require combat; heal works anytime.", InputSchema: objectSchema(
		prop("spell_id", "string", "Spell identifier", true),
		prop("target_id", "string", "Target enemy ID (for damage spells)", false),
	)},
	{Name: "save_game", Description: "Save the current game state to a named slot.", InputSchema: objectSchema(
		prop("slot", "string", "Save slot name (default: quicksave)", false),
	)},
	{Name: "load_game", Description: "Load a saved game state from a named slot.", InputSchema: objectSchema(
		prop("slot", "string", "Save slot name (default: quicksave)", false),
	)},
}

// handleConnection reads NDJSON requests from r, processes them, and writes
// NDJSON responses to w. It runs until r is closed or an I/O error occurs.
//
// In normal mode, the handshake phase (initialize, tools/list, new_game)
// runs as request-response. After new_game, the game loop takes over via
// the game goroutine's RunLoop and blocks until the player quits or the
// connection closes.
//
// In passthrough mode, every request goes through the game goroutine's
// command channel — there is no game loop or RPCRenderer.
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

		// In normal mode, after a successful new_game OR a resumed session
		// with an active game, hand the connection to the game loop via the
		// game goroutine's RunLoop. The loop blocks until quit/death/EOF.
		if !s.passthrough && sess != nil {
			if s.isNewGameSuccess(req, resp) {
				s.runGameLoop(scanner, w, sess)
				return
			}
			// Session resume: if initialize found an existing game, send the
			// current room and enter the game loop.
			s.mu.RLock()
			hasGame := sess.gameID != ""
			s.mu.RUnlock()
			if req.Method == "initialize" && resp.Error == nil && hasGame {
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
	case "initialize":
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

		s.mu.RLock()
		hasGame := (*sess).gameID != ""
		s.mu.RUnlock()

		log.Printf("daemon: session %s established (has_game=%v)", sid, hasGame)

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

	case "tools/list":
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]any{"tools": toolRegistry},
		}

	case "tools/call":
		if !s.passthrough {
			var params ToolCallParams
			if err := json.Unmarshal(req.Params, &params); err != nil {
				return Response{
					JSONRPC: "2.0",
					ID:      req.ID,
					Error:   &RPCError{Code: CodeInvalidParams, Message: "invalid tool call params: " + err.Error()},
				}
			}
			if params.Name == "new_game" {
				return s.handleNewGamePlay(req, sess)
			}
			return Response{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &RPCError{Code: CodeMethodNotFound, Message: "only new_game is available before the game loop starts — use play for text input"},
			}
		}
		return s.handleToolCall(req, *sess)

	default:
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: CodeMethodNotFound, Message: fmt.Sprintf("unknown method %q", req.Method)},
		}
	}
}

// isNewGameSuccess checks if a request/response pair is a successful new_game.
func (s *Server) isNewGameSuccess(req Request, resp Response) bool {
	if resp.Error != nil {
		return false
	}
	if req.Method == "tools/call" {
		var params ToolCallParams
		if err := json.Unmarshal(req.Params, &params); err == nil {
			return params.Name == "new_game"
		}
	}
	return false
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
// that already has an active game. Enters RunLoop without sending new_game —
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

// handleToolCall processes a tools/call request by dispatching to the game goroutine.
func (s *Server) handleToolCall(req Request, sess *Session) Response {
	var params ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: CodeInvalidParams, Message: "invalid tool call params: " + err.Error()},
		}
	}

	// new_game is special: it creates a game if there isn't one.
	if params.Name == "new_game" {
		return s.handleNewGamePassthrough(req, params, sess)
	}

	if sess == nil {
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: CodeInvalidRequest, Message: "call initialize first"},
		}
	}

	g, rpcErr := s.gameForSession(sess)
	if rpcErr != nil {
		return s.toolErrorResponse(req.ID, rpcErr)
	}

	result, rpcErr := g.Send(s.ctx, params.Name, params.Arguments)
	if rpcErr != nil {
		return s.toolErrorResponse(req.ID, rpcErr)
	}

	return s.toolSuccessResponse(req.ID, result)
}

// handleNewGamePassthrough creates a game (if needed) and sends new_game to it.
func (s *Server) handleNewGamePassthrough(req Request, params ToolCallParams, sess *Session) Response {
	if sess == nil {
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: CodeInvalidRequest, Message: "call initialize first"},
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

	result, rpcErr := g.Send(s.ctx, "new_game", params.Arguments)
	if rpcErr != nil {
		s.removeGame(g.id)
		g.Stop()
		return s.toolErrorResponse(req.ID, rpcErr)
	}

	// Bind the session to this game.
	s.mu.Lock()
	sess.gameID = g.id
	s.mu.Unlock()

	return s.toolSuccessResponse(req.ID, result)
}

// toolErrorResponse wraps an RPCError as a ToolResult with isError=true.
func (s *Server) toolErrorResponse(id json.RawMessage, rpcErr *RPCError) Response {
	errJSON, merr := json.Marshal(rpcErr)
	errText := rpcErr.Message
	if merr == nil {
		errText = string(errJSON)
	}
	return Response{
		JSONRPC: "2.0",
		ID:      id,
		Result: ToolResult{
			IsError: true,
			Content: []ToolContent{{Type: "text", Text: errText}},
		},
	}
}

// toolSuccessResponse wraps a result as a ToolResult.
func (s *Server) toolSuccessResponse(id json.RawMessage, result any) Response {
	data, err := json.Marshal(result)
	if err != nil {
		return Response{
			JSONRPC: "2.0",
			ID:      id,
			Error:   &RPCError{Code: CodeInternalError, Message: "marshal result: " + err.Error()},
		}
	}
	return Response{
		JSONRPC: "2.0",
		ID:      id,
		Result: ToolResult{
			Content: []ToolContent{{Type: "text", Text: string(data)}},
		},
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

// --- schema builder helpers ---

type propDef struct {
	name     string
	typ      string
	desc     string
	required bool
}

func prop(name, typ, desc string, required bool) propDef {
	return propDef{name: name, typ: typ, desc: desc, required: required}
}

func objectSchema(props ...propDef) map[string]any {
	schema := map[string]any{
		"type": "object",
	}
	if len(props) == 0 {
		schema["properties"] = map[string]any{}
		return schema
	}
	properties := make(map[string]any, len(props))
	var required []string
	for _, p := range props {
		properties[p.name] = map[string]any{
			"type":        p.typ,
			"description": p.desc,
		}
		if p.required {
			required = append(required, p.name)
		}
	}
	schema["properties"] = properties
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}
