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

func TestDaemon_ServeAndCallMethods(t *testing.T) {
	bin := serverBinary(t)
	root := repoRoot(t)

	// Use a short socket path (macOS 104-byte limit).
	dir, err := os.MkdirTemp("/tmp", "cryptd-e2e-*")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })
	sockPath := filepath.Join(dir, "d.sock")

	// Start daemon subprocess.
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
		conn, err := net.Dial("unix", sockPath)
		if err == nil {
			conn.Close()
			ready = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	require.True(t, ready, "daemon socket %s not ready within timeout", sockPath)

	// Connect and run a session.
	conn, err := net.Dial("unix", sockPath)
	require.NoError(t, err, "failed to connect to daemon socket")
	defer conn.Close()

	scanner := bufio.NewScanner(conn)

	send := func(id int, method string, params any) map[string]any {
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
		data, err := json.Marshal(req)
		require.NoError(t, err)
		data = append(data, '\n')
		_, err = conn.Write(data)
		require.NoError(t, err)

		require.True(t, scanner.Scan(), "expected response for id=%d", id)
		var resp map[string]any
		require.NoError(t, json.Unmarshal(scanner.Bytes(), &resp))
		return resp
	}

	// 1. Initialize in passthrough mode (direct game.* methods).
	resp := send(1, "session.init", map[string]string{"mode": "passthrough"})
	require.Nil(t, resp["error"])
	result, ok := resp["result"].(map[string]any)
	require.True(t, ok)
	serverInfo, ok := result["serverInfo"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "cryptd", serverInfo["name"])

	// 2. New game
	resp = send(2, "game.new", map[string]any{
		"scenario_id": "minimal", "character_name": "E2E Hero", "character_class": "fighter",
	})
	require.Nil(t, resp["error"])
	result, ok = resp["result"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "entrance", result["room"])

	// 3. Look
	resp = send(3, "game.look", nil)
	require.Nil(t, resp["error"])
	result, ok = resp["result"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Entrance Hall", result["name"])

	// 4. Pick up sword
	resp = send(4, "game.pick_up", map[string]any{"item_id": "short_sword"})
	require.Nil(t, resp["error"])
	result, ok = resp["result"].(map[string]any)
	require.True(t, ok)
	item, ok := result["item"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "short_sword", item["id"])

	// 5. Move south (triggers combat)
	resp = send(5, "game.move", map[string]any{"direction": "south"})
	require.Nil(t, resp["error"])
	result, ok = resp["result"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "goblin_lair", result["room"])
	_, hasCombat := result["combat"]
	assert.True(t, hasCombat, "expected combat to start in goblin_lair")
}
