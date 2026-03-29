package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/punt-labs/cryptd/internal/protocol"
)

// runMCP connects to the daemon and serves an MCP server on stdio.
// Returns an exit code (0 = clean, 1 = error).
func runMCP(socketPath, addr string) int {
	conn, err := dial(socketPath, addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "crypt mcp: %v\n", err)
		return 1
	}
	defer func() { _ = conn.Close() }()

	proxy := newDaemonProxy(conn)

	// Initialize session with the daemon.
	initResult, err := proxy.call("session.init", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "crypt mcp: session init failed: %v\n", err)
		return 1
	}
	var initResp protocol.InitializeResult
	if err := json.Unmarshal(initResult, &initResp); err == nil && initResp.SessionID != "" {
		fmt.Fprintf(os.Stderr, "crypt mcp: session %s\n", initResp.SessionID)
	}

	s := newMCPServer(proxy)

	if err := mcpserver.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "crypt mcp: %v\n", err)
		return 1
	}
	return 0
}

// daemonProxy is a thread-safe proxy for sending JSON-RPC requests to the daemon.
type daemonProxy struct {
	conn    net.Conn
	scanner *bufio.Scanner
	mu      sync.Mutex
	nextID  int
}

// newDaemonProxy creates a proxy for the given connection.
func newDaemonProxy(conn net.Conn) *daemonProxy {
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	return &daemonProxy{
		conn:    conn,
		scanner: scanner,
	}
}

// call sends a JSON-RPC request to the daemon and returns the result.
func (p *daemonProxy) call(method string, params json.RawMessage) (json.RawMessage, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.nextID++
	req := protocol.Request{
		JSONRPC: "2.0",
		ID:      json.RawMessage(fmt.Sprintf("%d", p.nextID)),
		Method:  method,
		Params:  params,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	data = append(data, '\n')
	if _, err := p.conn.Write(data); err != nil {
		return nil, fmt.Errorf("write: %w", err)
	}

	if !p.scanner.Scan() {
		if err := p.scanner.Err(); err != nil {
			return nil, fmt.Errorf("connection lost: %w", err)
		}
		return nil, fmt.Errorf("connection lost")
	}

	var resp struct {
		Result json.RawMessage    `json:"result"`
		Error  *protocol.RPCError `json:"error"`
	}
	if err := json.Unmarshal(p.scanner.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("%s", resp.Error.Message)
	}
	return resp.Result, nil
}

// newMCPServer creates an MCP server with all game tools registered.
func newMCPServer(proxy *daemonProxy) *mcpserver.MCPServer {
	s := mcpserver.NewMCPServer("crypt", "0.1.0", mcpserver.WithToolCapabilities(true))
	registerTools(s, proxy)
	return s
}

// toolDef describes an MCP tool to register.
type toolDef struct {
	name        string
	description string
	gameMethod  string
	opts        []mcplib.ToolOption
}

// gameTools returns the tool definitions for all game commands.
func gameTools() []toolDef {
	return []toolDef{
		{
			name:        "new_game",
			description: "Start a new game with a scenario, character name, and class.",
			gameMethod:  "new",
			opts: []mcplib.ToolOption{
				mcplib.WithString("scenario_id", mcplib.Required(), mcplib.Description("Scenario to load")),
				mcplib.WithString("character_name", mcplib.Required(), mcplib.Description("Character name")),
				mcplib.WithString("character_class", mcplib.Required(), mcplib.Description("Character class: fighter, mage, priest, thief")),
			},
		},
		{
			name:        "move",
			description: "Move in a direction.",
			gameMethod:  "move",
			opts: []mcplib.ToolOption{
				mcplib.WithString("direction", mcplib.Required(), mcplib.Description("Direction to move"), mcplib.Enum("north", "south", "east", "west", "up", "down")),
			},
		},
		{
			name:        "look",
			description: "Look around the current room.",
			gameMethod:  "look",
		},
		{
			name:        "pick_up",
			description: "Pick up an item.",
			gameMethod:  "pick_up",
			opts: []mcplib.ToolOption{
				mcplib.WithString("item_id", mcplib.Required(), mcplib.Description("Item to pick up")),
			},
		},
		{
			name:        "drop",
			description: "Drop an item from inventory.",
			gameMethod:  "drop",
			opts: []mcplib.ToolOption{
				mcplib.WithString("item_id", mcplib.Required(), mcplib.Description("Item to drop")),
			},
		},
		{
			name:        "equip",
			description: "Equip an item from inventory.",
			gameMethod:  "equip",
			opts: []mcplib.ToolOption{
				mcplib.WithString("item_id", mcplib.Required(), mcplib.Description("Item to equip")),
			},
		},
		{
			name:        "unequip",
			description: "Unequip an item from a slot.",
			gameMethod:  "unequip",
			opts: []mcplib.ToolOption{
				mcplib.WithString("slot", mcplib.Required(), mcplib.Description("Equipment slot to unequip")),
			},
		},
		{
			name:        "examine",
			description: "Examine an item closely.",
			gameMethod:  "examine",
			opts: []mcplib.ToolOption{
				mcplib.WithString("item_id", mcplib.Required(), mcplib.Description("Item to examine")),
			},
		},
		{
			name:        "inventory",
			description: "List items in your inventory.",
			gameMethod:  "inventory",
		},
		{
			name:        "attack",
			description: "Attack a target in combat.",
			gameMethod:  "attack",
			opts: []mcplib.ToolOption{
				mcplib.WithString("target_id", mcplib.Description("Target to attack (optional, defaults to current enemy)")),
			},
		},
		{
			name:        "defend",
			description: "Take a defensive stance in combat.",
			gameMethod:  "defend",
		},
		{
			name:        "flee",
			description: "Attempt to flee from combat.",
			gameMethod:  "flee",
		},
		{
			name:        "cast_spell",
			description: "Cast a spell.",
			gameMethod:  "cast_spell",
			opts: []mcplib.ToolOption{
				mcplib.WithString("spell_id", mcplib.Required(), mcplib.Description("Spell to cast")),
				mcplib.WithString("target_id", mcplib.Description("Target for the spell (optional)")),
			},
		},
		{
			name:        "save_game",
			description: "Save the current game.",
			gameMethod:  "save",
			opts: []mcplib.ToolOption{
				mcplib.WithString("slot", mcplib.Description("Save slot (optional)")),
			},
		},
		{
			name:        "load_game",
			description: "Load a saved game.",
			gameMethod:  "load",
			opts: []mcplib.ToolOption{
				mcplib.WithString("slot", mcplib.Description("Save slot to load (optional)")),
			},
		},
		{
			name:        "play",
			description: "Send natural language text to the game engine.",
			gameMethod:  "play",
			opts: []mcplib.ToolOption{
				mcplib.WithString("text", mcplib.Required(), mcplib.Description("Natural language command")),
			},
		},
	}
}

// registerTools adds all game tools to the MCP server.
func registerTools(s *mcpserver.MCPServer, proxy *daemonProxy) {
	for _, td := range gameTools() {
		opts := []mcplib.ToolOption{mcplib.WithDescription(td.description)}
		opts = append(opts, td.opts...)
		tool := mcplib.NewTool(td.name, opts...)
		s.AddTool(tool, makeGameHandler(proxy, td.gameMethod))
	}
}

// makeGameHandler returns an MCP tool handler that proxies to the daemon.
func makeGameHandler(proxy *daemonProxy, gameMethod string) mcpserver.ToolHandlerFunc {
	return func(_ context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		args := req.GetArguments()
		var params json.RawMessage
		if len(args) > 0 {
			var err error
			params, err = json.Marshal(args)
			if err != nil {
				return mcplib.NewToolResultError(fmt.Sprintf("marshal params: %v", err)), nil
			}
		}
		result, err := proxy.call("game."+gameMethod, params)
		if err != nil {
			return mcplib.NewToolResultError(err.Error()), nil
		}
		return mcplib.NewToolResultText(string(result)), nil
	}
}
