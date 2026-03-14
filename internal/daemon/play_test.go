package daemon

import (
	"bufio"
	"encoding/json"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/punt-labs/cryptd/internal/interpreter"
	"github.com/punt-labs/cryptd/internal/narrator"
	"github.com/punt-labs/cryptd/internal/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testNormalServer creates a normal-mode Server with Rules interpreter and
// Template narrator (no SLM/inference dependency).
func testNormalServer(t *testing.T) *Server {
	t.Helper()
	scenarioDir := filepath.Join(repoRoot(t), "testdata", "scenarios")
	rules := interpreter.NewRules()
	tmpl := narrator.NewTemplate()
	interp := interpreter.NewSLM(nil, rules)
	narr := narrator.NewSLM(nil, tmpl)
	return NewServer(
		filepath.Join(t.TempDir(), "test.sock"),
		scenarioDir,
		WithInterpreter(interp),
		WithNarrator(narr),
	)
}

func TestNormalMode_NewGameAndPlay(t *testing.T) {
	srv := testNormalServer(t)

	// Simulate a client connection with a pipe.
	clientR, serverW := io.Pipe()
	serverR, clientW := io.Pipe()
	defer clientR.Close()
	defer clientW.Close()

	// Run handleConnection in a goroutine.
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer serverW.Close()
		defer serverR.Close()
		srv.handleConnection(serverR, serverW)
	}()

	clientScanner := bufio.NewScanner(clientR)
	clientScanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	// Helper: send a request and read the response.
	send := func(method string, params any) protocol.Response {
		t.Helper()
		p, _ := json.Marshal(params)
		req := protocol.Request{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: method, Params: p}
		data, _ := json.Marshal(req)
		data = append(data, '\n')
		_, err := clientW.Write(data)
		require.NoError(t, err)

		require.True(t, clientScanner.Scan(), "expected response")
		var resp protocol.Response
		require.NoError(t, json.Unmarshal(clientScanner.Bytes(), &resp))
		return resp
	}

	// Initialize.
	resp := send("initialize", map[string]any{})
	require.Nil(t, resp.Error)

	// New game — returns PlayResponse with narrated room.
	resp = send("tools/call", map[string]any{
		"name": "new_game",
		"arguments": map[string]any{
			"scenario_id":     "minimal",
			"character_name":  "Tester",
			"character_class": "fighter",
		},
	})
	require.Nil(t, resp.Error, "new_game error: %+v", resp.Error)

	var playResp protocol.PlayResponse
	data, err := json.Marshal(resp.Result)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &playResp))
	assert.NotEmpty(t, playResp.Text, "expected narrated text from new_game")
	require.NotNil(t, playResp.State, "expected game state from new_game")
	assert.NotEmpty(t, playResp.State.Party, "expected party in game state")

	// After new_game, the game loop is running. Send a play request.
	resp = send("play", protocol.PlayRequest{Text: "look"})
	require.Nil(t, resp.Error, "play look error: %+v", resp.Error)

	data, err = json.Marshal(resp.Result)
	require.NoError(t, err)
	var lookResp protocol.PlayResponse
	require.NoError(t, json.Unmarshal(data, &lookResp))
	assert.NotEmpty(t, lookResp.Text, "expected narrated text from look")

	// Send quit to end the game loop.
	quitReq := protocol.Request{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: "quit"}
	quitData, _ := json.Marshal(quitReq)
	quitData = append(quitData, '\n')
	_, err = clientW.Write(quitData)
	require.NoError(t, err)

	// Read the quit response (game loop renders the farewell).
	require.True(t, clientScanner.Scan(), "expected quit response")
	var quitResp protocol.Response
	require.NoError(t, json.Unmarshal(clientScanner.Bytes(), &quitResp))

	// Wait for handleConnection to finish.
	clientW.Close()
	<-done
}

func TestNormalMode_PlayBeforeNewGame(t *testing.T) {
	srv := testNormalServer(t)

	// Before new_game, "play" method is unknown (game loop not started).
	resp := roundTrip(t, srv, Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "play",
		Params:  json.RawMessage(`{"text":"look"}`),
	})
	require.NotNil(t, resp.Error)
	assert.Equal(t, CodeMethodNotFound, resp.Error.Code)
}

func TestNormalMode_NonNewGameToolCallBlocked(t *testing.T) {
	srv := testNormalServer(t)

	// In normal mode, non-new_game tool calls are blocked.
	resp := roundTrip(t, srv, toolCall(1, "look", map[string]any{}))
	require.NotNil(t, resp.Error)
	assert.Equal(t, CodeMethodNotFound, resp.Error.Code)
	assert.True(t, strings.Contains(resp.Error.Message, "new_game"), resp.Error.Message)
}
