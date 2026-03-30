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

// LoadingMsg signals that a long-running operation (e.g., initial narration
// generation) has begun and the UI should show a loading indicator.
type LoadingMsg struct{}

// WelcomeMsg signals that no game is active and the UI should show a welcome
// message with instructions.
type WelcomeMsg struct{}

// ScenariosMsg arrives when game.list_scenarios completes.
type ScenariosMsg struct {
	Scenarios []protocol.ScenarioInfo
}

// SessionsMsg arrives when game.list_sessions completes.
type SessionsMsg struct {
	Sessions []protocol.SessionInfo
}

// StartCreationMsg signals a transition from lobby to game creation.
type StartCreationMsg struct {
	Scenarios []protocol.ScenarioInfo
}

// ResumeSessionMsg signals a transition from lobby to game with a saved session.
type ResumeSessionMsg struct {
	SessionID string
}

// CreationDoneMsg signals game creation is complete and the game should start.
type CreationDoneMsg struct {
	Scenario  string
	Name      string
	Class     string
}
