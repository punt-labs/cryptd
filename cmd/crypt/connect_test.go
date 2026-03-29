package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net"
	"testing"

	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSession_ResumeReadsInitialRoom verifies that the session() function,
// when called with a resumeSessionID, reads and displays the initial room
// response that the server sends after initialize.
func TestSession_ResumeReadsInitialRoom(t *testing.T) {
	// Create a pipe-based mock server.
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	const testSessionID = "test-resume-session-abc123"

	// Run the mock server in a goroutine: it reads requests and writes
	// canned responses to simulate a cryptd server in resume mode.
	go func() {
		defer serverConn.Close()
		scanner := bufio.NewScanner(serverConn)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

		writeResp := func(resp any) {
			data, _ := json.Marshal(resp)
			data = append(data, '\n')
			serverConn.Write(data)
		}

		// 1. Read and respond to initialize.
		if !scanner.Scan() {
			return
		}
		var initReq map[string]any
		json.Unmarshal(scanner.Bytes(), &initReq)

		writeResp(map[string]any{
			"jsonrpc": "2.0",
			"id":      initReq["id"],
			"result": protocol.InitializeResult{
				ProtocolVersion: "2024-11-05",
				ServerInfo:      protocol.ServerInfo{Name: "cryptd", Version: "0.1.0"},
				Capabilities:    map[string]any{"tools": map[string]any{}},
				SessionID:       testSessionID,
			},
		})

		// 2. The server sends the initial room render (unsolicited, from RunLoop).
		writeResp(map[string]any{
			"jsonrpc": "2.0",
			"id":      nil,
			"result": protocol.PlayResponse{
				Text: "You are in a dark cave. Water drips from the ceiling.",
				State: &model.GameState{
					Party: []model.Character{
						{Name: "Hero", Class: "fighter", HP: 20, MaxHP: 20},
					},
					Dungeon: model.DungeonState{
						CurrentRoom: "dark_cave",
					},
				},
			},
		})

		// 3. The client enters the REPL. Since stdin is closed (io.Reader
		// returns EOF), readline/scanner fails and session() returns.
		// Drain any remaining reads so the pipe doesn't block.
		for scanner.Scan() {
			// Read and discard any further requests.
		}
	}()

	// Call session() with the resume session ID. Use an empty reader for
	// stdin so the REPL exits immediately.
	var out bytes.Buffer
	var errOut bytes.Buffer

	code := session(clientConn, io.LimitReader(bytes.NewReader(nil), 0), &out, &errOut,
		"", "", "", testSessionID)

	// The session should exit cleanly (code 0) because stdin is empty.
	assert.Equal(t, 0, code, "session should exit cleanly; stderr: %s", errOut.String())

	// Verify the initial room was displayed.
	output := out.String()
	assert.Contains(t, output, "dark_cave", "should display the room header")
	assert.Contains(t, output, "You are in a dark cave", "should display the narration text")

	// Verify the session ID was printed to stderr.
	require.Contains(t, errOut.String(), testSessionID, "should print session ID to stderr")
}
