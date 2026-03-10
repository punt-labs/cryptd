package testutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
)

// FakeSLMServer is an httptest.Server that serves canned ollama JSON
// responses. Use URL() to wire the SLMInterpreter or SLMNarrator in tests.
type FakeSLMServer struct {
	mu        sync.Mutex
	responses []json.RawMessage
	pos       int
	server    *httptest.Server
}

// NewFakeSLMServer creates a FakeSLMServer that will return the provided JSON
// response bodies in order, cycling when exhausted.
// Each response should be a valid ollama /api/generate response object.
func NewFakeSLMServer(responses []json.RawMessage) *FakeSLMServer {
	f := &FakeSLMServer{responses: responses}
	f.server = httptest.NewServer(http.HandlerFunc(f.handle))
	return f
}

func (f *FakeSLMServer) handle(w http.ResponseWriter, _ *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.responses) == 0 {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	resp := f.responses[f.pos%len(f.responses)]
	f.pos++
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(resp)
}

// URL returns the base URL of the fake server (e.g. "http://127.0.0.1:PORT").
func (f *FakeSLMServer) URL() string { return f.server.URL }

// Close shuts down the fake server.
func (f *FakeSLMServer) Close() { f.server.Close() }
