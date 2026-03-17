package renderer

import (
	"fmt"
	"sync"

	"github.com/punt-labs/cryptd/internal/model"
)

// ShowRecord captures a single show() call with its translated element tree.
type ShowRecord struct {
	SceneID  string
	Elements []map[string]any
}

// LuxElementDisplay is a LuxDisplay implementation that translates LuxScene/
// LuxUpdate into Lux-native element dicts and records them for test assertions.
// It also translates injected Lux interaction events into InputEvents.
type LuxElementDisplay struct {
	mu      sync.Mutex
	shows   []ShowRecord
	updates [][]map[string]any
	events  chan model.InputEvent
}

// NewLuxElementDisplay creates a LuxElementDisplay with a buffered event channel.
func NewLuxElementDisplay() *LuxElementDisplay {
	return &LuxElementDisplay{
		events: make(chan model.InputEvent, 64),
	}
}

// RecordShow type-asserts the payload as LuxScene, translates it to elements,
// and records the result.
func (d *LuxElementDisplay) RecordShow(payload any) {
	scene, ok := payload.(LuxScene)
	if !ok {
		panic(fmt.Sprintf("LuxElementDisplay.RecordShow: expected LuxScene, got %T", payload))
	}
	elements := SceneToElements(scene)

	d.mu.Lock()
	defer d.mu.Unlock()
	d.shows = append(d.shows, ShowRecord{
		SceneID:  scene.Room,
		Elements: elements,
	})
}

// RecordUpdate type-asserts the payload as LuxUpdate, translates it to patches,
// and records the result.
func (d *LuxElementDisplay) RecordUpdate(payload any) {
	update, ok := payload.(LuxUpdate)
	if !ok {
		panic(fmt.Sprintf("LuxElementDisplay.RecordUpdate: expected LuxUpdate, got %T", payload))
	}
	patches := UpdateToPatches(update)

	d.mu.Lock()
	defer d.mu.Unlock()
	d.updates = append(d.updates, patches)
}

// Events returns the channel of InputEvents.
func (d *LuxElementDisplay) Events() <-chan model.InputEvent {
	return d.events
}

// Shows returns a snapshot of all recorded show calls.
func (d *LuxElementDisplay) Shows() []ShowRecord {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([]ShowRecord, len(d.shows))
	copy(out, d.shows)
	return out
}

// Updates returns a snapshot of all recorded update patch sets.
func (d *LuxElementDisplay) Updates() [][]map[string]any {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([][]map[string]any, len(d.updates))
	copy(out, d.updates)
	return out
}

// InjectInteraction translates a Lux interaction event map into an InputEvent
// and sends it to the event channel. No-op if the event is unrecognized.
// Panics if the channel buffer is full (programmer bug — too many events
// injected before the game loop drains them).
func (d *LuxElementDisplay) InjectInteraction(luxEvent map[string]any) {
	if input, ok := TranslateLuxEvent(luxEvent); ok {
		d.injectOrPanic(input)
	}
}

// InjectEvent sends an InputEvent directly (e.g. quit events).
// Panics if the channel buffer is full.
func (d *LuxElementDisplay) InjectEvent(e model.InputEvent) {
	d.injectOrPanic(e)
}

func (d *LuxElementDisplay) injectOrPanic(e model.InputEvent) {
	select {
	case d.events <- e:
	default:
		panic("LuxElementDisplay: event channel full (capacity 64) — test injects too many events before loop drains them")
	}
}
