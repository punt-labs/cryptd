package daemon

import (
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSessionID_Format(t *testing.T) {
	id, err := generateID()
	require.NoError(t, err)
	assert.Len(t, id, 32, "session ID should be 32 hex characters")

	_, decErr := hex.DecodeString(id)
	assert.NoError(t, decErr, "session ID should be valid hex")

	// Two calls should produce different IDs.
	id2, err := generateID()
	require.NoError(t, err)
	assert.NotEqual(t, id, id2, "session IDs should be unique")
}

func TestInitialize_AssignsSessionID(t *testing.T) {
	srv := testServer(t)
	idJSON, _ := json.Marshal(1)
	resp := roundTrip(t, srv, Request{JSONRPC: "2.0", ID: idJSON, Method: "initialize"})

	require.Nil(t, resp.Error)
	data, _ := json.Marshal(resp.Result)
	var init InitializeResult
	require.NoError(t, json.Unmarshal(data, &init))

	assert.NotEmpty(t, init.SessionID, "server should assign a session ID")
	assert.Len(t, init.SessionID, 32, "server-assigned session ID should be 32 hex chars")
}

func TestSanitizeSessionID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"valid", "abc-123", "abc-123"},
		{"empty", "", ""},
		{"too long", strings.Repeat("a", 129), ""},
		{"max length", strings.Repeat("a", 128), strings.Repeat("a", 128)},
		{"strips newlines", "abc\ndef", "abcdef"},
		{"strips tabs", "abc\tdef", "abcdef"},
		{"all control chars", "\n\r\t", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeSessionID(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestInitialize_EchoesClientSessionID(t *testing.T) {
	srv := testServer(t)
	idJSON, _ := json.Marshal(1)

	params, _ := json.Marshal(InitializeParams{SessionID: "test-123"})
	resp := roundTrip(t, srv, Request{
		JSONRPC: "2.0",
		ID:      idJSON,
		Method:  "initialize",
		Params:  params,
	})

	require.Nil(t, resp.Error)
	data, _ := json.Marshal(resp.Result)
	var init InitializeResult
	require.NoError(t, json.Unmarshal(data, &init))

	assert.Equal(t, "test-123", init.SessionID, "server should echo client-provided session ID")
}
