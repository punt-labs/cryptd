//go:build integration

package daemon

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/punt-labs/cryptd/internal/interpreter"
	"github.com/punt-labs/cryptd/internal/narrator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// startTestDaemon launches a daemon on a temp socket and returns the socket path.
// The daemon is stopped when the test completes.
func startTestDaemon(t *testing.T) string {
	t.Helper()
	// macOS limits Unix socket paths to 104 bytes. Use /tmp directly
	// instead of t.TempDir() which produces long paths under /var/folders.
	dir, err := os.MkdirTemp("/tmp", "cryptd-test-*")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })

	sockPath := filepath.Join(dir, "d.sock")
	scenarioDir := filepath.Join(repoRoot(t), "testdata", "scenarios")
	srv := NewServer(sockPath, scenarioDir)

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	// Poll until socket is ready.
	ready := false
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.Dial("unix", sockPath)
		if err == nil {
			conn.Close()
			ready = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	require.True(t, ready, "daemon socket %s not ready within timeout", sockPath)

	t.Cleanup(func() {
		if srv.listener != nil {
			srv.listener.Close()
		}
		<-errCh
	})

	return sockPath
}

// socketSession connects to the socket, sends session.init, and returns the
// session ID and open connection for further use.
func socketSession(t *testing.T, sockPath string) (string, net.Conn, *bufio.Scanner) {
	t.Helper()
	conn, err := net.Dial("unix", sockPath)
	require.NoError(t, err)

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	// Send session.init.
	idJSON, _ := json.Marshal(0)
	req := Request{JSONRPC: "2.0", ID: idJSON, Method: "session.init"}
	data, err := json.Marshal(req)
	require.NoError(t, err)
	data = append(data, '\n')
	_, err = conn.Write(data)
	require.NoError(t, err)

	require.True(t, scanner.Scan(), "expected session.init response")
	var resp Response
	require.NoError(t, json.Unmarshal(scanner.Bytes(), &resp))
	require.Nil(t, resp.Error)

	rdata, _ := json.Marshal(resp.Result)
	var init InitializeResult
	require.NoError(t, json.Unmarshal(rdata, &init))

	return init.SessionID, conn, scanner
}

// socketRoundTrip connects, sends session.init + a request, and reads responses.
func socketRoundTrip(t *testing.T, sockPath string, req Request) Response {
	t.Helper()
	conn, err := net.Dial("unix", sockPath)
	require.NoError(t, err)
	defer conn.Close()

	// Send session.init first.
	initReq := Request{JSONRPC: "2.0", ID: json.RawMessage(`0`), Method: "session.init"}
	data, err := json.Marshal(initReq)
	require.NoError(t, err)
	data = append(data, '\n')
	_, err = conn.Write(data)
	require.NoError(t, err)

	// Send the actual request.
	data, err = json.Marshal(req)
	require.NoError(t, err)
	data = append(data, '\n')
	_, err = conn.Write(data)
	require.NoError(t, err)

	scanner := bufio.NewScanner(conn)
	// Read session.init response.
	require.True(t, scanner.Scan(), "expected session.init response")
	// Read actual response.
	require.True(t, scanner.Scan(), "expected response")

	var resp Response
	require.NoError(t, json.Unmarshal(scanner.Bytes(), &resp))
	return resp
}

// socketMultiRoundTrip sends session.init + multiple requests on one connection.
func socketMultiRoundTrip(t *testing.T, sockPath string, reqs []Request) []Response {
	t.Helper()
	conn, err := net.Dial("unix", sockPath)
	require.NoError(t, err)
	defer conn.Close()

	// Prepend session.init.
	allReqs := append([]Request{
		{JSONRPC: "2.0", ID: json.RawMessage(`0`), Method: "session.init"},
	}, reqs...)

	for _, req := range allReqs {
		data, err := json.Marshal(req)
		require.NoError(t, err)
		data = append(data, '\n')
		_, err = conn.Write(data)
		require.NoError(t, err)
	}

	// Half-close write side to signal EOF to the daemon.
	if uc, ok := conn.(*net.UnixConn); ok {
		require.NoError(t, uc.CloseWrite())
	}

	var resps []Response
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		var resp Response
		require.NoError(t, json.Unmarshal(scanner.Bytes(), &resp))
		resps = append(resps, resp)
	}
	// Strip the session.init response.
	if len(resps) > 0 {
		resps = resps[1:]
	}
	return resps
}

func TestIntegration_SocketInitialize(t *testing.T) {
	sockPath := startTestDaemon(t)

	idJSON, _ := json.Marshal(1)
	resp := socketRoundTrip(t, sockPath, Request{
		JSONRPC: "2.0", ID: idJSON, Method: "session.init",
	})

	// socketRoundTrip sends its own session.init, then the request is another
	// session.init — both should succeed. Check the second one.
	require.Nil(t, resp.Error)
	data, _ := json.Marshal(resp.Result)
	var init InitializeResult
	require.NoError(t, json.Unmarshal(data, &init))
	assert.Equal(t, "cryptd", init.ServerInfo.Name)
}

func TestIntegration_SocketGameSession(t *testing.T) {
	sockPath := startTestDaemon(t)

	reqs := []Request{
		newGameCall(1, map[string]any{
			"scenario_id": "minimal", "character_name": "Hero", "character_class": "fighter",
		}),
		gameCall(2, "look", nil),
		gameCall(3, "pick_up", map[string]any{"item_id": "short_sword"}),
		gameCall(4, "inventory", nil),
	}

	resps := socketMultiRoundTrip(t, sockPath, reqs)
	require.Len(t, resps, 4)

	// new_game
	newGameResult := extractResult(t, resps[0])
	assert.Equal(t, "entrance", newGameResult["room"])

	// look
	lookResult := extractResult(t, resps[1])
	assert.Equal(t, "Entrance Hall", lookResult["name"])

	// pick_up
	pickUpResult := extractResult(t, resps[2])
	item, ok := pickUpResult["item"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "short_sword", item["id"])

	// inventory
	invResult := extractResult(t, resps[3])
	items, ok := invResult["items"].([]any)
	require.True(t, ok)
	assert.Len(t, items, 1)
}

// startTestTCPDaemon launches a daemon on an ephemeral TCP port and returns the address.
// It creates the listener synchronously to get the assigned port, then runs Serve in a goroutine.
func startTestTCPDaemon(t *testing.T) string {
	t.Helper()
	scenarioDir := filepath.Join(repoRoot(t), "testdata", "scenarios")
	srv := NewTCPServer(":0", scenarioDir)

	ln, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	addr := ln.Addr().String()

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(ln)
	}()

	// Poll until accepting connections.
	ready := false
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		conn, dialErr := net.Dial("tcp", addr)
		if dialErr == nil {
			conn.Close()
			ready = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	require.True(t, ready, "TCP server at %s not ready within timeout", addr)

	t.Cleanup(func() {
		ln.Close()
		<-errCh
	})

	return addr
}

// tcpRoundTrip connects over TCP, sends session.init + a request, and reads responses.
func tcpRoundTrip(t *testing.T, addr string, req Request) Response {
	t.Helper()
	conn, err := net.Dial("tcp", addr)
	require.NoError(t, err)
	defer conn.Close()

	// Send session.init first.
	initReq := Request{JSONRPC: "2.0", ID: json.RawMessage(`0`), Method: "session.init"}
	data, err := json.Marshal(initReq)
	require.NoError(t, err)
	data = append(data, '\n')
	_, err = conn.Write(data)
	require.NoError(t, err)

	// Send the actual request.
	data, err = json.Marshal(req)
	require.NoError(t, err)
	data = append(data, '\n')
	_, err = conn.Write(data)
	require.NoError(t, err)

	scanner := bufio.NewScanner(conn)
	// Read session.init response.
	require.True(t, scanner.Scan(), "expected session.init response")
	// Read actual response.
	require.True(t, scanner.Scan(), "expected response")

	var resp Response
	require.NoError(t, json.Unmarshal(scanner.Bytes(), &resp))
	return resp
}

// tcpSessionRoundTrip uses a specific session ID to reconnect.
func tcpSessionRoundTrip(t *testing.T, addr, sessionID string, req Request) Response {
	t.Helper()
	conn, err := net.Dial("tcp", addr)
	require.NoError(t, err)
	defer conn.Close()

	// Send session.init with session ID.
	params, _ := json.Marshal(InitializeParams{SessionID: sessionID})
	initReq := Request{JSONRPC: "2.0", ID: json.RawMessage(`0`), Method: "session.init", Params: params}
	data, err := json.Marshal(initReq)
	require.NoError(t, err)
	data = append(data, '\n')
	_, err = conn.Write(data)
	require.NoError(t, err)

	// Send the actual request.
	data, err = json.Marshal(req)
	require.NoError(t, err)
	data = append(data, '\n')
	_, err = conn.Write(data)
	require.NoError(t, err)

	scanner := bufio.NewScanner(conn)
	// Read session.init response.
	require.True(t, scanner.Scan(), "expected session.init response")
	// Read actual response.
	require.True(t, scanner.Scan(), "expected response")

	var resp Response
	require.NoError(t, json.Unmarshal(scanner.Bytes(), &resp))
	return resp
}

func TestIntegration_TCPInitialize(t *testing.T) {
	addr := startTestTCPDaemon(t)

	idJSON, _ := json.Marshal(1)
	resp := tcpRoundTrip(t, addr, Request{
		JSONRPC: "2.0", ID: idJSON, Method: "session.init",
	})

	require.Nil(t, resp.Error)
	data, _ := json.Marshal(resp.Result)
	var init InitializeResult
	require.NoError(t, json.Unmarshal(data, &init))
	assert.Equal(t, "cryptd", init.ServerInfo.Name)
}

func TestIntegration_TCPGameSession(t *testing.T) {
	addr := startTestTCPDaemon(t)

	// First connection: initialize and start a game.
	conn1, err := net.Dial("tcp", addr)
	require.NoError(t, err)
	defer conn1.Close()

	scanner1 := bufio.NewScanner(conn1)

	// Initialize.
	initReq := Request{JSONRPC: "2.0", ID: json.RawMessage(`0`), Method: "session.init"}
	data, _ := json.Marshal(initReq)
	data = append(data, '\n')
	conn1.Write(data)
	require.True(t, scanner1.Scan())
	var initResp Response
	require.NoError(t, json.Unmarshal(scanner1.Bytes(), &initResp))
	require.Nil(t, initResp.Error)
	rdata, _ := json.Marshal(initResp.Result)
	var init InitializeResult
	require.NoError(t, json.Unmarshal(rdata, &init))
	sessionID := init.SessionID

	// Start game.
	ngReq := newGameCall(1, map[string]any{
		"scenario_id": "minimal", "character_name": "Hero", "character_class": "fighter",
	})
	data, _ = json.Marshal(ngReq)
	data = append(data, '\n')
	conn1.Write(data)
	require.True(t, scanner1.Scan())
	var ngResp Response
	require.NoError(t, json.Unmarshal(scanner1.Bytes(), &ngResp))
	require.Nil(t, ngResp.Error)
	ngResult := extractResult(t, ngResp)
	assert.Equal(t, "entrance", ngResult["room"])
	conn1.Close()

	// Second TCP connection with same session ID — state persists.
	resp2 := tcpSessionRoundTrip(t, addr, sessionID, gameCall(2, "look", nil))
	require.Nil(t, resp2.Error)
	lookResult := extractResult(t, resp2)
	assert.Equal(t, "Entrance Hall", lookResult["name"])
}

func TestIntegration_SocketMultipleConnections(t *testing.T) {
	sockPath := startTestDaemon(t)

	// First connection: initialize and start game.
	sid, conn1, scanner1 := socketSession(t, sockPath)

	ngReq := newGameCall(1, map[string]any{
		"scenario_id": "minimal", "character_name": "Hero", "character_class": "fighter",
	})
	data, _ := json.Marshal(ngReq)
	data = append(data, '\n')
	conn1.Write(data)
	require.True(t, scanner1.Scan())
	var ngResp Response
	require.NoError(t, json.Unmarshal(scanner1.Bytes(), &ngResp))
	require.Nil(t, ngResp.Error)
	conn1.Close()

	// Second connection with same session ID — game state persists.
	conn2, err := net.Dial("unix", sockPath)
	require.NoError(t, err)
	defer conn2.Close()

	params, _ := json.Marshal(InitializeParams{SessionID: sid})
	initReq2 := Request{JSONRPC: "2.0", ID: json.RawMessage(`0`), Method: "session.init", Params: params}
	data, _ = json.Marshal(initReq2)
	data = append(data, '\n')
	conn2.Write(data)

	lookReq := gameCall(2, "look", nil)
	data, _ = json.Marshal(lookReq)
	data = append(data, '\n')
	conn2.Write(data)

	scanner2 := bufio.NewScanner(conn2)
	// Read session.init response.
	require.True(t, scanner2.Scan())
	// Read look response.
	require.True(t, scanner2.Scan())
	var multiLookResp Response
	require.NoError(t, json.Unmarshal(scanner2.Bytes(), &multiLookResp))
	require.Nil(t, multiLookResp.Error)
	multiResult := extractResult(t, multiLookResp)
	assert.Equal(t, "entrance", multiResult["room"])
}

func TestIntegration_ConcurrentSessionIsolation(t *testing.T) {
	addr := startTestTCPDaemon(t)

	const numClients = 2
	type clientResult struct {
		sessionID     string
		characterName string
		lookRoom      string
		err           error
	}

	results := make([]clientResult, numClients)
	var wg sync.WaitGroup

	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			charName := fmt.Sprintf("Hero%d", idx)

			conn, err := net.Dial("tcp", addr)
			if err != nil {
				results[idx] = clientResult{err: fmt.Errorf("dial: %w", err)}
				return
			}
			defer conn.Close()

			scanner := bufio.NewScanner(conn)
			scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

			sendAndRead := func(req Request) (Response, error) {
				data, merr := json.Marshal(req)
				if merr != nil {
					return Response{}, merr
				}
				data = append(data, '\n')
				if _, werr := conn.Write(data); werr != nil {
					return Response{}, werr
				}
				if !scanner.Scan() {
					return Response{}, fmt.Errorf("no response (scan err: %v)", scanner.Err())
				}
				var resp Response
				if uerr := json.Unmarshal(scanner.Bytes(), &resp); uerr != nil {
					return Response{}, uerr
				}
				return resp, nil
			}

			// 1. Initialize.
			initReq := Request{JSONRPC: "2.0", ID: json.RawMessage(`0`), Method: "session.init"}
			initResp, err := sendAndRead(initReq)
			if err != nil {
				results[idx] = clientResult{err: fmt.Errorf("session.init: %w", err)}
				return
			}
			rdata, _ := json.Marshal(initResp.Result)
			var init InitializeResult
			if err := json.Unmarshal(rdata, &init); err != nil {
				results[idx] = clientResult{err: fmt.Errorf("parse init: %w", err)}
				return
			}

			// 2. New game with unique character name.
			ngReq := newGameCall(1, map[string]any{
				"scenario_id":     "minimal",
				"character_name":  charName,
				"character_class": "fighter",
			})
			ngResp, err := sendAndRead(ngReq)
			if err != nil {
				results[idx] = clientResult{err: fmt.Errorf("game.new: %w", err)}
				return
			}
			if ngResp.Error != nil {
				results[idx] = clientResult{err: fmt.Errorf("game.new rpc error: %s", ngResp.Error.Message)}
				return
			}

			// 3. Look.
			lookReq := gameCall(2, "look", nil)
			lookResp, err := sendAndRead(lookReq)
			if err != nil {
				results[idx] = clientResult{err: fmt.Errorf("game.look: %w", err)}
				return
			}
			if lookResp.Error != nil {
				results[idx] = clientResult{err: fmt.Errorf("game.look rpc error: %s", lookResp.Error.Message)}
				return
			}

			// Extract character name from new_game result.
			ngData, _ := json.Marshal(ngResp.Result)
			var ngResult map[string]any
			_ = json.Unmarshal(ngData, &ngResult)
			hero, _ := ngResult["hero"].(map[string]any)
			heroName, _ := hero["name"].(string)

			// Extract room from look result.
			lookData, _ := json.Marshal(lookResp.Result)
			var lookResult map[string]any
			_ = json.Unmarshal(lookData, &lookResult)
			room, _ := lookResult["room"].(string)

			results[idx] = clientResult{
				sessionID:     init.SessionID,
				characterName: heroName,
				lookRoom:      room,
			}
		}(i)
	}

	wg.Wait()

	for i, r := range results {
		require.NoError(t, r.err, "client %d failed", i)
		expectedName := fmt.Sprintf("Hero%d", i)
		assert.Equal(t, expectedName, r.characterName, "client %d: character name mismatch", i)
		assert.Equal(t, "entrance", r.lookRoom, "client %d: room mismatch", i)
	}
	assert.NotEqual(t, results[0].sessionID, results[1].sessionID, "sessions should have different IDs")
}

func TestIntegration_SessionIsolation(t *testing.T) {
	sockPath := startTestDaemon(t)

	// Session A: start a game, move south.
	sidA, connA, scannerA := socketSession(t, sockPath)

	ngReq := newGameCall(1, map[string]any{
		"scenario_id": "minimal", "character_name": "HeroA", "character_class": "fighter",
	})
	data, _ := json.Marshal(ngReq)
	data = append(data, '\n')
	connA.Write(data)
	require.True(t, scannerA.Scan())
	connA.Close()

	// Session B: start a different game.
	sidB, connB, scannerB := socketSession(t, sockPath)
	ngReq2 := newGameCall(1, map[string]any{
		"scenario_id": "minimal", "character_name": "HeroB", "character_class": "mage",
	})
	data, _ = json.Marshal(ngReq2)
	data = append(data, '\n')
	connB.Write(data)
	require.True(t, scannerB.Scan())
	connB.Close()

	assert.NotEqual(t, sidA, sidB, "sessions should have different IDs")

	// Verify they have independent state: both should be in entrance.
	// Reconnect session A and look.
	conn3, err := net.Dial("unix", sockPath)
	require.NoError(t, err)
	defer conn3.Close()
	params, _ := json.Marshal(InitializeParams{SessionID: sidA})
	initReq3 := Request{JSONRPC: "2.0", ID: json.RawMessage(`0`), Method: "session.init", Params: params}
	data, _ = json.Marshal(initReq3)
	data = append(data, '\n')
	conn3.Write(data)
	lookReq := gameCall(2, "look", nil)
	data, _ = json.Marshal(lookReq)
	data = append(data, '\n')
	conn3.Write(data)
	scanner3 := bufio.NewScanner(conn3)
	require.True(t, scanner3.Scan()) // session.init
	require.True(t, scanner3.Scan()) // look
	var isoLookResp Response
	require.NoError(t, json.Unmarshal(scanner3.Bytes(), &isoLookResp))
	require.Nil(t, isoLookResp.Error)
	isoResult := extractResult(t, isoLookResp)
	assert.Equal(t, "entrance", isoResult["room"])
}

// startTestTCPNormalDaemon launches a normal-mode TCP daemon (with Rules
// interpreter + Template narrator, no SLM) and returns the address.
func startTestTCPNormalDaemon(t *testing.T) string {
	t.Helper()
	scenarioDir := filepath.Join(repoRoot(t), "testdata", "scenarios")

	rules := interpreter.NewRules()
	tmpl := narrator.NewTemplate()
	interp := interpreter.NewSLM(nil, rules)
	narr := narrator.NewSLM(nil, tmpl)

	srv := NewTCPServer(":0", scenarioDir,
		WithInterpreter(interp),
		WithNarrator(narr),
	)

	ln, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	addr := ln.Addr().String()

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(ln)
	}()

	// Poll until accepting connections.
	ready := false
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		conn, dialErr := net.Dial("tcp", addr)
		if dialErr == nil {
			conn.Close()
			ready = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	require.True(t, ready, "TCP normal server at %s not ready within timeout", addr)

	t.Cleanup(func() {
		ln.Close()
		<-errCh
	})

	return addr
}

func TestIntegration_SessionReconnect_StatePreserved(t *testing.T) {
	addr := startTestTCPNormalDaemon(t)

	// --- Connection 1: session.init -> game.new -> play "go south" -> disconnect ---
	conn1, err := net.Dial("tcp", addr)
	require.NoError(t, err)

	scanner1 := bufio.NewScanner(conn1)
	scanner1.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	sendAndRead := func(conn net.Conn, s *bufio.Scanner, req Request) Response {
		t.Helper()
		data, merr := json.Marshal(req)
		require.NoError(t, merr)
		data = append(data, '\n')
		_, werr := conn.Write(data)
		require.NoError(t, werr)
		require.True(t, s.Scan(), "expected response")
		var resp Response
		require.NoError(t, json.Unmarshal(s.Bytes(), &resp))
		return resp
	}

	// Initialize.
	reconInitResp := sendAndRead(conn1, scanner1, Request{
		JSONRPC: "2.0", ID: json.RawMessage(`0`), Method: "session.init",
	})
	require.Nil(t, reconInitResp.Error)
	sessionID := extractSessionID(t, reconInitResp)

	// New game (game.new in normal mode).
	ngParams, _ := json.Marshal(map[string]string{
		"scenario_id":     "minimal",
		"character_name":  "ReconnectHero",
		"character_class": "fighter",
	})
	reconNgResp := sendAndRead(conn1, scanner1, Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "game.new",
		Params:  ngParams,
	})
	require.Nil(t, reconNgResp.Error, "game.new error: %+v", reconNgResp.Error)

	// After game.new, the game loop is running. Send "go south".
	reconPlayResp := sendAndRead(conn1, scanner1, Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`2`),
		Method:  "game.play",
		Params:  json.RawMessage(`{"text":"go south"}`),
	})
	require.Nil(t, reconPlayResp.Error)

	// Verify we moved to goblin_lair.
	prData, _ := json.Marshal(reconPlayResp.Result)
	var pr PlayResponse
	require.NoError(t, json.Unmarshal(prData, &pr))
	require.NotNil(t, pr.State)
	assert.Equal(t, "goblin_lair", pr.State.Dungeon.CurrentRoom)

	// Disconnect.
	conn1.Close()

	// Wait a moment for the server to process the disconnection.
	time.Sleep(100 * time.Millisecond)

	// --- Connection 2: session.init with session ID -> read resumed room ---
	conn2, err := net.Dial("tcp", addr)
	require.NoError(t, err)
	defer conn2.Close()

	scanner2 := bufio.NewScanner(conn2)
	scanner2.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	// Send session.init with session ID.
	initParams, _ := json.Marshal(InitializeParams{SessionID: sessionID})
	reconInitResp2 := sendAndRead(conn2, scanner2, Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`0`),
		Method:  "session.init",
		Params:  initParams,
	})
	require.Nil(t, reconInitResp2.Error)
	assert.Equal(t, sessionID, extractSessionID(t, reconInitResp2))

	// Read the initial room render (resumeGameLoop sends it).
	require.True(t, scanner2.Scan(), "expected initial room render on resume")
	var roomResp Response
	require.NoError(t, json.Unmarshal(scanner2.Bytes(), &roomResp))

	roomData, _ := json.Marshal(roomResp.Result)
	var roomPlay PlayResponse
	require.NoError(t, json.Unmarshal(roomData, &roomPlay))
	require.NotNil(t, roomPlay.State, "expected game state in resumed room render")
	assert.Equal(t, "goblin_lair", roomPlay.State.Dungeon.CurrentRoom,
		"resumed session should be in goblin_lair")

	// Verify inventory is preserved: the hero should have the same character.
	require.NotEmpty(t, roomPlay.State.Party)
	assert.Equal(t, "ReconnectHero", roomPlay.State.Party[0].Name)
}

func TestIntegration_GracefulShutdown(t *testing.T) {
	addr := startTestTCPNormalDaemon(t)

	// Connect a client and start a game.
	conn, err := net.Dial("tcp", addr)
	require.NoError(t, err)

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	sendReq := func(req Request) {
		t.Helper()
		data, merr := json.Marshal(req)
		require.NoError(t, merr)
		data = append(data, '\n')
		_, werr := conn.Write(data)
		require.NoError(t, werr)
	}

	readResp := func() Response {
		t.Helper()
		require.True(t, scanner.Scan(), "expected response")
		var resp Response
		require.NoError(t, json.Unmarshal(scanner.Bytes(), &resp))
		return resp
	}

	// Initialize.
	sendReq(Request{JSONRPC: "2.0", ID: json.RawMessage(`0`), Method: "session.init"})
	shutdownInitResp := readResp()
	require.Nil(t, shutdownInitResp.Error)

	// New game.
	ngParams, _ := json.Marshal(map[string]string{
		"scenario_id":     "minimal",
		"character_name":  "ShutdownHero",
		"character_class": "fighter",
	})
	sendReq(Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "game.new",
		Params:  ngParams,
	})
	shutdownNgResp := readResp()
	require.Nil(t, shutdownNgResp.Error)

	// The game loop is now running. Close the connection to simulate shutdown.
	conn.Close()

	// The test passes if it completes without deadlock. The t.Cleanup from
	// startTestTCPNormalDaemon closes the listener and waits for Serve to
	// return. If there's a deadlock, the test will timeout.
}
