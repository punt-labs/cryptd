package daemon

import (
	"bufio"
	"encoding/json"
	"io"
	"testing"

	"github.com/punt-labs/cryptd/internal/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// standardSetup returns the init + game.new requests used by most contract tests.
func standardSetup() []Request {
	return []Request{
		initRequest(0),
		newGameCall(1, map[string]any{
			"scenario_id":     "minimal",
			"character_name":  "Tester",
			"character_class": "fighter",
		}),
	}
}

// setupAndCall sends init + game.new, then appends the given game call and
// returns all responses. Asserts setup succeeded.
func setupAndCall(t *testing.T, srv *Server, call Request) []Response {
	t.Helper()
	reqs := append(standardSetup(), call)
	resps := multiRoundTrip(t, srv, reqs)
	require.Len(t, resps, 3)
	require.Nil(t, resps[0].Error, "session.init failed")
	require.Nil(t, resps[1].Error, "game.new failed")
	return resps
}

func TestProtocol_DirectMethodRouting(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		params   any
		wantKeys []string // keys expected in the result map
	}{
		{
			name:     "game.move returns room",
			method:   "move",
			params:   map[string]any{"direction": "south"},
			wantKeys: []string{"room"},
		},
		{
			name:     "game.look returns room description and exits",
			method:   "look",
			params:   nil,
			wantKeys: []string{"room", "description", "exits"},
		},
		{
			name:     "game.inventory returns items and weight",
			method:   "inventory",
			params:   nil,
			wantKeys: []string{"items", "weight"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := testServer(t)
			resps := setupAndCall(t, srv, gameCall(2, tt.method, tt.params))
			result := extractResult(t, resps[2])
			for _, key := range tt.wantKeys {
				assert.Contains(t, result, key, "result missing key %q", key)
			}
		})
	}
}

func TestProtocol_ResponseFormatIsDirect(t *testing.T) {
	srv := testServer(t)
	resps := setupAndCall(t, srv, gameCall(2, "look", nil))

	result := extractResult(t, resps[2])

	// Direct JSON: has game-specific keys.
	assert.Contains(t, result, "room", "result should have 'room' key")

	// NOT MCP ToolResult format: must not have 'content' or 'isError' wrapper.
	assert.NotContains(t, result, "content", "result must not be MCP-wrapped (has 'content' key)")
	assert.NotContains(t, result, "isError", "result must not be MCP-wrapped (has 'isError' key)")
}

func TestProtocol_ErrorsAreJSONRPC(t *testing.T) {
	t.Run("invalid direction", func(t *testing.T) {
		srv := testServer(t)
		resps := setupAndCall(t, srv, gameCall(2, "move", map[string]any{"direction": "nowhere"}))

		resp := resps[2]
		require.NotNil(t, resp.Error, "expected JSON-RPC error for invalid direction")
		assert.Equal(t, CodeInvalidParams, resp.Error.Code, "expected CodeInvalidParams for invalid direction")
		assert.NotEmpty(t, resp.Error.Message, "error must have a Message")
		assert.Nil(t, resp.Result, "result must be nil when error is set")
	})

	t.Run("attack outside combat", func(t *testing.T) {
		srv := testServer(t)
		resps := setupAndCall(t, srv, gameCall(2, "attack", map[string]any{"target": "goblin"}))

		resp := resps[2]
		require.NotNil(t, resp.Error, "expected JSON-RPC error for attack outside combat")
		assert.Equal(t, CodeStateBlocked, resp.Error.Code, "expected CodeStateBlocked")
		assert.Nil(t, resp.Result, "result must be nil when error is set")
	})
}

func TestProtocol_UnknownGameMethod(t *testing.T) {
	srv := testServer(t)
	resps := setupAndCall(t, srv, gameCall(2, "nonexistent", nil))

	resp := resps[2]
	require.NotNil(t, resp.Error)
	assert.Equal(t, CodeMethodNotFound, resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "unknown command")
	assert.Nil(t, resp.Result)
}

func TestProtocol_GamePlayRejectedInPassthrough(t *testing.T) {
	srv := testServer(t) // testServer uses WithPassthrough()

	params, _ := json.Marshal(map[string]any{"text": "look around"})
	idJSON, _ := json.Marshal(2)
	playReq := Request{
		JSONRPC: "2.0",
		ID:      idJSON,
		Method:  "game.play",
		Params:  params,
	}

	reqs := append(standardSetup(), playReq)
	resps := multiRoundTrip(t, srv, reqs)
	require.Len(t, resps, 3)

	resp := resps[2]
	require.NotNil(t, resp.Error, "game.play must be rejected in passthrough mode")
	assert.Equal(t, CodeMethodNotFound, resp.Error.Code)
}

func TestProtocol_SessionInitRequired(t *testing.T) {
	srv := testServer(t)

	// Send game.new without session.init.
	resp := roundTrip(t, srv, newGameCall(1, map[string]any{
		"scenario_id":     "minimal",
		"character_name":  "Tester",
		"character_class": "fighter",
	}))

	require.NotNil(t, resp.Error)
	assert.Equal(t, CodeInvalidRequest, resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "call session.init first")
	assert.Nil(t, resp.Result)
}

func TestProtocol_SessionQuit(t *testing.T) {
	srv := testNormalServer(t)

	// Use pipes so the game loop can block waiting for input.
	clientR, serverW := io.Pipe()
	serverR, clientW := io.Pipe()
	defer func() { _ = clientR.Close() }()
	defer func() { _ = clientW.Close() }()

	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() { _ = serverW.Close() }()
		defer func() { _ = serverR.Close() }()
		srv.handleConnection(serverR, serverW)
	}()

	clientScanner := bufio.NewScanner(clientR)
	clientScanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	send := func(method string, params any) protocol.Response {
		t.Helper()
		p, err := json.Marshal(params)
		require.NoError(t, err)
		req := protocol.Request{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: method, Params: p}
		data, err := json.Marshal(req)
		require.NoError(t, err)
		data = append(data, '\n')
		_, err = clientW.Write(data)
		require.NoError(t, err)

		require.True(t, clientScanner.Scan(), "expected response for %s", method)
		var resp protocol.Response
		require.NoError(t, json.Unmarshal(clientScanner.Bytes(), &resp))
		return resp
	}

	// Handshake.
	resp := send("session.init", map[string]any{})
	require.Nil(t, resp.Error)

	// Start game — enters the game loop.
	resp = send("game.new", map[string]any{
		"scenario_id":     "minimal",
		"character_name":  "Tester",
		"character_class": "fighter",
	})
	require.Nil(t, resp.Error, "game.new error: %+v", resp.Error)

	// Send session.quit to end the game loop.
	quitReq := protocol.Request{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: "session.quit"}
	quitData, err := json.Marshal(quitReq)
	require.NoError(t, err)
	quitData = append(quitData, '\n')
	_, err = clientW.Write(quitData)
	require.NoError(t, err)

	// Read the quit narration response.
	require.True(t, clientScanner.Scan(), "expected quit narration response")
	var quitNarrResp protocol.Response
	require.NoError(t, json.Unmarshal(clientScanner.Bytes(), &quitNarrResp))

	// Read the final terminal response with Quit flag.
	require.True(t, clientScanner.Scan(), "expected final terminal response")
	var finalResp protocol.Response
	require.NoError(t, json.Unmarshal(clientScanner.Bytes(), &finalResp))
	finalData, err := json.Marshal(finalResp.Result)
	require.NoError(t, err)
	var finalPlay protocol.PlayResponse
	require.NoError(t, json.Unmarshal(finalData, &finalPlay))
	assert.True(t, finalPlay.Quit, "expected Quit flag in final response")

	// Close client write end and wait for handleConnection to finish.
	_ = clientW.Close()
	<-done
}
