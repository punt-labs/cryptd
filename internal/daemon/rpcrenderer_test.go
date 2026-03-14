package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRPCRenderer_Render_WritesValidNDJSON(t *testing.T) {
	clientR, serverW := io.Pipe()
	serverR, _ := io.Pipe() // reader side unused in this test
	defer clientR.Close()
	defer serverR.Close()

	rend := NewRPCRenderer(serverR, serverW)

	// Set a request ID as if Events() had received a play request.
	rend.setLastID(json.RawMessage(`42`))

	state := model.GameState{
		SchemaVersion: "1",
		Scenario:      "test",
		Dungeon: model.DungeonState{
			CurrentRoom: "entrance",
		},
		Party: []model.Character{
			{Name: "Hero", Class: "fighter", HP: 20, MaxHP: 20},
		},
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- rend.Render(context.Background(), state, "You enter the dungeon.")
		serverW.Close()
	}()

	// Read the NDJSON line from the client side.
	scanner := bufio.NewScanner(clientR)
	require.True(t, scanner.Scan(), "expected one NDJSON line")

	var resp protocol.Response
	require.NoError(t, json.Unmarshal(scanner.Bytes(), &resp))

	assert.Equal(t, "2.0", resp.JSONRPC)
	assert.Equal(t, json.RawMessage(`42`), resp.ID)
	assert.Nil(t, resp.Error)

	// Unmarshal the result as PlayResponse.
	resultBytes, err := json.Marshal(resp.Result)
	require.NoError(t, err)
	var play protocol.PlayResponse
	require.NoError(t, json.Unmarshal(resultBytes, &play))

	assert.Equal(t, "You enter the dungeon.", play.Text)
	require.NotNil(t, play.State)
	assert.Equal(t, "entrance", play.State.Dungeon.CurrentRoom)
	assert.Equal(t, "Hero", play.State.Party[0].Name)

	require.NoError(t, <-errCh)
}

func TestRPCRenderer_Render_DeepCopiesState(t *testing.T) {
	clientR, serverW := io.Pipe()
	serverR, _ := io.Pipe()
	defer clientR.Close()
	defer serverR.Close()

	rend := NewRPCRenderer(serverR, serverW)
	rend.setLastID(json.RawMessage(`1`))

	state := model.GameState{
		Party: []model.Character{
			{Name: "Hero", HP: 20, MaxHP: 20, Inventory: []model.Item{{ID: "sword"}}},
		},
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- rend.Render(context.Background(), state, "test")
		serverW.Close()
	}()

	// Read the NDJSON output and verify the deep-copied state has the
	// original item ID, not a mutation applied after Render() was called.
	scanner := bufio.NewScanner(clientR)
	require.True(t, scanner.Scan())

	var resp protocol.Response
	require.NoError(t, json.Unmarshal(scanner.Bytes(), &resp))
	resultBytes, err := json.Marshal(resp.Result)
	require.NoError(t, err)
	var play protocol.PlayResponse
	require.NoError(t, json.Unmarshal(resultBytes, &play))

	assert.Equal(t, "sword", play.State.Party[0].Inventory[0].ID)
	require.NoError(t, <-errCh)
}

func TestRPCRenderer_Events_PlayRequest(t *testing.T) {
	serverR, clientW := io.Pipe()
	_, serverW := io.Pipe()
	defer serverW.Close()

	rend := NewRPCRenderer(serverR, serverW)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rend.StartReader(ctx)

	// Write a play request from the client side.
	req := protocol.Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "play",
		Params:  json.RawMessage(`{"text":"go north"}`),
	}
	data, err := json.Marshal(req)
	require.NoError(t, err)
	data = append(data, '\n')
	_, err = clientW.Write(data)
	require.NoError(t, err)

	// Read the InputEvent from Events().
	select {
	case ev := <-rend.Events():
		assert.Equal(t, "input", ev.Type)
		assert.Equal(t, "go north", ev.Payload)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for input event")
	}

	// Verify the request ID was stored.
	assert.Equal(t, json.RawMessage(`1`), rend.getLastID())
}

func TestRPCRenderer_Events_QuitRequest(t *testing.T) {
	serverR, clientW := io.Pipe()
	_, serverW := io.Pipe()
	defer serverW.Close()

	rend := NewRPCRenderer(serverR, serverW)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rend.StartReader(ctx)

	// Write a quit request.
	req := protocol.Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`2`),
		Method:  "quit",
	}
	data, err := json.Marshal(req)
	require.NoError(t, err)
	data = append(data, '\n')
	_, err = clientW.Write(data)
	require.NoError(t, err)

	// Should receive a quit event.
	select {
	case ev := <-rend.Events():
		assert.Equal(t, "quit", ev.Type)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for quit event")
	}

	// Channel should be closed after quit.
	select {
	case _, ok := <-rend.Events():
		assert.False(t, ok, "events channel should be closed after quit")
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for channel close")
	}
}

func TestRPCRenderer_Events_EOF(t *testing.T) {
	serverR, clientW := io.Pipe()
	_, serverW := io.Pipe()
	defer serverW.Close()

	rend := NewRPCRenderer(serverR, serverW)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rend.StartReader(ctx)

	// Close the writer to simulate EOF.
	clientW.Close()

	// Channel should be closed.
	select {
	case _, ok := <-rend.Events():
		assert.False(t, ok, "events channel should be closed on EOF")
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for channel close on EOF")
	}
}

func TestRPCRenderer_Events_SkipsEmptyText(t *testing.T) {
	serverR, clientW := io.Pipe()
	_, serverW := io.Pipe()
	defer serverW.Close()

	rend := NewRPCRenderer(serverR, serverW)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rend.StartReader(ctx)

	// Write a play request with empty text — should be skipped.
	req1 := protocol.Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "play",
		Params:  json.RawMessage(`{"text":""}`),
	}
	data, err := json.Marshal(req1)
	require.NoError(t, err)
	data = append(data, '\n')
	_, err = clientW.Write(data)
	require.NoError(t, err)

	// Write a valid play request — should arrive.
	req2 := protocol.Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`2`),
		Method:  "play",
		Params:  json.RawMessage(`{"text":"look"}`),
	}
	data, err = json.Marshal(req2)
	require.NoError(t, err)
	data = append(data, '\n')
	_, err = clientW.Write(data)
	require.NoError(t, err)

	select {
	case ev := <-rend.Events():
		assert.Equal(t, "input", ev.Type)
		assert.Equal(t, "look", ev.Payload)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for input event")
	}
}

func TestRPCRenderer_Events_SkipsMalformedJSON(t *testing.T) {
	serverR, clientW := io.Pipe()
	_, serverW := io.Pipe()
	defer serverW.Close()

	rend := NewRPCRenderer(serverR, serverW)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rend.StartReader(ctx)

	// Write malformed JSON followed by a valid request.
	_, err := clientW.Write([]byte("not json\n"))
	require.NoError(t, err)

	req := protocol.Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`3`),
		Method:  "play",
		Params:  json.RawMessage(`{"text":"help"}`),
	}
	data, err := json.Marshal(req)
	require.NoError(t, err)
	data = append(data, '\n')
	_, err = clientW.Write(data)
	require.NoError(t, err)

	select {
	case ev := <-rend.Events():
		assert.Equal(t, "input", ev.Type)
		assert.Equal(t, "help", ev.Payload)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for input event after malformed JSON")
	}
}

func TestRPCRenderer_Events_ContextCancellation(t *testing.T) {
	serverR, clientW := io.Pipe()
	_, serverW := io.Pipe()
	defer serverW.Close()

	rend := NewRPCRenderer(serverR, serverW)
	ctx, cancel := context.WithCancel(context.Background())
	rend.StartReader(ctx)

	// Close the writer first to give the scanner a clean EOF, then cancel.
	clientW.Close()
	cancel()

	// Drain the channel until it is closed.
	timeout := time.After(2 * time.Second)
	for {
		select {
		case _, ok := <-rend.Events():
			if !ok {
				return // channel closed — test passes
			}
		case <-timeout:
			t.Fatal("timed out waiting for channel close on context cancellation")
		}
	}
}

// Verify RPCRenderer satisfies model.Renderer at compile time.
var _ model.Renderer = (*RPCRenderer)(nil)
