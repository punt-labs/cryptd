package inference_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/punt-labs/cryptd/internal/inference"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeOllamaServer returns an httptest.Server that serves /api/tags with
// the given model names.
func fakeOllamaServer(t *testing.T, models []string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/tags", func(w http.ResponseWriter, _ *http.Request) {
		type model struct {
			Name string `json:"name"`
		}
		ms := make([]model, len(models))
		for i, m := range models {
			ms[i] = model{Name: m}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"models": ms})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// fakeLlamaCppServer returns an httptest.Server that serves /v1/models with
// the given model IDs.
func fakeLlamaCppServer(t *testing.T, models []string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/models", func(w http.ResponseWriter, _ *http.Request) {
		type model struct {
			ID     string `json:"id"`
			Object string `json:"object"`
		}
		ms := make([]model, len(models))
		for i, m := range models {
			ms[i] = model{ID: m, Object: "model"}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"object": "list", "data": ms})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// closedServerURL returns a URL that is guaranteed to refuse connections.
func closedServerURL(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()
	return srv.URL
}

func TestProbe_OllamaFirst(t *testing.T) {
	ollama := fakeOllamaServer(t, []string{"smollm2:135m"})
	llama := fakeLlamaCppServer(t, []string{"SmolLM2-135M-Q4"})

	endpoints := []inference.Endpoint{
		{Name: "ollama", BaseURL: ollama.URL, HealthPath: "/api/tags", ModelExtractor: inference.OllamaModels},
		{Name: "llama.cpp", BaseURL: llama.URL, HealthPath: "/v1/models", ModelExtractor: inference.OpenAIModels},
	}

	r := inference.Probe(context.Background(), endpoints, time.Second)
	require.NotNil(t, r)
	assert.Equal(t, "ollama", r.Name)
	assert.Equal(t, ollama.URL, r.BaseURL)
	assert.Equal(t, "smollm2:135m", r.Model)
}

func TestProbe_FallsThrough(t *testing.T) {
	downURL := closedServerURL(t)
	llama := fakeLlamaCppServer(t, []string{"SmolLM2-135M-Q4"})

	endpoints := []inference.Endpoint{
		{Name: "ollama", BaseURL: downURL, HealthPath: "/api/tags", ModelExtractor: inference.OllamaModels},
		{Name: "llama.cpp", BaseURL: llama.URL, HealthPath: "/v1/models", ModelExtractor: inference.OpenAIModels},
	}

	r := inference.Probe(context.Background(), endpoints, time.Second)
	require.NotNil(t, r)
	assert.Equal(t, "llama.cpp", r.Name)
	assert.Equal(t, "SmolLM2-135M-Q4", r.Model)
}

func TestProbe_NoneAvailable(t *testing.T) {
	downURL1 := closedServerURL(t)
	downURL2 := closedServerURL(t)

	endpoints := []inference.Endpoint{
		{Name: "ollama", BaseURL: downURL1, HealthPath: "/api/tags", ModelExtractor: inference.OllamaModels},
		{Name: "llama.cpp", BaseURL: downURL2, HealthPath: "/v1/models", ModelExtractor: inference.OpenAIModels},
	}

	r := inference.Probe(context.Background(), endpoints, time.Second)
	assert.Nil(t, r)
}

func TestProbe_EmptyModelList(t *testing.T) {
	ollama := fakeOllamaServer(t, []string{})

	endpoints := []inference.Endpoint{
		{Name: "ollama", BaseURL: ollama.URL, HealthPath: "/api/tags", ModelExtractor: inference.OllamaModels},
	}

	r := inference.Probe(context.Background(), endpoints, time.Second)
	assert.Nil(t, r, "server with no models should not be returned")
}

func TestProbe_ContextCancelled(t *testing.T) {
	ollama := fakeOllamaServer(t, []string{"smollm2:135m"})

	endpoints := []inference.Endpoint{
		{Name: "ollama", BaseURL: ollama.URL, HealthPath: "/api/tags", ModelExtractor: inference.OllamaModels},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled

	r := inference.Probe(ctx, endpoints, time.Second)
	assert.Nil(t, r)
}

func TestProbe_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	endpoints := []inference.Endpoint{
		{Name: "broken", BaseURL: srv.URL, HealthPath: "/api/tags", ModelExtractor: inference.OllamaModels},
	}

	r := inference.Probe(context.Background(), endpoints, time.Second)
	assert.Nil(t, r)
}

func TestProbe_TrailingSlashInBaseURL(t *testing.T) {
	// Use a strict handler that only responds to the exact path /api/tags.
	// http.ServeMux auto-redirects //api/tags → /api/tags, which would mask
	// the bug. This handler rejects any path that isn't exactly /api/tags.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models":[{"name":"smollm2:135m"}]}`))
	}))
	t.Cleanup(srv.Close)

	// Trailing slash on BaseURL should not produce //api/tags.
	endpoints := []inference.Endpoint{
		{Name: "ollama", BaseURL: srv.URL + "/", HealthPath: "/api/tags", ModelExtractor: inference.OllamaModels},
	}

	r := inference.Probe(context.Background(), endpoints, time.Second)
	require.NotNil(t, r, "trailing slash must not break probe")
	assert.Equal(t, "smollm2:135m", r.Model)
}

func TestDefaultEndpoints_ReturnsCopy(t *testing.T) {
	a := inference.DefaultEndpoints()
	b := inference.DefaultEndpoints()
	a[0].Name = "mutated"
	assert.Equal(t, "ollama", b[0].Name, "mutation of one copy must not affect another")
}
