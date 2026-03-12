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

func TestDaemon_ServeAndCallTools(t *testing.T) {
	bin := binary(t)
	root := repoRoot(t)

	// Use a short socket path (macOS 104-byte limit).
	dir, err := os.MkdirTemp("/tmp", "cryptd-e2e-*")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })
	sockPath := filepath.Join(dir, "d.sock")

	// Start daemon subprocess.
	cmd := exec.Command(bin, "serve", "--socket", sockPath)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "CRYPT_SCENARIO_DIR=testdata/scenarios")
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		cmd.Process.Signal(os.Interrupt)
		cmd.Wait()
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

	toolCallParams := func(name string, args any) map[string]any {
		argsJSON, _ := json.Marshal(args)
		return map[string]any{
			"name":      name,
			"arguments": json.RawMessage(argsJSON),
		}
	}

	// 1. Initialize
	resp := send(1, "initialize", nil)
	require.Nil(t, resp["error"])
	result, ok := resp["result"].(map[string]any)
	require.True(t, ok)
	serverInfo, ok := result["serverInfo"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "cryptd", serverInfo["name"])

	// 2. List tools
	resp = send(2, "tools/list", nil)
	require.Nil(t, resp["error"])
	result, ok = resp["result"].(map[string]any)
	require.True(t, ok)
	tools, ok := result["tools"].([]any)
	require.True(t, ok)
	assert.Len(t, tools, 15)

	// 3. New game
	resp = send(3, "tools/call", toolCallParams("new_game", map[string]any{
		"scenario_id": "minimal", "character_name": "E2E Hero", "character_class": "fighter",
	}))
	require.Nil(t, resp["error"])
	toolResult := extractE2EToolResult(t, resp)
	assert.Equal(t, "entrance", toolResult["room"])

	// 4. Look
	resp = send(4, "tools/call", toolCallParams("look", nil))
	require.Nil(t, resp["error"])
	toolResult = extractE2EToolResult(t, resp)
	assert.Equal(t, "Entrance Hall", toolResult["name"])

	// 5. Pick up sword
	resp = send(5, "tools/call", toolCallParams("pick_up", map[string]any{"item_id": "short_sword"}))
	require.Nil(t, resp["error"])
	toolResult = extractE2EToolResult(t, resp)
	item, ok := toolResult["item"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "short_sword", item["id"])

	// 6. Move south (triggers combat)
	resp = send(6, "tools/call", toolCallParams("move", map[string]any{"direction": "south"}))
	require.Nil(t, resp["error"])
	toolResult = extractE2EToolResult(t, resp)
	assert.Equal(t, "goblin_lair", toolResult["room"])
	_, hasCombat := toolResult["combat"]
	assert.True(t, hasCombat, "expected combat to start in goblin_lair")
}

// extractE2EToolResult extracts the parsed JSON from a tools/call response.
func extractE2EToolResult(t *testing.T, resp map[string]any) map[string]any {
	t.Helper()
	result, ok := resp["result"].(map[string]any)
	require.True(t, ok, "expected result object")
	content, ok := result["content"].([]any)
	require.True(t, ok, "expected content array")
	require.Len(t, content, 1)
	entry, ok := content[0].(map[string]any)
	require.True(t, ok)
	text, ok := entry["text"].(string)
	require.True(t, ok)
	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(text), &parsed))
	return parsed
}
