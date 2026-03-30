package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// InputBar wraps a textinput for the command prompt at the bottom of the screen.
type InputBar struct {
	input   textinput.Model
	waiting bool
	width   int
}

// NewInputBar creates an InputBar with gold "> " prompt, focused, 256 char limit.
func NewInputBar() InputBar {
	ti := textinput.New()
	ti.Prompt = "> "
	ti.PromptStyle = StylePrompt
	ti.Focus() // return value intentionally discarded during construction
	ti.CharLimit = 256
	return InputBar{input: ti}
}

// Update handles key events for the input bar.
func (b InputBar) Update(msg tea.Msg) (InputBar, tea.Cmd) {
	if b.waiting {
		if msg, ok := msg.(tea.KeyMsg); ok {
			if msg.Type == tea.KeyCtrlC {
				return b, tea.Quit
			}
		}
		return b, nil
	}

	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.Type {
		case tea.KeyCtrlC:
			return b, tea.Quit
		case tea.KeyEnter:
			text := strings.TrimSpace(b.input.Value())
			if text == "" {
				return b, nil
			}
			b.input.Reset()
			return b, func() tea.Msg { return SendCmdMsg{Text: text} }
		}
	}

	var cmd tea.Cmd
	b.input, cmd = b.input.Update(msg)
	return b, cmd
}

// View renders the input bar.
func (b InputBar) View() string {
	if b.waiting {
		return StyleSystem.Render("> sending...")
	}
	return b.input.View()
}

// SetWaiting toggles the waiting state.
func (b *InputBar) SetWaiting(v bool) {
	b.waiting = v
}

// SetWidth updates the width and resizes the textinput.
func (b *InputBar) SetWidth(w int) {
	b.width = w
	b.input.Width = w
}

// Focus delegates to the textinput.
func (b *InputBar) Focus() tea.Cmd {
	return b.input.Focus()
}

// Blur delegates to the textinput.
func (b *InputBar) Blur() {
	b.input.Blur()
}

// Focused returns whether the textinput is focused.
func (b InputBar) Focused() bool {
	return b.input.Focused()
}
