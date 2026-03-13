// Package daemon implements a JSON-RPC 2.0 server over Unix sockets,
// exposing the game engine's 15 MCP tools as RPC methods.
//
// Wire protocol types live in internal/protocol. This file re-exports
// them so existing daemon code compiles without import changes.
package daemon

import "github.com/punt-labs/cryptd/internal/protocol"

// Re-export wire protocol types. Daemon-internal code uses these directly;
// external consumers (cmd/crypt) import internal/protocol instead.
type (
	Request          = protocol.Request
	Response         = protocol.Response
	RPCError         = protocol.RPCError
	PlayRequest      = protocol.PlayRequest
	PlayResponse     = protocol.PlayResponse
	ToolCallParams   = protocol.ToolCallParams
	ToolResult       = protocol.ToolResult
	ToolContent      = protocol.ToolContent
	ToolInfo         = protocol.ToolInfo
	InitializeResult = protocol.InitializeResult
	ServerInfo       = protocol.ServerInfo
)

// DefaultSocketPath re-exports protocol.DefaultSocketPath.
var DefaultSocketPath = protocol.DefaultSocketPath

// Re-export error code constants.
const (
	CodeParseError     = protocol.CodeParseError
	CodeInvalidRequest = protocol.CodeInvalidRequest
	CodeMethodNotFound = protocol.CodeMethodNotFound
	CodeInvalidParams  = protocol.CodeInvalidParams
	CodeInternalError  = protocol.CodeInternalError
	CodeStateBlocked   = protocol.CodeStateBlocked
	CodeGameOver       = protocol.CodeGameOver
	CodeNoActiveGame   = protocol.CodeNoActiveGame
)
