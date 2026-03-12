package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/punt-labs/cryptd/internal/daemon"
)

// errConnLost signals that the server connection dropped.
var errConnLost = errors.New("connection lost")

func runConnect(args []string) {
	fs := flag.NewFlagSet("connect", flag.ContinueOnError)
	socketPath := fs.String("socket", "", "Unix socket path (default ~/.crypt/daemon.sock)")
	addr := fs.String("addr", "", "TCP address (e.g. host:9000)")
	raw := fs.Bool("raw", false, "print raw JSON responses")
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if *socketPath != "" && *addr != "" {
		fmt.Fprintln(os.Stderr, "error: --socket and --addr are mutually exclusive")
		os.Exit(1)
	}

	var conn net.Conn
	var err error

	if *addr != "" {
		// TCP: connect directly, no auto-start.
		conn, err = net.DialTimeout("tcp", *addr, 5*time.Second)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: cannot connect to %s: %v\n", *addr, err)
			os.Exit(1)
		}
	} else {
		// Unix socket: auto-start server if needed.
		sock := *socketPath
		if sock == "" {
			sock, err = daemon.DefaultSocketPath()
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: cannot determine home directory: %v\n", err)
				os.Exit(1)
			}
		}
		conn, err = dialOrStart(sock)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	}
	defer conn.Close()

	os.Exit(connectSession(conn, os.Stdin, os.Stdout, os.Stderr, *raw))
}

// dialOrStart connects to the Unix socket, starting cryptd serve if needed.
func dialOrStart(socketPath string) (net.Conn, error) {
	// Try connecting first.
	conn, err := net.DialTimeout("unix", socketPath, time.Second)
	if err == nil {
		return conn, nil
	}

	// Not running — start the server.
	cryptd, err := exec.LookPath("cryptd")
	if err != nil {
		return nil, fmt.Errorf("cryptd not found in PATH — install it or start the server manually")
	}

	cmd := exec.Command(cryptd, "serve", "--socket", socketPath)
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start cryptd: %w", err)
	}

	// Poll for the socket to appear.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		conn, err = net.DialTimeout("unix", socketPath, 500*time.Millisecond)
		if err == nil {
			return conn, nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return nil, fmt.Errorf("cryptd started but socket %s not ready within 5s", socketPath)
}

// connectSession runs the interactive REPL on an open connection.
// Returns exit code 0 for clean exit, 1 for errors.
func connectSession(conn net.Conn, in io.Reader, out, errOut io.Writer, raw bool) int {
	scanner := bufio.NewScanner(conn)
	reqID := 0

	send := func(method string, params any) (json.RawMessage, error) {
		reqID++
		req := map[string]any{
			"jsonrpc": "2.0",
			"id":      reqID,
			"method":  method,
		}
		if params != nil {
			p, err := json.Marshal(params)
			if err != nil {
				return nil, fmt.Errorf("marshal params: %w", err)
			}
			req["params"] = json.RawMessage(p)
		}
		data, err := json.Marshal(req)
		if err != nil {
			return nil, err
		}
		data = append(data, '\n')
		if _, err := conn.Write(data); err != nil {
			return nil, fmt.Errorf("%w: write: %w", errConnLost, err)
		}
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return nil, fmt.Errorf("%w: read: %w", errConnLost, err)
			}
			return nil, errConnLost
		}
		var resp struct {
			Result json.RawMessage `json:"result"`
			Error  *daemon.RPCError `json:"error"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
			return nil, fmt.Errorf("parse response: %w", err)
		}
		if resp.Error != nil {
			return nil, fmt.Errorf("server error: %s", resp.Error.Message)
		}
		return resp.Result, nil
	}

	// MCP initialize handshake.
	if _, err := send("initialize", nil); err != nil {
		fmt.Fprintf(errOut, "error: %v\n", err)
		return 1
	}

	// Tool call helper.
	callTool := func(name string, args any) (json.RawMessage, error) {
		argsJSON, err := json.Marshal(args)
		if err != nil {
			return nil, fmt.Errorf("marshal args: %w", err)
		}
		params := map[string]any{
			"name":      name,
			"arguments": json.RawMessage(argsJSON),
		}
		result, err := send("tools/call", params)
		if err != nil {
			return nil, err
		}
		// Extract text from ToolResult content array.
		var tr struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		}
		if err := json.Unmarshal(result, &tr); err != nil {
			return result, nil // Return raw if not ToolResult format.
		}
		if tr.IsError && len(tr.Content) > 0 {
			return nil, extractErrorMessage(tr.Content[0].Text)
		}
		if len(tr.Content) > 0 {
			return json.RawMessage(tr.Content[0].Text), nil
		}
		return result, nil
	}

	inputScanner := bufio.NewScanner(in)
	fmt.Fprintln(out, "Connected to cryptd. Type 'help' for commands, 'quit' to exit.")

	for {
		fmt.Fprint(out, "> ")
		if !inputScanner.Scan() {
			break
		}
		line := strings.TrimSpace(inputScanner.Text())
		if line == "" {
			continue
		}

		if line == "quit" || line == "exit" {
			fmt.Fprintln(out, "Goodbye.")
			return 0
		}

		tool, toolArgs, err := parseCommand(line)
		if err != nil {
			fmt.Fprintf(errOut, "%v\n", err)
			continue
		}

		result, err := callTool(tool, toolArgs)
		if err != nil {
			if errors.Is(err, errConnLost) {
				fmt.Fprintf(errOut, "Error: %v\n", err)
				return 1
			}
			fmt.Fprintf(errOut, "Error: %v\n", err)
			continue
		}

		if raw {
			var pretty json.RawMessage
			if json.Unmarshal(result, &pretty) == nil {
				formatted, _ := json.MarshalIndent(pretty, "", "  ")
				fmt.Fprintln(out, string(formatted))
			}
		} else {
			formatToolResult(out, tool, result)
		}
	}
	return 0
}

// parseCommand maps user text to a tool name and arguments.
func parseCommand(line string) (string, map[string]any, error) {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return "", nil, fmt.Errorf("empty command")
	}
	verb := strings.ToLower(parts[0])
	rest := parts[1:]

	switch verb {
	case "new", "start":
		// new <scenario_id> <name> <class>
		if len(rest) < 3 {
			return "", nil, fmt.Errorf("usage: new <scenario_id> <character_name> <character_class>")
		}
		return "new_game", map[string]any{
			"scenario_id":    rest[0],
			"character_name": rest[1],
			"character_class": rest[2],
		}, nil

	case "go", "move", "walk":
		if len(rest) < 1 {
			return "", nil, fmt.Errorf("usage: go <direction>")
		}
		return "move", map[string]any{"direction": directionAlias(rest[0])}, nil

	case "n", "s", "e", "w", "u", "d",
		"north", "south", "east", "west", "up", "down":
		return "move", map[string]any{"direction": directionAlias(verb)}, nil

	case "l", "look":
		return "look", map[string]any{}, nil

	case "take", "get", "grab", "pick":
		if len(rest) < 1 {
			return "", nil, fmt.Errorf("usage: take <item_id>")
		}
		return "pick_up", map[string]any{"item_id": strings.Join(rest, "_")}, nil

	case "drop":
		if len(rest) < 1 {
			return "", nil, fmt.Errorf("usage: drop <item_id>")
		}
		return "drop", map[string]any{"item_id": strings.Join(rest, "_")}, nil

	case "equip", "wear", "wield":
		if len(rest) < 1 {
			return "", nil, fmt.Errorf("usage: equip <item_id>")
		}
		return "equip", map[string]any{"item_id": strings.Join(rest, "_")}, nil

	case "unequip", "remove":
		if len(rest) < 1 {
			return "", nil, fmt.Errorf("usage: unequip <slot>")
		}
		return "unequip", map[string]any{"slot": rest[0]}, nil

	case "examine", "inspect", "x":
		if len(rest) < 1 {
			return "", nil, fmt.Errorf("usage: examine <item_id>")
		}
		return "examine", map[string]any{"item_id": strings.Join(rest, "_")}, nil

	case "i", "inv", "inventory":
		return "inventory", map[string]any{}, nil

	case "attack", "hit", "strike", "kill":
		args := map[string]any{}
		if len(rest) > 0 {
			args["target_id"] = strings.Join(rest, "_")
		}
		return "attack", args, nil

	case "defend", "guard", "block":
		return "defend", map[string]any{}, nil

	case "flee", "run", "escape":
		return "flee", map[string]any{}, nil

	case "cast":
		if len(rest) < 1 {
			return "", nil, fmt.Errorf("usage: cast <spell_id> [target_id]")
		}
		args := map[string]any{"spell_id": rest[0]}
		if len(rest) > 1 {
			args["target_id"] = strings.Join(rest[1:], "_")
		}
		return "cast_spell", args, nil

	case "save":
		args := map[string]any{}
		if len(rest) > 0 {
			args["slot"] = rest[0]
		}
		return "save_game", args, nil

	case "load":
		args := map[string]any{}
		if len(rest) > 0 {
			args["slot"] = rest[0]
		}
		return "load_game", args, nil

	case "help", "?":
		return "", nil, errors.New(helpText)

	default:
		return "", nil, fmt.Errorf("unknown command %q — type 'help' for commands", verb)
	}
}

// directionAlias normalises direction abbreviations.
func directionAlias(s string) string {
	switch strings.ToLower(s) {
	case "n", "north":
		return "north"
	case "s", "south":
		return "south"
	case "e", "east":
		return "east"
	case "w", "west":
		return "west"
	case "u", "up":
		return "up"
	case "d", "down":
		return "down"
	default:
		return s
	}
}

const helpText = `commands:
  new <scenario> <name> <class>   Start a new game
  go/move <direction>             Move (north/south/east/west/up/down)
  n/s/e/w/u/d                     Direction shortcuts
  look/l                          Look around
  take/get <item>                 Pick up an item
  drop <item>                     Drop an item
  equip <item>                    Equip an item
  unequip <slot>                  Unequip a slot
  examine/x <item>                Examine an item
  inventory/i                     Show inventory
  attack [target]                 Attack (default: first enemy)
  defend                          Raise guard
  flee                            Attempt to flee
  cast <spell> [target]           Cast a spell
  save [slot]                     Save game
  load [slot]                     Load game
  quit/exit                       Disconnect`

// formatToolResult prints a human-readable summary of a tool result.
func formatToolResult(out io.Writer, tool string, data json.RawMessage) {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		fmt.Fprintln(out, string(data))
		return
	}

	switch tool {
	case "new_game", "look":
		formatRoom(out, m)
	case "move":
		formatRoom(out, m)
		formatCombat(out, m)
	case "pick_up":
		if item, ok := m["item"].(map[string]any); ok {
			fmt.Fprintf(out, "Picked up: %s\n", item["name"])
		}
	case "drop":
		if item, ok := m["item"].(map[string]any); ok {
			fmt.Fprintf(out, "Dropped: %s\n", item["name"])
		}
	case "equip":
		if item, ok := m["item"].(map[string]any); ok {
			fmt.Fprintf(out, "Equipped %s in %s slot\n", item["name"], m["slot"])
		}
	case "unequip":
		if item, ok := m["item"].(map[string]any); ok {
			fmt.Fprintf(out, "Unequipped %s from %s slot\n", item["name"], m["slot"])
		}
	case "examine":
		if item, ok := m["item"].(map[string]any); ok {
			fmt.Fprintf(out, "%s: %s\n", item["name"], item["description"])
		}
	case "inventory":
		formatInventory(out, m)
	case "attack":
		formatAttack(out, m)
	case "defend":
		fmt.Fprintln(out, "You raise your guard.")
		formatEnemyTurns(out, m)
	case "flee":
		if success, _ := m["success"].(bool); success {
			fmt.Fprintln(out, "You flee from combat!")
		} else {
			fmt.Fprintln(out, "You fail to flee!")
			formatEnemyTurns(out, m)
		}
	case "cast_spell":
		formatSpell(out, m)
	case "save_game":
		fmt.Fprintf(out, "Game saved to slot: %s\n", m["slot"])
	case "load_game":
		fmt.Fprintf(out, "Game loaded from slot: %s\n", m["slot"])
		formatRoom(out, m)
	default:
		formatted, _ := json.MarshalIndent(m, "", "  ")
		fmt.Fprintln(out, string(formatted))
	}

	// Show hero status if present.
	if hero, ok := m["hero"].(map[string]any); ok {
		hp, _ := hero["hp"].(float64)
		maxHP, _ := hero["max_hp"].(float64)
		fmt.Fprintf(out, "[HP: %.0f/%.0f", hp, maxHP)
		if mp, ok := hero["mp"].(float64); ok && mp > 0 {
			maxMP, _ := hero["max_mp"].(float64)
			fmt.Fprintf(out, " MP: %.0f/%.0f", mp, maxMP)
		}
		fmt.Fprintln(out, "]")
	}
}

func formatRoom(out io.Writer, m map[string]any) {
	name, _ := m["name"].(string)
	desc, _ := m["description"].(string)
	if name != "" {
		fmt.Fprintf(out, "=== %s ===\n", name)
	}
	if desc != "" {
		fmt.Fprintln(out, desc)
	}
	if exits, ok := m["exits"].([]any); ok && len(exits) > 0 {
		var dirs []string
		for _, e := range exits {
			if s, ok := e.(string); ok {
				dirs = append(dirs, s)
			}
		}
		fmt.Fprintf(out, "Exits: %s\n", strings.Join(dirs, ", "))
	}
	if items, ok := m["items"].([]any); ok && len(items) > 0 {
		var names []string
		for _, item := range items {
			if s, ok := item.(string); ok {
				names = append(names, s)
			}
		}
		if len(names) > 0 {
			fmt.Fprintf(out, "Items here: %s\n", strings.Join(names, ", "))
		}
	}
}

func formatCombat(out io.Writer, m map[string]any) {
	combat, ok := m["combat"].(map[string]any)
	if !ok {
		return
	}
	if enemies, ok := combat["enemies"].([]any); ok {
		fmt.Fprint(out, "Combat! Enemies: ")
		var names []string
		for _, e := range enemies {
			if em, ok := e.(map[string]any); ok {
				name, _ := em["name"].(string)
				hp, _ := em["hp"].(float64)
				names = append(names, fmt.Sprintf("%s (HP: %.0f)", name, hp))
			}
		}
		fmt.Fprintln(out, strings.Join(names, ", "))
	}
	formatEnemyTurns(out, m)
}

func formatAttack(out io.Writer, m map[string]any) {
	target, _ := m["target"].(string)
	damage, _ := m["damage"].(float64)
	killed, _ := m["killed"].(bool)
	if killed {
		xp, _ := m["xp_awarded"].(float64)
		fmt.Fprintf(out, "You kill %s! (+%.0f XP)\n", target, xp)
	} else {
		fmt.Fprintf(out, "You hit %s for %.0f damage.\n", target, damage)
	}
	if over, _ := m["combat_over"].(bool); over {
		fmt.Fprintln(out, "Combat is over!")
		if lu, ok := m["level_up"].(map[string]any); ok {
			newLvl, _ := lu["new_level"].(float64)
			fmt.Fprintf(out, "Level up! You are now level %.0f!\n", newLvl)
		}
	}
	formatEnemyTurns(out, m)
}

func formatSpell(out io.Writer, m map[string]any) {
	spell, _ := m["spell"].(string)
	effect, _ := m["effect"].(string)
	power, _ := m["power"].(float64)
	switch effect {
	case "damage":
		target, _ := m["target"].(string)
		fmt.Fprintf(out, "%s deals %.0f damage to %s!\n", spell, power, target)
	case "heal":
		fmt.Fprintf(out, "%s heals you for %.0f HP!\n", spell, power)
	}
	if over, _ := m["combat_over"].(bool); over {
		fmt.Fprintln(out, "Combat is over!")
	}
	formatEnemyTurns(out, m)
}

func formatInventory(out io.Writer, m map[string]any) {
	items, _ := m["items"].([]any)
	if len(items) == 0 {
		fmt.Fprintln(out, "Your inventory is empty.")
		return
	}
	fmt.Fprintln(out, "Inventory:")
	for _, item := range items {
		if im, ok := item.(map[string]any); ok {
			fmt.Fprintf(out, "  - %s (%s)\n", im["name"], im["id"])
		}
	}
	if eq, ok := m["equipped"].(map[string]any); ok {
		var slots []string
		for slot, id := range eq {
			if s, ok := id.(string); ok && s != "" {
				slots = append(slots, fmt.Sprintf("%s=%s", slot, s))
			}
		}
		if len(slots) > 0 {
			fmt.Fprintf(out, "Equipped: %s\n", strings.Join(slots, ", "))
		}
	}
}

// extractErrorMessage tries to parse an RPCError JSON blob and return just
// the human-readable message. Falls back to the raw string if not JSON.
func extractErrorMessage(text string) error {
	var rpcErr struct {
		Message string `json:"message"`
	}
	if json.Unmarshal([]byte(text), &rpcErr) == nil && rpcErr.Message != "" {
		return errors.New(rpcErr.Message)
	}
	return errors.New(text)
}

func formatEnemyTurns(out io.Writer, m map[string]any) {
	turns, ok := m["enemy_turns"].([]any)
	if !ok || len(turns) == 0 {
		return
	}
	for _, t := range turns {
		turn, ok := t.(map[string]any)
		if !ok {
			continue
		}
		enemy, _ := turn["enemy"].(string)
		action, _ := turn["action"].(string)
		switch action {
		case "attack":
			damage, _ := turn["damage"].(float64)
			fmt.Fprintf(out, "%s attacks you for %.0f damage!\n", enemy, damage)
			if dead, _ := turn["hero_dead"].(bool); dead {
				fmt.Fprintln(out, "You have been slain!")
			}
		case "flee":
			fmt.Fprintf(out, "%s flees!\n", enemy)
		}
	}
}
