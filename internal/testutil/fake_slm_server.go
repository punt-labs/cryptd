package testutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
)

// FakeSLMServer is an httptest.Server that serves OpenAI-compatible
// /v1/chat/completions responses. Use URL() to wire the inference.Client
// in tests.
type FakeSLMServer struct {
	mu        sync.Mutex
	responses []string // raw content strings to return
	pos       int
	calls     []FakeSLMCall
	server    *httptest.Server
}

// FakeSLMCall records a single request to the fake server.
type FakeSLMCall struct {
	Model    string            `json:"model"`
	Messages []json.RawMessage `json:"messages"`
}

// NewFakeSLMServer creates a FakeSLMServer that will return the provided
// content strings as chat completion responses, cycling when exhausted.
func NewFakeSLMServer(responses []string) *FakeSLMServer {
	f := &FakeSLMServer{responses: responses}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/completions", f.handleChatCompletions)
	mux.HandleFunc("/v1/models", f.handleModels)
	f.server = httptest.NewServer(mux)
	return f
}

func (f *FakeSLMServer) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Record the call.
	var req struct {
		Model    string            `json:"model"`
		Messages []json.RawMessage `json:"messages"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
		f.calls = append(f.calls, FakeSLMCall{Model: req.Model, Messages: req.Messages})
	}

	if len(f.responses) == 0 {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	content := f.responses[f.pos%len(f.responses)]
	f.pos++

	resp := map[string]any{
		"id":      "fake-completion",
		"object":  "chat.completion",
		"model":   req.Model,
		"choices": []map[string]any{{
			"index": 0,
			"message": map[string]string{
				"role":    "assistant",
				"content": content,
			},
			"finish_reason": "stop",
		}},
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (f *FakeSLMServer) handleModels(w http.ResponseWriter, _ *http.Request) {
	resp := map[string]any{
		"object": "list",
		"data": []map[string]string{{
			"id":     "fake-model",
			"object": "model",
		}},
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// URL returns the base URL of the fake server (e.g. "http://127.0.0.1:PORT").
func (f *FakeSLMServer) URL() string { return f.server.URL }

// Close shuts down the fake server.
func (f *FakeSLMServer) Close() { f.server.Close() }

// Calls returns a copy of all recorded requests.
func (f *FakeSLMServer) Calls() []FakeSLMCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]FakeSLMCall, len(f.calls))
	copy(out, f.calls)
	return out
}
