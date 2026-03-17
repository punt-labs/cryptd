package lux

import (
	"encoding/json"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// miniServer spins up a Unix socket server that speaks the Lux wire protocol.
// It sends a ReadyMessage on connect and records received frames.
type miniServer struct {
	t        *testing.T
	listener net.Listener
	sockPath string
	received []map[string]any
	mu       sync.Mutex
	conn     net.Conn
	done     chan struct{}
}

func newMiniServer(t *testing.T) *miniServer {
	t.Helper()
	// Unix socket paths are limited to ~104 bytes on macOS.
	// t.TempDir() paths are too long; use /tmp with a short name.
	dir, err := os.MkdirTemp("/tmp", "lux-t-")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })
	sock := filepath.Join(dir, "t.sock")
	ln, err := net.Listen("unix", sock)
	require.NoError(t, err)
	t.Cleanup(func() { ln.Close() })
	return &miniServer{
		t:        t,
		listener: ln,
		sockPath: sock,
		done:     make(chan struct{}),
	}
}

func (s *miniServer) acceptAndHandshake() {
	s.t.Helper()
	conn, err := s.listener.Accept()
	require.NoError(s.t, err)
	s.conn = conn

	ready := ReadyMessage{
		Type:         "ready",
		Version:      ProtocolVersion,
		Capabilities: []string{"scene", "update"},
	}
	frame, err := EncodeFrame(ready)
	require.NoError(s.t, err)
	_, err = conn.Write(frame)
	require.NoError(s.t, err)
}

func (s *miniServer) readFrame() map[string]any {
	s.t.Helper()
	header := make([]byte, HeaderSize)
	_, err := io.ReadFull(s.conn, header)
	require.NoError(s.t, err)
	length := int(header[0])<<24 | int(header[1])<<16 | int(header[2])<<8 | int(header[3])
	payload := make([]byte, length)
	_, err = io.ReadFull(s.conn, payload)
	require.NoError(s.t, err)
	var msg map[string]any
	require.NoError(s.t, json.Unmarshal(payload, &msg))
	return msg
}

func (s *miniServer) sendFrame(msg any) {
	s.t.Helper()
	frame, err := EncodeFrame(msg)
	require.NoError(s.t, err)
	_, err = s.conn.Write(frame)
	require.NoError(s.t, err)
}

func (s *miniServer) close() {
	if s.conn != nil {
		s.conn.Close()
	}
	s.listener.Close()
}

// --- Protocol tests ---

func TestEncodeDecodeFrame(t *testing.T) {
	original := map[string]any{
		"type":    "ping",
		"ts":      1234.5,
		"message": "hello",
	}
	frame, err := EncodeFrame(original)
	require.NoError(t, err)

	// Verify header: payload length = total - 4 bytes
	assert.Equal(t, len(frame)-HeaderSize, len(frame)-4)

	decoded, rest, err := DecodeFrame(frame)
	require.NoError(t, err)
	assert.Empty(t, rest)
	assert.Equal(t, "ping", decoded["type"])
	assert.InDelta(t, 1234.5, decoded["ts"], 0.01)
}

func TestDecodeFrame_Incomplete(t *testing.T) {
	frame, err := EncodeFrame(map[string]any{"type": "ping"})
	require.NoError(t, err)

	// Only header, no payload
	_, _, err = DecodeFrame(frame[:HeaderSize])
	assert.ErrorIs(t, err, ErrIncompleteFrame)

	// Partial header
	_, _, err = DecodeFrame(frame[:2])
	assert.ErrorIs(t, err, ErrIncompleteFrame)
}

func TestFrameReader_MultipleMessages(t *testing.T) {
	f1, err := EncodeFrame(map[string]any{"type": "ping", "ts": 1.0})
	require.NoError(t, err)
	f2, err := EncodeFrame(map[string]any{"type": "clear"})
	require.NoError(t, err)

	var reader FrameReader
	// Feed both frames at once
	reader.Feed(append(f1, f2...))
	msgs, err := reader.Drain()
	require.NoError(t, err)
	assert.Len(t, msgs, 2)
	assert.Equal(t, "ping", msgs[0]["type"])
	assert.Equal(t, "clear", msgs[1]["type"])
}

func TestFrameReader_PartialFeed(t *testing.T) {
	frame, err := EncodeFrame(map[string]any{"type": "ping"})
	require.NoError(t, err)

	var reader FrameReader
	// Feed first half
	mid := len(frame) / 2
	reader.Feed(frame[:mid])
	msgs, err := reader.Drain()
	require.NoError(t, err)
	assert.Empty(t, msgs)

	// Feed second half
	reader.Feed(frame[mid:])
	msgs, err = reader.Drain()
	require.NoError(t, err)
	assert.Len(t, msgs, 1)
}

func TestParseInbound_Ready(t *testing.T) {
	raw := map[string]any{
		"type":         "ready",
		"version":      "0.1",
		"capabilities": []any{"scene", "update"},
	}
	parsed := ParseInbound(raw)
	ready, ok := parsed.(*ReadyMessage)
	require.True(t, ok)
	assert.Equal(t, "0.1", ready.Version)
	assert.Equal(t, []string{"scene", "update"}, ready.Capabilities)
}

func TestParseInbound_Ack(t *testing.T) {
	errStr := "test error"
	raw := map[string]any{
		"type":     "ack",
		"scene_id": "s1",
		"ts":       123.0,
		"error":    errStr,
	}
	parsed := ParseInbound(raw)
	ack, ok := parsed.(*AckMessage)
	require.True(t, ok)
	assert.Equal(t, "s1", ack.SceneID)
	require.NotNil(t, ack.Error)
	assert.Equal(t, "test error", *ack.Error)
}

func TestParseInbound_Interaction(t *testing.T) {
	raw := map[string]any{
		"type":       "interaction",
		"element_id": "act_attack",
		"action":     "clicked",
		"ts":         1.0,
	}
	parsed := ParseInbound(raw)
	inter, ok := parsed.(*InteractionMessage)
	require.True(t, ok)
	assert.Equal(t, "act_attack", inter.ElementID)
	assert.Equal(t, "clicked", inter.Action)
}

func TestParseInbound_Unknown(t *testing.T) {
	raw := map[string]any{"type": "window", "event": "resize"}
	assert.Nil(t, ParseInbound(raw))
}

// --- Client tests ---

func TestClient_ConnectHandshake(t *testing.T) {
	srv := newMiniServer(t)
	defer srv.close()

	c := NewClient(
		WithSocketPath(srv.sockPath),
		WithName("test-client"),
		WithConnectTimeout(2*time.Second),
	)

	go srv.acceptAndHandshake()

	err := c.Connect()
	require.NoError(t, err)
	defer c.Close()

	assert.True(t, c.IsConnected())

	// Server should receive ConnectMessage
	msg := srv.readFrame()
	assert.Equal(t, "connect", msg["type"])
	assert.Equal(t, "test-client", msg["name"])
}

func TestClient_ShowAck(t *testing.T) {
	srv := newMiniServer(t)
	defer srv.close()

	c := NewClient(
		WithSocketPath(srv.sockPath),
		WithRecvTimeout(2*time.Second),
	)

	go srv.acceptAndHandshake()
	require.NoError(t, c.Connect())
	defer c.Close()

	// Send show in background, read it on server, reply with ack
	var ack *AckMessage
	var showErr error
	done := make(chan struct{})
	go func() {
		defer close(done)
		ack, showErr = c.Show("scene1", []map[string]any{
			{"kind": "text", "id": "t1", "content": "Hello"},
		}, nil)
	}()

	// Server reads the scene message
	msg := srv.readFrame()
	assert.Equal(t, "scene", msg["type"])
	assert.Equal(t, "scene1", msg["id"])

	// Server sends ack
	srv.sendFrame(AckMessage{Type: "ack", SceneID: "scene1", TS: 1.0})

	<-done
	require.NoError(t, showErr)
	require.NotNil(t, ack)
	assert.Equal(t, "scene1", ack.SceneID)
}

func TestClient_UpdateAck(t *testing.T) {
	srv := newMiniServer(t)
	defer srv.close()

	c := NewClient(
		WithSocketPath(srv.sockPath),
		WithRecvTimeout(2*time.Second),
	)

	go srv.acceptAndHandshake()
	require.NoError(t, c.Connect())
	defer c.Close()

	var ack *AckMessage
	var updateErr error
	done := make(chan struct{})
	go func() {
		defer close(done)
		ack, updateErr = c.Update("scene1", []map[string]any{
			{"id": "narration", "set": map[string]any{"content": "updated"}},
		})
	}()

	msg := srv.readFrame()
	assert.Equal(t, "update", msg["type"])
	assert.Equal(t, "scene1", msg["scene_id"])

	srv.sendFrame(AckMessage{Type: "ack", SceneID: "scene1"})

	<-done
	require.NoError(t, updateErr)
	require.NotNil(t, ack)
}

func TestClient_RecvInteraction(t *testing.T) {
	srv := newMiniServer(t)
	defer srv.close()

	c := NewClient(
		WithSocketPath(srv.sockPath),
		WithRecvTimeout(2*time.Second),
	)

	go srv.acceptAndHandshake()
	require.NoError(t, c.Connect())
	defer c.Close()

	srv.sendFrame(InteractionMessage{
		Type:      "interaction",
		ElementID: "act_attack",
		Action:    "clicked",
		TS:        1.0,
	})

	inter, err := c.Recv(2 * time.Second)
	require.NoError(t, err)
	require.NotNil(t, inter)
	assert.Equal(t, "act_attack", inter.ElementID)
	assert.Equal(t, "clicked", inter.Action)
}

func TestClient_RecvTimeout(t *testing.T) {
	srv := newMiniServer(t)
	defer srv.close()

	c := NewClient(
		WithSocketPath(srv.sockPath),
		WithRecvTimeout(100*time.Millisecond),
	)

	go srv.acceptAndHandshake()
	require.NoError(t, c.Connect())
	defer c.Close()

	start := time.Now()
	inter, err := c.Recv(100 * time.Millisecond)
	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.Nil(t, inter)
	assert.Greater(t, elapsed, 50*time.Millisecond)
}

func TestClient_SocketPathResolution(t *testing.T) {
	// Env var takes precedence
	dir := t.TempDir()
	envPath := filepath.Join(dir, "custom.sock")
	t.Setenv("LUX_SOCKET", envPath)
	t.Setenv("XDG_RUNTIME_DIR", "")

	path := DefaultSocketPath()
	assert.Equal(t, envPath, path)

	// XDG fallback
	t.Setenv("LUX_SOCKET", "")
	xdgDir := filepath.Join(dir, "xdg")
	t.Setenv("XDG_RUNTIME_DIR", xdgDir)

	path = DefaultSocketPath()
	assert.Equal(t, filepath.Join(xdgDir, "lux", "display.sock"), path)

	// /tmp fallback
	t.Setenv("XDG_RUNTIME_DIR", "")
	user := os.Getenv("USER")
	if user == "" {
		user = "unknown"
	}
	path = DefaultSocketPath()
	assert.Equal(t, filepath.Join("/tmp", "lux-"+user, "display.sock"), path)
}

func TestClient_Ping(t *testing.T) {
	srv := newMiniServer(t)
	defer srv.close()

	c := NewClient(
		WithSocketPath(srv.sockPath),
		WithRecvTimeout(2*time.Second),
	)

	go srv.acceptAndHandshake()
	require.NoError(t, c.Connect())
	defer c.Close()

	done := make(chan error, 1)
	go func() {
		done <- c.Ping()
	}()

	msg := srv.readFrame()
	assert.Equal(t, "ping", msg["type"])

	// No pong message type in our protocol — ping is fire-and-forget.
	// The client just sends it; no response expected.
	err := <-done
	require.NoError(t, err)
}

func TestClient_ShowAckTimeout(t *testing.T) {
	srv := newMiniServer(t)
	defer srv.close()

	c := NewClient(
		WithSocketPath(srv.sockPath),
		WithRecvTimeout(200*time.Millisecond),
	)

	go srv.acceptAndHandshake()
	require.NoError(t, c.Connect())
	defer c.Close()

	// Server reads the scene but never sends an ack
	var ack *AckMessage
	var showErr error
	done := make(chan struct{})
	go func() {
		defer close(done)
		ack, showErr = c.Show("scene1", []map[string]any{
			{"kind": "text", "id": "t1", "content": "Hello"},
		}, nil)
	}()

	srv.readFrame() // consume the scene

	<-done
	assert.ErrorIs(t, showErr, ErrAckTimeout)
	assert.Nil(t, ack)
}

func TestClient_Clear(t *testing.T) {
	srv := newMiniServer(t)
	defer srv.close()

	c := NewClient(
		WithSocketPath(srv.sockPath),
		WithRecvTimeout(2*time.Second),
	)

	go srv.acceptAndHandshake()
	require.NoError(t, c.Connect())
	defer c.Close()

	require.NoError(t, c.Clear())

	msg := srv.readFrame()
	assert.Equal(t, "clear", msg["type"])
}
