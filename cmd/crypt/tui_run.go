package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/punt-labs/cryptd/cmd/crypt/tui"
	"github.com/punt-labs/cryptd/internal/protocol"
)

func runTUI(socketPath, addr, scenario, charName, charClass, sessionID string) int {
	conn, err := dial(socketPath, addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	defer func() { _ = conn.Close() }()

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	reqID := 0

	var mu sync.Mutex
	send := tui.SendFn(func(method string, params any) (json.RawMessage, error) {
		mu.Lock()
		defer mu.Unlock()
		reqID++
		req := map[string]any{
			"jsonrpc": "2.0",
			"id":      reqID,
			"method":  method,
		}
		if params != nil {
			p, err := json.Marshal(params)
			if err != nil {
				return nil, fmt.Errorf("marshal params: %w", err)
			}
			req["params"] = json.RawMessage(p)
		}
		data, err := json.Marshal(req)
		if err != nil {
			return nil, err
		}
		data = append(data, '\n')
		if _, err := conn.Write(data); err != nil {
			return nil, fmt.Errorf("%w: write: %w", tui.ErrConnLost, err)
		}
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return nil, fmt.Errorf("%w: read: %w", tui.ErrConnLost, err)
			}
			return nil, tui.ErrConnLost
		}
		var resp struct {
			Result json.RawMessage    `json:"result"`
			Error  *protocol.RPCError `json:"error"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
			return nil, fmt.Errorf("parse response: %w", err)
		}
		if resp.Error != nil {
			return nil, fmt.Errorf("server error: %s", resp.Error.Message)
		}
		return resp.Result, nil
	})

	// Handshake: session.init before starting Bubble Tea.
	var initParams any
	if sessionID != "" {
		initParams = protocol.InitializeParams{SessionID: sessionID}
	}
	initResult, err := send("session.init", initParams)
	if err != nil {
		fmt.Fprintf(os.Stderr, "session init failed: %v\n", err)
		return 1
	}
	var initResp protocol.InitializeResult
	if err := json.Unmarshal(initResult, &initResp); err != nil {
		fmt.Fprintf(os.Stderr, "parse init response: %v\n", err)
		return 1
	}

	// If resuming an existing game, consume the unsolicited initial PlayResponse.
	var initialResp *protocol.PlayResponse
	if initResp.HasGame {
		if !scanner.Scan() {
			fmt.Fprintf(os.Stderr, "connection lost during resume\n")
			return 1
		}
		var envelope struct {
			Result json.RawMessage `json:"result"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &envelope); err == nil && envelope.Result != nil {
			var pr protocol.PlayResponse
			if err := json.Unmarshal(envelope.Result, &pr); err == nil {
				initialResp = &pr
			}
		}
	}

	app := tui.NewApp(send, initResp.SessionID, scenario, charName, charClass, initialResp)
	p := tea.NewProgram(app, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	return 0
}
