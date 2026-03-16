package daemon

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/punt-labs/cryptd/internal/engine"
	"github.com/punt-labs/cryptd/internal/model"
	"golang.org/x/term"
)

// Server is the game server that exposes MCP tools over a network connection.
// It supports both Unix sockets and TCP (DES-025).
//
// Two modes (DES-025 revised):
//   - Normal: interpreter → engine → narrator → display text (for crypt CLI)
//   - Passthrough: raw MCP tool surface with structured JSON (for Claude Code)
type Server struct {
	eng         *engine.Engine
	state       *model.GameState
	interp      model.CommandInterpreter
	narr        model.Narrator
	passthrough     bool   // true = structured JSON, false = interpreted + narrated text
	network         string // "unix" or "tcp"
	address         string // socket path or host:port
	scenarioDir     string
	defaultScenario string // scenario ID used when new_game omits scenario_id
	listener    net.Listener
	mu          sync.Mutex // guards eng and state
}

// ServerOption configures a Server.
type ServerOption func(*Server)

// WithPassthrough enables passthrough mode: raw MCP tool surface with structured
// JSON responses. When disabled (the default), the server interprets text input
// and returns narrated display text.
func WithPassthrough() ServerOption {
	return func(s *Server) { s.passthrough = true }
}

// WithInterpreter sets the command interpreter for normal mode.
func WithInterpreter(interp model.CommandInterpreter) ServerOption {
	return func(s *Server) { s.interp = interp }
}

// WithNarrator sets the narrator for normal mode.
func WithNarrator(narr model.Narrator) ServerOption {
	return func(s *Server) { s.narr = narr }
}

// WithDefaultScenario sets the scenario ID used when new_game is called
// without a scenario_id argument.
func WithDefaultScenario(id string) ServerOption {
	return func(s *Server) { s.defaultScenario = id }
}

// NewServer creates a Server that listens on a Unix socket at the given path.
// scenarioDir is the directory to search for scenario YAML files.
func NewServer(socketPath, scenarioDir string, opts ...ServerOption) *Server {
	s := &Server{
		network:     "unix",
		address:     socketPath,
		scenarioDir: scenarioDir,
	}
	for _, o := range opts {
		o(s)
	}
	s.defaultPassthrough()
	return s
}

// NewTCPServer creates a Server that listens on a TCP address (e.g. ":9000").
// scenarioDir is the directory to search for scenario YAML files.
func NewTCPServer(listenAddr, scenarioDir string, opts ...ServerOption) *Server {
	s := &Server{
		network:     "tcp",
		address:     listenAddr,
		scenarioDir: scenarioDir,
	}
	for _, o := range opts {
		o(s)
	}
	s.defaultPassthrough()
	return s
}

// defaultPassthrough enables passthrough mode when neither interpreter nor narrator
// is configured. Without this, normal mode would panic on nil interp/narr.
func (s *Server) defaultPassthrough() {
	if !s.passthrough && (s.interp == nil || s.narr == nil) {
		s.passthrough = true
	}
}

// ListenAndServe starts listening and blocks until interrupted by
// SIGINT/SIGTERM. It accepts one connection at a time (single-session for M8).
func (s *Server) ListenAndServe() error {
	if s.network == "unix" {
		return s.listenUnix()
	}
	return s.listenTCP()
}

func (s *Server) listenUnix() error {
	// Ensure parent directory exists.
	dir := filepath.Dir(s.address)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create socket directory: %w", err)
	}

	// Remove stale socket file if it exists.
	if err := os.Remove(s.address); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove stale socket: %w", err)
	}

	ln, err := net.Listen("unix", s.address)
	if err != nil {
		return fmt.Errorf("listen unix: %w", err)
	}

	// Restrict socket permissions to owner only, independent of umask.
	if err := os.Chmod(s.address, 0o600); err != nil {
		ln.Close()
		return fmt.Errorf("set unix socket permissions: %w", err)
	}

	// Clean up socket file on shutdown.
	defer os.Remove(s.address)

	return s.Serve(ln)
}

func (s *Server) listenTCP() error {
	ln, err := net.Listen("tcp", s.address)
	if err != nil {
		return fmt.Errorf("listen tcp: %w", err)
	}
	log.Println("daemon: WARNING: TCP mode has no authentication — do not expose to untrusted networks")
	return s.Serve(ln)
}

// Serve runs the accept loop on an already-opened listener.
// Use this when you need to control listener creation (e.g., for ":0" port assignment).
func (s *Server) Serve(ln net.Listener) error {
	s.listener = ln
	s.address = ln.Addr().String()

	defer ln.Close()

	// Handle signals for graceful shutdown.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer func() {
		signal.Stop(sigCh)
		close(sigCh)
	}()
	go func() {
		sig, ok := <-sigCh
		if !ok {
			return
		}
		// When stderr is a terminal, print a newline so the shutdown
		// message starts on a clean line after the terminal's ^C echo.
		if term.IsTerminal(int(os.Stderr.Fd())) {
			fmt.Fprintln(os.Stderr)
		}
		log.Printf("daemon: received %s, shutting down", sig)
		ln.Close()
	}()

	log.Printf("daemon: listening on %s %s", s.network, s.address)

	for {
		conn, err := ln.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil // intentional shutdown
			}
			return fmt.Errorf("accept: %w", err)
		}
		log.Println("daemon: client connected")
		s.handleConnection(conn, conn)
		if err := conn.Close(); err != nil {
			log.Printf("daemon: conn close: %v", err)
		}
		log.Println("daemon: client disconnected")
	}
}

// Address returns the configured listen address.
func (s *Server) Address() string {
	return s.address
}
