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

// ListScenariosCmd returns a tea.Cmd that sends game.list_scenarios.
func ListScenariosCmd(send SendFn) tea.Cmd {
	return func() tea.Msg {
		result, err := send("game.list_scenarios", nil)
		if err != nil {
			if errors.Is(err, ErrConnLost) {
				return ConnLostMsg{Err: err}
			}
			return ServerErrMsg{Err: err}
		}
		var resp protocol.ListScenariosResult
		if err := json.Unmarshal(result, &resp); err != nil {
			return ServerErrMsg{Err: fmt.Errorf("parse list_scenarios response: %w", err)}
		}
		return ScenariosMsg{Scenarios: resp.Scenarios}
	}
}

// ListSessionsCmd returns a tea.Cmd that sends game.list_sessions.
func ListSessionsCmd(send SendFn) tea.Cmd {
	return func() tea.Msg {
		result, err := send("game.list_sessions", nil)
		if err != nil {
			if errors.Is(err, ErrConnLost) {
				return ConnLostMsg{Err: err}
			}
			return ServerErrMsg{Err: err}
		}
		var resp protocol.ListSessionsResult
		if err := json.Unmarshal(result, &resp); err != nil {
			return ServerErrMsg{Err: fmt.Errorf("parse list_sessions response: %w", err)}
		}
		return SessionsMsg{Sessions: resp.Sessions}
	}
}

