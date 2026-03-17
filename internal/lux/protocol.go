package lux

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
)

const (
	// HeaderSize is the 4-byte big-endian uint32 length prefix.
	HeaderSize = 4
	// MaxMessageSize is the maximum payload size (16 MiB), matching the Python protocol.
	MaxMessageSize = 16 << 20
	// ProtocolVersion is the current wire protocol version.
	ProtocolVersion = "0.1"
)

// --- Outbound messages (client → display) ---

// SceneMessage sends a full element tree to the display.
type SceneMessage struct {
	Type       string           `json:"type"`
	ID         string           `json:"id"`
	Elements   []map[string]any `json:"elements"`
	Layout     string           `json:"layout,omitempty"`
	Title      *string          `json:"title,omitempty"`
	FrameID    *string          `json:"frame_id,omitempty"`
	FrameTitle *string          `json:"frame_title,omitempty"`
	FrameSize  []int            `json:"frame_size,omitempty"`
}

// UpdateMessage sends incremental patches to the display.
type UpdateMessage struct {
	Type    string           `json:"type"`
	SceneID string           `json:"scene_id"`
	Patches []map[string]any `json:"patches"`
}

// PingMessage is a keepalive probe.
type PingMessage struct {
	Type string  `json:"type"`
	TS   float64 `json:"ts"`
}

// ClearMessage removes all content from the display.
type ClearMessage struct {
	Type string `json:"type"`
}

// ConnectMessage identifies the client after handshake.
type ConnectMessage struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

// --- Inbound messages (display → client) ---

// ReadyMessage is sent by the display after connection.
type ReadyMessage struct {
	Type         string   `json:"type"`
	Version      string   `json:"version"`
	Capabilities []string `json:"capabilities,omitempty"`
}

// AckMessage acknowledges a scene or update.
type AckMessage struct {
	Type    string  `json:"type"`
	SceneID string  `json:"scene_id"`
	TS      float64 `json:"ts,omitempty"`
	Error   *string `json:"error,omitempty"`
}

// InteractionMessage is a user interaction from the display.
type InteractionMessage struct {
	Type      string  `json:"type"`
	ElementID string  `json:"element_id"`
	Action    string  `json:"action"`
	TS        float64 `json:"ts,omitempty"`
	Value     any     `json:"value,omitempty"`
}

// EncodeFrame serializes msg as JSON and prepends the 4-byte length prefix.
func EncodeFrame(msg any) ([]byte, error) {
	payload, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	if len(payload) > MaxMessageSize {
		return nil, fmt.Errorf("message size %d exceeds maximum %d", len(payload), MaxMessageSize)
	}
	frame := make([]byte, HeaderSize+len(payload))
	binary.BigEndian.PutUint32(frame[:HeaderSize], uint32(len(payload)))
	copy(frame[HeaderSize:], payload)
	return frame, nil
}

// DecodeFrame parses one length-prefixed frame from data, returning the
// decoded JSON object and any remaining bytes. Returns ErrIncompleteFrame
// if the buffer doesn't contain a full frame yet.
func DecodeFrame(data []byte) (map[string]any, []byte, error) {
	if len(data) < HeaderSize {
		return nil, data, ErrIncompleteFrame
	}
	length := binary.BigEndian.Uint32(data[:HeaderSize])
	if length > MaxMessageSize {
		return nil, nil, fmt.Errorf("message size %d exceeds maximum %d", length, MaxMessageSize)
	}
	total := HeaderSize + int(length)
	if len(data) < total {
		return nil, data, ErrIncompleteFrame
	}
	var msg map[string]any
	if err := json.Unmarshal(data[HeaderSize:total], &msg); err != nil {
		return nil, data[total:], fmt.Errorf("unmarshal: %w", err)
	}
	return msg, data[total:], nil
}

// ErrIncompleteFrame indicates the buffer doesn't contain a complete frame.
var ErrIncompleteFrame = errors.New("incomplete frame")

// FrameReader accumulates bytes and yields complete messages.
// Feed it with data from net.Conn.Read(), then call Drain() for decoded messages.
type FrameReader struct {
	buf []byte
}

// Feed appends raw bytes to the internal buffer.
func (r *FrameReader) Feed(data []byte) {
	r.buf = append(r.buf, data...)
}

// Drain extracts all complete messages from the buffer.
func (r *FrameReader) Drain() ([]map[string]any, error) {
	var msgs []map[string]any
	for {
		msg, rest, err := DecodeFrame(r.buf)
		if errors.Is(err, ErrIncompleteFrame) {
			return msgs, nil
		}
		if err != nil {
			return msgs, err
		}
		msgs = append(msgs, msg)
		r.buf = rest
	}
}

// ParseInbound classifies a raw JSON dict into a typed inbound message.
// Returns nil for unrecognized message types.
func ParseInbound(raw map[string]any) any {
	typ, _ := raw["type"].(string)
	switch typ {
	case "ready":
		msg := ReadyMessage{Type: "ready"}
		msg.Version, _ = raw["version"].(string)
		if caps, ok := raw["capabilities"].([]any); ok {
			for _, c := range caps {
				if s, ok := c.(string); ok {
					msg.Capabilities = append(msg.Capabilities, s)
				}
			}
		}
		return &msg
	case "ack":
		msg := AckMessage{Type: "ack"}
		msg.SceneID, _ = raw["scene_id"].(string)
		msg.TS, _ = raw["ts"].(float64)
		if e, ok := raw["error"].(string); ok {
			msg.Error = &e
		}
		return &msg
	case "interaction":
		msg := InteractionMessage{Type: "interaction"}
		msg.ElementID, _ = raw["element_id"].(string)
		msg.Action, _ = raw["action"].(string)
		msg.TS, _ = raw["ts"].(float64)
		msg.Value = raw["value"]
		return &msg
	default:
		return nil
	}
}
