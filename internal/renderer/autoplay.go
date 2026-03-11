package renderer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/punt-labs/cryptd/internal/model"
)

// TranscriptEntry is one request-response pair in an autoplay transcript.
type TranscriptEntry struct {
	Command  string `json:"command"`
	Room     string `json:"room"`
	Response string `json:"response"`
}

// Autoplay is a Renderer that feeds commands from a queue and writes a
// formatted transcript. It implements the same Renderer interface as CLI,
// so the game loop runs identically.
type Autoplay struct {
	out      io.Writer
	commands []string
	idx      int
	events   chan model.InputEvent
	json     bool
	quitSent bool

	// Transcript collects all entries for JSON output.
	Transcript []TranscriptEntry
}

// NewAutoplay creates an Autoplay renderer that will execute the given commands
// in order. If jsonMode is true, output is JSON; otherwise plain text.
func NewAutoplay(out io.Writer, commands []string, jsonMode bool) *Autoplay {
	return &Autoplay{
		out:      out,
		commands: commands,
		events:   make(chan model.InputEvent, 1),
		json:     jsonMode,
	}
}

// Render writes the game's response and queues the next command.
func (a *Autoplay) Render(ctx context.Context, state model.GameState, narration string) error {
	// Once quit has been sent, don't record or queue anything further.
	if a.quitSent {
		return nil
	}

	// The first Render call is the initial room description (no command yet).
	// Subsequent calls are responses to commands we sent.
	if a.idx == 0 {
		// Initial state — record it but don't attribute to a command.
		entry := TranscriptEntry{
			Room:     state.Dungeon.CurrentRoom,
			Response: narration,
		}
		a.Transcript = append(a.Transcript, entry)
		if !a.json {
			if _, err := fmt.Fprintf(a.out, "[%s]\n%s\n", entry.Room, entry.Response); err != nil {
				return err
			}
		}
	} else {
		entry := TranscriptEntry{
			Command:  a.commands[a.idx-1],
			Room:     state.Dungeon.CurrentRoom,
			Response: narration,
		}
		a.Transcript = append(a.Transcript, entry)
		if !a.json {
			if _, err := fmt.Fprintf(a.out, "\n> %s\n[%s]\n%s\n", entry.Command, entry.Room, entry.Response); err != nil {
				return err
			}
		}
	}

	// Queue the next command, or signal end of input.
	if a.idx < len(a.commands) {
		cmd := a.commands[a.idx]
		a.idx++
		select {
		case a.events <- model.InputEvent{Type: "input", Payload: cmd}:
		case <-ctx.Done():
			return ctx.Err()
		}
	} else {
		// No more commands — send quit.
		a.quitSent = true
		select {
		case a.events <- model.InputEvent{Type: "quit"}:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

// Events returns the channel on which InputEvents are sent.
func (a *Autoplay) Events() <-chan model.InputEvent {
	return a.events
}

// WriteJSON writes the full transcript as JSON to the output writer.
func (a *Autoplay) WriteJSON() error {
	enc := json.NewEncoder(a.out)
	enc.SetIndent("", "  ")
	return enc.Encode(a.Transcript)
}
