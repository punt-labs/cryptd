package lux

import (
	"testing"
	"time"

	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/renderer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDisplay_ShowTranslatesScene(t *testing.T) {
	srv := newMiniServer(t)
	defer srv.close()

	c := NewClient(
		WithSocketPath(srv.sockPath),
		WithRecvTimeout(2*time.Second),
	)

	go srv.acceptAndHandshake()
	require.NoError(t, c.Connect())

	display := NewDisplay(c, "test-scene", nil)
	defer display.Close()

	// Server goroutine: read scene, send ack
	done := make(chan map[string]any, 1)
	go func() {
		msg := srv.readFrame()
		srv.sendFrame(AckMessage{Type: "ack", SceneID: "test-scene"})
		done <- msg
	}()

	scene := renderer.LuxScene{
		Room:      "entrance",
		Narration: "You stand at the entrance.",
		Party: []renderer.LuxHero{
			{Name: "Hero", Class: "Fighter", Level: 1, HP: 100, MaxHP: 100},
		},
		Actions: []string{"look", "inventory"},
	}
	display.RecordShow(scene)

	msg := <-done
	assert.Equal(t, "scene", msg["type"])
	assert.Equal(t, "test-scene", msg["id"])

	// Verify elements were translated (should have room_header, party, narration, buttons)
	elements, ok := msg["elements"].([]any)
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(elements), 3)

	assert.NoError(t, display.WriteErr())
}

func TestDisplay_UpdateTranslatesPatches(t *testing.T) {
	srv := newMiniServer(t)
	defer srv.close()

	c := NewClient(
		WithSocketPath(srv.sockPath),
		WithRecvTimeout(2*time.Second),
	)

	go srv.acceptAndHandshake()
	require.NoError(t, c.Connect())

	display := NewDisplay(c, "test-scene", nil)
	defer display.Close()

	done := make(chan map[string]any, 1)
	go func() {
		msg := srv.readFrame()
		srv.sendFrame(AckMessage{Type: "ack", SceneID: "test-scene"})
		done <- msg
	}()

	update := renderer.LuxUpdate{
		Type:    "narration",
		Content: "You look around the room.",
	}
	display.RecordUpdate(update)

	msg := <-done
	assert.Equal(t, "update", msg["type"])
	assert.Equal(t, "test-scene", msg["scene_id"])
	assert.NoError(t, display.WriteErr())
}

func TestDisplay_EventsTranslateInteractions(t *testing.T) {
	srv := newMiniServer(t)
	defer srv.close()

	c := NewClient(
		WithSocketPath(srv.sockPath),
		WithRecvTimeout(2*time.Second),
	)

	go srv.acceptAndHandshake()
	require.NoError(t, c.Connect())

	display := NewDisplay(c, "test-scene", nil)
	defer display.Close()

	// Server sends an interaction
	srv.sendFrame(InteractionMessage{
		Type:      "interaction",
		ElementID: "act_attack",
		Action:    "clicked",
	})

	// Read from the display's event channel
	select {
	case evt := <-display.Events():
		assert.Equal(t, model.InputEvent{Type: "input", Payload: "attack"}, evt)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestDisplay_TextInputBuffering(t *testing.T) {
	srv := newMiniServer(t)
	defer srv.close()

	c := NewClient(
		WithSocketPath(srv.sockPath),
		WithRecvTimeout(2*time.Second),
	)

	go srv.acceptAndHandshake()
	require.NoError(t, c.Connect())

	display := NewDisplay(c, "test-scene", nil)
	defer display.Close()

	// Send text input "changed" event followed by Send button click.
	srv.sendFrame(InteractionMessage{
		Type:      "interaction",
		ElementID: "cmd_input",
		Action:    "changed",
		Value:     "go north",
	})
	srv.sendFrame(InteractionMessage{
		Type:      "interaction",
		ElementID: "act_send",
		Action:    "act_send",
	})

	select {
	case evt := <-display.Events():
		assert.Equal(t, model.InputEvent{Type: "input", Payload: "go north"}, evt)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for text input event")
	}
}

func TestDisplay_EmptySendIgnored(t *testing.T) {
	srv := newMiniServer(t)
	defer srv.close()

	c := NewClient(
		WithSocketPath(srv.sockPath),
		WithRecvTimeout(2*time.Second),
	)

	go srv.acceptAndHandshake()
	require.NoError(t, c.Connect())

	display := NewDisplay(c, "test-scene", nil)
	defer display.Close()

	// Send button click without prior text input — should be ignored.
	srv.sendFrame(InteractionMessage{
		Type:      "interaction",
		ElementID: "act_send",
		Action:    "act_send",
	})
	// Then send a regular button click to verify the channel isn't blocked.
	srv.sendFrame(InteractionMessage{
		Type:      "interaction",
		ElementID: "act_look",
		Action:    "act_look",
	})

	select {
	case evt := <-display.Events():
		assert.Equal(t, model.InputEvent{Type: "input", Payload: "look"}, evt)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out — empty send may have produced garbage event")
	}
}

func TestDisplay_WriteErrReportsAckError(t *testing.T) {
	srv := newMiniServer(t)
	defer srv.close()

	c := NewClient(
		WithSocketPath(srv.sockPath),
		WithRecvTimeout(2*time.Second),
	)

	go srv.acceptAndHandshake()
	require.NoError(t, c.Connect())

	display := NewDisplay(c, "test-scene", nil)
	defer display.Close()

	// Server sends ack with error
	go func() {
		srv.readFrame()
		errStr := "bad scene"
		srv.sendFrame(AckMessage{Type: "ack", SceneID: "test-scene", Error: &errStr})
	}()

	scene := renderer.LuxScene{Room: "entrance", Narration: "test"}
	display.RecordShow(scene)

	err := display.WriteErr()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad scene")
}
