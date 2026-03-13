// Package daemon implements a JSON-RPC 2.0 server over Unix sockets,
// exposing the game engine's 15 MCP tools as RPC methods.
package daemon

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
	Text  string          `json:"text"`
	State *model.GameState `json:"state,omitempty"`
	Dead  bool            `json:"dead,omitempty"`
	Quit  bool            `json:"quit,omitempty"`
}

// MCP tool call/result types (subset of MCP spec used by tools/call).

// ToolCallParams is the params object for a tools/call request.
type ToolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// ToolResult is the result object for a tools/call response.
type ToolResult struct {
	Content []ToolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// ToolContent is one element in a tool result's content array.
type ToolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ToolInfo describes one tool for tools/list.
type ToolInfo struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

// InitializeResult is the result of the initialize method.
type InitializeResult struct {
	ProtocolVersion string         `json:"protocolVersion"`
	ServerInfo      ServerInfo     `json:"serverInfo"`
	Capabilities    map[string]any `json:"capabilities"`
}

// ServerInfo identifies the server in the initialize handshake.
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}
