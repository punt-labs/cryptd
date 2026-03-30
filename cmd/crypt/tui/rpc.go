package tui

import (
	"encoding/json"
	"errors"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/punt-labs/cryptd/internal/protocol"
)

// SendFn is the JSON-RPC send function signature, matching the closure
// created in cmd/crypt/connect.go session().
type SendFn func(method string, params any) (json.RawMessage, error)

// ErrConnLost is the sentinel for a dropped server connection.
var ErrConnLost = errors.New("connection lost")

// PlayCmd returns a tea.Cmd that sends game.play with the given text.
func PlayCmd(send SendFn, text string) tea.Cmd {
	return func() tea.Msg {
		result, err := send("game.play", protocol.PlayRequest{Text: text})
		if err != nil {
			if errors.Is(err, ErrConnLost) {
				return ConnLostMsg{Err: err}
			}
			return ServerErrMsg{Err: err}
		}
		var resp protocol.PlayResponse
		if err := json.Unmarshal(result, &resp); err != nil {
			return ServerErrMsg{Err: fmt.Errorf("parse play response: %w", err)}
		}
		return ServerResponseMsg{Response: resp}
	}
}

// NewGameCmd returns a tea.Cmd that sends game.new.
func NewGameCmd(send SendFn, scenario, name, class string) tea.Cmd {
	return func() tea.Msg {
		result, err := send("game.new", map[string]string{
			"scenario_id":     scenario,
			"character_name":  name,
			"character_class": class,
		})
		if err != nil {
			if errors.Is(err, ErrConnLost) {
				return ConnLostMsg{Err: err}
			}
			return ServerErrMsg{Err: err}
		}
		var resp protocol.PlayResponse
		if err := json.Unmarshal(result, &resp); err != nil {
			return ServerErrMsg{Err: fmt.Errorf("parse new game response: %w", err)}
		}
		return GameStartMsg{Response: resp}
	}
}

// InitCmd returns a tea.Cmd that sends session.init.
func InitCmd(send SendFn, sessionID string) tea.Cmd {
	return func() tea.Msg {
		var params any
		if sessionID != "" {
			params = protocol.InitializeParams{SessionID: sessionID}
		}
		result, err := send("session.init", params)
		if err != nil {
			if errors.Is(err, ErrConnLost) {
				return ConnLostMsg{Err: err}
			}
			return ServerErrMsg{Err: err}
		}
		var resp protocol.InitializeResult
		if err := json.Unmarshal(result, &resp); err != nil {
			return ServerErrMsg{Err: fmt.Errorf("parse init response: %w", err)}
		}
		return SessionReadyMsg{
			SessionID: resp.SessionID,
			HasGame:   resp.HasGame,
		}
	}
}
