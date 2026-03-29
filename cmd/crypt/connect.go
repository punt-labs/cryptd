package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/ergochat/readline"
	"github.com/punt-labs/cryptd/internal/protocol"
)

// errConnLost signals that the server connection dropped.
var errConnLost = errors.New("connection lost")

// run connects to the server and starts the interactive session.
// Returns an exit code (0 = clean, 1 = error).
func run(socketPath, addr, scenario, charName, charClass, sessionID string) int {
	conn, err := dial(socketPath, addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 1
	}
	defer func() { _ = conn.Close() }()

	return session(conn, os.Stdin, os.Stdout, os.Stderr, scenario, charName, charClass, sessionID)
}

// dial connects to the server via TCP or Unix socket.
func dial(socketPath, addr string) (net.Conn, error) {
	return dialWithFlags(socketPath, addr)
}

// dialMCP connects to the server for MCP mode. If auto-starting, it uses
// --passthrough because MCP clients send game.* methods that require passthrough
// dispatch. Normal mode only accepts game.new and game.play.
func dialMCP(socketPath, addr string) (net.Conn, error) {
	return dialWithFlags(socketPath, addr, "--passthrough")
}

// dialWithFlags connects to the server via TCP (if addr is set) or Unix socket,
// auto-starting cryptd serve with the given extra flags if the socket is not
// already listening.
func dialWithFlags(socketPath, addr string, extraFlags ...string) (net.Conn, error) {
	if addr != "" {
		conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
		if err != nil {
			return nil, fmt.Errorf("cannot connect to %s: %w", addr, err)
		}
		return conn, nil
	}

	sock := socketPath
	if sock == "" {
		var err error
		sock, err = protocol.DefaultSocketPath()
		if err != nil {
			return nil, fmt.Errorf("cannot determine home directory: %w", err)
		}
	}
	return dialOrStart(sock, extraFlags...)
}

// dialOrStart connects to the Unix socket, starting cryptd serve if needed.
// Any extraFlags are appended to the serve command (e.g. "--passthrough").
func dialOrStart(socketPath string, extraFlags ...string) (net.Conn, error) {
	conn, err := net.DialTimeout("unix", socketPath, time.Second)
	if err == nil {
		return conn, nil
	}

	cryptd, err := exec.LookPath("cryptd")
	if err != nil {
		return nil, fmt.Errorf("cryptd not found in PATH — install it or start the server manually")
	}

	// Start in foreground mode so this process IS the server — kill and
	// wait target the right PID (not a daemonize parent that already exited).
	// Capture stderr in a buffer — server log output on the client terminal
	// corrupts readline's raw-mode state, but we need diagnostics on failure.
	args := []string{"serve", "-f", "--socket", socketPath}
	args = append(args, extraFlags...)
	var stderrBuf bytes.Buffer
	cmd := exec.Command(cryptd, args...)
	cmd.Stderr = &stderrBuf
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start cryptd: %w", err)
	}

	// Reap the child in the background to avoid zombies.
	waitCh := make(chan error, 1)
	go func() { waitCh <- cmd.Wait() }()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		// Check if the server exited before the socket appeared.
		select {
		case err := <-waitCh:
			msg := strings.TrimSpace(stderrBuf.String())
			if msg != "" {
				return nil, fmt.Errorf("cryptd exited before socket ready: %s", msg)
			}
			return nil, fmt.Errorf("cryptd exited before socket ready: %v", err)
		default:
		}
		conn, err = net.DialTimeout("unix", socketPath, 500*time.Millisecond)
		if err == nil {
			return conn, nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Socket never appeared — kill the server.
	_ = cmd.Process.Kill()
	<-waitCh
	return nil, fmt.Errorf("cryptd started but socket %s not ready within 5s", socketPath)
}

// session runs the interactive REPL on an established connection.
func session(conn net.Conn, in io.Reader, out, errOut io.Writer, scenario, charName, charClass, resumeSessionID string) int {
	scanner := bufio.NewScanner(conn)
	// Match the server's 1 MiB buffer to handle large narrated responses.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
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
			Result json.RawMessage  `json:"result"`
			Error  *protocol.RPCError `json:"error"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
			return nil, fmt.Errorf("parse response: %w", err)
		}
		if resp.Error != nil {
			return nil, fmt.Errorf("server error: %s", resp.Error.Message)
		}
		return resp.Result, nil
	}

	// Session init handshake.
	var initParams any
	if resumeSessionID != "" {
		initParams = protocol.InitializeParams{SessionID: resumeSessionID}
	}
	initResult, err := send("session.init", initParams)
	if err != nil {
		fmt.Fprintf(errOut, "error: %v\n", err)
		return 1
	}
	var initResp protocol.InitializeResult
	if err := json.Unmarshal(initResult, &initResp); err == nil && initResp.SessionID != "" {
		fmt.Fprintf(errOut, "crypt: session %s\n", initResp.SessionID)
	}
	resuming := resumeSessionID != "" && initResp.HasGame

	if resuming {
		// The server enters RunLoop immediately on resume and sends the current
		// room as the initial render. Read and display it before entering the REPL.
		if !scanner.Scan() {
			fmt.Fprintf(errOut, "error: connection lost during session resume\n")
			return 1
		}
		var resp struct {
			Result json.RawMessage    `json:"result"`
			Error  *protocol.RPCError `json:"error"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
			fmt.Fprintf(errOut, "error: invalid response during resume: %v\n", err)
			return 1
		}
		if resp.Error != nil {
			fmt.Fprintf(errOut, "error: resume failed: %s\n", resp.Error.Message)
			return 1
		}
		var playResp protocol.PlayResponse
		if err := json.Unmarshal(resp.Result, &playResp); err == nil {
			displayPlayResponse(out, playResp)
		}
	}

	// Auto-start game if --scenario given (skip if resuming an existing session).
	if scenario != "" && !resuming {
		if err := startGame(send, out, errOut, scenario, charName, charClass); err != nil {
			if errors.Is(err, errConnLost) {
				fmt.Fprintf(errOut, "error: %v\n", err)
				return 1
			}
			fmt.Fprintf(errOut, "error: %v\n", err)
		}
	}

	// Interactive REPL with readline for line editing and history.
	fmt.Fprintln(out, "Type 'help' for commands, 'quit' to exit.")

	rl, rlErr := readline.NewFromConfig(&readline.Config{
		Prompt:          "> ",
		InterruptPrompt: "^C",
		EOFPrompt:       "quit",
		Stdin:           in,
		Stdout:          out,
		Stderr:          errOut,
	})
	if rlErr != nil {
		// Readline failed (e.g. not a terminal) — fall back to plain scanner.
		return replScanner(in, out, errOut, send)
	}
	defer func() { _ = rl.Close() }()

	for {
		line, err := rl.Readline()
		if err != nil {
			if errors.Is(err, readline.ErrInterrupt) || errors.Is(err, io.EOF) {
				fmt.Fprintln(out, "Farewell, adventurer.")
				return 0
			}
			// Terminal I/O failure — surface the error.
			fmt.Fprintf(errOut, "input error: %v\n", err)
			return 1
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if code, done := handleLine(line, send, out, errOut); done {
			return code
		}
	}
}

// handleLine processes a single REPL input line. Returns (exitCode, true) if
// the session should end, or (0, false) to continue the loop.
func handleLine(line string, send func(string, any) (json.RawMessage, error), out, errOut io.Writer) (int, bool) {
	if line == "quit" || line == "exit" {
		fmt.Fprintln(out, "Farewell, adventurer.")
		return 0, true
	}

	if line == "help" || line == "?" {
		fmt.Fprintln(out, helpText)
		return 0, false
	}

	// "new <scenario> <name> <class>" is handled client-side.
	if strings.HasPrefix(line, "new ") || strings.HasPrefix(line, "start ") {
		parts := strings.Fields(line)
		if len(parts) < 4 {
			fmt.Fprintln(errOut, "usage: new <scenario_id> <character_name> <character_class>")
			return 0, false
		}
		if err := startGame(send, out, errOut, parts[1], parts[2], parts[3]); err != nil {
			if errors.Is(err, errConnLost) {
				fmt.Fprintf(errOut, "Error: %v\n", err)
				return 1, true
			}
			fmt.Fprintf(errOut, "Error: %v\n", err)
		}
		return 0, false
	}

	// Everything else: send to server as play text.
	result, err := send("game.play", map[string]string{"text": line})
	if err != nil {
		if errors.Is(err, errConnLost) {
			fmt.Fprintf(errOut, "Error: %v\n", err)
			return 1, true
		}
		fmt.Fprintf(errOut, "Error: %v\n", err)
		return 0, false
	}

	var playResp protocol.PlayResponse
	if err := json.Unmarshal(result, &playResp); err != nil {
		// Not a play response — print raw.
		fmt.Fprintln(out, string(result))
		return 0, false
	}
	if displayPlayResponse(out, playResp) {
		return 0, true
	}
	return 0, false
}

// replScanner is the fallback REPL for non-terminal input (pipes, tests).
func replScanner(in io.Reader, out, errOut io.Writer, send func(string, any) (json.RawMessage, error)) int {
	inputScanner := bufio.NewScanner(in)
	for {
		_, _ = fmt.Fprint(out, "> ") // prompt write failure is non-fatal
		if !inputScanner.Scan() {
			break
		}
		line := strings.TrimSpace(inputScanner.Text())
		if line == "" {
			continue
		}
		if code, done := handleLine(line, send, out, errOut); done {
			return code
		}
	}
	return 0
}

// startGame sends a game.new request and displays the initial narration.
func startGame(send func(string, any) (json.RawMessage, error), out, errOut io.Writer, scenario, name, class string) error {
	result, err := send("game.new", map[string]string{
		"scenario_id":     scenario,
		"character_name":  name,
		"character_class": class,
	})
	if err != nil {
		return err
	}

	var playResp protocol.PlayResponse
	if err := json.Unmarshal(result, &playResp); err != nil {
		// Not a play response — print raw to errOut since this is unexpected.
		fmt.Fprintf(errOut, "warning: unexpected response format: %s\n", string(result))
		return nil
	}
	displayPlayResponse(out, playResp)
	return nil
}


const helpText = `commands:
  new <scenario> <name> <class>   Start a new game
  quit/exit                       Disconnect

Everything else is sent to the server as natural language.
The server interprets your commands — try "go north",
"look around", "pick up the sword", "attack", etc.

To resume a previous session: crypt --session <id>`
