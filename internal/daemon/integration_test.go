//go:build integration

package daemon

import (
	"bufio"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

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
	srv := NewServer(sockPath, scenarioDir, WithPassthrough())

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

// socketRoundTrip connects, sends a request, and reads one response.
func socketRoundTrip(t *testing.T, sockPath string, req Request) Response {
	t.Helper()
	conn, err := net.Dial("unix", sockPath)
	require.NoError(t, err)
	defer conn.Close()

	data, err := json.Marshal(req)
	require.NoError(t, err)
	data = append(data, '\n')
	_, err = conn.Write(data)
	require.NoError(t, err)

	scanner := bufio.NewScanner(conn)
	require.True(t, scanner.Scan(), "expected response")

	var resp Response
	require.NoError(t, json.Unmarshal(scanner.Bytes(), &resp))
	return resp
}

// socketMultiRoundTrip sends multiple requests on one connection.
func socketMultiRoundTrip(t *testing.T, sockPath string, reqs []Request) []Response {
	t.Helper()
	conn, err := net.Dial("unix", sockPath)
	require.NoError(t, err)
	defer conn.Close()

	for _, req := range reqs {
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
	return resps
}

func TestIntegration_SocketInitialize(t *testing.T) {
	sockPath := startTestDaemon(t)

	idJSON, _ := json.Marshal(1)
	resp := socketRoundTrip(t, sockPath, Request{
		JSONRPC: "2.0", ID: idJSON, Method: "initialize",
	})

	require.Nil(t, resp.Error)
	data, _ := json.Marshal(resp.Result)
	var init InitializeResult
	require.NoError(t, json.Unmarshal(data, &init))
	assert.Equal(t, "cryptd", init.ServerInfo.Name)
}

func TestIntegration_SocketGameSession(t *testing.T) {
	sockPath := startTestDaemon(t)

	reqs := []Request{
		toolCall(1, "new_game", map[string]any{
			"scenario_id": "minimal", "character_name": "Hero", "character_class": "fighter",
		}),
		toolCall(2, "look", nil),
		toolCall(3, "pick_up", map[string]any{"item_id": "short_sword"}),
		toolCall(4, "inventory", nil),
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
	srv := NewTCPServer(":0", scenarioDir, WithPassthrough())

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

// tcpRoundTrip connects over TCP, sends a request, and reads one response.
func tcpRoundTrip(t *testing.T, addr string, req Request) Response {
	t.Helper()
	conn, err := net.Dial("tcp", addr)
	require.NoError(t, err)
	defer conn.Close()

	data, err := json.Marshal(req)
	require.NoError(t, err)
	data = append(data, '\n')
	_, err = conn.Write(data)
	require.NoError(t, err)

	scanner := bufio.NewScanner(conn)
	require.True(t, scanner.Scan(), "expected response")

	var resp Response
	require.NoError(t, json.Unmarshal(scanner.Bytes(), &resp))
	return resp
}

func TestIntegration_TCPInitialize(t *testing.T) {
	addr := startTestTCPDaemon(t)

	idJSON, _ := json.Marshal(1)
	resp := tcpRoundTrip(t, addr, Request{
		JSONRPC: "2.0", ID: idJSON, Method: "initialize",
	})

	require.Nil(t, resp.Error)
	data, _ := json.Marshal(resp.Result)
	var init InitializeResult
	require.NoError(t, json.Unmarshal(data, &init))
	assert.Equal(t, "cryptd", init.ServerInfo.Name)
}

func TestIntegration_TCPGameSession(t *testing.T) {
	addr := startTestTCPDaemon(t)

	// Start a game.
	resp := tcpRoundTrip(t, addr, toolCall(1, "new_game", map[string]any{
		"scenario_id": "minimal", "character_name": "Hero", "character_class": "fighter",
	}))
	require.Nil(t, resp.Error)
	result := extractResult(t, resp)
	assert.Equal(t, "entrance", result["room"])

	// Look from a second TCP connection (state persists on server).
	resp2 := tcpRoundTrip(t, addr, toolCall(2, "look", nil))
	require.Nil(t, resp2.Error)
	lookResult := extractResult(t, resp2)
	assert.Equal(t, "Entrance Hall", lookResult["name"])
}

func TestIntegration_SocketMultipleConnections(t *testing.T) {
	sockPath := startTestDaemon(t)

	// First connection: start game.
	resp1 := socketRoundTrip(t, sockPath, toolCall(1, "new_game", map[string]any{
		"scenario_id": "minimal", "character_name": "Hero", "character_class": "fighter",
	}))
	require.Nil(t, resp1.Error)

	// Second connection: state persists (same server).
	resp2 := socketRoundTrip(t, sockPath, toolCall(2, "look", nil))
	require.Nil(t, resp2.Error)
	result := extractResult(t, resp2)
	assert.Equal(t, "entrance", result["room"])
}
