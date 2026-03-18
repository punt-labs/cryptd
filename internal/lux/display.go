package lux

import (
	"fmt"
	"sync"

	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/renderer"
)

// Display implements renderer.LuxDisplay using a real Lux Client.
// It translates LuxScene/LuxUpdate payloads into wire protocol messages
// and forwards InteractionMessages as InputEvents.
type Display struct {
	client   *Client
	sceneID  string
	showOpts *ShowOpts
	events   chan model.InputEvent
	done     chan struct{}
	once     sync.Once
	writeErr error
	mu       sync.Mutex
}

// NewDisplay creates a Display backed by the given Client.
// sceneID is the stable scene identifier used for all show/update calls.
// opts configures the Lux frame (pass nil for main window, but callers
// should always set frame_id to avoid colliding with other Lux clients).
func NewDisplay(client *Client, sceneID string, opts *ShowOpts) *Display {
	d := &Display{
		client:   client,
		sceneID:  sceneID,
		showOpts: opts,
		events:   make(chan model.InputEvent, 64),
		done:     make(chan struct{}),
	}
	go d.eventLoop()
	return d
}

// RecordShow translates a LuxScene into elements and sends via client.Show().
func (d *Display) RecordShow(payload any) {
	scene, ok := payload.(renderer.LuxScene)
	if !ok {
		panic(fmt.Sprintf("Display.RecordShow: expected LuxScene, got %T", payload))
	}
	elements := renderer.SceneToElements(scene)
	ack, err := d.client.Show(d.sceneID, elements, d.showOpts)
	if err != nil {
		d.setWriteErr(err)
		return
	}
	if ack != nil && ack.Error != nil {
		d.setWriteErr(fmt.Errorf("display error: %s", *ack.Error))
	}
}

// RecordUpdate translates a LuxUpdate into patches and sends via client.Update().
func (d *Display) RecordUpdate(payload any) {
	update, ok := payload.(renderer.LuxUpdate)
	if !ok {
		panic(fmt.Sprintf("Display.RecordUpdate: expected LuxUpdate, got %T", payload))
	}
	patches := renderer.UpdateToPatches(update)
	ack, err := d.client.Update(d.sceneID, patches)
	if err != nil {
		d.setWriteErr(err)
		return
	}
	if ack != nil && ack.Error != nil {
		d.setWriteErr(fmt.Errorf("display error: %s", *ack.Error))
	}
}

// Events returns the channel of InputEvents translated from Lux interactions.
func (d *Display) Events() <-chan model.InputEvent {
	return d.events
}

// WriteErr returns the first write error, if any. Implements the optional
// errReporter interface checked by renderer.Lux.displayErr().
func (d *Display) WriteErr() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.writeErr
}

// Close stops the event loop goroutine and closes the client.
func (d *Display) Close() error {
	d.once.Do(func() { close(d.done) })
	return d.client.Close()
}

func (d *Display) setWriteErr(err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.writeErr == nil {
		d.writeErr = err
	}
}

// eventLoop selects on the client's interaction channel, the reader-done
// signal, and the display-done signal. Translates InteractionMessages to
// InputEvents via TranslateLuxEvent. Buffers text input and submits on
// Send button click.
func (d *Display) eventLoop() {
	defer close(d.events)
	var pendingText string
	for {
		select {
		case <-d.done:
			return
		case <-d.client.ReaderDone():
			if err := d.client.ReaderErr(); err != nil {
				d.setWriteErr(fmt.Errorf("lux connection lost: %w", err))
			}
			return
		case inter, ok := <-d.client.Interactions():
			if !ok {
				return
			}
			luxEvent := map[string]any{
				"element_id": inter.ElementID,
				"action":     inter.Action,
			}
			if inter.Value != nil {
				luxEvent["value"] = inter.Value
			}

			// Buffer text input changes.
			if text, ok := renderer.TranslateLuxTextInput(luxEvent); ok {
				pendingText = text
				continue
			}

			// Send button submits buffered text (skip if empty or non-click action).
			if inter.ElementID == "act_send" && (inter.Action == "clicked" || inter.Action == "act_send") {
				if pendingText == "" {
					continue
				}
				input := model.InputEvent{Type: "input", Payload: pendingText}
				pendingText = ""
				select {
				case d.events <- input:
				case <-d.done:
					return
				}
				continue
			}

			// Other button clicks.
			if input, ok := renderer.TranslateLuxEvent(luxEvent); ok {
				select {
				case d.events <- input:
				case <-d.done:
					return
				}
			}
		}
	}
}
