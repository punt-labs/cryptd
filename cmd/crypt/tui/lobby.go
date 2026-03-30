package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/punt-labs/cryptd/internal/protocol"
)

// lobbyMenuItem identifies a menu option in the lobby.
type lobbyMenuItem int

const (
	menuNewGame lobbyMenuItem = iota
	menuResume
	menuQuit
	menuCount // sentinel: total number of menu items
)

// Lobby is the welcome screen sub-model. It shows a title, menu, and
// optionally a list of saved sessions for resumption.
type Lobby struct {
	send SendFn

	menuIndex    int
	scenarios    []protocol.ScenarioInfo
	sessions     []protocol.SessionInfo
	scenarioErr  bool
	sessionErr   bool
	loadingScen  bool
	loadingSess  bool
	sessionIndex int // selected session when browsing resume list
	browsing     bool // true when the resume sub-list is expanded

	width, height int
}

// NewLobby creates a Lobby wired to the given RPC send function.
func NewLobby(send SendFn) Lobby {
	return Lobby{
		send:        send,
		loadingScen: true,
		loadingSess: true,
	}
}

// Init dispatches the two list RPCs to populate scenarios and sessions.
func (l Lobby) Init() tea.Cmd {
	return tea.Batch(
		ListScenariosCmd(l.send),
		ListSessionsCmd(l.send),
	)
}

// Update handles lobby key events and RPC responses.
func (l Lobby) Update(msg tea.Msg) (Lobby, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		l.width = msg.Width
		l.height = msg.Height
		return l, nil

	case ScenariosMsg:
		l.scenarios = msg.Scenarios
		l.loadingScen = false
		l.scenarioErr = false
		return l, nil

	case ScenarioErrMsg:
		l.loadingScen = false
		l.scenarioErr = true
		return l, nil

	case SessionsMsg:
		l.sessions = msg.Sessions
		l.loadingSess = false
		l.sessionErr = false
		return l, nil

	case SessionErrMsg:
		l.loadingSess = false
		l.sessionErr = true
		return l, nil

	case tea.KeyMsg:
		return l.handleKey(msg)
	}
	return l, nil
}

// handleKey processes keyboard input for menu navigation.
func (l Lobby) handleKey(msg tea.KeyMsg) (Lobby, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return l, tea.Quit

	case tea.KeyUp, tea.KeyShiftTab:
		if l.browsing {
			l.sessionIndex--
			if l.sessionIndex < 0 {
				l.sessionIndex = len(l.sessions) - 1
			}
			return l, nil
		}
		l.menuIndex--
		if l.menuIndex < 0 {
			l.menuIndex = int(menuCount) - 1
		}
		return l, nil

	case tea.KeyDown, tea.KeyTab:
		if l.browsing {
			l.sessionIndex++
			if l.sessionIndex >= len(l.sessions) {
				l.sessionIndex = 0
			}
			return l, nil
		}
		l.menuIndex++
		if l.menuIndex >= int(menuCount) {
			l.menuIndex = 0
		}
		return l, nil

	case tea.KeyEscape:
		if l.browsing {
			l.browsing = false
			return l, nil
		}
		return l, nil

	case tea.KeyEnter:
		return l.selectMenuItem()
	}

	// Handle j/k vim-style navigation.
	switch msg.String() {
	case "j":
		if l.browsing {
			l.sessionIndex++
			if l.sessionIndex >= len(l.sessions) {
				l.sessionIndex = 0
			}
		} else {
			l.menuIndex++
			if l.menuIndex >= int(menuCount) {
				l.menuIndex = 0
			}
		}
	case "k":
		if l.browsing {
			l.sessionIndex--
			if l.sessionIndex < 0 {
				l.sessionIndex = len(l.sessions) - 1
			}
		} else {
			l.menuIndex--
			if l.menuIndex < 0 {
				l.menuIndex = int(menuCount) - 1
			}
		}
	case "q":
		if !l.browsing {
			return l, tea.Quit
		}
	}

	return l, nil
}

// selectMenuItem handles Enter on the current menu item.
func (l Lobby) selectMenuItem() (Lobby, tea.Cmd) {
	if l.browsing {
		if len(l.sessions) > 0 && l.sessionIndex < len(l.sessions) {
			sess := l.sessions[l.sessionIndex]
			return l, func() tea.Msg {
				return ResumeSessionMsg{SessionID: sess.ID}
			}
		}
		return l, nil
	}

	switch lobbyMenuItem(l.menuIndex) {
	case menuNewGame:
		if l.loadingScen || l.scenarioErr || len(l.scenarios) == 0 {
			return l, nil // not ready yet
		}
		scenarios := l.scenarios
		return l, func() tea.Msg {
			return StartCreationMsg{Scenarios: scenarios}
		}
	case menuResume:
		if len(l.sessions) == 0 {
			return l, nil // nothing to resume
		}
		l.browsing = true
		l.sessionIndex = 0
		return l, nil
	case menuQuit:
		return l, tea.Quit
	}
	return l, nil
}

// View renders the lobby screen.
func (l Lobby) View() string {
	// Title banner.
	title := StyleLobbyTitle.Render("  CRYPT  ")
	subtitle := StyleLobbySubtitle.Render("A dungeon awaits")

	// Menu items.
	var menuLines []string
	for i := 0; i < int(menuCount); i++ {
		item := lobbyMenuItem(i)
		label := l.menuLabel(item)
		dimmed := l.isMenuDimmed(item)
		if i == l.menuIndex {
			menuLines = append(menuLines, StyleMenuSelected.Render("> "+label))
		} else if dimmed {
			menuLines = append(menuLines, StyleMenuDim.Render("  "+label))
		} else {
			menuLines = append(menuLines, StyleMenuNormal.Render("  "+label))
		}
	}
	menu := strings.Join(menuLines, "\n")

	// Session list (shown when browsing resume).
	var sessionView string
	if l.browsing && len(l.sessions) > 0 {
		sessionView = "\n" + l.renderSessionList()
	}

	// Loading indicator.
	var loadingView string
	if l.loadingScen || l.loadingSess {
		loadingView = "\n" + StyleSystem.Render("Loading...")
	}

	// Hint line.
	hint := StyleCreationHint.Render("arrow keys to navigate, enter to select, q to quit")

	// Compose the content.
	content := lipgloss.JoinVertical(lipgloss.Left,
		"",
		title,
		subtitle,
		"",
		menu,
		sessionView,
		loadingView,
		"",
		hint,
	)

	// Place in a bordered box, centered on screen.
	box := StyleLobbyBox.Render(content)

	return lipgloss.Place(l.width, l.height,
		lipgloss.Center, lipgloss.Center,
		box)
}

// menuLabel returns the display text for a menu item.
func (l Lobby) menuLabel(item lobbyMenuItem) string {
	switch item {
	case menuNewGame:
		if l.loadingScen {
			return "New Game  (loading...)"
		}
		if l.scenarioErr {
			return "New Game  (unavailable)"
		}
		n := len(l.scenarios)
		if n == 1 {
			return "New Game  (1 scenario)"
		}
		return fmt.Sprintf("New Game  (%d scenarios)", n)
	case menuResume:
		n := len(l.sessions)
		if l.loadingSess {
			return "Resume  (loading...)"
		}
		if l.sessionErr {
			return "Resume  (unavailable)"
		}
		if n == 0 {
			return "Resume  (no saves)"
		}
		if n == 1 {
			return "Resume  (1 save)"
		}
		return fmt.Sprintf("Resume  (%d saves)", n)
	case menuQuit:
		return "Quit"
	}
	return ""
}

// isMenuDimmed returns true when a menu item should be rendered with dim style.
func (l Lobby) isMenuDimmed(item lobbyMenuItem) bool {
	switch item {
	case menuNewGame:
		return l.scenarioErr || (!l.loadingScen && len(l.scenarios) == 0)
	case menuResume:
		return l.sessionErr || (!l.loadingSess && len(l.sessions) == 0)
	}
	return false
}

// renderSessionList renders the expandable list of saved sessions.
func (l Lobby) renderSessionList() string {
	var lines []string
	lines = append(lines, StyleSidebarLabel.Render("SAVED SESSIONS"))
	lines = append(lines, "")

	for i, s := range l.sessions {
		name := s.CharacterName
		if name == "" {
			name = "Unknown"
		}
		class := s.CharacterClass
		if class == "" {
			class = "???"
		}

		line := fmt.Sprintf("%s the %s  Lv %d", name, class, s.Level)
		if s.RoomID != "" {
			line += "  " + StyleSessionDetail.Render("@ "+s.RoomID)
		}

		if i == l.sessionIndex {
			lines = append(lines, StyleMenuSelected.Render("> "+line))
		} else {
			lines = append(lines, StyleSessionItem.Render("  "+line))
		}
	}

	lines = append(lines, "")
	lines = append(lines, StyleCreationHint.Render("enter to resume, esc to go back"))

	return strings.Join(lines, "\n")
}
