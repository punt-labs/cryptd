//go:build e2e

package e2e

import (
	"bufio"
	"encoding/json"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_SessionReconnect verifies end-to-end that a client can disconnect and
// reconnect with the same session ID to resume a game in normal mode.
//
// Flow:
//   1. Start cryptd serve -f --socket <tmp>
//   2. Client 1: initialize -> new_game -> play "go south" -> disconnect
//   3. Client 2: initialize with session ID from client 1 -> read initial room
//      -> assert room is goblin_lair (not entrance)
func TestE2E_SessionReconnect(t *testing.T) {
	bin := serverBinary(t)
	root := repoRoot(t)

	// Use a short socket path (macOS 104-byte limit).
	dir, err := os.MkdirTemp("/tmp", "cryptd-e2e-session-*")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })
	sockPath := filepath.Join(dir, "d.sock")

	// Start daemon subprocess in normal mode (no --passthrough).
	cmd := exec.Command(bin, "serve", "-f", "--socket", sockPath)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "CRYPT_SCENARIO_DIR=testdata/scenarios")
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		_ = cmd.Process.Signal(os.Interrupt)
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		select {
		case <-time.After(5 * time.Second):
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		case <-done:
		}
	})

	// Wait for socket to appear.
	ready := false
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		conn, dialErr := net.Dial("unix", sockPath)
		if dialErr == nil {
			conn.Close()
			ready = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	require.True(t, ready, "daemon socket %s not ready within timeout", sockPath)

	// --- Client 1: initialize -> new_game -> play "go south" -> disconnect ---
	conn1, err := net.Dial("unix", sockPath)
	require.NoError(t, err)

	scanner1 := bufio.NewScanner(conn1)
	scanner1.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	send := func(conn net.Conn, s *bufio.Scanner, id int, method string, params any) map[string]any {
		t.Helper()
		req := map[string]any{
			"jsonrpc": "2.0",
			"id":      id,
			"method":  method,
		}
		if params != nil {
			p, _ := json.Marshal(params)
			req["params"] = json.RawMessage(p)
		}
		data, merr := json.Marshal(req)
		require.NoError(t, merr)
		data = append(data, '\n')
		_, werr := conn.Write(data)
		require.NoError(t, werr)

		require.True(t, s.Scan(), "expected response for id=%d method=%s", id, method)
		var resp map[string]any
		require.NoError(t, json.Unmarshal(s.Bytes(), &resp))
		return resp
	}

	// Initialize.
	initResp := send(conn1, scanner1, 1, "initialize", nil)
	require.Nil(t, initResp["error"])
	initResult, ok := initResp["result"].(map[string]any)
	require.True(t, ok)
	sessionID, ok := initResult["session_id"].(string)
	require.True(t, ok)
	require.NotEmpty(t, sessionID)

	// New game via tools/call.
	argsJSON, _ := json.Marshal(map[string]string{
		"scenario_id":     "minimal",
		"character_name":  "E2E ReconnectHero",
		"character_class": "fighter",
	})
	ngResp := send(conn1, scanner1, 2, "tools/call", map[string]any{
		"name":      "new_game",
		"arguments": json.RawMessage(argsJSON),
	})
	require.Nil(t, ngResp["error"], "new_game error: %+v", ngResp["error"])

	// After new_game, the game loop is running. Send "go south".
	playResp := send(conn1, scanner1, 3, "play", map[string]string{"text": "go south"})
	require.Nil(t, playResp["error"])

	// Verify the response contains the current room state.
	playResult, ok := playResp["result"].(map[string]any)
	require.True(t, ok)
	state, ok := playResult["state"].(map[string]any)
	require.True(t, ok, "expected state in play response")
	dungeon, ok := state["dungeon"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "goblin_lair", dungeon["current_room"])

	// Disconnect client 1.
	conn1.Close()

	// Wait for the server to process the disconnection.
	time.Sleep(200 * time.Millisecond)

	// --- Client 2: initialize with session ID -> read resumed room ---
	conn2, err := net.Dial("unix", sockPath)
	require.NoError(t, err)
	defer conn2.Close()

	scanner2 := bufio.NewScanner(conn2)
	scanner2.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	// Send initialize with session ID.
	initResp2 := send(conn2, scanner2, 1, "initialize", map[string]string{
		"session_id": sessionID,
	})
	require.Nil(t, initResp2["error"])

	// After initialize with an existing game, the server enters resumeGameLoop
	// and sends the current room as the initial render. Read it.
	require.True(t, scanner2.Scan(), "expected initial room render on resume")
	var roomResp map[string]any
	require.NoError(t, json.Unmarshal(scanner2.Bytes(), &roomResp))

	roomResult, ok := roomResp["result"].(map[string]any)
	require.True(t, ok, "expected result in room response")
	roomState, ok := roomResult["state"].(map[string]any)
	require.True(t, ok, "expected state in resumed room render")
	roomDungeon, ok := roomState["dungeon"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "goblin_lair", roomDungeon["current_room"],
		"resumed session should be in goblin_lair, not entrance")
}
