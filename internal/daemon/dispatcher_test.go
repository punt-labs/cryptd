package daemon

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/punt-labs/cryptd/internal/engine"
	"github.com/punt-labs/cryptd/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testServer creates a Server wired to the minimal scenario in testdata.
// No interpreter or narrator is configured, so sessions default to passthrough
// mode (the server has no normal-mode capability).
func testServer(t *testing.T) *Server {
	t.Helper()
	// Find testdata relative to repo root.
	scenarioDir := filepath.Join(repoRoot(t), "testdata", "scenarios")
	return NewServer(filepath.Join(t.TempDir(), "test.sock"), scenarioDir)
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	// internal/daemon/ → repo root
	return filepath.Join(wd, "..", "..")
}

// initRequest returns a session.init request with the given ID.
func initRequest(id int) Request {
	idJSON, _ := json.Marshal(id)
	return Request{JSONRPC: "2.0", ID: idJSON, Method: "session.init"}
}

// initRequestWithSession returns a session.init request that provides a session ID.
func initRequestWithSession(id int, sessionID string) Request {
	idJSON, _ := json.Marshal(id)
	params, _ := json.Marshal(InitializeParams{SessionID: sessionID})
	return Request{JSONRPC: "2.0", ID: idJSON, Method: "session.init", Params: params}
}

// roundTrip sends a JSON-RPC request and reads the response.
// Uses bytes.Buffer — no goroutines or pipes needed.
func roundTrip(t *testing.T, srv *Server, req Request) Response {
	t.Helper()
	resps := multiRoundTrip(t, srv, []Request{req})
	require.Len(t, resps, 1)
	return resps[0]
}

// multiRoundTrip sends multiple requests on a single connection and returns responses in order.
// All requests are written to a buffer, then handleConnection reads them synchronously.
func multiRoundTrip(t *testing.T, srv *Server, reqs []Request) []Response {
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
	return resps
}

// gameCall builds a direct game.* method request.
// command is the short name (e.g. "move", "look") which gets prefixed with "game.".
func gameCall(id int, command string, args any) Request {
	var params json.RawMessage
	if args != nil {
		params, _ = json.Marshal(args)
	}
	idJSON, _ := json.Marshal(id)
	return Request{
		JSONRPC: "2.0",
		ID:      idJSON,
		Method:  "game." + command,
		Params:  params,
	}
}

// newGameCall builds a game.new request with the standard new-game arguments.
func newGameCall(id int, args map[string]any) Request {
	params, _ := json.Marshal(args)
	idJSON, _ := json.Marshal(id)
	return Request{
		JSONRPC: "2.0",
		ID:      idJSON,
		Method:  "game.new",
		Params:  params,
	}
}

// extractResult extracts the direct JSON result from a response.
func extractResult(t *testing.T, resp Response) map[string]any {
	t.Helper()
	require.Nil(t, resp.Error, "unexpected RPC error: %+v", resp.Error)

	data, err := json.Marshal(resp.Result)
	require.NoError(t, err)
	var result map[string]any
	require.NoError(t, json.Unmarshal(data, &result))
	return result
}

// extractSessionID pulls the session ID from a session.init response.
func extractSessionID(t *testing.T, resp Response) string {
	t.Helper()
	require.Nil(t, resp.Error)
	data, _ := json.Marshal(resp.Result)
	var init InitializeResult
	require.NoError(t, json.Unmarshal(data, &init))
	require.NotEmpty(t, init.SessionID)
	return init.SessionID
}

// activeGame returns the first (and typically only) game in the server registry.
func activeGame(t *testing.T, srv *Server) *Game {
	t.Helper()
	srv.mu.RLock()
	defer srv.mu.RUnlock()
	for _, g := range srv.games {
		return g
	}
	t.Fatal("no active game in server registry")
	return nil
}

func TestInitialize(t *testing.T) {
	srv := testServer(t)
	idJSON, _ := json.Marshal(1)
	resp := roundTrip(t, srv, Request{JSONRPC: "2.0", ID: idJSON, Method: "session.init"})

	require.Nil(t, resp.Error)
	data, _ := json.Marshal(resp.Result)
	var init InitializeResult
	require.NoError(t, json.Unmarshal(data, &init))
	assert.Equal(t, "cryptd", init.ServerInfo.Name)
	assert.NotEmpty(t, init.ProtocolVersion)
}

func TestListScenarios(t *testing.T) {
	srv := testServer(t)
	// game.list_scenarios works without a session — send session.init first
	// (required by our handler routing), then list_scenarios.
	reqs := []Request{
		initRequest(0),
		{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: "game.list_scenarios"},
	}
	resps := multiRoundTrip(t, srv, reqs)
	require.Len(t, resps, 2)
	require.Nil(t, resps[1].Error)

	data, err := json.Marshal(resps[1].Result)
	require.NoError(t, err)
	var result ListScenariosResult
	require.NoError(t, json.Unmarshal(data, &result))

	// Should contain at least "minimal".
	var ids []string
	for _, s := range result.Scenarios {
		ids = append(ids, s.ID)
	}
	assert.Contains(t, ids, "minimal")

	// Verify "minimal" has the correct title.
	for _, s := range result.Scenarios {
		if s.ID == "minimal" {
			assert.Equal(t, "Minimal Dungeon", s.Title)
		}
	}
}

func TestNewGame(t *testing.T) {
	srv := testServer(t)
	reqs := []Request{
		initRequest(0),
		newGameCall(1, map[string]any{
			"scenario_id":     "minimal",
			"character_name":  "Tester",
			"character_class": "fighter",
		}),
	}
	resps := multiRoundTrip(t, srv, reqs)
	require.Len(t, resps, 2)
	result := extractResult(t, resps[1])
	assert.Equal(t, "entrance", result["room"])
	assert.Equal(t, "Entrance Hall", result["name"])

	hero, ok := result["hero"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Tester", hero["name"])
	assert.Equal(t, "fighter", hero["class"])
}

func TestNoActiveGame(t *testing.T) {
	srv := testServer(t)
	reqs := []Request{
		initRequest(0),
		gameCall(1, "look", nil),
	}
	resps := multiRoundTrip(t, srv, reqs)
	require.Len(t, resps, 2)

	require.NotNil(t, resps[1].Error)
	assert.Contains(t, resps[1].Error.Message, "no active game")
}

func TestLook(t *testing.T) {
	srv := testServer(t)
	reqs := []Request{
		initRequest(0),
		newGameCall(1, map[string]any{
			"scenario_id": "minimal", "character_name": "Tester", "character_class": "fighter",
		}),
		gameCall(2, "look", nil),
	}
	resps := multiRoundTrip(t, srv, reqs)
	require.Len(t, resps, 3)

	result := extractResult(t, resps[2])
	assert.Equal(t, "entrance", result["room"])
	assert.Equal(t, "Entrance Hall", result["name"])
}

func TestMove(t *testing.T) {
	srv := testServer(t)
	reqs := []Request{
		initRequest(0),
		newGameCall(1, map[string]any{
			"scenario_id": "minimal", "character_name": "Tester", "character_class": "fighter",
		}),
		gameCall(2, "move", map[string]any{"direction": "south"}),
	}
	resps := multiRoundTrip(t, srv, reqs)
	require.Len(t, resps, 3)

	result := extractResult(t, resps[2])
	assert.Equal(t, "goblin_lair", result["room"])
	// Should have auto-started combat (goblin_lair has enemies).
	combat, ok := result["combat"].(map[string]any)
	require.True(t, ok, "expected combat to start")
	enemies, ok := combat["enemies"].([]any)
	require.True(t, ok)
	assert.Len(t, enemies, 1)
}

func TestMoveNoExit(t *testing.T) {
	srv := testServer(t)
	reqs := []Request{
		initRequest(0),
		newGameCall(1, map[string]any{
			"scenario_id": "minimal", "character_name": "Tester", "character_class": "fighter",
		}),
		gameCall(2, "move", map[string]any{"direction": "east"}),
	}
	resps := multiRoundTrip(t, srv, reqs)
	require.Len(t, resps, 3)

	require.NotNil(t, resps[2].Error)
	assert.Contains(t, resps[2].Error.Message, "no exit")
}

func TestPickUpAndDrop(t *testing.T) {
	srv := testServer(t)
	reqs := []Request{
		initRequest(0),
		newGameCall(1, map[string]any{
			"scenario_id": "minimal", "character_name": "Tester", "character_class": "fighter",
		}),
		gameCall(2, "pick_up", map[string]any{"item_id": "short_sword"}),
		gameCall(3, "drop", map[string]any{"item_id": "short_sword"}),
	}
	resps := multiRoundTrip(t, srv, reqs)
	require.Len(t, resps, 4)

	pickResult := extractResult(t, resps[2])
	item, ok := pickResult["item"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "short_sword", item["id"])

	dropResult := extractResult(t, resps[3])
	item, ok = dropResult["item"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "short_sword", item["id"])
}

func TestEquipAndUnequip(t *testing.T) {
	srv := testServer(t)
	reqs := []Request{
		initRequest(0),
		newGameCall(1, map[string]any{
			"scenario_id": "minimal", "character_name": "Tester", "character_class": "fighter",
		}),
		gameCall(2, "pick_up", map[string]any{"item_id": "short_sword"}),
		gameCall(3, "equip", map[string]any{"item_id": "short_sword"}),
		gameCall(4, "unequip", map[string]any{"slot": "weapon"}),
	}
	resps := multiRoundTrip(t, srv, reqs)
	require.Len(t, resps, 5)

	equipResult := extractResult(t, resps[3])
	assert.Equal(t, "weapon", equipResult["slot"])

	unequipResult := extractResult(t, resps[4])
	assert.Equal(t, "weapon", unequipResult["slot"])
}

func TestUseItem(t *testing.T) {
	srv := testServer(t)
	reqs := []Request{
		initRequest(0),
		newGameCall(1, map[string]any{
			"scenario_id": "minimal", "character_name": "Tester", "character_class": "fighter",
		}),
		gameCall(2, "pick_up", map[string]any{"item_id": "health_potion"}),
		gameCall(3, "use_item", map[string]any{"item_id": "health_potion"}),
	}
	resps := multiRoundTrip(t, srv, reqs)
	require.Len(t, resps, 4)

	useResult := extractResult(t, resps[3])
	assert.Equal(t, "heal", useResult["effect"])
	assert.Equal(t, "Health Potion", useResult["item"])
	power, ok := useResult["power"].(float64) // JSON numbers are float64
	require.True(t, ok)
	assert.GreaterOrEqual(t, power, 1.0)
}

func TestUseItem_NotConsumable(t *testing.T) {
	srv := testServer(t)
	reqs := []Request{
		initRequest(0),
		newGameCall(1, map[string]any{
			"scenario_id": "minimal", "character_name": "Tester", "character_class": "fighter",
		}),
		gameCall(2, "pick_up", map[string]any{"item_id": "short_sword"}),
		gameCall(3, "use_item", map[string]any{"item_id": "short_sword"}),
	}
	resps := multiRoundTrip(t, srv, reqs)
	require.Len(t, resps, 4)

	// Engine errors are now standard JSON-RPC errors.
	require.NotNil(t, resps[3].Error)
	assert.Contains(t, resps[3].Error.Message, "weapon")
}

func TestExamine(t *testing.T) {
	srv := testServer(t)
	reqs := []Request{
		initRequest(0),
		newGameCall(1, map[string]any{
			"scenario_id": "minimal", "character_name": "Tester", "character_class": "fighter",
		}),
		gameCall(2, "examine", map[string]any{"item_id": "short_sword"}),
	}
	resps := multiRoundTrip(t, srv, reqs)
	require.Len(t, resps, 3)

	result := extractResult(t, resps[2])
	item, ok := result["item"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Short Sword", item["name"])
}

func TestInventory(t *testing.T) {
	srv := testServer(t)
	reqs := []Request{
		initRequest(0),
		newGameCall(1, map[string]any{
			"scenario_id": "minimal", "character_name": "Tester", "character_class": "fighter",
		}),
		gameCall(2, "pick_up", map[string]any{"item_id": "short_sword"}),
		gameCall(3, "inventory", nil),
	}
	resps := multiRoundTrip(t, srv, reqs)
	require.Len(t, resps, 4)

	result := extractResult(t, resps[3])
	items, ok := result["items"].([]any)
	require.True(t, ok)
	assert.Len(t, items, 1)
}

func TestSaveAndLoad(t *testing.T) {
	srv := testServer(t)
	saveDir := t.TempDir()

	reqs := []Request{
		initRequest(0),
		newGameCall(1, map[string]any{
			"scenario_id": "minimal", "character_name": "Tester", "character_class": "fighter",
		}),
	}
	resps := multiRoundTrip(t, srv, reqs)
	require.Len(t, resps, 2)

	// Set save dir on the engine via Inspect (game goroutine owns eng).
	g := activeGame(t, srv)
	ctx := context.Background()
	require.NoError(t, g.Inspect(ctx, func(eng *engine.Engine, _ *model.GameState) {
		eng.SaveDir = saveDir
	}))

	// Use the session from the first connection for subsequent calls.
	sid := extractSessionID(t, resps[0])

	saveResps := multiRoundTrip(t, srv, []Request{
		initRequestWithSession(0, sid),
		gameCall(2, "save_game", map[string]any{"slot": "test"}),
	})
	require.Len(t, saveResps, 2)
	saveResult := extractResult(t, saveResps[1])
	assert.Equal(t, "test", saveResult["slot"])

	loadResps := multiRoundTrip(t, srv, []Request{
		initRequestWithSession(0, sid),
		gameCall(3, "load_game", map[string]any{"slot": "test"}),
	})
	require.Len(t, loadResps, 2)
	loadResult := extractResult(t, loadResps[1])
	assert.Equal(t, "test", loadResult["slot"])
	assert.Equal(t, "entrance", loadResult["room"])
}

func TestCombatBlockedActions(t *testing.T) {
	srv := testServer(t)
	reqs := []Request{
		initRequest(0),
		newGameCall(1, map[string]any{
			"scenario_id": "minimal", "character_name": "Tester", "character_class": "fighter",
		}),
		gameCall(2, "move", map[string]any{"direction": "south"}), // enters combat
	}
	resps := multiRoundTrip(t, srv, reqs)
	require.Len(t, resps, 3)

	sid := extractSessionID(t, resps[0])

	// Combat should be active — try blocked actions.
	for _, cmd := range []string{"move", "pick_up", "drop", "equip", "unequip", "examine"} {
		var args map[string]any
		switch cmd {
		case "move":
			args = map[string]any{"direction": "north"}
		case "pick_up", "drop", "equip", "examine":
			args = map[string]any{"item_id": "short_sword"}
		case "unequip":
			args = map[string]any{"slot": "weapon"}
		}
		blockResps := multiRoundTrip(t, srv, []Request{
			initRequestWithSession(0, sid),
			gameCall(10, cmd, args),
		})
		require.Len(t, blockResps, 2, "cmd=%s", cmd)
		require.NotNil(t, blockResps[1].Error, "expected %s to be blocked during combat", cmd)
		assert.NotEmpty(t, blockResps[1].Error.Message, "expected %s to have error message", cmd)
	}
}

func TestDefend(t *testing.T) {
	srv := testServer(t)
	reqs := []Request{
		initRequest(0),
		newGameCall(1, map[string]any{
			"scenario_id": "minimal", "character_name": "Tester", "character_class": "fighter",
		}),
		gameCall(2, "move", map[string]any{"direction": "south"}),
	}
	resps := multiRoundTrip(t, srv, reqs)
	require.Len(t, resps, 3)

	sid := extractSessionID(t, resps[0])

	// Force hero's turn for deterministic test via Inspect.
	g := activeGame(t, srv)
	ctx := context.Background()
	require.NoError(t, g.Inspect(ctx, func(eng *engine.Engine, state *model.GameState) {
		if !eng.IsHeroTurn(state) {
			state.Dungeon.Combat.CurrentTurn = 0
			state.Dungeon.Combat.TurnOrder[0] = "hero"
		}
	}))

	defendResps := multiRoundTrip(t, srv, []Request{
		initRequestWithSession(0, sid),
		gameCall(3, "defend", nil),
	})
	require.Len(t, defendResps, 2)

	resp := defendResps[1]
	if resp.Error != nil {
		// Hero might be dead from enemy turns — that's a valid game over.
		assert.Contains(t, resp.Error.Message, "dead")
	} else {
		result := extractResult(t, resp)
		assert.Equal(t, true, result["defending"])
	}
}

func TestUnknownCommand(t *testing.T) {
	srv := testServer(t)
	reqs := []Request{
		initRequest(0),
		newGameCall(1, map[string]any{
			"scenario_id": "minimal", "character_name": "Tester", "character_class": "fighter",
		}),
		gameCall(2, "nonexistent_command", nil),
	}
	resps := multiRoundTrip(t, srv, reqs)
	require.Len(t, resps, 3)
	require.NotNil(t, resps[2].Error)
	assert.Contains(t, resps[2].Error.Message, "unknown command")
}

func TestInvalidJSONRPC(t *testing.T) {
	srv := testServer(t)
	idJSON, _ := json.Marshal(1)
	resp := roundTrip(t, srv, Request{JSONRPC: "1.0", ID: idJSON, Method: "session.init"})
	require.NotNil(t, resp.Error)
	assert.Equal(t, CodeInvalidRequest, resp.Error.Code)
}

func TestUnknownMethod(t *testing.T) {
	srv := testServer(t)
	idJSON, _ := json.Marshal(1)
	resp := roundTrip(t, srv, Request{JSONRPC: "2.0", ID: idJSON, Method: "unknown/method"})
	require.NotNil(t, resp.Error)
	assert.Equal(t, CodeMethodNotFound, resp.Error.Code)
}

func TestGameCallBeforeInit(t *testing.T) {
	srv := testServer(t)
	// Send game.new without any prior session.init.
	resp := roundTrip(t, srv, newGameCall(1, map[string]any{
		"scenario_id":     "minimal",
		"character_name":  "Tester",
		"character_class": "fighter",
	}))

	// Should get a JSON-RPC error telling us to init first.
	require.NotNil(t, resp.Error)
	assert.Equal(t, CodeInvalidRequest, resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "call session.init first")
}

func TestRepeatedNewGame_CleansUpOldGame(t *testing.T) {
	srv := testServer(t)
	reqs := []Request{
		initRequest(0),
		newGameCall(1, map[string]any{
			"scenario_id": "minimal", "character_name": "First", "character_class": "fighter",
		}),
	}
	resps := multiRoundTrip(t, srv, reqs)
	require.Len(t, resps, 2)
	require.Nil(t, resps[1].Error)

	// Record old game count.
	sid := extractSessionID(t, resps[0])

	srv.mu.RLock()
	oldCount := len(srv.games)
	srv.mu.RUnlock()
	assert.Equal(t, 1, oldCount, "expected 1 game after first new_game")

	// Second new_game on the same session should replace the game.
	reqs2 := []Request{
		initRequestWithSession(0, sid),
		newGameCall(2, map[string]any{
			"scenario_id": "minimal", "character_name": "Second", "character_class": "mage",
		}),
	}
	resps2 := multiRoundTrip(t, srv, reqs2)
	require.Len(t, resps2, 2)
	require.Nil(t, resps2[1].Error)

	// Old game should be removed; only the new game remains.
	srv.mu.RLock()
	newCount := len(srv.games)
	srv.mu.RUnlock()
	assert.Equal(t, 1, newCount, "expected 1 game after second new_game (old game cleaned up)")

	// Verify the new game works — look should succeed.
	reqs3 := []Request{
		initRequestWithSession(0, sid),
		gameCall(3, "look", nil),
	}
	resps3 := multiRoundTrip(t, srv, reqs3)
	require.Len(t, resps3, 2)
	result := extractResult(t, resps3[1])
	assert.Equal(t, "entrance", result["room"])
}
