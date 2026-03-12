package inference

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Runtime represents a discovered inference server.
type Runtime struct {
	Name    string // "ollama" or "llama.cpp"
	BaseURL string // e.g. "http://localhost:11434"
	Model   string // selected model ID
}

// PreferredModels lists models in descending priority for auto-detection.
// Medium tier (richer narration) is preferred over small tier (faster).
var PreferredModels = []string{
	"gemma3:1b",
	"llama3.2:3b",
	"smollm2:135m",
}

// defaultEndpoints are the well-known local endpoints for inference runtimes,
// in priority order (ollama first, then llama.cpp).
var defaultEndpoints = []Endpoint{
	{Name: "ollama", BaseURL: "http://localhost:11434", HealthPath: "/api/tags", ModelLister: OllamaModels, Preferred: PreferredModels},
	{Name: "llama.cpp", BaseURL: "http://localhost:8080", HealthPath: "/v1/models", ModelLister: OpenAIModels},
}

// DefaultEndpoints returns the well-known local endpoints for inference
// runtimes, in priority order (ollama first, then llama.cpp). Each call
// returns a deep copy safe to mutate.
func DefaultEndpoints() []Endpoint {
	eps := make([]Endpoint, len(defaultEndpoints))
	copy(eps, defaultEndpoints)
	for i := range eps {
		if len(eps[i].Preferred) > 0 {
			eps[i].Preferred = append([]string(nil), eps[i].Preferred...)
		}
	}
	return eps
}

// Endpoint describes how to probe a single inference runtime.
type Endpoint struct {
	Name        string
	BaseURL     string
	HealthPath  string
	ModelLister func([]byte) ([]string, error)
	Preferred   []string // ranked model preferences; first match wins
}

// Probe checks the given endpoints in order and returns the first responding
// runtime, or nil if none are available. Each endpoint gets its own timeout.
func Probe(ctx context.Context, endpoints []Endpoint, timeout time.Duration) *Runtime {
	for _, ep := range endpoints {
		if r := probeEndpoint(ctx, ep, timeout); r != nil {
			return r
		}
	}
	return nil
}

func probeEndpoint(ctx context.Context, ep Endpoint, timeout time.Duration) *Runtime {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	base, err := url.Parse(ep.BaseURL)
	if err != nil {
		return nil
	}
	base.Path = strings.TrimRight(base.Path, "/") + ep.HealthPath

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base.String(), nil)
	if err != nil {
		return nil
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	const maxBody = 1 << 20 // 1 MiB
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return nil
	}

	if ep.ModelLister == nil {
		return nil
	}

	models, err := ep.ModelLister(body)
	if err != nil || len(models) == 0 {
		return nil
	}

	model := selectModel(models, ep.Preferred)

	return &Runtime{
		Name:    ep.Name,
		BaseURL: ep.BaseURL,
		Model:   model,
	}
}

// selectModel returns the highest-priority preferred model that is available,
// or the first available model if no preferences match.
func selectModel(available []string, preferred []string) string {
	if len(preferred) > 0 {
		set := make(map[string]bool, len(available))
		for _, m := range available {
			set[m] = true
		}
		for _, p := range preferred {
			if set[p] {
				return p
			}
		}
	}
	return available[0]
}

// OllamaModels extracts all model names from an ollama /api/tags response.
func OllamaModels(body []byte) ([]string, error) {
	var resp struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse ollama response: %w", err)
	}
	models := make([]string, len(resp.Models))
	for i, m := range resp.Models {
		models[i] = m.Name
	}
	return models, nil
}

// OpenAIModels extracts all model IDs from an OpenAI-compatible
// /v1/models response (used by llama.cpp and compatible servers).
func OpenAIModels(body []byte) ([]string, error) {
	var resp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse openai models response: %w", err)
	}
	models := make([]string, len(resp.Data))
	for i, m := range resp.Data {
		models[i] = m.ID
	}
	return models, nil
}
