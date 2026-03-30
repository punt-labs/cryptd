package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/punt-labs/cryptd/internal/protocol"
)

const maxLines = 500

// NarrationPane is the scrolling narration log on the left side.
type NarrationPane struct {
	viewport viewport.Model
	lines    []string
	lastRoom string
	width    int
	height   int
}

// NewNarrationPane creates a narration pane with the given dimensions.
func NewNarrationPane(width, height int) NarrationPane {
	vp := viewport.New(width, height)
	return NarrationPane{
		viewport: vp,
		width:    width,
		height:   height,
	}
}

// AppendText appends a styled line and scrolls to the bottom.
func (p *NarrationPane) AppendText(text string, style lipgloss.Style) {
	p.lines = append(p.lines, style.Render(text))
	p.trimLines()
	p.syncViewport()
}

// AppendResponse appends narration from a PlayResponse.
func (p *NarrationPane) AppendResponse(resp protocol.PlayResponse) {
	if resp.State != nil {
		room := resp.State.Dungeon.CurrentRoom
		if room != "" && room != p.lastRoom {
			p.lines = append(p.lines, StyleRoomName.Render(room))
			p.lastRoom = room
		}
	}

	if resp.Text != "" {
		for _, line := range strings.Split(resp.Text, "\n") {
			p.lines = append(p.lines, StyleNarration.Render(line))
		}
	}

	if resp.Dead {
		p.lines = append(p.lines, StyleDamage.Render("*** You have died ***"))
		p.lines = append(p.lines, StyleSystem.Render("Press Enter to exit."))
	}

	p.trimLines()
	p.syncViewport()
}

// Update forwards messages to the viewport for scroll handling.
func (p NarrationPane) Update(msg tea.Msg) (NarrationPane, tea.Cmd) {
	var cmd tea.Cmd
	p.viewport, cmd = p.viewport.Update(msg)
	return p, cmd
}

// View returns the viewport view.
func (p NarrationPane) View() string {
	return p.viewport.View()
}

// SetSize updates the viewport dimensions.
func (p *NarrationPane) SetSize(width, height int) {
	p.width = width
	p.height = height
	p.viewport.Width = width
	p.viewport.Height = height
}

// trimLines caps the line buffer at maxLines.
func (p *NarrationPane) trimLines() {
	if len(p.lines) > maxLines {
		p.lines = p.lines[len(p.lines)-maxLines:]
	}
}

// syncViewport joins lines and scrolls to the bottom.
func (p *NarrationPane) syncViewport() {
	p.viewport.SetContent(strings.Join(p.lines, "\n"))
	p.viewport.GotoBottom()
}
