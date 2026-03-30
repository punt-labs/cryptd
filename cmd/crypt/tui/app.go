package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/punt-labs/cryptd/internal/protocol"
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
func NewApp(send SendFn, sessionID, scenario, charName, charClass string, initialResp *protocol.PlayResponse) App {
	return App{
		send:        send,
		sessionID:   sessionID,
		scenario:    scenario,
		charName:    charName,
		charClass:   charClass,
		initialResp: initialResp,
		narration:   NewNarrationPane(80, 20),
		sidebar:     NewSidebarPane(SidebarWidth, 20),
		combat:      NewCombatOverlay(80),
		input:       NewInputBar(),
	}
}

// Init returns the initial command. For resumed sessions, it produces a
// GameStartMsg from the pre-fetched response. For new games with a scenario,
// it sends game.new. Otherwise it returns nil.
func (a App) Init() tea.Cmd {
	if a.initialResp != nil {
		resp := *a.initialResp
		return func() tea.Msg { return GameStartMsg{Response: resp} }
	}
	if a.scenario != "" {
		a.waiting = true
		a.input.SetWaiting(true)
		return NewGameCmd(a.send, a.scenario, a.charName, a.charClass)
	}
	return nil
}

// Update routes messages to sub-components.
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.narrWidth = max(a.width-SidebarWidth-3, 10)
		a.mainHeight = max(a.height-3, 1)
		a.narration.SetSize(a.narrWidth, a.mainHeight)
		a.sidebar = NewSidebarPane(SidebarWidth, a.mainHeight)
		a.combat = NewCombatOverlay(a.narrWidth)
		a.input.SetWidth(a.width)

		var cmd tea.Cmd
		a.narration, cmd = a.narration.Update(msg)
		var inputCmd tea.Cmd
		a.input, inputCmd = a.input.Update(msg)
		return &a, tea.Batch(cmd, inputCmd)

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
		if a.sessionID == "" {
			// Init failed — fatal.
			a.narration.AppendText(fmt.Sprintf("Failed to connect: %v", msg.Err), StyleDamage)
			a.narration.AppendText("Press Ctrl+C to exit.", StyleSystem)
			a.input.Blur()
			a.err = msg.Err
			return &a, nil
		}
		a.narration.AppendText(fmt.Sprintf("Error: %v", msg.Err), StyleSystem)
		return &a, nil

	case ConnLostMsg:
		a.err = msg.Err
		a.narration.AppendText("Connection lost.", StyleDamage)
		a.narration.AppendText("Press Ctrl+C to exit.", StyleSystem)
		a.input.Blur()
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

// View renders the full TUI layout.
func (a App) View() string {
	if a.quitting {
		return ""
	}

	// Header.
	header := a.renderHeader()

	// Main area.
	narrView := a.narration.View()
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

// renderHeader builds the top status line.
func (a App) renderHeader() string {
	if a.lastResp == nil || a.lastResp.State == nil || len(a.lastResp.State.Party) == 0 {
		return StyleHeader.Width(a.width).Render(StyleSystem.Render("Connecting..."))
	}
	hero := a.lastResp.State.Party[0]
	text := fmt.Sprintf("%s %s · Lv %d · %d XP   %s",
		StyleCharName.Render(hero.Name),
		StyleCharClass.Render("the "+hero.Class),
		hero.Level,
		hero.XP,
		StyleSessionID.Render("session "+a.sessionID),
	)
	return StyleHeader.Width(a.width).Render(text)
}
