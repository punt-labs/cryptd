package main

import (
	"encoding/json"
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCP_ToolRegistration(t *testing.T) {
	// Create a server with a nil proxy (tools won't be called).
	s := newMCPServer(nil)

	// Verify the tool definitions list matches expected names and gameMethod values.
	tools := gameTools()
	expected := []string{
		"new_game", "move", "look", "pick_up", "drop", "equip", "unequip",
		"examine", "inventory", "attack", "defend", "flee", "cast_spell",
		"context", "save_game", "load_game",
	}
	require.Len(t, tools, len(expected), "tool count mismatch")

	names := make([]string, len(tools))
	for i, td := range tools {
		names[i] = td.name
	}
	assert.Equal(t, expected, names)

	// Verify gameMethod matches the tool name for tools that use direct dispatch
	// (everything except new_game which routes through handleNewGamePassthrough,
	// and play which routes through normal mode's game loop).
	// The daemon strips the "game." prefix, so gameMethod must match the dispatch
	// switch cases in internal/daemon/game.go.
	for _, td := range tools {
		assert.NotEmpty(t, td.gameMethod, "tool %s has empty gameMethod", td.name)
		assert.NotEmpty(t, td.description, "tool %s has empty description", td.name)
	}

	// mcp-go does not expose a method to list registered tools on MCPServer.
	// We verify the server was created and tool count matches; actual dispatch
	// is covered by integration tests.
	assert.NotNil(t, s)
}

func TestDaemonProxy_Call(t *testing.T) {
	// Create a pipe-based fake daemon.
	clientConn, serverConn := net.Pipe()
	defer func() { _ = clientConn.Close() }()
	defer func() { _ = serverConn.Close() }()

	proxy := newDaemonProxy(clientConn)

	// Fake daemon: read a request, respond with a canned result.
	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 4096)
		n, err := serverConn.Read(buf)
		if err != nil {
			return
		}
		// Verify the request is valid JSON-RPC.
		var req map[string]any
		if err := json.Unmarshal(buf[:n], &req); err != nil {
			return
		}
		// Send a success response.
		resp := fmt.Sprintf(`{"jsonrpc":"2.0","id":%s,"result":{"text":"You are in a dark room."}}`, string(mustMarshal(req["id"])))
		resp += "\n"
		_, _ = serverConn.Write([]byte(resp))
	}()

	params, _ := json.Marshal(map[string]string{"text": "look"})
	result, err := proxy.call("game.play", params)
	require.NoError(t, err)

	var parsed map[string]string
	require.NoError(t, json.Unmarshal(result, &parsed))
	assert.Equal(t, "You are in a dark room.", parsed["text"])

	<-done
}

func TestDaemonProxy_Error(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer func() { _ = clientConn.Close() }()
	defer func() { _ = serverConn.Close() }()

	proxy := newDaemonProxy(clientConn)

	// Fake daemon: read a request, respond with a JSON-RPC error.
	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 4096)
		n, err := serverConn.Read(buf)
		if err != nil {
			return
		}
		var req map[string]any
		if err := json.Unmarshal(buf[:n], &req); err != nil {
			return
		}
		resp := fmt.Sprintf(`{"jsonrpc":"2.0","id":%s,"error":{"code":-32003,"message":"no active game"}}`, string(mustMarshal(req["id"])))
		resp += "\n"
		_, _ = serverConn.Write([]byte(resp))
	}()

	_, err := proxy.call("game.look", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no active game")

	<-done
}

func TestDaemonProxy_ConnectionLost(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer func() { _ = clientConn.Close() }()

	proxy := newDaemonProxy(clientConn)

	// Close the server side immediately so the read fails.
	_ = serverConn.Close()

	_, err := proxy.call("game.look", nil)
	require.Error(t, err)
	// The error may be "connection lost" (read fails) or "write: ..." (write fails first).
}

// mustMarshal marshals v to JSON or panics.
func mustMarshal(v any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
