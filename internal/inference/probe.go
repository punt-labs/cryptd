package inference

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Runtime represents a discovered inference server.
type Runtime struct {
	Name    string // "ollama" or "llama.cpp"
	BaseURL string // e.g. "http://localhost:11434"
	Model   string // first available model ID
}

// defaultEndpoints are the well-known local endpoints for inference runtimes,
// in priority order (ollama first, then llama.cpp).
var defaultEndpoints = []Endpoint{
	{Name: "ollama", BaseURL: "http://localhost:11434", HealthPath: "/api/tags", ModelExtractor: OllamaModels},
	{Name: "llama.cpp", BaseURL: "http://localhost:8080", HealthPath: "/v1/models", ModelExtractor: OpenAIModels},
}

// DefaultEndpoints returns the well-known local endpoints for inference
// runtimes, in priority order (ollama first, then llama.cpp).
func DefaultEndpoints() []Endpoint {
	eps := make([]Endpoint, len(defaultEndpoints))
	copy(eps, defaultEndpoints)
	return eps
}

// Endpoint describes how to probe a single inference runtime.
type Endpoint struct {
	Name           string
	BaseURL        string
	HealthPath     string
	ModelExtractor func([]byte) (string, error)
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
	base.Path = base.Path + ep.HealthPath

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

	model, err := ep.ModelExtractor(body)
	if err != nil || model == "" {
		return nil
	}

	return &Runtime{
		Name:    ep.Name,
		BaseURL: ep.BaseURL,
		Model:   model,
	}
}

// OllamaModels extracts the first model name from an ollama /api/tags response.
func OllamaModels(body []byte) (string, error) {
	var resp struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("parse ollama response: %w", err)
	}
	if len(resp.Models) == 0 {
		return "", nil
	}
	return resp.Models[0].Name, nil
}

// OpenAIModels extracts the first model ID from an OpenAI-compatible
// /v1/models response (used by llama.cpp and compatible servers).
func OpenAIModels(body []byte) (string, error) {
	var resp struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("parse openai models response: %w", err)
	}
	if len(resp.Data) == 0 {
		return "", nil
	}
	return resp.Data[0].ID, nil
}
