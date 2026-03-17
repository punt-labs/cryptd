package lux

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ErrAckTimeout is returned when an ack is not received within recvTimeout.
var ErrAckTimeout = errors.New("ack timeout")

// Client connects to a Lux display server over Unix domain socket
// and speaks the length-prefixed JSON wire protocol.
//
// After Connect(), a background reader goroutine demultiplexes incoming
// messages: AckMessages go to an internal ack channel, InteractionMessages
// go to an interaction channel, and everything else is discarded.
type Client struct {
	socketPath     string
	name           string
	connectTimeout time.Duration
	recvTimeout    time.Duration

	mu           sync.Mutex
	conn         net.Conn
	ready        *ReadyMessage
	closed       bool
	acks         chan *AckMessage
	interactions chan *InteractionMessage
	readerDone   chan struct{}
	readerErr    error // terminal error from readLoop, protected by mu
}

// Option configures a Client.
type Option func(*Client)

// WithSocketPath overrides the default socket path.
func WithSocketPath(path string) Option {
	return func(c *Client) { c.socketPath = path }
}

// WithName sets the client identity sent in the ConnectMessage.
func WithName(name string) Option {
	return func(c *Client) { c.name = name }
}

// WithConnectTimeout sets the handshake timeout.
func WithConnectTimeout(d time.Duration) Option {
	return func(c *Client) { c.connectTimeout = d }
}

// WithRecvTimeout sets the default receive timeout.
func WithRecvTimeout(d time.Duration) Option {
	return func(c *Client) { c.recvTimeout = d }
}

// NewClient creates a Client with the given options.
func NewClient(opts ...Option) *Client {
	c := &Client{
		connectTimeout: 5 * time.Second,
		recvTimeout:    5 * time.Second,
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// DefaultSocketPath returns the socket path using the same resolution order
// as the Python client: $LUX_SOCKET > $XDG_RUNTIME_DIR/lux/display.sock >
// /tmp/lux-$USER/display.sock.
func DefaultSocketPath() string {
	if env := os.Getenv("LUX_SOCKET"); env != "" {
		return env
	}
	if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
		return filepath.Join(xdg, "lux", "display.sock")
	}
	user := os.Getenv("USER")
	if user == "" {
		user = "unknown"
	}
	return filepath.Join("/tmp", "lux-"+user, "display.sock")
}

// Connect dials the Unix socket and performs the ReadyMessage handshake.
// Starts a background reader goroutine that demultiplexes incoming messages.
// Not safe for concurrent use — call once before using other methods.
func (c *Client) Connect() error {
	c.mu.Lock()
	if c.conn != nil {
		c.mu.Unlock()
		return nil
	}
	c.mu.Unlock()

	path := c.socketPath
	if path == "" {
		path = DefaultSocketPath()
	}

	conn, err := net.DialTimeout("unix", path, c.connectTimeout)
	if err != nil {
		return fmt.Errorf("connect to %s: %w", path, err)
	}

	// Read the ReadyMessage synchronously before starting the reader goroutine.
	readyMsg, err := readHandshake(conn, c.connectTimeout)
	if err != nil {
		conn.Close()
		return fmt.Errorf("handshake: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.ready = readyMsg
	c.acks = make(chan *AckMessage, 16)
	c.interactions = make(chan *InteractionMessage, 64)
	c.readerDone = make(chan struct{})
	c.readerErr = nil
	c.mu.Unlock()

	// Send ConnectMessage if name is set
	if c.name != "" {
		if err := c.send(ConnectMessage{Type: "connect", Name: c.name}); err != nil {
			conn.Close()
			c.mu.Lock()
			c.conn = nil
			c.mu.Unlock()
			return fmt.Errorf("send connect: %w", err)
		}
	}

	// Start background reader that demuxes acks vs interactions
	go c.readLoop()

	return nil
}

// readHandshake reads the first frame and expects a ReadyMessage.
func readHandshake(conn net.Conn, timeout time.Duration) (*ReadyMessage, error) {
	if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return nil, err
	}
	defer conn.SetReadDeadline(time.Time{})

	var reader FrameReader
	buf := make([]byte, 65536)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			return nil, err
		}
		reader.Feed(buf[:n])
		msgs, err := reader.Drain()
		if err != nil {
			return nil, err
		}
		for _, raw := range msgs {
			parsed := ParseInbound(raw)
			if msg, ok := parsed.(*ReadyMessage); ok {
				return msg, nil
			}
		}
	}
}

// readLoop is the background goroutine that reads frames and routes them.
// On any error, stores it in readerErr and exits.
func (c *Client) readLoop() {
	defer close(c.readerDone)

	var reader FrameReader
	buf := make([]byte, 65536)

	for {
		c.mu.Lock()
		conn := c.conn
		c.mu.Unlock()
		if conn == nil {
			return
		}

		n, err := conn.Read(buf)
		if err != nil {
			c.mu.Lock()
			c.readerErr = fmt.Errorf("read loop: %w", err)
			c.mu.Unlock()
			return
		}
		reader.Feed(buf[:n])
		msgs, err := reader.Drain()
		if err != nil {
			c.mu.Lock()
			c.readerErr = fmt.Errorf("read loop drain: %w", err)
			c.mu.Unlock()
			return
		}
		for _, raw := range msgs {
			parsed := ParseInbound(raw)
			switch msg := parsed.(type) {
			case *AckMessage:
				// Block until consumed — never drop acks.
				c.acks <- msg
			case *InteractionMessage:
				select {
				case c.interactions <- msg:
				default:
					// Interaction channel full — drop oldest to make room.
					// This is less dangerous than dropping acks; interactions
					// are user input and the player can re-click.
					<-c.interactions
					c.interactions <- msg
				}
			}
		}
	}
}

// Close shuts down the connection and waits for the reader to exit.
func (c *Client) Close() error {
	c.mu.Lock()
	c.closed = true
	conn := c.conn
	readerDone := c.readerDone
	c.conn = nil
	c.mu.Unlock()

	if conn == nil {
		return nil
	}
	err := conn.Close()
	if readerDone != nil {
		<-readerDone
	}
	return err
}

// IsConnected returns whether the client has an active connection.
func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn != nil
}

// Interactions returns the channel of InteractionMessages from the display.
// Used by Display.eventLoop to select directly without polling.
func (c *Client) Interactions() <-chan *InteractionMessage {
	return c.interactions
}

// ReaderErr returns the terminal error from the reader goroutine, if any.
func (c *Client) ReaderErr() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.readerErr
}

// ShowOpts configures optional fields for a Show call.
type ShowOpts struct {
	FrameID    string
	FrameTitle string
	FrameSize  [2]int // width, height; zero means unset
}

// Show sends a SceneMessage and waits for an AckMessage.
func (c *Client) Show(sceneID string, elements []map[string]any, opts *ShowOpts) (*AckMessage, error) {
	msg := SceneMessage{
		Type:     "scene",
		ID:       sceneID,
		Elements: elements,
		Layout:   "single",
	}
	if opts != nil {
		if opts.FrameID != "" {
			msg.FrameID = &opts.FrameID
		}
		if opts.FrameTitle != "" {
			msg.FrameTitle = &opts.FrameTitle
		}
		if opts.FrameSize != [2]int{} {
			msg.FrameSize = []int{opts.FrameSize[0], opts.FrameSize[1]}
		}
	}
	if err := c.send(msg); err != nil {
		return nil, err
	}
	return c.waitAck()
}

// Update sends an UpdateMessage and waits for an AckMessage.
func (c *Client) Update(sceneID string, patches []map[string]any) (*AckMessage, error) {
	msg := UpdateMessage{
		Type:    "update",
		SceneID: sceneID,
		Patches: patches,
	}
	if err := c.send(msg); err != nil {
		return nil, err
	}
	return c.waitAck()
}

// Ping sends a PingMessage. Fire-and-forget — no response expected.
func (c *Client) Ping() error {
	return c.send(PingMessage{
		Type: "ping",
		TS:   float64(time.Now().UnixMilli()) / 1000.0,
	})
}

// Clear sends a ClearMessage to remove all display content.
func (c *Client) Clear() error {
	return c.send(ClearMessage{Type: "clear"})
}

// Recv waits for the next InteractionMessage. Returns nil, nil if timeout
// expires with no interaction available.
func (c *Client) Recv(timeout time.Duration) (*InteractionMessage, error) {
	select {
	case msg := <-c.interactions:
		return msg, nil
	case <-time.After(timeout):
		return nil, nil
	}
}

// send encodes and writes a framed message.
func (c *Client) send(msg any) error {
	frame, err := EncodeFrame(msg)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return fmt.Errorf("not connected")
	}
	_, err = c.conn.Write(frame)
	return err
}

// waitAck waits for an AckMessage from the reader goroutine.
// Returns ErrAckTimeout if no ack arrives within recvTimeout.
func (c *Client) waitAck() (*AckMessage, error) {
	select {
	case ack := <-c.acks:
		return ack, nil
	case <-c.readerDone:
		// Reader exited — return stored error
		c.mu.Lock()
		err := c.readerErr
		c.mu.Unlock()
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("connection closed")
	case <-time.After(c.recvTimeout):
		return nil, ErrAckTimeout
	}
}
