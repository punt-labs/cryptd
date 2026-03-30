package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/punt-labs/cryptd/internal/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testScenarios() []protocol.ScenarioInfo {
	return []protocol.ScenarioInfo{
		{ID: "dungeon", Title: "The Dungeon", Description: "A dark dungeon crawl."},
		{ID: "forest", Title: "Dark Forest", Description: "Lost in the woods."},
	}
}

func testSessions() []protocol.SessionInfo {
	return []protocol.SessionInfo{
		{
			ID: "sess-1", ScenarioID: "dungeon",
			CharacterName: "Thorn", CharacterClass: "fighter",
			Level: 3, RoomID: "Entry Hall",
		},
		{
			ID: "sess-2", ScenarioID: "forest",
			CharacterName: "Wisp", CharacterClass: "mage",
			Level: 1, RoomID: "Clearing",
		},
	}
}

func TestLobby_Init(t *testing.T) {
	l := NewLobby(mockSend(`{"scenarios":[]}`))
	cmd := l.Init()
	assert.NotNil(t, cmd, "Init should return a batch Cmd for fetching scenarios and sessions")
}

func TestLobby_ScenariosMsg(t *testing.T) {
	l := NewLobby(mockSend(`{}`))
	l, _ = l.Update(ScenariosMsg{Scenarios: testScenarios()})
	assert.Len(t, l.scenarios, 2)
	assert.False(t, l.loadingScen)
}

func TestLobby_SessionsMsg(t *testing.T) {
	l := NewLobby(mockSend(`{}`))
	l, _ = l.Update(SessionsMsg{Sessions: testSessions()})
	assert.Len(t, l.sessions, 2)
	assert.False(t, l.loadingSess)
}

func TestLobby_MenuNavigation(t *testing.T) {
	l := NewLobby(mockSend(`{}`))
	l.loadingScen = false
	l.loadingSess = false
	l.scenarios = testScenarios()

	// Start at index 0 (New Game).
	assert.Equal(t, 0, l.menuIndex)

	// Down arrow moves to Resume.
	l, _ = l.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 1, l.menuIndex)

	// Down again to Quit.
	l, _ = l.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 2, l.menuIndex)

	// Down wraps to New Game.
	l, _ = l.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 0, l.menuIndex)

	// Up wraps to Quit.
	l, _ = l.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, 2, l.menuIndex)
}

func TestLobby_VimNavigation(t *testing.T) {
	l := NewLobby(mockSend(`{}`))
	l.loadingScen = false

	l, _ = l.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	assert.Equal(t, 1, l.menuIndex)

	l, _ = l.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	assert.Equal(t, 0, l.menuIndex)
}

func TestLobby_NewGameTransition(t *testing.T) {
	l := NewLobby(mockSend(`{}`))
	l.loadingScen = false
	l.scenarios = testScenarios()
	l.menuIndex = 0

	l, cmd := l.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	msg := cmd()
	startMsg, ok := msg.(StartCreationMsg)
	require.True(t, ok, "expected StartCreationMsg, got %T", msg)
	assert.Len(t, startMsg.Scenarios, 2)
}

func TestLobby_NewGameBlockedWhileLoading(t *testing.T) {
	l := NewLobby(mockSend(`{}`))
	// loadingScen is true by default.
	l.menuIndex = 0

	_, cmd := l.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd, "should not produce StartCreationMsg while loading")
}

func TestLobby_ResumeEntersBrowseMode(t *testing.T) {
	l := NewLobby(mockSend(`{}`))
	l.loadingSess = false
	l.sessions = testSessions()
	l.menuIndex = 1 // Resume

	l, cmd := l.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd, "entering browse mode should not produce a command")
	assert.True(t, l.browsing)
	assert.Equal(t, 0, l.sessionIndex)
}

func TestLobby_ResumeSelectSession(t *testing.T) {
	l := NewLobby(mockSend(`{}`))
	l.loadingSess = false
	l.sessions = testSessions()
	l.browsing = true
	l.sessionIndex = 1

	l, cmd := l.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	msg := cmd()
	resumeMsg, ok := msg.(ResumeSessionMsg)
	require.True(t, ok, "expected ResumeSessionMsg, got %T", msg)
	assert.Equal(t, "sess-2", resumeMsg.SessionID)
}

func TestLobby_ResumeEscExitsBrowse(t *testing.T) {
	l := NewLobby(mockSend(`{}`))
	l.browsing = true
	l.sessions = testSessions()

	l, _ = l.Update(tea.KeyMsg{Type: tea.KeyEscape})
	assert.False(t, l.browsing)
}

func TestLobby_ResumeNoSaves(t *testing.T) {
	l := NewLobby(mockSend(`{}`))
	l.loadingSess = false
	l.menuIndex = 1 // Resume

	_, cmd := l.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd, "resume with no saves should be a no-op")
}

func TestLobby_QuitProducesQuitCmd(t *testing.T) {
	l := NewLobby(mockSend(`{}`))
	l.menuIndex = 2 // Quit

	_, cmd := l.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	assert.True(t, ok, "expected tea.QuitMsg, got %T", msg)
}

func TestLobby_QShortcut(t *testing.T) {
	l := NewLobby(mockSend(`{}`))

	_, cmd := l.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	assert.True(t, ok, "q should quit from main menu")
}

func TestLobby_QDoesNotQuitWhileBrowsing(t *testing.T) {
	l := NewLobby(mockSend(`{}`))
	l.browsing = true
	l.sessions = testSessions()

	_, cmd := l.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	assert.Nil(t, cmd, "q should not quit while browsing sessions")
}

func TestLobby_SessionNavigationWraps(t *testing.T) {
	l := NewLobby(mockSend(`{}`))
	l.browsing = true
	l.sessions = testSessions()
	l.sessionIndex = 0

	// Up wraps to last.
	l, _ = l.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, 1, l.sessionIndex)

	// Down wraps to first.
	l, _ = l.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 0, l.sessionIndex)
}

func TestLobby_View(t *testing.T) {
	l := NewLobby(mockSend(`{}`))
	l.width = 80
	l.height = 30
	l.loadingScen = false
	l.loadingSess = false
	l.scenarios = testScenarios()
	l.sessions = testSessions()

	v := l.View()
	assert.Contains(t, v, "CRYPT")
	assert.Contains(t, v, "New Game")
	assert.Contains(t, v, "Resume")
	assert.Contains(t, v, "Quit")
}

func TestLobby_ScenarioErrMsgClearsLoading(t *testing.T) {
	l := NewLobby(mockSend(`{}`))
	assert.True(t, l.loadingScen)

	l, _ = l.Update(ScenarioErrMsg{Err: assert.AnError})
	assert.False(t, l.loadingScen)
	assert.True(t, l.scenarioErr)
	// Session loading is unaffected.
	assert.True(t, l.loadingSess)
}

func TestLobby_SessionErrMsgClearsLoading(t *testing.T) {
	l := NewLobby(mockSend(`{}`))
	assert.True(t, l.loadingSess)

	l, _ = l.Update(SessionErrMsg{Err: assert.AnError})
	assert.False(t, l.loadingSess)
	assert.True(t, l.sessionErr)
	// Scenario loading is unaffected.
	assert.True(t, l.loadingScen)
}

func TestLobby_NewGameBlockedWhenUnavailable(t *testing.T) {
	l := NewLobby(mockSend(`{}`))
	l.loadingScen = false
	l.scenarioErr = true
	l.menuIndex = 0

	_, cmd := l.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd, "should not produce StartCreationMsg when scenarios errored")
}

func TestLobby_NewGameBlockedWhenNoScenarios(t *testing.T) {
	l := NewLobby(mockSend(`{}`))
	l.loadingScen = false
	l.scenarios = nil
	l.menuIndex = 0

	_, cmd := l.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd, "should not produce StartCreationMsg with no scenarios")
}
