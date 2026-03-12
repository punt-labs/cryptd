package inference_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/punt-labs/cryptd/internal/inference"
	"github.com/punt-labs/cryptd/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChatCompletion_HappyPath(t *testing.T) {
	srv := testutil.NewFakeSLMServer([]string{`{"action":"move","direction":"north"}`})
	defer srv.Close()

	client := inference.NewClient(srv.URL(), "smollm2:135m", 5*time.Second)
	resp, err := client.ChatCompletion(context.Background(), []inference.Message{
		{Role: inference.RoleSystem, Content: "You are a game parser."},
		{Role: inference.RoleUser, Content: "go north"},
	}, nil)

	require.NoError(t, err)
	assert.Equal(t, `{"action":"move","direction":"north"}`, resp)
}

func TestChatCompletion_SendsCorrectRequest(t *testing.T) {
	srv := testutil.NewFakeSLMServer([]string{"ok"})
	defer srv.Close()

	client := inference.NewClient(srv.URL(), "test-model", 5*time.Second)
	_, err := client.ChatCompletion(context.Background(), []inference.Message{
		{Role: inference.RoleSystem, Content: "sys"},
		{Role: inference.RoleUser, Content: "usr"},
	}, nil)
	require.NoError(t, err)

	calls := srv.Calls()
	require.Len(t, calls, 1)
	assert.Equal(t, "test-model", calls[0].Model)
	require.Len(t, calls[0].Messages, 2)

	var msg0, msg1 struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	require.NoError(t, json.Unmarshal(calls[0].Messages[0], &msg0))
	require.NoError(t, json.Unmarshal(calls[0].Messages[1], &msg1))
	assert.Equal(t, "system", msg0.Role)
	assert.Equal(t, "sys", msg0.Content)
	assert.Equal(t, "user", msg1.Role)
	assert.Equal(t, "usr", msg1.Content)
}

func TestChatCompletion_WithOptions(t *testing.T) {
	srv := testutil.NewFakeSLMServer([]string{"result"})
	defer srv.Close()

	temp := 0.1
	client := inference.NewClient(srv.URL(), "m", 5*time.Second)
	resp, err := client.ChatCompletion(context.Background(), []inference.Message{
		{Role: inference.RoleUser, Content: "test"},
	}, &inference.Options{Temperature: &temp, MaxTokens: 50})

	require.NoError(t, err)
	assert.Equal(t, "result", resp)
}

func TestChatCompletion_CyclesResponses(t *testing.T) {
	srv := testutil.NewFakeSLMServer([]string{"first", "second"})
	defer srv.Close()

	client := inference.NewClient(srv.URL(), "m", 5*time.Second)

	r1, err := client.ChatCompletion(context.Background(), []inference.Message{{Role: inference.RoleUser, Content: "a"}}, nil)
	require.NoError(t, err)
	assert.Equal(t, "first", r1)

	r2, err := client.ChatCompletion(context.Background(), []inference.Message{{Role: inference.RoleUser, Content: "b"}}, nil)
	require.NoError(t, err)
	assert.Equal(t, "second", r2)

	r3, err := client.ChatCompletion(context.Background(), []inference.Message{{Role: inference.RoleUser, Content: "c"}}, nil)
	require.NoError(t, err)
	assert.Equal(t, "first", r3)
}

func TestChatCompletion_EmptyResponses(t *testing.T) {
	srv := testutil.NewFakeSLMServer(nil)
	defer srv.Close()

	client := inference.NewClient(srv.URL(), "m", 5*time.Second)
	_, err := client.ChatCompletion(context.Background(), []inference.Message{{Role: inference.RoleUser, Content: "x"}}, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "503")
}

func TestChatCompletion_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"model not loaded"}`))
	}))
	defer srv.Close()

	client := inference.NewClient(srv.URL, "m", 5*time.Second)
	_, err := client.ChatCompletion(context.Background(), []inference.Message{{Role: inference.RoleUser, Content: "x"}}, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
	assert.Contains(t, err.Error(), "model not loaded")
}

func TestChatCompletion_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`not json`))
	}))
	defer srv.Close()

	client := inference.NewClient(srv.URL, "m", 5*time.Second)
	_, err := client.ChatCompletion(context.Background(), []inference.Message{{Role: inference.RoleUser, Content: "x"}}, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal")
}

func TestChatCompletion_EmptyChoices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	defer srv.Close()

	client := inference.NewClient(srv.URL, "m", 5*time.Second)
	_, err := client.ChatCompletion(context.Background(), []inference.Message{{Role: inference.RoleUser, Content: "x"}}, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no choices")
}

func TestChatCompletion_ContextCancelled(t *testing.T) {
	srv := testutil.NewFakeSLMServer([]string{"ok"})
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	client := inference.NewClient(srv.URL(), "m", 5*time.Second)
	_, err := client.ChatCompletion(ctx, []inference.Message{{Role: inference.RoleUser, Content: "x"}}, nil)

	require.Error(t, err)
}

func TestChatCompletion_Timeout(t *testing.T) {
	started := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		time.Sleep(2 * time.Second) // longer than client timeout
	}))
	defer srv.Close()

	client := inference.NewClient(srv.URL, "m", 50*time.Millisecond)
	_, err := client.ChatCompletion(context.Background(), []inference.Message{{Role: inference.RoleUser, Content: "x"}}, nil)

	<-started // ensure handler was entered
	require.Error(t, err)
}

func TestClientAccessors(t *testing.T) {
	client := inference.NewClient("http://localhost:8080", "smollm2:135m", 5*time.Second)
	assert.Equal(t, "http://localhost:8080", client.BaseURL())
	assert.Equal(t, "smollm2:135m", client.Model())
}
