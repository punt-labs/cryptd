package daemon

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/punt-labs/cryptd/internal/interpreter"
	"github.com/punt-labs/cryptd/internal/narrator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testNormalServer creates a normal-mode Server with Rules interpreter and
// Template narrator (no SLM/inference dependency).
func testNormalServer(t *testing.T) *Server {
	t.Helper()
	scenarioDir := filepath.Join(repoRoot(t), "testdata", "scenarios")
	rules := interpreter.NewRules()
	tmpl := narrator.NewTemplate()
	interp := interpreter.NewSLM(nil, rules)
	narr := narrator.NewSLM(nil, tmpl)
	return NewServer(
		filepath.Join(t.TempDir(), "test.sock"),
		scenarioDir,
		WithInterpreter(interp),
		WithNarrator(narr),
	)
}

func playReq(id int, text string) Request {
	params, _ := json.Marshal(PlayRequest{Text: text})
	idJSON, _ := json.Marshal(id)
	return Request{
		JSONRPC: "2.0",
		ID:      idJSON,
		Method:  "play",
		Params:  params,
	}
}

func TestPlayNoGame(t *testing.T) {
	srv := testNormalServer(t)
	resp := roundTrip(t, srv, playReq(1, "look around"))
	require.NotNil(t, resp.Error)
	assert.Equal(t, CodeNoActiveGame, resp.Error.Code)
}

func TestPlayRequiresText(t *testing.T) {
	srv := testNormalServer(t)
	resp := roundTrip(t, srv, playReq(1, ""))
	require.NotNil(t, resp.Error)
	assert.Equal(t, CodeInvalidParams, resp.Error.Code)
}

func TestPlayNewGameAndLook(t *testing.T) {
	srv := testNormalServer(t)

	// In normal mode, new_game via tools/call is intercepted and returns narrated text.
	newGameReq := toolCall(1, "new_game", map[string]any{
		"scenario_id":     "minimal",
		"character_name":  "Tester",
		"character_class": "fighter",
	})
	resp := roundTrip(t, srv, newGameReq)
	require.Nil(t, resp.Error, "new_game error: %+v", resp.Error)

	var result PlayResponse
	data, err := json.Marshal(resp.Result)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &result))

	// Normal mode returns narrated text and full game state.
	assert.NotEmpty(t, result.Text, "expected narrated text from new_game")
	assert.NotEmpty(t, result.State.Party, "expected party in game state from new_game")

	// Now play "look"
	lookResp := roundTrip(t, srv, playReq(2, "look"))
	require.Nil(t, lookResp.Error, "play look error: %+v", lookResp.Error)

	var lookResult PlayResponse
	data, err = json.Marshal(lookResp.Result)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &lookResult))
	assert.NotEmpty(t, lookResult.Text, "expected narrated text from look")
}

func TestPlayMove(t *testing.T) {
	srv := testNormalServer(t)

	reqs := []Request{
		toolCall(1, "new_game", map[string]any{
			"scenario_id": "minimal", "character_name": "Tester", "character_class": "fighter",
		}),
	}
	resps := multiRoundTrip(t, srv, reqs)
	require.Len(t, resps, 1)
	require.Nil(t, resps[0].Error, "new_game error: %+v", resps[0].Error)

	// Move south via play.
	moveResp := roundTrip(t, srv, playReq(2, "go south"))
	require.Nil(t, moveResp.Error, "play move error: %+v", moveResp.Error)

	var result PlayResponse
	data, err := json.Marshal(moveResp.Result)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &result))
	assert.NotEmpty(t, result.Text, "expected narrated text from move")
}

func TestPlayPassthroughBlocked(t *testing.T) {
	srv := testServer(t) // passthrough mode
	resp := roundTrip(t, srv, playReq(1, "look"))
	require.NotNil(t, resp.Error)
	assert.Equal(t, CodeMethodNotFound, resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "passthrough")
}
