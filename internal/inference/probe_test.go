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
		{Name: "ollama", BaseURL: ollama.URL, HealthPath: "/api/tags", ModelLister: inference.OllamaModels},
		{Name: "llama.cpp", BaseURL: llama.URL, HealthPath: "/v1/models", ModelLister: inference.OpenAIModels},
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
		{Name: "ollama", BaseURL: downURL, HealthPath: "/api/tags", ModelLister: inference.OllamaModels},
		{Name: "llama.cpp", BaseURL: llama.URL, HealthPath: "/v1/models", ModelLister: inference.OpenAIModels},
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
		{Name: "ollama", BaseURL: downURL1, HealthPath: "/api/tags", ModelLister: inference.OllamaModels},
		{Name: "llama.cpp", BaseURL: downURL2, HealthPath: "/v1/models", ModelLister: inference.OpenAIModels},
	}

	r := inference.Probe(context.Background(), endpoints, time.Second)
	assert.Nil(t, r)
}

func TestProbe_EmptyModelList(t *testing.T) {
	ollama := fakeOllamaServer(t, []string{})

	endpoints := []inference.Endpoint{
		{Name: "ollama", BaseURL: ollama.URL, HealthPath: "/api/tags", ModelLister: inference.OllamaModels},
	}

	r := inference.Probe(context.Background(), endpoints, time.Second)
	assert.Nil(t, r, "server with no models should not be returned")
}

func TestProbe_ContextCancelled(t *testing.T) {
	ollama := fakeOllamaServer(t, []string{"smollm2:135m"})

	endpoints := []inference.Endpoint{
		{Name: "ollama", BaseURL: ollama.URL, HealthPath: "/api/tags", ModelLister: inference.OllamaModels},
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
		{Name: "broken", BaseURL: srv.URL, HealthPath: "/api/tags", ModelLister: inference.OllamaModels},
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
		{Name: "ollama", BaseURL: srv.URL + "/", HealthPath: "/api/tags", ModelLister: inference.OllamaModels},
	}

	r := inference.Probe(context.Background(), endpoints, time.Second)
	require.NotNil(t, r, "trailing slash must not break probe")
	assert.Equal(t, "smollm2:135m", r.Model)
}

func TestDefaultEndpoints_ReturnsCopy(t *testing.T) {
	a := inference.DefaultEndpoints()
	b := inference.DefaultEndpoints()

	require.NotEmpty(t, a, "default endpoints must contain at least one endpoint")
	require.NotEmpty(t, a[0].Preferred, "default endpoint must have at least one preferred model")

	// Mutating scalar field must not leak between copies.
	a[0].Name = "mutated"
	assert.Equal(t, "ollama", b[0].Name, "mutation of one copy must not affect another (Name)")

	// Mutating slice element must also not leak between copies.
	originalPreferred := b[0].Preferred[0]
	a[0].Preferred[0] = "mutated-model"
	assert.Equal(t, originalPreferred, b[0].Preferred[0], "mutation of one copy must not affect another (Preferred)")
}

func TestProbe_NilModelListerSkipsEndpoint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models":[{"name":"smollm2:135m"}]}`))
	}))
	t.Cleanup(srv.Close)

	endpoints := []inference.Endpoint{
		{Name: "no-lister", BaseURL: srv.URL, HealthPath: "/api/tags"},
	}

	r := inference.Probe(context.Background(), endpoints, time.Second)
	assert.Nil(t, r, "nil ModelLister should be treated as unavailable")
}

func TestProbe_PrefersHigherPriorityModel(t *testing.T) {
	// Server has multiple models; gemma3:1b should be selected over smollm2:135m.
	ollama := fakeOllamaServer(t, []string{"smollm2:135m", "gemma3:1b", "llama3.2:3b"})

	endpoints := []inference.Endpoint{
		{
			Name:        "ollama",
			BaseURL:     ollama.URL,
			HealthPath:  "/api/tags",
			ModelLister: inference.OllamaModels,
			Preferred:   inference.PreferredModels,
		},
	}

	r := inference.Probe(context.Background(), endpoints, time.Second)
	require.NotNil(t, r)
	assert.Equal(t, "gemma3:1b", r.Model, "should prefer gemma3:1b (highest priority)")
}

func TestProbe_FallsToSecondPreference(t *testing.T) {
	// Server has llama3.2:3b but not gemma3:1b.
	ollama := fakeOllamaServer(t, []string{"smollm2:135m", "llama3.2:3b"})

	endpoints := []inference.Endpoint{
		{
			Name:        "ollama",
			BaseURL:     ollama.URL,
			HealthPath:  "/api/tags",
			ModelLister: inference.OllamaModels,
			Preferred:   inference.PreferredModels,
		},
	}

	r := inference.Probe(context.Background(), endpoints, time.Second)
	require.NotNil(t, r)
	assert.Equal(t, "llama3.2:3b", r.Model, "should fall to second preference")
}

func TestProbe_NoPreferenceMatchUsesFirst(t *testing.T) {
	// Server has models not in the preference list.
	ollama := fakeOllamaServer(t, []string{"mistral:7b", "phi3:mini"})

	endpoints := []inference.Endpoint{
		{
			Name:        "ollama",
			BaseURL:     ollama.URL,
			HealthPath:  "/api/tags",
			ModelLister: inference.OllamaModels,
			Preferred:   inference.PreferredModels,
		},
	}

	r := inference.Probe(context.Background(), endpoints, time.Second)
	require.NotNil(t, r)
	assert.Equal(t, "mistral:7b", r.Model, "should fall back to first available model")
}

func TestProbe_NoPreferencesUsesFirst(t *testing.T) {
	// Endpoint with no Preferred list — just picks first model.
	ollama := fakeOllamaServer(t, []string{"phi3:mini", "gemma3:1b"})

	endpoints := []inference.Endpoint{
		{
			Name:        "ollama",
			BaseURL:     ollama.URL,
			HealthPath:  "/api/tags",
			ModelLister: inference.OllamaModels,
			// No Preferred field
		},
	}

	r := inference.Probe(context.Background(), endpoints, time.Second)
	require.NotNil(t, r)
	assert.Equal(t, "phi3:mini", r.Model, "with no preferences, should use first model")
}

func TestDefaultEndpoints_IncludesPreferences(t *testing.T) {
	eps := inference.DefaultEndpoints()
	require.Len(t, eps, 2)

	// Ollama endpoint should have model preferences.
	assert.Equal(t, "ollama", eps[0].Name)
	assert.NotEmpty(t, eps[0].Preferred, "ollama endpoint should have preferred models")
	assert.Equal(t, "gemma3:1b", eps[0].Preferred[0], "gemma3:1b should be top preference")

	// llama.cpp endpoint should have no preferences (single model loaded).
	assert.Equal(t, "llama.cpp", eps[1].Name)
	assert.Empty(t, eps[1].Preferred, "llama.cpp endpoint should have no preferences")
}
