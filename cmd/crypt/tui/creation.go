package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/punt-labs/cryptd/internal/protocol"
)

// creationStep tracks progress through the character creation wizard.
type creationStep int

const (
	stepScenario creationStep = iota
	stepName
	stepClass
	stepStats
	stepCount // sentinel
)

// classInfo holds display data for a character class.
type classInfo struct {
	name     string
	desc     string
	hp       string
	mp       string
	primary  string
}

var classes = []classInfo{
	{"fighter", "Master of arms. Tough, strong, and relentless in melee combat.", "+8 HP/lv", "No MP", "STR, CON"},
	{"mage", "Wielder of arcane power. Fragile but devastatingly powerful.", "+4 HP/lv", "+4 MP/lv", "INT, WIS"},
	{"priest", "Divine servant. Heals allies and wards against the dark.", "+6 HP/lv", "+3 MP/lv", "WIS, CHA"},
	{"thief", "Shadow walker. Quick, cunning, and deadly from behind.", "+6 HP/lv", "No MP", "DEX, CHA"},
}

// statNames is the display order for the six attributes.
var statNames = [6]string{"STR", "DEX", "CON", "INT", "WIS", "CHA"}

// BaseStatValue and PointBuyPool mirror internal/engine/leveling.go constants.
// Duplicated here to avoid a dependency from TUI → engine.
const (
	baseStatValue = 10
	pointBuyPool  = 8
)

// GameCreation is the character creation wizard sub-model.
type GameCreation struct {
	send SendFn

	step       creationStep
	scenarios  []protocol.ScenarioInfo
	scenIndex  int
	nameInput  textinput.Model
	classIndex int
	statPoints [6]int // bonus above base for each stat (index matches statNames)
	statIndex  int    // which stat is currently selected

	width, height int
}

// NewGameCreation creates a GameCreation wizard with the given scenarios.
func NewGameCreation(send SendFn, scenarios []protocol.ScenarioInfo) GameCreation {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = "Adventurer"
	ti.CharLimit = 24
	ti.Width = 24

	return GameCreation{
		send:      send,
		scenarios: scenarios,
		nameInput: ti,
	}
}

// Init returns nil; the creation screen needs no async work.
func (c GameCreation) Init() tea.Cmd {
	return nil
}

// Update handles key events and step transitions for the creation wizard.
func (c GameCreation) Update(msg tea.Msg) (GameCreation, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		c.width = msg.Width
		c.height = msg.Height
		return c, nil

	case tea.KeyMsg:
		return c.handleKey(msg)
	}

	// Forward to text input when on name step.
	if c.step == stepName {
		var cmd tea.Cmd
		c.nameInput, cmd = c.nameInput.Update(msg)
		return c, cmd
	}
	return c, nil
}

// handleKey processes keyboard input for all creation steps.
func (c GameCreation) handleKey(msg tea.KeyMsg) (GameCreation, tea.Cmd) {
	// Global keys.
	switch msg.Type {
	case tea.KeyCtrlC:
		return c, tea.Quit
	case tea.KeyEscape:
		return c.goBack()
	}

	switch c.step {
	case stepScenario:
		return c.handleScenarioKey(msg)
	case stepName:
		return c.handleNameKey(msg)
	case stepClass:
		return c.handleClassKey(msg)
	case stepStats:
		return c.handleStatsKey(msg)
	}
	return c, nil
}

// goBack moves to the previous step, or does nothing at the first step.
func (c GameCreation) goBack() (GameCreation, tea.Cmd) {
	if c.step > stepScenario {
		c.step--
		if c.step == stepName {
			return c, c.nameInput.Focus()
		}
	}
	return c, nil
}

// handleScenarioKey handles keys in the scenario selection step.
func (c GameCreation) handleScenarioKey(msg tea.KeyMsg) (GameCreation, tea.Cmd) {
	switch msg.Type {
	case tea.KeyUp:
		c.scenIndex--
		if c.scenIndex < 0 {
			c.scenIndex = len(c.scenarios) - 1
		}
	case tea.KeyDown:
		c.scenIndex++
		if c.scenIndex >= len(c.scenarios) {
			c.scenIndex = 0
		}
	case tea.KeyEnter:
		if len(c.scenarios) > 0 {
			c.step = stepName
			return c, c.nameInput.Focus()
		}
	default:
		switch msg.String() {
		case "j":
			c.scenIndex++
			if c.scenIndex >= len(c.scenarios) {
				c.scenIndex = 0
			}
		case "k":
			c.scenIndex--
			if c.scenIndex < 0 {
				c.scenIndex = len(c.scenarios) - 1
			}
		}
	}
	return c, nil
}

// handleNameKey handles keys in the name input step.
func (c GameCreation) handleNameKey(msg tea.KeyMsg) (GameCreation, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		c.nameInput.Blur()
		c.step = stepClass
		return c, nil
	default:
		var cmd tea.Cmd
		c.nameInput, cmd = c.nameInput.Update(msg)
		return c, cmd
	}
}

// handleClassKey handles keys in the class selection step.
func (c GameCreation) handleClassKey(msg tea.KeyMsg) (GameCreation, tea.Cmd) {
	switch msg.Type {
	case tea.KeyUp:
		c.classIndex--
		if c.classIndex < 0 {
			c.classIndex = len(classes) - 1
		}
	case tea.KeyDown:
		c.classIndex++
		if c.classIndex >= len(classes) {
			c.classIndex = 0
		}
	case tea.KeyEnter:
		c.step = stepStats
		return c, nil
	default:
		switch msg.String() {
		case "j":
			c.classIndex++
			if c.classIndex >= len(classes) {
				c.classIndex = 0
			}
		case "k":
			c.classIndex--
			if c.classIndex < 0 {
				c.classIndex = len(classes) - 1
			}
		}
	}
	return c, nil
}

// handleStatsKey handles keys in the stat allocation step.
func (c GameCreation) handleStatsKey(msg tea.KeyMsg) (GameCreation, tea.Cmd) {
	switch msg.Type {
	case tea.KeyUp:
		c.statIndex--
		if c.statIndex < 0 {
			c.statIndex = 5
		}
	case tea.KeyDown:
		c.statIndex++
		if c.statIndex > 5 {
			c.statIndex = 0
		}
	case tea.KeyRight:
		if c.totalStatPoints() < pointBuyPool {
			c.statPoints[c.statIndex]++
		}
	case tea.KeyLeft:
		if c.statPoints[c.statIndex] > 0 {
			c.statPoints[c.statIndex]--
		}
	case tea.KeyEnter:
		if c.totalStatPoints() == pointBuyPool {
			return c, c.finish()
		}
	default:
		switch msg.String() {
		case "j":
			c.statIndex++
			if c.statIndex > 5 {
				c.statIndex = 0
			}
		case "k":
			c.statIndex--
			if c.statIndex < 0 {
				c.statIndex = 5
			}
		case "l", "+", "=":
			if c.totalStatPoints() < pointBuyPool {
				c.statPoints[c.statIndex]++
			}
		case "h", "-":
			if c.statPoints[c.statIndex] > 0 {
				c.statPoints[c.statIndex]--
			}
		}
	}
	return c, nil
}

// totalStatPoints returns the sum of all allocated bonus points.
func (c GameCreation) totalStatPoints() int {
	total := 0
	for _, p := range c.statPoints {
		total += p
	}
	return total
}

// finish produces the CreationDoneMsg with the chosen parameters.
func (c GameCreation) finish() tea.Cmd {
	name := strings.TrimSpace(c.nameInput.Value())
	if name == "" {
		name = "Adventurer"
	}
	scenario := ""
	if c.scenIndex < len(c.scenarios) {
		scenario = c.scenarios[c.scenIndex].ID
	}
	class := classes[c.classIndex].name

	return func() tea.Msg {
		return CreationDoneMsg{
			Scenario: scenario,
			Name:     name,
			Class:    class,
		}
	}
}

// View renders the creation wizard.
func (c GameCreation) View() string {
	// Step indicator.
	steps := c.renderStepIndicator()

	// Current step content.
	var content string
	switch c.step {
	case stepScenario:
		content = c.viewScenario()
	case stepName:
		content = c.viewName()
	case stepClass:
		content = c.viewClass()
	case stepStats:
		content = c.viewStats()
	}

	// Compose.
	inner := lipgloss.JoinVertical(lipgloss.Left,
		"",
		StyleCreationHeader.Render("CREATE YOUR CHARACTER"),
		"",
		steps,
		"",
		content,
		"",
		c.stepHint(),
	)

	box := StyleCreationBox.Render(inner)

	return lipgloss.Place(c.width, c.height,
		lipgloss.Center, lipgloss.Center,
		box)
}

// renderStepIndicator shows the 4-step progress indicator.
func (c GameCreation) renderStepIndicator() string {
	labels := [4]string{"Scenario", "Name", "Class", "Stats"}
	var parts []string
	for i, label := range labels {
		step := creationStep(i)
		var style lipgloss.Style
		switch {
		case step < c.step:
			style = StyleStepDone
			label = "[x] " + label
		case step == c.step:
			style = StyleStepActive
			label = "[>] " + label
		default:
			style = StyleStepPending
			label = "[ ] " + label
		}
		parts = append(parts, style.Render(label))
	}
	return strings.Join(parts, StyleStepPending.Render("  "))
}

// viewScenario renders the scenario selection list.
func (c GameCreation) viewScenario() string {
	if len(c.scenarios) == 0 {
		return StyleSystem.Render("No scenarios available.")
	}

	var lines []string
	for i, s := range c.scenarios {
		title := s.Title
		if title == "" {
			title = s.ID
		}
		if i == c.scenIndex {
			lines = append(lines, StyleMenuSelected.Render("> "+title))
			if s.Description != "" {
				wrapped := WrapText(s.Description, 50)
				lines = append(lines, StyleClassDesc.Render("  "+wrapped))
			}
		} else {
			lines = append(lines, StyleMenuNormal.Render("  "+title))
		}
	}
	return strings.Join(lines, "\n")
}

// viewName renders the name input step.
func (c GameCreation) viewName() string {
	var lines []string
	lines = append(lines, StyleMenuNormal.Render("Enter your character's name:"))
	lines = append(lines, "")
	lines = append(lines, "  "+c.nameInput.View())
	return strings.Join(lines, "\n")
}

// viewClass renders the class selection step.
func (c GameCreation) viewClass() string {
	var lines []string
	lines = append(lines, StyleMenuNormal.Render("Choose your class:"))
	lines = append(lines, "")

	for i, cl := range classes {
		title := strings.ToUpper(cl.name[:1]) + cl.name[1:]
		if i == c.classIndex {
			lines = append(lines, StyleMenuSelected.Render("> "+title))
			lines = append(lines, StyleClassDesc.Render("  "+cl.desc))
			detail := fmt.Sprintf("  %s  %s  Primary: %s", cl.hp, cl.mp, cl.primary)
			lines = append(lines, StyleSessionDetail.Render(detail))
			lines = append(lines, "")
		} else {
			lines = append(lines, StyleMenuNormal.Render("  "+title))
		}
	}
	return strings.Join(lines, "\n")
}

// viewStats renders the stat allocation step.
func (c GameCreation) viewStats() string {
	remaining := pointBuyPool - c.totalStatPoints()

	var lines []string
	lines = append(lines, StyleMenuNormal.Render(
		fmt.Sprintf("Distribute %d points (base %d each):", pointBuyPool, baseStatValue)))
	lines = append(lines, StyleStatPoints.Render(
		fmt.Sprintf("Remaining: %d", remaining)))
	lines = append(lines, "")

	for i, name := range statNames {
		value := baseStatValue + c.statPoints[i]
		bar := strings.Repeat("*", c.statPoints[i])
		if bar == "" {
			bar = "-"
		}

		line := fmt.Sprintf("  %s  %2d  ", name, value)

		if i == c.statIndex {
			lines = append(lines, StyleMenuSelected.Render("> "+line)+StyleStatBar.Render(bar))
		} else {
			lines = append(lines, StyleMenuNormal.Render("  "+line)+StyleStatPoints.Render(bar))
		}
	}

	lines = append(lines, "")
	if remaining == 0 {
		lines = append(lines, StyleStepDone.Render("All points allocated. Press Enter to begin."))
	} else {
		lines = append(lines, StyleCreationHint.Render("Use left/right to adjust, allocate all points to continue."))
	}

	return strings.Join(lines, "\n")
}

// stepHint returns the context-sensitive help text for the current step.
func (c GameCreation) stepHint() string {
	switch c.step {
	case stepScenario:
		return StyleCreationHint.Render("arrow keys to select, enter to confirm, esc to go back")
	case stepName:
		return StyleCreationHint.Render("type a name, enter to confirm, esc to go back")
	case stepClass:
		return StyleCreationHint.Render("arrow keys to select, enter to confirm, esc to go back")
	case stepStats:
		return StyleCreationHint.Render("up/down to select stat, left/right to adjust, enter when done, esc to go back")
	}
	return ""
}
