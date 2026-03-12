package daemon

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
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
func (s *Server) handleConnection(r io.Reader, w io.Writer) {
	scanner := bufio.NewScanner(r)
	// Allow up to 1 MB per line (generous for JSON-RPC).
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

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

		resp := s.safeHandleRequest(req)

		// JSON-RPC 2.0: requests without an ID are notifications — no response.
		if req.ID == nil {
			continue
		}
		s.writeResponse(w, resp)
	}

	if err := scanner.Err(); err != nil {
		log.Printf("daemon: connection read error: %v", err)
	}
}

// safeHandleRequest wraps handleRequest with panic recovery so one bad request
// does not crash the daemon or drop the connection.
func (s *Server) safeHandleRequest(req Request) (resp Response) {
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
	return s.handleRequest(req)
}

// handleRequest routes a single JSON-RPC request to the appropriate handler.
func (s *Server) handleRequest(req Request) Response {
	switch req.Method {
	case "initialize":
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: InitializeResult{
				ProtocolVersion: "2024-11-05",
				ServerInfo:      ServerInfo{Name: "cryptd", Version: "0.1.0"},
				Capabilities:    map[string]any{"tools": map[string]any{}},
			},
		}

	case "tools/list":
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]any{"tools": toolRegistry},
		}

	case "tools/call":
		return s.handleToolCall(req)

	default:
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: CodeMethodNotFound, Message: fmt.Sprintf("unknown method %q", req.Method)},
		}
	}
}

// handleToolCall processes a tools/call request by dispatching to the engine.
func (s *Server) handleToolCall(req Request) Response {
	var params ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: CodeInvalidParams, Message: "invalid tool call params: " + err.Error()},
		}
	}

	result, rpcErr := s.dispatch(params.Name, params.Arguments)
	if rpcErr != nil {
		// MCP convention: tool execution errors are returned as ToolResult
		// with isError=true (not JSON-RPC errors). This lets MCP clients
		// distinguish protocol failures from game-logic errors.
		// Serialize the full RPCError so clients can use error codes.
		errJSON, merr := json.Marshal(rpcErr)
		errText := rpcErr.Message
		if merr == nil {
			errText = string(errJSON)
		}
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: ToolResult{
				IsError: true,
				Content: []ToolContent{{Type: "text", Text: errText}},
			},
		}
	}

	// Marshal the result to JSON text for the MCP content array.
	data, err := json.Marshal(result)
	if err != nil {
		return Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: CodeInternalError, Message: "marshal result: " + err.Error()},
		}
	}

	return Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: ToolResult{
			Content: []ToolContent{{Type: "text", Text: string(data)}},
		},
	}
}

// writeResponse serialises a Response as a single NDJSON line.
// Not concurrency-safe: callers must serialize writes or add a per-connection mutex (M10).
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
