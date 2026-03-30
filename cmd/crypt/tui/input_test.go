package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInputBar(t *testing.T) {
	tests := []struct {
		name  string
		setup func(InputBar) InputBar
		msg   tea.Msg
		check func(t *testing.T, bar InputBar, cmd tea.Cmd)
	}{
		{
			name: "enter with text emits SendCmdMsg and resets",
			setup: func(b InputBar) InputBar {
				// Type some text into the input.
				b.input.SetValue("go north")
				return b
			},
			msg: tea.KeyMsg{Type: tea.KeyEnter},
			check: func(t *testing.T, bar InputBar, cmd tea.Cmd) {
				require.NotNil(t, cmd)
				msg := cmd()
				sendMsg, ok := msg.(SendCmdMsg)
				require.True(t, ok, "expected SendCmdMsg, got %T", msg)
				assert.Equal(t, "go north", sendMsg.Text)
				assert.Empty(t, bar.input.Value(), "input should be reset")
			},
		},
		{
			name: "enter with empty text does nothing",
			setup: func(b InputBar) InputBar {
				b.input.SetValue("   ")
				return b
			},
			msg: tea.KeyMsg{Type: tea.KeyEnter},
			check: func(t *testing.T, bar InputBar, cmd tea.Cmd) {
				assert.Nil(t, cmd)
			},
		},
		{
			name:  "ctrl+c returns quit",
			setup: func(b InputBar) InputBar { return b },
			msg:   tea.KeyMsg{Type: tea.KeyCtrlC},
			check: func(t *testing.T, bar InputBar, cmd tea.Cmd) {
				require.NotNil(t, cmd)
				msg := cmd()
				_, ok := msg.(tea.QuitMsg)
				assert.True(t, ok, "expected tea.QuitMsg, got %T", msg)
			},
		},
		{
			name: "waiting state changes view",
			setup: func(b InputBar) InputBar {
				b.SetWaiting(true)
				return b
			},
			msg: nil,
			check: func(t *testing.T, bar InputBar, cmd tea.Cmd) {
				assert.Contains(t, bar.View(), "sending")
			},
		},
		{
			name:  "regular keys update textinput value",
			setup: func(b InputBar) InputBar { return b },
			msg:   tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}},
			check: func(t *testing.T, bar InputBar, cmd tea.Cmd) {
				assert.Equal(t, "h", bar.input.Value())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bar := NewInputBar()
			bar = tt.setup(bar)

			if tt.msg != nil {
				var cmd tea.Cmd
				bar, cmd = bar.Update(tt.msg)
				tt.check(t, bar, cmd)
			} else {
				tt.check(t, bar, nil)
			}
		})
	}
}
