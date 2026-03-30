package tui

import (
	"encoding/json"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mockSend(result string) SendFn {
	return func(method string, params any) (json.RawMessage, error) {
		return json.RawMessage(result), nil
	}
}

func testPlayResponse() protocol.PlayResponse {
	return protocol.PlayResponse{
		Text: "You enter a dark room.",
		State: &model.GameState{
			Party: []model.Character{
				{
					Name:  "Thorn",
					Class: "fighter",
					Level: 3,
					HP:    25, MaxHP: 30,
					XP: 150,
				},
			},
			Dungeon: model.DungeonState{
				CurrentRoom: "entry_hall",
			},
		},
		Exits:       []string{"north", "east"},
		NextLevelXP: 300,
	}
}

func combatPlayResponse() protocol.PlayResponse {
	resp := testPlayResponse()
	resp.State.Dungeon.Combat = model.CombatState{
		Active:    true,
		Round:     1,
		TurnOrder: []string{"hero", "goblin"},
		Enemies: []model.EnemyInstance{
			{Name: "Goblin", HP: 10, MaxHP: 10},
		},
	}
	return resp
}

func TestApp(t *testing.T) {
	tests := []struct {
		name  string
		setup func() App
		msg   tea.Msg
		check func(t *testing.T, a *App, cmd tea.Cmd)
	}{
		{
			name: "Init returns non-nil Cmd",
			setup: func() App {
				return NewApp(mockSend(`{}`), "sess-1", "dungeon", "Thorn", "fighter")
			},
			msg: nil, // use Init() instead
			check: func(t *testing.T, a *App, cmd tea.Cmd) {
				initCmd := a.Init()
				assert.NotNil(t, initCmd)
			},
		},
		{
			name: "WindowSizeMsg updates dimensions",
			setup: func() App {
				return NewApp(mockSend(`{}`), "sess-1", "", "Thorn", "fighter")
			},
			msg: tea.WindowSizeMsg{Width: 120, Height: 40},
			check: func(t *testing.T, a *App, cmd tea.Cmd) {
				assert.Equal(t, 120, a.width)
				assert.Equal(t, 40, a.height)
			},
		},
		{
			name: "SessionReadyMsg with scenario returns non-nil Cmd",
			setup: func() App {
				return NewApp(mockSend(`{}`), "", "dungeon", "Thorn", "fighter")
			},
			msg: SessionReadyMsg{SessionID: "sess-42"},
			check: func(t *testing.T, a *App, cmd tea.Cmd) {
				assert.Equal(t, "sess-42", a.sessionID)
				assert.NotNil(t, cmd, "should return NewGameCmd when scenario is set")
			},
		},
		{
			name: "SessionReadyMsg without scenario returns nil Cmd",
			setup: func() App {
				return NewApp(mockSend(`{}`), "", "", "Thorn", "fighter")
			},
			msg: SessionReadyMsg{SessionID: "sess-42"},
			check: func(t *testing.T, a *App, cmd tea.Cmd) {
				assert.Equal(t, "sess-42", a.sessionID)
				assert.Nil(t, cmd)
			},
		},
		{
			name: "ServerResponseMsg updates state",
			setup: func() App {
				a := NewApp(mockSend(`{}`), "sess-1", "", "Thorn", "fighter")
				a.waiting = true
				return a
			},
			msg: ServerResponseMsg{Response: testPlayResponse()},
			check: func(t *testing.T, a *App, cmd tea.Cmd) {
				require.NotNil(t, a.lastResp)
				assert.Equal(t, "Thorn", a.lastResp.State.Party[0].Name)
				assert.False(t, a.waiting)
				assert.False(t, a.dead)
			},
		},
		{
			name: "combat hotkey a dispatches attack",
			setup: func() App {
				a := NewApp(mockSend(`{}`), "sess-1", "", "Thorn", "fighter")
				resp := combatPlayResponse()
				a.lastResp = &resp
				a.input.Blur() // hotkey mode
				return a
			},
			msg: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}},
			check: func(t *testing.T, a *App, cmd tea.Cmd) {
				require.NotNil(t, cmd)
				msg := cmd()
				sendMsg, ok := msg.(SendCmdMsg)
				require.True(t, ok, "expected SendCmdMsg, got %T", msg)
				assert.Equal(t, "attack", sendMsg.Text)
			},
		},
		{
			name: "combat hotkey ignored when input focused",
			setup: func() App {
				a := NewApp(mockSend(`{}`), "sess-1", "", "Thorn", "fighter")
				resp := combatPlayResponse()
				a.lastResp = &resp
				// input starts focused by default
				return a
			},
			msg: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}},
			check: func(t *testing.T, a *App, cmd tea.Cmd) {
				// 'a' should go to input, not produce a combat command.
				// The cmd, if any, should NOT produce SendCmdMsg{Text: "attack"}.
				if cmd != nil {
					msg := cmd()
					sendMsg, ok := msg.(SendCmdMsg)
					if ok {
						assert.NotEqual(t, "attack", sendMsg.Text,
							"'a' should go to input when focused, not dispatch attack")
					}
				}
			},
		},
		{
			name: "dead state: enter quits",
			setup: func() App {
				a := NewApp(mockSend(`{}`), "sess-1", "", "Thorn", "fighter")
				a.dead = true
				return a
			},
			msg: tea.KeyMsg{Type: tea.KeyEnter},
			check: func(t *testing.T, a *App, cmd tea.Cmd) {
				require.NotNil(t, cmd)
				// tea.Quit returns a special QuitMsg.
				msg := cmd()
				_, ok := msg.(tea.QuitMsg)
				assert.True(t, ok, "expected tea.QuitMsg, got %T", msg)
			},
		},
		{
			name: "dead state: random key does not quit",
			setup: func() App {
				a := NewApp(mockSend(`{}`), "sess-1", "", "Thorn", "fighter")
				a.dead = true
				return a
			},
			msg: tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}},
			check: func(t *testing.T, a *App, cmd tea.Cmd) {
				assert.Nil(t, cmd, "random key should not quit when dead")
			},
		},
		{
			name: "SendCmdMsg sets waiting",
			setup: func() App {
				return NewApp(mockSend(`{}`), "sess-1", "", "Thorn", "fighter")
			},
			msg: SendCmdMsg{Text: "look"},
			check: func(t *testing.T, a *App, cmd tea.Cmd) {
				assert.True(t, a.waiting)
				assert.NotNil(t, cmd, "should return PlayCmd")
			},
		},
		{
			name: "SendCmdMsg ignored when dead",
			setup: func() App {
				a := NewApp(mockSend(`{}`), "sess-1", "", "Thorn", "fighter")
				a.dead = true
				return a
			},
			msg: SendCmdMsg{Text: "look"},
			check: func(t *testing.T, a *App, cmd tea.Cmd) {
				assert.False(t, a.waiting)
				assert.Nil(t, cmd)
			},
		},
		{
			name: "SendCmdMsg ignored when already waiting",
			setup: func() App {
				a := NewApp(mockSend(`{}`), "sess-1", "", "Thorn", "fighter")
				a.waiting = true
				return a
			},
			msg: SendCmdMsg{Text: "look"},
			check: func(t *testing.T, a *App, cmd tea.Cmd) {
				assert.True(t, a.waiting)
				assert.Nil(t, cmd)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := tt.setup()
			if tt.msg == nil {
				tt.check(t, &a, nil)
				return
			}
			result, cmd := a.Update(tt.msg)
			appPtr, ok := result.(*App)
			require.True(t, ok, "Update must return *App")
			tt.check(t, appPtr, cmd)
		})
	}
}

func TestAppGameStartMsg(t *testing.T) {
	a := NewApp(mockSend(`{}`), "sess-1", "dungeon", "Thorn", "fighter")
	resp := testPlayResponse()
	result, cmd := a.Update(GameStartMsg{Response: resp})
	appPtr := result.(*App)
	require.NotNil(t, appPtr.lastResp)
	assert.Equal(t, "You enter a dark room.", appPtr.lastResp.Text)
	assert.Nil(t, cmd)
}

func TestAppServerErrMsg(t *testing.T) {
	a := NewApp(mockSend(`{}`), "sess-1", "", "Thorn", "fighter")
	a.waiting = true
	result, _ := a.Update(ServerErrMsg{Err: assert.AnError})
	appPtr := result.(*App)
	assert.False(t, appPtr.waiting)
}

func TestAppConnLostMsg(t *testing.T) {
	a := NewApp(mockSend(`{}`), "sess-1", "", "Thorn", "fighter")
	result, _ := a.Update(ConnLostMsg{Err: assert.AnError})
	appPtr := result.(*App)
	assert.NotNil(t, appPtr.err)
	assert.False(t, appPtr.input.Focused())
}

func TestAppViewConnecting(t *testing.T) {
	a := NewApp(mockSend(`{}`), "sess-1", "", "Thorn", "fighter")
	a.width = 100
	a.height = 30
	v := a.View()
	assert.Contains(t, v, "Connecting...")
}

func TestAppViewWithState(t *testing.T) {
	a := NewApp(mockSend(`{}`), "sess-1", "", "Thorn", "fighter")
	a.width = 100
	a.height = 30
	resp := testPlayResponse()
	a.lastResp = &resp
	v := a.View()
	assert.Contains(t, v, "Thorn")
	assert.Contains(t, v, "fighter")
}

func TestAppCombatHotkeyD(t *testing.T) {
	a := NewApp(mockSend(`{}`), "sess-1", "", "Thorn", "fighter")
	resp := combatPlayResponse()
	a.lastResp = &resp
	a.input.Blur()
	result, cmd := a.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	_ = result
	require.NotNil(t, cmd)
	msg := cmd()
	sendMsg, ok := msg.(SendCmdMsg)
	require.True(t, ok)
	assert.Equal(t, "defend", sendMsg.Text)
}

func TestAppCombatTabFocusesInput(t *testing.T) {
	a := NewApp(mockSend(`{}`), "sess-1", "", "Thorn", "fighter")
	resp := combatPlayResponse()
	a.lastResp = &resp
	a.input.Blur()
	assert.False(t, a.input.Focused())
	result, _ := a.Update(tea.KeyMsg{Type: tea.KeyTab})
	appPtr := result.(*App)
	assert.True(t, appPtr.input.Focused())
}

func TestAppCombatEscBlursInput(t *testing.T) {
	a := NewApp(mockSend(`{}`), "sess-1", "", "Thorn", "fighter")
	resp := combatPlayResponse()
	a.lastResp = &resp
	// input starts focused
	assert.True(t, a.input.Focused())
	result, _ := a.Update(tea.KeyMsg{Type: tea.KeyEsc})
	appPtr := result.(*App)
	assert.False(t, appPtr.input.Focused())
}
