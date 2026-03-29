//go:build integration

package daemon

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/protocol"
	"github.com/punt-labs/cryptd/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testDMServer creates a normal-mode Server with FakeLLMInterpreter +
// FakeLLMNarrator, wired to the minimal scenario.
func testDMServer(t *testing.T, interp model.CommandInterpreter, narr model.Narrator) *Server {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	scenarioDir := filepath.Join(wd, "..", "..", "testdata", "scenarios")
	return NewServer(
		filepath.Join(t.TempDir(), "dm.sock"),
		scenarioDir,
		WithInterpreter(interp),
		WithNarrator(narr),
		WithDefaultScenario("minimal"),
	)
}

// dmRoundTrip sends all requests on a single connection (new_game first, then
// play requests) and collects all responses. This exercises the full normal-mode
// flow: new_game → game loop via RPCRenderer.
func dmRoundTrip(t *testing.T, srv *Server, reqs []Request) []Response {
	t.Helper()
	var input bytes.Buffer
	for _, req := range reqs {
		data, err := json.Marshal(req)
		require.NoError(t, err)
		input.Write(data)
		input.WriteByte('\n')
	}

	var output bytes.Buffer
	srv.handleConnection(&input, &output)

	var resps []Response
	scanner := bufio.NewScanner(&output)
	for scanner.Scan() {
		var resp Response
		require.NoError(t, json.Unmarshal(scanner.Bytes(), &resp))
		resps = append(resps, resp)
	}
	require.NoError(t, scanner.Err())
	return resps
}

func playRequest(id int, text string) Request {
	params, _ := json.Marshal(protocol.PlayRequest{Text: text})
	idJSON, _ := json.Marshal(id)
	return Request{
		JSONRPC: "2.0",
		ID:      idJSON,
		Method:  "play",
		Params:  params,
	}
}

func quitRequest(id int) Request {
	idJSON, _ := json.Marshal(id)
	return Request{
		JSONRPC: "2.0",
		ID:      idJSON,
		Method:  "quit",
	}
}

func TestDMDaemon_NewGameAndPlay(t *testing.T) {
	interp := testutil.NewFakeLLMInterpreter([]model.EngineAction{
		{Type: "look"},
		{Type: "quit"},
	})
	narr := testutil.NewFakeLLMNarrator([]string{
		"The entrance hall stretches before you.",
		"You look around the hall.",
		"Farewell, adventurer.",
	})

	srv := testDMServer(t, interp, narr)

	reqs := []Request{
		toolCall(1, "new_game", map[string]any{
			"scenario_id": "minimal", "character_name": "Hero", "character_class": "fighter",
		}),
		playRequest(2, "look around"),
		quitRequest(3),
	}

	resps := dmRoundTrip(t, srv, reqs)
	// new_game response + play response + quit response.
	require.GreaterOrEqual(t, len(resps), 2, "expected at least new_game + quit responses")

	// First response is new_game with narrated text.
	data, err := json.Marshal(resps[0].Result)
	require.NoError(t, err)
	var playResp PlayResponse
	require.NoError(t, json.Unmarshal(data, &playResp))
	assert.NotEmpty(t, playResp.Text)
}

func TestDMDaemon_PlayMovement(t *testing.T) {
	interp := testutil.NewFakeLLMInterpreter([]model.EngineAction{
		{Type: "move", Direction: "south"},
		// Combat in goblin_lair — need attacks.
		{Type: "attack"},
		{Type: "attack"},
		{Type: "attack"},
		{Type: "attack"},
		{Type: "attack"},
		{Type: "attack"},
		{Type: "attack"},
		{Type: "attack"},
		{Type: "quit"},
	})
	narr := testutil.NewFakeLLMNarrator([]string{
		"Narration.",
	})

	srv := testDMServer(t, interp, narr)

	reqs := []Request{
		toolCall(1, "new_game", map[string]any{
			"scenario_id": "minimal", "character_name": "Hero", "character_class": "fighter",
		}),
		playRequest(2, "go south"),
		playRequest(3, "attack"),
		playRequest(4, "attack"),
		playRequest(5, "attack"),
		playRequest(6, "attack"),
		playRequest(7, "attack"),
		playRequest(8, "attack"),
		playRequest(9, "attack"),
		playRequest(10, "attack"),
		quitRequest(11),
	}

	resps := dmRoundTrip(t, srv, reqs)
	require.GreaterOrEqual(t, len(resps), 2)

	// Verify all responses have no RPC errors.
	for i, resp := range resps {
		assert.Nil(t, resp.Error, "response %d has error: %+v", i, resp.Error)
	}
}
