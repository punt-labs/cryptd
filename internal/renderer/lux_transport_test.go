package renderer_test

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/punt-labs/cryptd/internal/renderer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONTransport_ShowWritesJSON(t *testing.T) {
	var buf bytes.Buffer
	tr := renderer.NewJSONTransport(&buf, strings.NewReader(""))

	scene := renderer.LuxScene{
		Room:      "entrance",
		Narration: "You arrive.",
		Party:     []renderer.LuxHero{{Name: "Adventurer", Class: "Fighter", HP: 20, MaxHP: 20}},
	}
	tr.RecordShow(scene)

	line := strings.TrimSpace(buf.String())
	require.NotEmpty(t, line)

	var msg struct {
		Method  string           `json:"method"`
		Payload renderer.LuxScene `json:"payload"`
	}
	require.NoError(t, json.Unmarshal([]byte(line), &msg))
	assert.Equal(t, "show", msg.Method)
	assert.Equal(t, "entrance", msg.Payload.Room)
	assert.Equal(t, "You arrive.", msg.Payload.Narration)
	require.Len(t, msg.Payload.Party, 1)
	assert.Equal(t, "Adventurer", msg.Payload.Party[0].Name)
}

func TestJSONTransport_UpdateWritesJSON(t *testing.T) {
	var buf bytes.Buffer
	tr := renderer.NewJSONTransport(&buf, strings.NewReader(""))

	hero := renderer.LuxHero{Name: "Adventurer", HP: 15, MaxHP: 20}
	update := renderer.LuxUpdate{
		Type:    "narration",
		Content: "You take damage.",
		Hero:    &hero,
	}
	tr.RecordUpdate(update)

	line := strings.TrimSpace(buf.String())
	var msg struct {
		Method  string             `json:"method"`
		Payload renderer.LuxUpdate `json:"payload"`
	}
	require.NoError(t, json.Unmarshal([]byte(line), &msg))
	assert.Equal(t, "update", msg.Method)
	assert.Equal(t, "narration", msg.Payload.Type)
	assert.Equal(t, "You take damage.", msg.Payload.Content)
	require.NotNil(t, msg.Payload.Hero)
	assert.Equal(t, 15, msg.Payload.Hero.HP)
}

func TestJSONTransport_ReadsInputEvents(t *testing.T) {
	input := `{"type":"input","payload":"go north"}` + "\n" + `{"type":"quit"}` + "\n"
	tr := renderer.NewJSONTransport(io.Discard, strings.NewReader(input))

	ev1 := <-tr.Events()
	assert.Equal(t, "input", ev1.Type)
	assert.Equal(t, "go north", ev1.Payload)

	ev2 := <-tr.Events()
	assert.Equal(t, "quit", ev2.Type)
}

func TestJSONTransport_ChannelClosesOnEOF(t *testing.T) {
	tr := renderer.NewJSONTransport(io.Discard, strings.NewReader(""))

	_, ok := <-tr.Events()
	assert.False(t, ok, "channel should be closed after reader EOF")
}

func TestJSONTransport_SkipsMalformedInput(t *testing.T) {
	input := "not json\n" + `{"type":"quit"}` + "\n"
	tr := renderer.NewJSONTransport(io.Discard, strings.NewReader(input))

	ev := <-tr.Events()
	assert.Equal(t, "quit", ev.Type, "malformed line should be skipped")
}

func TestJSONTransport_MultipleShowsWriteSeparateLines(t *testing.T) {
	var buf bytes.Buffer
	tr := renderer.NewJSONTransport(&buf, strings.NewReader(""))

	tr.RecordShow(renderer.LuxScene{Room: "room1", Narration: "First."})
	tr.RecordShow(renderer.LuxScene{Room: "room2", Narration: "Second."})

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Len(t, lines, 2)

	var msg1, msg2 struct {
		Payload renderer.LuxScene `json:"payload"`
	}
	require.NoError(t, json.Unmarshal([]byte(lines[0]), &msg1))
	require.NoError(t, json.Unmarshal([]byte(lines[1]), &msg2))
	assert.Equal(t, "room1", msg1.Payload.Room)
	assert.Equal(t, "room2", msg2.Payload.Room)
}
