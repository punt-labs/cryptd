package testutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"
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
	delay     time.Duration // artificial delay before responding
}

// FakeSLMCall records a single request to the fake server.
type FakeSLMCall struct {
	Model       string            `json:"model"`
	Messages    []json.RawMessage `json:"messages"`
	Temperature *float64          `json:"temperature,omitempty"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
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

// SetDelay configures an artificial delay before each response. Use with
// a short client timeout to test timeout→fallback behavior.
func (f *FakeSLMServer) SetDelay(d time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.delay = d
}

func (f *FakeSLMServer) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Decode and record the request before any delay, so timeout calls
	// are still visible in Calls().
	var req struct {
		Model       string            `json:"model"`
		Messages    []json.RawMessage `json:"messages"`
		Temperature *float64          `json:"temperature,omitempty"`
		MaxTokens   int               `json:"max_tokens,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	f.mu.Lock()
	f.calls = append(f.calls, FakeSLMCall{
		Model:       req.Model,
		Messages:    req.Messages,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	})
	delay := f.delay

	// Reserve the response before delay so timed-out requests consume
	// their slot and keep ordering deterministic for later calls.
	var content string
	hasResponse := len(f.responses) > 0
	if hasResponse {
		content = f.responses[f.pos%len(f.responses)]
		f.pos++
	}
	f.mu.Unlock()

	if delay > 0 {
		timer := time.NewTimer(delay)
		select {
		case <-timer.C:
		case <-r.Context().Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return
		}
	}

	if !hasResponse {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

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
	for i, c := range f.calls {
		msgs := make([]json.RawMessage, len(c.Messages))
		for j, m := range c.Messages {
			if m != nil {
				b := make([]byte, len(m))
				copy(b, m)
				msgs[j] = json.RawMessage(b)
			}
		}
		var tempCopy *float64
		if c.Temperature != nil {
			v := *c.Temperature
			tempCopy = &v
		}
		out[i] = FakeSLMCall{
			Model:       c.Model,
			Messages:    msgs,
			Temperature: tempCopy,
			MaxTokens:   c.MaxTokens,
		}
	}
	return out
}
