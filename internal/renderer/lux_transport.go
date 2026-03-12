package renderer

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/punt-labs/cryptd/internal/model"
)

// luxMessage is the JSON-lines wire format for Lux display communication.
// Each message is a single JSON object followed by a newline.
type luxMessage struct {
	Method  string `json:"method"`  // "show" or "update"
	Payload any    `json:"payload"` // LuxScene or LuxUpdate
}

// JSONTransport implements LuxDisplay over a JSON-lines protocol on
// arbitrary io.Reader/io.Writer streams. Show and update calls write
// newline-delimited JSON to the writer; input events are read as
// newline-delimited JSON from the reader.
type JSONTransport struct {
	mu     sync.Mutex
	w      io.Writer
	events chan model.InputEvent
}

// NewJSONTransport creates a LuxDisplay that writes scene/update payloads as
// JSON lines to w and reads InputEvents as JSON lines from r. The reader
// goroutine runs until r is closed or returns an error.
func NewJSONTransport(w io.Writer, r io.Reader) *JSONTransport {
	t := &JSONTransport{
		w:      w,
		events: make(chan model.InputEvent, 64),
	}
	go t.readEvents(r)
	return t
}

// RecordShow writes a show message to the output stream.
func (t *JSONTransport) RecordShow(payload any) {
	t.writeMessage("show", payload)
}

// RecordUpdate writes an update message to the output stream.
func (t *JSONTransport) RecordUpdate(payload any) {
	t.writeMessage("update", payload)
}

// Events returns the channel of InputEvents read from the input stream.
func (t *JSONTransport) Events() <-chan model.InputEvent {
	return t.events
}

func (t *JSONTransport) writeMessage(method string, payload any) {
	data, err := json.Marshal(luxMessage{Method: method, Payload: payload})
	if err != nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	fmt.Fprintf(t.w, "%s\n", data)
}

func (t *JSONTransport) readEvents(r io.Reader) {
	defer close(t.events)
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		var ev model.InputEvent
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			continue
		}
		t.events <- ev
	}
}
