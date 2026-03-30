package daemon

import (
	"encoding/json"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListSessions_Empty(t *testing.T) {
	srv := testServer(t)

	resp := roundTrip(t, srv, Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "game.list_sessions",
	})
	require.Nil(t, resp.Error)

	data, err := json.Marshal(resp.Result)
	require.NoError(t, err)
	var result ListSessionsResult
	require.NoError(t, json.Unmarshal(data, &result))
	assert.Equal(t, []SessionInfo{}, result.Sessions)
}

func TestListSessions_MultipleSessions(t *testing.T) {
	srv := testServer(t)

	// Start two games on two different sessions.
	sess1Reqs := []Request{
		initRequestWithSession(0, "session-alpha"),
		newGameCall(1, map[string]any{
			"scenario_id":     "minimal",
			"character_name":  "Kael",
			"character_class": "mage",
		}),
	}
	resps1 := multiRoundTrip(t, srv, sess1Reqs)
	require.Len(t, resps1, 2)
	require.Nil(t, resps1[0].Error, "session.init failed")
	require.Nil(t, resps1[1].Error, "game.new failed: %+v", resps1[1].Error)

	sess2Reqs := []Request{
		initRequestWithSession(0, "session-beta"),
		newGameCall(1, map[string]any{
			"scenario_id":     "minimal",
			"character_name":  "Liora",
			"character_class": "fighter",
		}),
	}
	resps2 := multiRoundTrip(t, srv, sess2Reqs)
	require.Len(t, resps2, 2)
	require.Nil(t, resps2[0].Error, "session.init failed")
	require.Nil(t, resps2[1].Error, "game.new failed: %+v", resps2[1].Error)

	// Third connection: list sessions without starting a game.
	listReqs := []Request{
		initRequestWithSession(0, "session-gamma"),
		{
			JSONRPC: "2.0",
			ID:      json.RawMessage(`1`),
			Method:  "game.list_sessions",
		},
	}
	resps3 := multiRoundTrip(t, srv, listReqs)
	require.Len(t, resps3, 2)
	require.Nil(t, resps3[1].Error)

	data, err := json.Marshal(resps3[1].Result)
	require.NoError(t, err)
	var result ListSessionsResult
	require.NoError(t, json.Unmarshal(data, &result))
	require.Len(t, result.Sessions, 2)

	// Sort by session ID for deterministic comparison.
	sort.Slice(result.Sessions, func(i, j int) bool {
		return result.Sessions[i].ID < result.Sessions[j].ID
	})

	alpha := result.Sessions[0]
	assert.Equal(t, "session-alpha", alpha.ID)
	assert.Equal(t, "minimal", alpha.ScenarioID)
	assert.Equal(t, "Kael", alpha.CharacterName)
	assert.Equal(t, "mage", alpha.CharacterClass)
	assert.Equal(t, 1, alpha.Level)
	assert.Equal(t, "entrance", alpha.RoomName)

	beta := result.Sessions[1]
	assert.Equal(t, "session-beta", beta.ID)
	assert.Equal(t, "minimal", beta.ScenarioID)
	assert.Equal(t, "Liora", beta.CharacterName)
	assert.Equal(t, "fighter", beta.CharacterClass)
	assert.Equal(t, 1, beta.Level)
	assert.Equal(t, "entrance", beta.RoomName)
}

func TestListSessions_ExcludesSessionsWithoutGame(t *testing.T) {
	srv := testServer(t)

	// Session with a game.
	withGame := []Request{
		initRequestWithSession(0, "has-game"),
		newGameCall(1, map[string]any{
			"scenario_id":     "minimal",
			"character_name":  "Hero",
			"character_class": "thief",
		}),
	}
	resps := multiRoundTrip(t, srv, withGame)
	require.Nil(t, resps[1].Error)

	// Session without a game.
	noGame := []Request{
		initRequestWithSession(0, "no-game"),
	}
	multiRoundTrip(t, srv, noGame)

	// List should only return the session with a game.
	listReqs := []Request{{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "game.list_sessions",
	}}
	listResps := multiRoundTrip(t, srv, listReqs)
	require.Len(t, listResps, 1)
	require.Nil(t, listResps[0].Error)

	data, _ := json.Marshal(listResps[0].Result)
	var result ListSessionsResult
	require.NoError(t, json.Unmarshal(data, &result))
	require.Len(t, result.Sessions, 1)
	assert.Equal(t, "has-game", result.Sessions[0].ID)
}

func TestListSessions_NoSessionRequired(t *testing.T) {
	srv := testServer(t)

	// Call list_sessions without session.init first.
	resp := roundTrip(t, srv, Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "game.list_sessions",
	})
	require.Nil(t, resp.Error, "list_sessions should work without session.init")
}
