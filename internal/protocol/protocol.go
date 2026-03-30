// Package protocol defines the JSON-RPC 2.0 wire types shared between
// the cryptd server and crypt client.
package protocol

import (
	"encoding/json"

	"github.com/punt-labs/cryptd/internal/model"
)

// JSON-RPC 2.0 wire types.

// Request is a JSON-RPC 2.0 request.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response is a JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError is a JSON-RPC 2.0 error object.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Standard JSON-RPC error codes.
const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603
)

// Application-specific error codes.
const (
	CodeStateBlocked = -32001 // combat/turn/lock constraints
	CodeGameOver     = -32002 // hero is dead
	CodeNoActiveGame = -32003 // tool called before new_game
)

// PlayRequest is the JSON-RPC params for the "play" method.
type PlayRequest struct {
	Text string `json:"text"`
}

// PlayResponse is the JSON-RPC result for the "play" method.
type PlayResponse struct {
	Text        string           `json:"text"`
	State       *model.GameState `json:"state,omitempty"`
	Dead        bool             `json:"dead,omitempty"`
	Quit        bool             `json:"quit,omitempty"`
	Exits       []string         `json:"exits,omitempty"`        // transient: available exits from current room
	NextLevelXP int              `json:"next_level_xp,omitempty"` // transient: XP needed for next level
}

// InitializeParams is the optional params for the "session.init" method.
type InitializeParams struct {
	SessionID string `json:"session_id,omitempty"`
	Mode      string `json:"mode,omitempty"` // "passthrough" or "" (normal)
}

// InitializeResult is the result of the initialize method.
type InitializeResult struct {
	ProtocolVersion string         `json:"protocolVersion"`
	ServerInfo      ServerInfo     `json:"serverInfo"`
	Capabilities    map[string]any `json:"capabilities"`
	SessionID       string         `json:"session_id"`
	HasGame         bool           `json:"has_game,omitempty"`
}

// ServerInfo identifies the server in the initialize handshake.
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ScenarioInfo describes a single available scenario for the client.
type ScenarioInfo struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

// ListScenariosResult is the result of the game.list_scenarios method.
type ListScenariosResult struct {
	Scenarios []ScenarioInfo `json:"scenarios"`
}

// SessionInfo describes one active game session for the lobby screen.
type SessionInfo struct {
	ID             string `json:"id"`
	ScenarioID     string `json:"scenario_id,omitempty"`
	CharacterName  string `json:"character_name,omitempty"`
	CharacterClass string `json:"character_class,omitempty"`
	Level          int    `json:"level,omitempty"`
	RoomName       string `json:"room_name,omitempty"`
}

// ListSessionsResult is the result of the game.list_sessions method.
type ListSessionsResult struct {
	Sessions []SessionInfo `json:"sessions"`
}
