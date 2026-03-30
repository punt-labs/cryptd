package tui

import "github.com/punt-labs/cryptd/internal/protocol"

// ServerResponseMsg arrives when a JSON-RPC call completes successfully.
type ServerResponseMsg struct {
	Response protocol.PlayResponse
}

// ServerErrMsg arrives when the server returns an error.
type ServerErrMsg struct {
	Err error
}

// ConnLostMsg arrives when the server connection drops.
type ConnLostMsg struct {
	Err error
}

// SendCmdMsg is dispatched by InputBar or combat shortcuts to trigger a play command.
type SendCmdMsg struct {
	Text string
}

// GameStartMsg arrives after a successful game.new response.
type GameStartMsg struct {
	Response protocol.PlayResponse
}
