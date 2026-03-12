// Package inference provides an OpenAI-compatible HTTP client for chat
// completions. It works with llama.cpp, ollama, and any server implementing
// the /v1/chat/completions endpoint.
package inference

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Role constants for chat messages.
const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

// Message is a single message in a chat conversation.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Options configures a single chat completion request.
type Options struct {
	// Temperature controls randomness (0.0 = deterministic, 1.0 = creative).
	Temperature *float64 `json:"temperature,omitempty"`
	// MaxTokens limits the response length.
	MaxTokens int `json:"max_tokens,omitempty"`
}

// Client calls an OpenAI-compatible /v1/chat/completions endpoint.
type Client struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewClient creates an inference client targeting the given base URL and model.
// The base URL should not include a trailing slash (e.g. "http://localhost:8080").
func NewClient(baseURL, model string, timeout time.Duration) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// chatRequest is the JSON body sent to /v1/chat/completions.
type chatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature *float64  `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

// chatResponse is the JSON body returned from /v1/chat/completions.
type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// ChatCompletion sends a chat completion request and returns the model's
// response text. Callers own JSON parsing of the response content.
func (c *Client) ChatCompletion(ctx context.Context, messages []Message, opts *Options) (string, error) {
	req := chatRequest{
		Model:    c.model,
		Messages: messages,
	}
	if opts != nil {
		req.Temperature = opts.Temperature
		req.MaxTokens = opts.MaxTokens
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("inference: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("inference: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("inference: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("inference: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("inference: server returned %d: %s", resp.StatusCode, truncate(respBody, 200))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("inference: unmarshal response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("inference: response contained no choices")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// BaseURL returns the client's configured base URL.
func (c *Client) BaseURL() string { return c.baseURL }

// Model returns the client's configured model name.
func (c *Client) Model() string { return c.model }

func truncate(b []byte, max int) string {
	if len(b) <= max {
		return string(b)
	}
	return string(b[:max]) + "..."
}
