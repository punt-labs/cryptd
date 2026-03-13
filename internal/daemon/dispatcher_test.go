package daemon

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testServer creates a Server wired to the minimal scenario in testdata.
func testServer(t *testing.T) *Server {
	t.Helper()
	// Find testdata relative to repo root.
	scenarioDir := filepath.Join(repoRoot(t), "testdata", "scenarios")
	return NewServer(filepath.Join(t.TempDir(), "test.sock"), scenarioDir, WithPassthrough())
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	// internal/daemon/ → repo root
	return filepath.Join(wd, "..", "..")
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

func toolCall(id int, name string, args any) Request {
	argsJSON, _ := json.Marshal(args)
	params, _ := json.Marshal(ToolCallParams{Name: name, Arguments: argsJSON})
	idJSON, _ := json.Marshal(id)
	return Request{
		JSONRPC: "2.0",
		ID:      idJSON,
		Method:  "tools/call",
		Params:  params,
	}
}

func extractResult(t *testing.T, resp Response) map[string]any {
	t.Helper()
	require.Nil(t, resp.Error, "unexpected RPC error: %+v", resp.Error)

	// Result is a ToolResult with Content[0].Text as JSON.
	data, err := json.Marshal(resp.Result)
	require.NoError(t, err)
	var tr ToolResult
	require.NoError(t, json.Unmarshal(data, &tr))
	require.Len(t, tr.Content, 1)

	var result map[string]any
	require.NoError(t, json.Unmarshal([]byte(tr.Content[0].Text), &result))
	return result
}

func TestInitialize(t *testing.T) {
	srv := testServer(t)
	idJSON, _ := json.Marshal(1)
	resp := roundTrip(t, srv, Request{JSONRPC: "2.0", ID: idJSON, Method: "initialize"})

	require.Nil(t, resp.Error)
	data, _ := json.Marshal(resp.Result)
	var init InitializeResult
	require.NoError(t, json.Unmarshal(data, &init))
	assert.Equal(t, "cryptd", init.ServerInfo.Name)
	assert.NotEmpty(t, init.ProtocolVersion)
}

func TestToolsList(t *testing.T) {
	srv := testServer(t)
	idJSON, _ := json.Marshal(1)
	resp := roundTrip(t, srv, Request{JSONRPC: "2.0", ID: idJSON, Method: "tools/list"})

	require.Nil(t, resp.Error)
	data, _ := json.Marshal(resp.Result)
	var result struct {
		Tools []ToolInfo `json:"tools"`
	}
	require.NoError(t, json.Unmarshal(data, &result))
	assert.Len(t, result.Tools, 15)

	// Check first and last tool names.
	assert.Equal(t, "new_game", result.Tools[0].Name)
	assert.Equal(t, "load_game", result.Tools[14].Name)
}

func TestNewGame(t *testing.T) {
	srv := testServer(t)
	resp := roundTrip(t, srv, toolCall(1, "new_game", map[string]any{
		"scenario_id":     "minimal",
		"character_name":  "Tester",
		"character_class": "fighter",
	}))
	result := extractResult(t, resp)
	assert.Equal(t, "entrance", result["room"])
	assert.Equal(t, "Entrance Hall", result["name"])

	hero, ok := result["hero"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Tester", hero["name"])
	assert.Equal(t, "fighter", hero["class"])
}

// extractToolError extracts the error text from a ToolResult with isError=true.
func extractToolError(t *testing.T, resp Response) string {
	t.Helper()
	require.Nil(t, resp.Error, "expected no JSON-RPC error, got: %+v", resp.Error)
	data, err := json.Marshal(resp.Result)
	require.NoError(t, err)
	var tr ToolResult
	require.NoError(t, json.Unmarshal(data, &tr))
	require.True(t, tr.IsError, "expected isError=true in ToolResult")
	require.Len(t, tr.Content, 1)
	return tr.Content[0].Text
}

func TestNoActiveGame(t *testing.T) {
	srv := testServer(t)
	resp := roundTrip(t, srv, toolCall(1, "look", nil))

	errText := extractToolError(t, resp)
	assert.Contains(t, errText, "no active game")
}

func TestLook(t *testing.T) {
	srv := testServer(t)
	reqs := []Request{
		toolCall(1, "new_game", map[string]any{
			"scenario_id": "minimal", "character_name": "Tester", "character_class": "fighter",
		}),
		toolCall(2, "look", nil),
	}
	resps := multiRoundTrip(t, srv, reqs)
	require.Len(t, resps, 2)

	result := extractResult(t, resps[1])
	assert.Equal(t, "entrance", result["room"])
	assert.Equal(t, "Entrance Hall", result["name"])
}

func TestMove(t *testing.T) {
	srv := testServer(t)
	reqs := []Request{
		toolCall(1, "new_game", map[string]any{
			"scenario_id": "minimal", "character_name": "Tester", "character_class": "fighter",
		}),
		toolCall(2, "move", map[string]any{"direction": "south"}),
	}
	resps := multiRoundTrip(t, srv, reqs)
	require.Len(t, resps, 2)

	result := extractResult(t, resps[1])
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
		toolCall(1, "new_game", map[string]any{
			"scenario_id": "minimal", "character_name": "Tester", "character_class": "fighter",
		}),
		toolCall(2, "move", map[string]any{"direction": "east"}),
	}
	resps := multiRoundTrip(t, srv, reqs)
	require.Len(t, resps, 2)

	errText := extractToolError(t, resps[1])
	assert.Contains(t, errText, "no exit")
}

func TestPickUpAndDrop(t *testing.T) {
	srv := testServer(t)
	reqs := []Request{
		toolCall(1, "new_game", map[string]any{
			"scenario_id": "minimal", "character_name": "Tester", "character_class": "fighter",
		}),
		toolCall(2, "pick_up", map[string]any{"item_id": "short_sword"}),
	}
	resps := multiRoundTrip(t, srv, reqs)
	require.Len(t, resps, 2)

	pickResult := extractResult(t, resps[1])
	item, ok := pickResult["item"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "short_sword", item["id"])

	// Now drop it.
	dropResp := roundTrip(t, srv, toolCall(3, "drop", map[string]any{"item_id": "short_sword"}))
	dropResult := extractResult(t, dropResp)
	item, ok = dropResult["item"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "short_sword", item["id"])
}

func TestEquipAndUnequip(t *testing.T) {
	srv := testServer(t)
	reqs := []Request{
		toolCall(1, "new_game", map[string]any{
			"scenario_id": "minimal", "character_name": "Tester", "character_class": "fighter",
		}),
		toolCall(2, "pick_up", map[string]any{"item_id": "short_sword"}),
		toolCall(3, "equip", map[string]any{"item_id": "short_sword"}),
	}
	resps := multiRoundTrip(t, srv, reqs)
	require.Len(t, resps, 3)

	equipResult := extractResult(t, resps[2])
	assert.Equal(t, "weapon", equipResult["slot"])

	// Unequip.
	unequipResp := roundTrip(t, srv, toolCall(4, "unequip", map[string]any{"slot": "weapon"}))
	unequipResult := extractResult(t, unequipResp)
	assert.Equal(t, "weapon", unequipResult["slot"])
}

func TestExamine(t *testing.T) {
	srv := testServer(t)
	reqs := []Request{
		toolCall(1, "new_game", map[string]any{
			"scenario_id": "minimal", "character_name": "Tester", "character_class": "fighter",
		}),
		toolCall(2, "examine", map[string]any{"item_id": "short_sword"}),
	}
	resps := multiRoundTrip(t, srv, reqs)
	require.Len(t, resps, 2)

	result := extractResult(t, resps[1])
	item, ok := result["item"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Short Sword", item["name"])
}

func TestInventory(t *testing.T) {
	srv := testServer(t)
	reqs := []Request{
		toolCall(1, "new_game", map[string]any{
			"scenario_id": "minimal", "character_name": "Tester", "character_class": "fighter",
		}),
		toolCall(2, "pick_up", map[string]any{"item_id": "short_sword"}),
		toolCall(3, "inventory", nil),
	}
	resps := multiRoundTrip(t, srv, reqs)
	require.Len(t, resps, 3)

	result := extractResult(t, resps[2])
	items, ok := result["items"].([]any)
	require.True(t, ok)
	assert.Len(t, items, 1)
}

func TestSaveAndLoad(t *testing.T) {
	srv := testServer(t)
	// Override save dir so we don't pollute the real filesystem.
	saveDir := t.TempDir()

	reqs := []Request{
		toolCall(1, "new_game", map[string]any{
			"scenario_id": "minimal", "character_name": "Tester", "character_class": "fighter",
		}),
	}
	resps := multiRoundTrip(t, srv, reqs)
	require.Len(t, resps, 1)

	// Set save dir on the engine (we have access since it's in-process).
	srv.mu.Lock()
	srv.eng.SaveDir = saveDir
	srv.mu.Unlock()

	saveResp := roundTrip(t, srv, toolCall(2, "save_game", map[string]any{"slot": "test"}))
	saveResult := extractResult(t, saveResp)
	assert.Equal(t, "test", saveResult["slot"])

	loadResp := roundTrip(t, srv, toolCall(3, "load_game", map[string]any{"slot": "test"}))
	loadResult := extractResult(t, loadResp)
	assert.Equal(t, "test", loadResult["slot"])
	assert.Equal(t, "entrance", loadResult["room"])
}

func TestCombatBlockedActions(t *testing.T) {
	srv := testServer(t)
	reqs := []Request{
		toolCall(1, "new_game", map[string]any{
			"scenario_id": "minimal", "character_name": "Tester", "character_class": "fighter",
		}),
		toolCall(2, "move", map[string]any{"direction": "south"}), // enters combat
	}
	resps := multiRoundTrip(t, srv, reqs)
	require.Len(t, resps, 2)

	// Combat should be active — try blocked actions.
	for _, tool := range []string{"move", "pick_up", "drop", "equip", "unequip", "examine"} {
		var args map[string]any
		switch tool {
		case "move":
			args = map[string]any{"direction": "north"}
		case "pick_up", "drop", "equip", "examine":
			args = map[string]any{"item_id": "short_sword"}
		case "unequip":
			args = map[string]any{"slot": "weapon"}
		}
		resp := roundTrip(t, srv, toolCall(10, tool, args))
		errText := extractToolError(t, resp)
		assert.NotEmpty(t, errText, "expected %s to be blocked during combat", tool)
	}
}

func TestDefend(t *testing.T) {
	srv := testServer(t)
	reqs := []Request{
		toolCall(1, "new_game", map[string]any{
			"scenario_id": "minimal", "character_name": "Tester", "character_class": "fighter",
		}),
		toolCall(2, "move", map[string]any{"direction": "south"}),
	}
	resps := multiRoundTrip(t, srv, reqs)
	require.Len(t, resps, 2)

	// We're in combat. If it's not hero's turn (enemies went first), we need
	// to check. If it is hero's turn, defend should work.
	srv.mu.Lock()
	// Force hero's turn for deterministic test.
	if !srv.eng.IsHeroTurn(srv.state) {
		srv.state.Dungeon.Combat.CurrentTurn = 0
		srv.state.Dungeon.Combat.TurnOrder[0] = "hero"
	}
	srv.mu.Unlock()

	resp := roundTrip(t, srv, toolCall(3, "defend", nil))
	// Tool errors now come as ToolResult with isError.
	data, err := json.Marshal(resp.Result)
	require.NoError(t, err)
	var tr ToolResult
	require.NoError(t, json.Unmarshal(data, &tr))
	if tr.IsError {
		// Hero might be dead from enemy turns — that's a valid game over.
		assert.Contains(t, tr.Content[0].Text, "dead")
	} else {
		result := extractResult(t, resp)
		assert.Equal(t, true, result["defending"])
	}
}

func TestUnknownTool(t *testing.T) {
	srv := testServer(t)
	// Start a game first.
	reqs := []Request{
		toolCall(1, "new_game", map[string]any{
			"scenario_id": "minimal", "character_name": "Tester", "character_class": "fighter",
		}),
		toolCall(2, "nonexistent_tool", nil),
	}
	resps := multiRoundTrip(t, srv, reqs)
	require.Len(t, resps, 2)
	errText := extractToolError(t, resps[1])
	assert.Contains(t, errText, "unknown tool")
}

func TestInvalidJSONRPC(t *testing.T) {
	srv := testServer(t)
	idJSON, _ := json.Marshal(1)
	resp := roundTrip(t, srv, Request{JSONRPC: "1.0", ID: idJSON, Method: "initialize"})
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
