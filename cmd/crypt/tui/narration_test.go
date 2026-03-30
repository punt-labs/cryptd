package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNarrationPane(t *testing.T) {
	tests := []struct {
		name  string
		run   func(p *NarrationPane)
		check func(t *testing.T, p *NarrationPane)
	}{
		{
			name: "AppendText adds line visible in viewport",
			run: func(p *NarrationPane) {
				p.AppendText("hello dungeon", lipgloss.NewStyle())
			},
			check: func(t *testing.T, p *NarrationPane) {
				assert.Len(t, p.lines, 1)
				assert.Contains(t, p.lines[0], "hello dungeon")
			},
		},
		{
			name: "AppendResponse with room change adds room header",
			run: func(p *NarrationPane) {
				p.AppendResponse(protocol.PlayResponse{
					Text: "A dark corridor.",
					State: &model.GameState{
						Dungeon: model.DungeonState{CurrentRoom: "Crypt Entrance"},
					},
				})
			},
			check: func(t *testing.T, p *NarrationPane) {
				// First line is the room name, second is the narration.
				assert.GreaterOrEqual(t, len(p.lines), 2)
				assert.Contains(t, p.lines[0], "Crypt Entrance")
				assert.Contains(t, p.lines[1], "A dark corridor.")
			},
		},
		{
			name: "AppendResponse with dead adds death notice",
			run: func(p *NarrationPane) {
				p.AppendResponse(protocol.PlayResponse{
					Text: "The dragon strikes.",
					Dead: true,
				})
			},
			check: func(t *testing.T, p *NarrationPane) {
				require.GreaterOrEqual(t, len(p.lines), 2)
				deathLine := p.lines[len(p.lines)-2]
				assert.Contains(t, deathLine, "You have died")
				exitLine := p.lines[len(p.lines)-1]
				assert.Contains(t, exitLine, "Press Enter to exit")
			},
		},
		{
			name: "line buffer caps at maxLines",
			run: func(p *NarrationPane) {
				for i := 0; i < maxLines+50; i++ {
					p.AppendText("line", lipgloss.NewStyle())
				}
			},
			check: func(t *testing.T, p *NarrationPane) {
				assert.Equal(t, maxLines, len(p.lines))
			},
		},
		{
			name: "SetSize updates viewport dimensions",
			run: func(p *NarrationPane) {
				p.SetSize(100, 50)
			},
			check: func(t *testing.T, p *NarrationPane) {
				assert.Equal(t, 100, p.viewport.Width)
				assert.Equal(t, 50, p.viewport.Height)
				assert.Equal(t, 100, p.width)
				assert.Equal(t, 50, p.height)
			},
		},
		{
			name: "AppendResponse same room does not duplicate header",
			run: func(p *NarrationPane) {
				state := &model.GameState{
					Dungeon: model.DungeonState{CurrentRoom: "Hall"},
				}
				p.AppendResponse(protocol.PlayResponse{Text: "first", State: state})
				p.AppendResponse(protocol.PlayResponse{Text: "second", State: state})
			},
			check: func(t *testing.T, p *NarrationPane) {
				// "Hall" should appear exactly once across all lines.
				count := 0
				for _, line := range p.lines {
					if strings.Contains(line, "Hall") {
						count++
					}
				}
				assert.Equal(t, 1, count, "room header should appear once")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewNarrationPane(80, 24)
			tt.run(&p)
			tt.check(t, &p)
		})
	}
}
