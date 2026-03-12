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
)

// Server is the game server that exposes MCP tools over a network connection.
// It supports both Unix sockets and TCP (DES-025).
type Server struct {
	eng         *engine.Engine
	state       *model.GameState
	network     string // "unix" or "tcp"
	address     string // socket path or host:port
	scenarioDir string
	listener    net.Listener
	mu          sync.Mutex // guards eng and state
}

// NewServer creates a Server that listens on a Unix socket at the given path.
// scenarioDir is the directory to search for scenario YAML files.
func NewServer(socketPath, scenarioDir string) *Server {
	return &Server{
		network:     "unix",
		address:     socketPath,
		scenarioDir: scenarioDir,
	}
}

// NewTCPServer creates a Server that listens on a TCP address (e.g. ":9000").
// scenarioDir is the directory to search for scenario YAML files.
func NewTCPServer(listenAddr, scenarioDir string) *Server {
	return &Server{
		network:     "tcp",
		address:     listenAddr,
		scenarioDir: scenarioDir,
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

	defer ln.Close()

	// Handle signals for graceful shutdown.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)
	go func() {
		sig, ok := <-sigCh
		if !ok {
			return
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
