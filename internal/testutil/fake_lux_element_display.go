package testutil

import (
	"fmt"
	"sync"

	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/renderer"
)

// LuxShowRecord captures a single show() call with its translated element tree.
type LuxShowRecord struct {
	SceneID  string
	Elements []map[string]any
}

// FakeLuxElementDisplay is a LuxDisplay implementation that translates LuxScene/
// LuxUpdate into Lux-native element dicts and records them for test assertions.
// It also translates injected Lux interaction events into InputEvents.
type FakeLuxElementDisplay struct {
	mu      sync.Mutex
	shows   []LuxShowRecord
	updates [][]map[string]any
	events  chan model.InputEvent
}

// NewFakeLuxElementDisplay creates a FakeLuxElementDisplay with a buffered event channel.
func NewFakeLuxElementDisplay() *FakeLuxElementDisplay {
	return &FakeLuxElementDisplay{
		events: make(chan model.InputEvent, 64),
	}
}

// RecordShow type-asserts the payload as LuxScene, translates it to elements,
// and records the result. Panics on wrong type (programmer bug in test wiring).
func (d *FakeLuxElementDisplay) RecordShow(payload any) {
	scene, ok := payload.(renderer.LuxScene)
	if !ok {
		panic(fmt.Sprintf("FakeLuxElementDisplay.RecordShow: expected LuxScene, got %T", payload))
	}
	elements := renderer.SceneToElements(scene)

	d.mu.Lock()
	defer d.mu.Unlock()
	d.shows = append(d.shows, LuxShowRecord{
		SceneID:  scene.Room,
		Elements: elements,
	})
}

// RecordUpdate type-asserts the payload as LuxUpdate, translates it to patches,
// and records the result. Panics on wrong type.
func (d *FakeLuxElementDisplay) RecordUpdate(payload any) {
	update, ok := payload.(renderer.LuxUpdate)
	if !ok {
		panic(fmt.Sprintf("FakeLuxElementDisplay.RecordUpdate: expected LuxUpdate, got %T", payload))
	}
	patches := renderer.UpdateToPatches(update)

	d.mu.Lock()
	defer d.mu.Unlock()
	d.updates = append(d.updates, patches)
}

// Events returns the channel of InputEvents.
func (d *FakeLuxElementDisplay) Events() <-chan model.InputEvent {
	return d.events
}

// Shows returns a snapshot of all recorded show calls.
func (d *FakeLuxElementDisplay) Shows() []LuxShowRecord {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([]LuxShowRecord, len(d.shows))
	copy(out, d.shows)
	return out
}

// Updates returns a snapshot of all recorded update patch sets.
func (d *FakeLuxElementDisplay) Updates() [][]map[string]any {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([][]map[string]any, len(d.updates))
	copy(out, d.updates)
	return out
}

// InjectInteraction translates a Lux interaction event map into an InputEvent
// and sends it to the event channel. No-op if the event is unrecognized.
// Panics if the channel buffer is full.
func (d *FakeLuxElementDisplay) InjectInteraction(luxEvent map[string]any) {
	if input, ok := renderer.TranslateLuxEvent(luxEvent); ok {
		d.injectOrPanic(input)
	}
}

// InjectEvent sends an InputEvent directly (e.g. quit events).
// Panics if the channel buffer is full.
func (d *FakeLuxElementDisplay) InjectEvent(e model.InputEvent) {
	d.injectOrPanic(e)
}

func (d *FakeLuxElementDisplay) injectOrPanic(e model.InputEvent) {
	select {
	case d.events <- e:
	default:
		panic("FakeLuxElementDisplay: event channel full (capacity 64) — test injects too many events before loop drains them")
	}
}
