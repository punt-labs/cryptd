package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/punt-labs/cryptd/internal/protocol"
)

// appState tracks which top-level screen is active.
type appState int

const (
	stateLobby    appState = iota
	stateCreation
	stateGame
)

// App is the top-level Bubble Tea model. It composes the four leaf components
// and routes all messages.
type App struct {
	send      SendFn
	sessionID string
	scenario  string
	charName  string
	charClass string

	initialResp *protocol.PlayResponse

	state    appState
	lobby    Lobby
	creation GameCreation

	width, height int
	narrWidth     int
	mainHeight    int

	lastResp  *protocol.PlayResponse
	dead      bool
	quitting  bool
	waiting   bool
	err       error

	narration NarrationPane
	sidebar   SidebarPane
	combat    CombatOverlay
	input     InputBar
}

// NewApp creates an App wired to the given RPC send function.
// If initialResp is non-nil (session resume), Init() will use it directly
// instead of making a network call.
// When scenario and sessionID are both empty, the app starts in lobby mode.
func NewApp(send SendFn, sessionID, scenario, charName, charClass string, initialResp *protocol.PlayResponse) App {
	startState := stateGame
	if scenario == "" && initialResp == nil {
		startState = stateLobby
	}

	return App{
		send:        send,
		sessionID:   sessionID,
		scenario:    scenario,
		charName:    charName,
		charClass:   charClass,
		initialResp: initialResp,
		state:       startState,
		lobby:       NewLobby(send),
		narration:   NewNarrationPane(80, 20),
		sidebar:     NewSidebarPane(SidebarWidth, 20),
		combat:      NewCombatOverlay(80),
		input:       NewInputBar(),
	}
}

// Init returns the initial command. For resumed sessions, it produces a
// GameStartMsg from the pre-fetched response. For new games with a scenario,
// it sends game.new. For lobby mode, it fetches scenarios and sessions.
func (a App) Init() tea.Cmd {
	if a.state == stateLobby {
		return a.lobby.Init()
	}
	if a.initialResp != nil {
		resp := *a.initialResp
		return func() tea.Msg { return GameStartMsg{Response: resp} }
	}
	if a.scenario != "" {
		return tea.Batch(
			func() tea.Msg { return LoadingMsg{} },
			NewGameCmd(a.send, a.scenario, a.charName, a.charClass, nil),
		)
	}
	return func() tea.Msg { return WelcomeMsg{} }
}

// Update routes messages to sub-components based on current state.
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// WindowSizeMsg goes to all states.
	if wsm, ok := msg.(tea.WindowSizeMsg); ok {
		a.width = wsm.Width
		a.height = wsm.Height
		a.lobby.width = wsm.Width
		a.lobby.height = wsm.Height
		a.creation.width = wsm.Width
		a.creation.height = wsm.Height

		// Game layout calculations.
		sidebarTotal := SidebarWidth + 4
		a.narrWidth = max(a.width-sidebarTotal-1, 10)
		a.mainHeight = max(a.height-4, 1)
		a.narration.SetSize(a.narrWidth, a.mainHeight)
		a.sidebar = NewSidebarPane(SidebarWidth, a.mainHeight)
		a.combat = NewCombatOverlay(a.narrWidth)
		a.input.SetWidth(a.width)

		var cmd tea.Cmd
		a.narration, cmd = a.narration.Update(msg)
		var inputCmd tea.Cmd
		a.input, inputCmd = a.input.Update(msg)
		return &a, tea.Batch(cmd, inputCmd)
	}

	switch a.state {
	case stateLobby:
		return a.updateLobby(msg)
	case stateCreation:
		return a.updateCreation(msg)
	case stateGame:
		return a.updateGame(msg)
	}
	return &a, nil
}

// updateLobby handles messages while in lobby state.
func (a App) updateLobby(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case StartCreationMsg:
		a.state = stateCreation
		a.creation = NewGameCreation(a.send, msg.Scenarios)
		return &a, a.creation.Init()

	case ResumeSessionMsg:
		// Transition to game state with the selected session.
		// Re-bind the server connection by sending session.init with
		// the chosen session ID.
		a.state = stateGame
		a.sessionID = msg.SessionID
		a.waiting = true
		a.input.SetWaiting(true)
		return &a, SessionInitCmd(a.send, msg.SessionID)
	}

	var cmd tea.Cmd
	a.lobby, cmd = a.lobby.Update(msg)
	return &a, cmd
}

// updateCreation handles messages while in creation state.
func (a App) updateCreation(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case CreationDoneMsg:
		a.state = stateGame
		a.scenario = msg.Scenario
		a.charName = msg.Name
		a.charClass = msg.Class
		a.waiting = true
		a.input.SetWaiting(true)
		a.narration.AppendText("Starting game...", StyleSystem)
		return &a, NewGameCmd(a.send, msg.Scenario, msg.Name, msg.Class, msg.Stats)
	}

	// Handle esc at first step to go back to lobby.
	if km, ok := msg.(tea.KeyMsg); ok && km.Type == tea.KeyEscape && a.creation.step == stepScenario {
		a.state = stateLobby
		return &a, nil
	}

	var cmd tea.Cmd
	a.creation, cmd = a.creation.Update(msg)
	return &a, cmd
}

// updateGame handles messages while in game state (existing behavior).
func (a App) updateGame(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case SessionInitDoneMsg:
		a.sessionID = msg.SessionID
		if msg.HasGame {
			// The server will send an unsolicited PlayResponse via the
			// game loop. For now, keep waiting — the next ServerResponseMsg
			// or GameStartMsg will clear it.
			return &a, nil
		}
		// Session exists but has no active game — show welcome.
		a.waiting = false
		a.input.SetWaiting(false)
		a.narration.AppendText("Session has no active game.", StyleSystem)
		a.narration.AppendText("Type 'new <scenario> <name> <class>' to start.", StyleSystem)
		return &a, nil

	case GameStartMsg:
		resp := msg.Response
		a.waiting = false
		a.input.SetWaiting(false)
		a.lastResp = &resp
		a.narration.AppendResponse(resp)
		if a.combatActive() {
			a.input.Blur()
		}
		return &a, nil

	case ServerResponseMsg:
		resp := msg.Response
		wasCombat := a.combatActive()
		a.waiting = false
		a.lastResp = &resp
		a.narration.AppendResponse(resp)
		a.input.SetWaiting(false)

		if resp.Dead {
			a.dead = true
			a.input.Blur()
			return &a, nil
		}
		if resp.Quit {
			a.quitting = true
			return &a, tea.Quit
		}
		nowCombat := a.combatActive()
		var cmd tea.Cmd
		if wasCombat && !nowCombat {
			cmd = a.input.Focus()
		}
		if !wasCombat && nowCombat {
			a.input.Blur()
		}
		return &a, cmd

	case ServerErrMsg:
		a.waiting = false
		a.input.SetWaiting(false)
		a.narration.AppendText(fmt.Sprintf("Error: %v", msg.Err), StyleSystem)
		if a.lastResp == nil {
			// No game started yet — fatal, no recovery possible.
			a.narration.AppendText("Press Ctrl+C to exit.", StyleSystem)
			a.input.Blur()
			a.err = msg.Err
		}
		return &a, nil

	case ConnLostMsg:
		a.err = msg.Err
		a.narration.AppendText("Connection lost.", StyleDamage)
		a.narration.AppendText("Press Ctrl+C to exit.", StyleSystem)
		a.input.Blur()
		return &a, nil

	case LoadingMsg:
		a.waiting = true
		a.input.SetWaiting(true)
		a.narration.AppendText("Starting game...", StyleSystem)
		return &a, nil

	case WelcomeMsg:
		a.narration.AppendText("Welcome to Crypt.", StyleNarration)
		a.narration.AppendText("", StyleNarration)
		a.narration.AppendText("Type 'new <scenario> <name> <class>' to start a game.", StyleSystem)
		a.narration.AppendText("Type 'help' for available commands.", StyleSystem)
		return &a, nil

	case SendCmdMsg:
		if a.dead || a.waiting {
			return &a, nil
		}
		a.waiting = true
		a.input.SetWaiting(true)
		return &a, PlayCmd(a.send, msg.Text)

	case tea.KeyMsg:
		return a.handleKey(msg)
	}

	// Default: forward to narration for scroll handling.
	var cmd tea.Cmd
	a.narration, cmd = a.narration.Update(msg)
	return &a, cmd
}

// handleKey routes key events between combat hotkeys, input bar, and narration.
func (a App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if a.dead {
		switch msg.Type {
		case tea.KeyEnter, tea.KeySpace, tea.KeyEscape, tea.KeyCtrlC:
			return &a, tea.Quit
		case tea.KeyPgUp, tea.KeyPgDown, tea.KeyHome, tea.KeyEnd:
			var cmd tea.Cmd
			a.narration, cmd = a.narration.Update(msg)
			return &a, cmd
		}
		return &a, nil
	}

	combat := a.combatActive()

	if combat && !a.input.Focused() && !a.waiting {
		switch msg.String() {
		case "a":
			return &a, a.combatCmd("attack")
		case "d":
			return &a, a.combatCmd("defend")
		case "f":
			return &a, a.combatCmd("flee")
		case "u":
			return &a, a.combatCmd("use item")
		case "tab", "i":
			cmd := a.input.Focus()
			return &a, cmd
		}
	}

	if combat && a.input.Focused() {
		if msg.String() == "esc" {
			a.input.Blur()
			return &a, nil
		}
	}

	// Delegate to input bar.
	var inputCmd tea.Cmd
	a.input, inputCmd = a.input.Update(msg)

	// Also forward to narration for scroll (PgUp/PgDn/mouse).
	var narrCmd tea.Cmd
	a.narration, narrCmd = a.narration.Update(msg)

	return &a, tea.Batch(inputCmd, narrCmd)
}

// combatCmd returns a Cmd that produces a SendCmdMsg with the given text.
func (a App) combatCmd(text string) tea.Cmd {
	return func() tea.Msg { return SendCmdMsg{Text: text} }
}

// combatActive reports whether combat is currently active.
func (a App) combatActive() bool {
	return a.lastResp != nil &&
		a.lastResp.State != nil &&
		a.lastResp.State.Dungeon.Combat.Active
}

// View renders the full TUI layout based on current state.
func (a App) View() string {
	if a.quitting {
		return ""
	}

	switch a.state {
	case stateLobby:
		return a.lobby.View()
	case stateCreation:
		return a.creation.View()
	case stateGame:
		return a.viewGame()
	}
	return ""
}

// viewGame renders the game screen (narration + sidebar + input).
func (a App) viewGame() string {
	// Header.
	header := a.renderHeader()

	// Main area: narration pane with border + sidebar with border.
	narrView := StyleNarrationPane.
		Width(a.narrWidth).
		Height(a.mainHeight).
		Render(a.narration.View())

	sidebarView := StyleSidebar.
		Width(SidebarWidth).
		Height(a.mainHeight).
		Render(a.sidebar.Render(a.lastResp))

	var mainArea string
	if a.combatActive() {
		combatView := a.combat.Render(
			a.lastResp.State.Dungeon.Combat, a.narrWidth, a.mainHeight)
		mainArea = lipgloss.JoinHorizontal(lipgloss.Top, combatView, sidebarView)
	} else {
		mainArea = lipgloss.JoinHorizontal(lipgloss.Top, narrView, sidebarView)
	}

	// Input.
	inputArea := a.input.View()

	return lipgloss.JoinVertical(lipgloss.Left, header, mainArea, inputArea)
}

// renderHeader builds the top status line with character info left-aligned
// and session ID right-aligned.
func (a App) renderHeader() string {
	if a.lastResp == nil || a.lastResp.State == nil || len(a.lastResp.State.Party) == 0 {
		return StyleHeader.Width(a.width).Render(StyleSystem.Render("Connecting..."))
	}
	hero := a.lastResp.State.Party[0]

	left := fmt.Sprintf("%s %s %s",
		StyleCharName.Render(hero.Name),
		StyleCharClass.Render("the "+hero.Class),
		StyleLevelXP.Render(fmt.Sprintf("Lv %d  %d XP", hero.Level, hero.XP)),
	)
	right := StyleSessionID.Render("session " + a.sessionID)

	// Calculate gap to right-align session ID.
	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)
	gap := a.width - leftW - rightW - 2 // -2 for padding
	if gap < 1 {
		gap = 1
	}

	return StyleHeader.Width(a.width).Render(left + strings.Repeat(" ", gap) + right)
}
