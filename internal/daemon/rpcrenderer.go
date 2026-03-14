package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/protocol"
)

// RPCRenderer implements model.Renderer over a JSON-RPC 2.0 / NDJSON
// connection. It bridges the game loop to the network: Render() writes
// PlayResponse NDJSON lines; Events() reads play requests and converts
// them to InputEvents.
//
// RPCRenderer only handles the play loop. Handshake methods (initialize,
// tools/list, new_game) are handled by the daemon before the game loop starts.
type RPCRenderer struct {
	scanner   *bufio.Scanner
	writer    io.Writer
	events    chan model.InputEvent
	startOnce sync.Once

	// mu guards lastID so the reader goroutine and Render() do not race.
	mu     sync.Mutex
	lastID json.RawMessage
}

// NewRPCRenderer creates an RPCRenderer that reads JSON-RPC requests from r
// and writes JSON-RPC responses to w.
func NewRPCRenderer(r io.Reader, w io.Writer) *RPCRenderer {
	scanner := bufio.NewScanner(r)
	// 1 MiB buffer to match the server/client convention.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	return &RPCRenderer{
		scanner: scanner,
		writer:  w,
		events:  make(chan model.InputEvent, 1),
	}
}

// Render writes a PlayResponse as a JSON-RPC 2.0 NDJSON line, echoing the
// request ID from the most recent Events() input.
func (r *RPCRenderer) Render(_ context.Context, state model.GameState, narration string) error {
	stateCopy := deepCopyState(&state)

	resp := protocol.Response{
		JSONRPC: "2.0",
		ID:      r.getLastID(),
		Result: protocol.PlayResponse{
			Text:  narration,
			State: stateCopy,
		},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("rpcrenderer: marshal response: %w", err)
	}
	data = append(data, '\n')
	_, err = r.writer.Write(data)
	if err != nil {
		return fmt.Errorf("rpcrenderer: write response: %w", err)
	}
	return nil
}

// Events returns the channel on which InputEvents are sent. A background
// goroutine (started once) reads JSON-RPC requests from the scanner and
// converts play requests into InputEvents.
func (r *RPCRenderer) Events() <-chan model.InputEvent {
	return r.events
}

// StartReader launches the background input reader. It must be called with a
// context that the game loop owns so the goroutine can observe cancellation.
// The game loop calls Events() to get the channel, but the reader goroutine
// needs explicit startup because it requires a context.
func (r *RPCRenderer) StartReader(ctx context.Context) {
	r.startOnce.Do(func() {
		go r.readLoop(ctx)
	})
}

// readLoop reads NDJSON lines from the scanner and dispatches play requests
// as InputEvents. It closes the events channel on EOF or context cancellation.
func (r *RPCRenderer) readLoop(ctx context.Context) {
	defer close(r.events)

	for r.scanner.Scan() {
		line := r.scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req protocol.Request
		if err := json.Unmarshal(line, &req); err != nil {
			// Malformed JSON — skip the line. The game loop does not have a
			// way to report protocol errors; those belong to the daemon layer.
			continue
		}

		switch req.Method {
		case "play":
			var params protocol.PlayRequest
			if err := json.Unmarshal(req.Params, &params); err != nil {
				continue
			}
			if params.Text == "" {
				continue
			}
			r.setLastID(req.ID)
			ev := model.InputEvent{Type: "input", Payload: params.Text}
			select {
			case r.events <- ev:
			case <-ctx.Done():
				return
			}

		case "quit":
			r.setLastID(req.ID)
			ev := model.InputEvent{Type: "quit"}
			select {
			case r.events <- ev:
			case <-ctx.Done():
			}
			return

		default:
			// Unknown method during the play loop — ignore.
		}
	}

	// Scanner stopped: EOF or read error.
	if err := r.scanner.Err(); err != nil {
		select {
		case r.events <- model.InputEvent{Type: "error", Payload: err.Error()}:
		case <-ctx.Done():
		}
	}
}

func (r *RPCRenderer) setLastID(id json.RawMessage) {
	r.mu.Lock()
	r.lastID = id
	r.mu.Unlock()
}

func (r *RPCRenderer) getLastID() json.RawMessage {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.lastID
}
