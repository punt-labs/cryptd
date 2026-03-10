package testutil

import (
	"sync"

	"github.com/punt-labs/cryptd/internal/model"
)

// LuxCall records a single call made by LuxRenderer to the Lux MCP tool.
type LuxCall struct {
	Method  string // "show" or "update"
	Payload any
}

// FakeLuxServer is an in-process test double for the Lux MCP display server.
// It records every show() and update() call and can inject synthetic
// InputEvents to drive the game loop in tests.
type FakeLuxServer struct {
	mu     sync.Mutex
	calls  []LuxCall
	events chan model.InputEvent
}

// NewFakeLuxServer creates a FakeLuxServer with a buffered event channel.
func NewFakeLuxServer() *FakeLuxServer {
	return &FakeLuxServer{
		events: make(chan model.InputEvent, 64),
	}
}

// RecordShow records a Lux show() call with the given payload.
func (f *FakeLuxServer) RecordShow(payload any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, LuxCall{Method: "show", Payload: payload})
}

// RecordUpdate records a Lux update() call with the given payload.
func (f *FakeLuxServer) RecordUpdate(payload any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, LuxCall{Method: "update", Payload: payload})
}

// Calls returns a snapshot of all recorded calls.
func (f *FakeLuxServer) Calls() []LuxCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]LuxCall, len(f.calls))
	copy(out, f.calls)
	return out
}

// InjectEvent queues a synthetic InputEvent for the renderer to consume.
func (f *FakeLuxServer) InjectEvent(e model.InputEvent) {
	f.events <- e
}

// Events returns the channel of injected InputEvents.
func (f *FakeLuxServer) Events() <-chan model.InputEvent {
	return f.events
}
