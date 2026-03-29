package daemon

import (
	"bufio"
	"encoding/json"
	"io"
	"path/filepath"
	"testing"

	"github.com/punt-labs/cryptd/internal/interpreter"
	"github.com/punt-labs/cryptd/internal/narrator"
	"github.com/punt-labs/cryptd/internal/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResumeGameLoop_NormalMode verifies that a reconnected session with an
// active game enters the game loop automatically and renders the current room.
//
// Flow:
//
//	Connection 1: initialize -> new_game -> play "go south" -> disconnect
//	Connection 2: initialize with same session ID -> read initial room render
//	  (should be goblin_lair, not entrance) -> play "look" -> read response -> quit
func TestResumeGameLoop_NormalMode(t *testing.T) {
	scenarioDir := filepath.Join(repoRoot(t), "testdata", "scenarios")
	rules := interpreter.NewRules()
	tmpl := narrator.NewTemplate()
	interp := interpreter.NewSLM(nil, rules)
	narr := narrator.NewSLM(nil, tmpl)
	srv := NewServer(
		filepath.Join(t.TempDir(), "test.sock"),
		scenarioDir,
		WithInterpreter(interp),
		WithNarrator(narr),
	)

	// --- Connection 1: initialize -> new_game -> play "go south" -> disconnect ---
	clientR1, serverW1 := io.Pipe()
	serverR1, clientW1 := io.Pipe()

	done1 := make(chan struct{})
	go func() {
		defer close(done1)
		defer serverW1.Close()
		defer serverR1.Close()
		srv.handleConnection(serverR1, serverW1)
	}()

	scanner1 := bufio.NewScanner(clientR1)
	scanner1.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	send1 := func(method string, params any) protocol.Response {
		t.Helper()
		p, _ := json.Marshal(params)
		req := protocol.Request{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: method, Params: p}
		data, _ := json.Marshal(req)
		data = append(data, '\n')
		_, err := clientW1.Write(data)
		require.NoError(t, err)
		require.True(t, scanner1.Scan(), "expected response for %s", method)
		var resp protocol.Response
		require.NoError(t, json.Unmarshal(scanner1.Bytes(), &resp))
		return resp
	}

	// Initialize.
	initResp := send1("initialize", nil)
	require.Nil(t, initResp.Error)
	initData, _ := json.Marshal(initResp.Result)
	var initResult protocol.InitializeResult
	require.NoError(t, json.Unmarshal(initData, &initResult))
	sessionID := initResult.SessionID
	require.NotEmpty(t, sessionID)

	// New game.
	ngResp := send1("tools/call", map[string]any{
		"name": "new_game",
		"arguments": json.RawMessage(`{"scenario_id":"minimal","character_name":"Tester","character_class":"fighter"}`),
	})
	require.Nil(t, ngResp.Error, "new_game error: %+v", ngResp.Error)

	// After new_game, the server enters the game loop. Send "go south".
	moveResp := send1("play", protocol.PlayRequest{Text: "go south"})
	require.Nil(t, moveResp.Error)

	// Verify we moved to goblin_lair via the game state in the response.
	moveData, _ := json.Marshal(moveResp.Result)
	var movePlay protocol.PlayResponse
	require.NoError(t, json.Unmarshal(moveData, &movePlay))
	require.NotNil(t, movePlay.State)
	assert.Equal(t, "goblin_lair", movePlay.State.Dungeon.CurrentRoom)

	// Disconnect: close the client write pipe -> scanner EOF -> handleConnection returns.
	// Also close the client read pipe so the server's writes don't block.
	clientW1.Close()
	clientR1.Close()
	<-done1

	// Verify the game is still in the server registry with room = goblin_lair.
	srv.mu.RLock()
	sess, ok := srv.sessions[sessionID]
	srv.mu.RUnlock()
	require.True(t, ok, "session should exist after disconnect")
	require.NotEmpty(t, sess.gameID, "session should have a game after disconnect")

	// --- Connection 2: initialize with session ID -> resume -> look -> quit ---
	clientR2, serverW2 := io.Pipe()
	serverR2, clientW2 := io.Pipe()

	done2 := make(chan struct{})
	go func() {
		defer close(done2)
		defer serverW2.Close()
		defer serverR2.Close()
		srv.handleConnection(serverR2, serverW2)
	}()

	scanner2 := bufio.NewScanner(clientR2)
	scanner2.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	// Send initialize with session ID.
	initParams, _ := json.Marshal(protocol.InitializeParams{SessionID: sessionID})
	initReq := protocol.Request{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: "initialize", Params: initParams}
	initReqData, _ := json.Marshal(initReq)
	initReqData = append(initReqData, '\n')
	_, err := clientW2.Write(initReqData)
	require.NoError(t, err)

	// Read initialize response.
	require.True(t, scanner2.Scan(), "expected initialize response")
	var initResp2 protocol.Response
	require.NoError(t, json.Unmarshal(scanner2.Bytes(), &initResp2))
	require.Nil(t, initResp2.Error)

	// Verify HasGame is true for a resumed session with an active game.
	initData2, _ := json.Marshal(initResp2.Result)
	var initResult2 protocol.InitializeResult
	require.NoError(t, json.Unmarshal(initData2, &initResult2))
	assert.True(t, initResult2.HasGame, "resumed session with active game should have has_game=true")

	// After initialize with an existing game, the server enters resumeGameLoop
	// which runs RunLoop with SkipInitialRender=false, meaning it sends the
	// initial room render. Read it.
	require.True(t, scanner2.Scan(), "expected initial room render on resume")
	var roomResp protocol.Response
	require.NoError(t, json.Unmarshal(scanner2.Bytes(), &roomResp))

	roomData, _ := json.Marshal(roomResp.Result)
	var roomPlay protocol.PlayResponse
	require.NoError(t, json.Unmarshal(roomData, &roomPlay))
	require.NotNil(t, roomPlay.State, "expected game state in resumed room render")
	assert.Equal(t, "goblin_lair", roomPlay.State.Dungeon.CurrentRoom,
		"resumed session should be in goblin_lair, not entrance")

	// Send "look" to verify the game loop is working.
	lookReq := protocol.Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`3`),
		Method:  "play",
		Params:  json.RawMessage(`{"text":"look"}`),
	}
	lookData, _ := json.Marshal(lookReq)
	lookData = append(lookData, '\n')
	_, err = clientW2.Write(lookData)
	require.NoError(t, err)

	require.True(t, scanner2.Scan(), "expected look response")
	var lookResp protocol.Response
	require.NoError(t, json.Unmarshal(scanner2.Bytes(), &lookResp))
	require.Nil(t, lookResp.Error)

	lookRespData, _ := json.Marshal(lookResp.Result)
	var lookPlay protocol.PlayResponse
	require.NoError(t, json.Unmarshal(lookRespData, &lookPlay))
	require.NotNil(t, lookPlay.State)
	assert.Equal(t, "goblin_lair", lookPlay.State.Dungeon.CurrentRoom)

	// Disconnect by closing both ends of the pipe.
	clientW2.Close()
	clientR2.Close()
	<-done2
}

// TestResumeGameLoop_NoGame verifies that initialize with an unknown session ID
// (that has no game) does not enter the game loop. The server should create a
// new session and let the client proceed normally (e.g. send new_game).
func TestResumeGameLoop_NoGame(t *testing.T) {
	srv := testNormalServer(t)

	reqs := []Request{
		initRequestWithSession(0, "nonexistent-session-id"),
	}

	// multiRoundTrip uses buffer-based handleConnection -- if the server tried
	// to enter a game loop, it would block. With no game, it returns normally.
	resps := multiRoundTrip(t, srv, reqs)
	require.Len(t, resps, 1)
	require.Nil(t, resps[0].Error)

	// The server creates a new session with the provided ID.
	data, _ := json.Marshal(resps[0].Result)
	var init InitializeResult
	require.NoError(t, json.Unmarshal(data, &init))
	assert.Equal(t, "nonexistent-session-id", init.SessionID)
	assert.False(t, init.HasGame, "new session without a game should have has_game=false")
}

// TestResumeGameLoop_SkipInitialRender verifies that the new_game path sets
// SkipInitialRender=true so the initial room is not rendered twice (once by
// handleNewGamePlay and once by RunLoop).
func TestResumeGameLoop_SkipInitialRender(t *testing.T) {
	scenarioDir := filepath.Join(repoRoot(t), "testdata", "scenarios")
	rules := interpreter.NewRules()
	tmpl := narrator.NewTemplate()
	interp := interpreter.NewSLM(nil, rules)
	narr := narrator.NewSLM(nil, tmpl)
	srv := NewServer(
		filepath.Join(t.TempDir(), "test.sock"),
		scenarioDir,
		WithInterpreter(interp),
		WithNarrator(narr),
	)

	clientR, serverW := io.Pipe()
	serverR, clientW := io.Pipe()

	done := make(chan struct{})
	go func() {
		defer close(done)
		defer serverW.Close()
		defer serverR.Close()
		srv.handleConnection(serverR, serverW)
	}()

	scanner := bufio.NewScanner(clientR)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	sendReq := func(req protocol.Request) {
		t.Helper()
		data, _ := json.Marshal(req)
		data = append(data, '\n')
		_, err := clientW.Write(data)
		require.NoError(t, err)
	}

	// Initialize.
	sendReq(protocol.Request{JSONRPC: "2.0", ID: json.RawMessage(`0`), Method: "initialize"})
	require.True(t, scanner.Scan(), "expected initialize response")

	// New game via tools/call.
	argsJSON, _ := json.Marshal(map[string]string{
		"scenario_id":     "minimal",
		"character_name":  "Tester",
		"character_class": "fighter",
	})
	tcParams, _ := json.Marshal(protocol.ToolCallParams{Name: "new_game", Arguments: argsJSON})
	sendReq(protocol.Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "tools/call",
		Params:  tcParams,
	})

	// Read new_game response (the PlayResponse from handleNewGamePlay).
	require.True(t, scanner.Scan(), "expected new_game response")
	var ngResp protocol.Response
	require.NoError(t, json.Unmarshal(scanner.Bytes(), &ngResp))
	require.Nil(t, ngResp.Error)

	ngData, _ := json.Marshal(ngResp.Result)
	var ngPlay protocol.PlayResponse
	require.NoError(t, json.Unmarshal(ngData, &ngPlay))
	assert.NotEmpty(t, ngPlay.Text, "new_game should return narrated text")

	// The game loop is now running with SkipInitialRender=true. The next
	// response should come only when we send a play request -- no unsolicited
	// initial render. Send "look" and verify we get exactly one response.
	sendReq(protocol.Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`2`),
		Method:  "play",
		Params:  json.RawMessage(`{"text":"look"}`),
	})

	require.True(t, scanner.Scan(), "expected look response")
	var lookResp protocol.Response
	require.NoError(t, json.Unmarshal(scanner.Bytes(), &lookResp))
	require.Nil(t, lookResp.Error)

	lookRespData, _ := json.Marshal(lookResp.Result)
	var lookPlay protocol.PlayResponse
	require.NoError(t, json.Unmarshal(lookRespData, &lookPlay))
	assert.NotEmpty(t, lookPlay.Text, "look should return narrated text")
	require.NotNil(t, lookPlay.State)
	assert.Equal(t, "entrance", lookPlay.State.Dungeon.CurrentRoom)

	// Close both ends of the pipe to clean up.
	clientW.Close()
	clientR.Close()
	<-done
}
