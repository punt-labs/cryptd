package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func creationScenarios() []protocol.ScenarioInfo {
	return []protocol.ScenarioInfo{
		{ID: "dungeon", Title: "The Dungeon", Description: "A dark dungeon."},
		{ID: "forest", Title: "Dark Forest", Description: "Lost in the woods."},
	}
}

func TestCreation_Init(t *testing.T) {
	c := NewGameCreation(mockSend(`{}`), creationScenarios())
	cmd := c.Init()
	assert.Nil(t, cmd, "creation Init should return nil")
}

func TestCreation_ScenarioNavigation(t *testing.T) {
	c := NewGameCreation(mockSend(`{}`), creationScenarios())
	assert.Equal(t, stepScenario, c.step)
	assert.Equal(t, 0, c.scenIndex)

	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 1, c.scenIndex)

	// Wrap.
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 0, c.scenIndex)

	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, 1, c.scenIndex)
}

func TestCreation_ScenarioToName(t *testing.T) {
	c := NewGameCreation(mockSend(`{}`), creationScenarios())
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, stepName, c.step)
}

func TestCreation_NameToClass(t *testing.T) {
	c := NewGameCreation(mockSend(`{}`), creationScenarios())
	c.step = stepName

	// Type a name then press enter.
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, stepClass, c.step)
}

func TestCreation_ClassNavigation(t *testing.T) {
	c := NewGameCreation(mockSend(`{}`), creationScenarios())
	c.step = stepClass
	assert.Equal(t, 0, c.classIndex)

	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 1, c.classIndex)

	// Wrap around.
	c.classIndex = 3
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 0, c.classIndex)
}

func TestCreation_ClassToStats(t *testing.T) {
	c := NewGameCreation(mockSend(`{}`), creationScenarios())
	c.step = stepClass

	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, stepStats, c.step)
}

func TestCreation_StatAllocation(t *testing.T) {
	c := NewGameCreation(mockSend(`{}`), creationScenarios())
	c.step = stepStats
	c.statIndex = 0

	// Add a point to STR.
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyRight})
	assert.Equal(t, 1, c.statPoints[0])
	assert.Equal(t, 1, c.totalStatPoints())

	// Remove a point from STR.
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyLeft})
	assert.Equal(t, 0, c.statPoints[0])

	// Can't go below zero.
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyLeft})
	assert.Equal(t, 0, c.statPoints[0])
}

func TestCreation_StatAllocationCap(t *testing.T) {
	c := NewGameCreation(mockSend(`{}`), creationScenarios())
	c.step = stepStats

	// Fill up all 8 points on one stat.
	for i := 0; i < pointBuyPool+2; i++ {
		c, _ = c.Update(tea.KeyMsg{Type: tea.KeyRight})
	}
	assert.Equal(t, pointBuyPool, c.statPoints[0], "should cap at pool size")
	assert.Equal(t, pointBuyPool, c.totalStatPoints())
}

func TestCreation_StatNavigation(t *testing.T) {
	c := NewGameCreation(mockSend(`{}`), creationScenarios())
	c.step = stepStats
	assert.Equal(t, 0, c.statIndex)

	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 1, c.statIndex)

	// Wrap.
	c.statIndex = 5
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 0, c.statIndex)

	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, 5, c.statIndex)
}

func TestCreation_FinishRequiresAllPoints(t *testing.T) {
	c := NewGameCreation(mockSend(`{}`), creationScenarios())
	c.step = stepStats
	// Only allocate 3 points.
	c.statPoints = [6]int{3, 0, 0, 0, 0, 0}

	_, cmd := c.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd, "should not finish with unallocated points")
}

func TestCreation_FinishWithAllPoints(t *testing.T) {
	c := NewGameCreation(mockSend(`{}`), creationScenarios())
	c.step = stepStats
	c.classIndex = 2 // priest
	c.statPoints = [6]int{2, 2, 2, 1, 1, 0} // total = 8

	_, cmd := c.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	msg := cmd()
	doneMsg, ok := msg.(CreationDoneMsg)
	require.True(t, ok, "expected CreationDoneMsg, got %T", msg)
	assert.Equal(t, "dungeon", doneMsg.Scenario)
	assert.Equal(t, "priest", doneMsg.Class)
	assert.Equal(t, "Adventurer", doneMsg.Name) // default, since we didn't type anything
	require.NotNil(t, doneMsg.Stats, "stats must be included in CreationDoneMsg")
	assert.Equal(t, &model.Stats{
		STR: 12, DEX: 12, CON: 12, INT: 11, WIS: 11, CHA: 10,
	}, doneMsg.Stats)
}

func TestCreation_EscGoesBack(t *testing.T) {
	c := NewGameCreation(mockSend(`{}`), creationScenarios())
	c.step = stepClass

	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEscape})
	assert.Equal(t, stepName, c.step)

	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEscape})
	assert.Equal(t, stepScenario, c.step)

	// At first step, esc does nothing (handled by App).
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyEscape})
	assert.Equal(t, stepScenario, c.step)
}

func TestCreation_VimKeys(t *testing.T) {
	c := NewGameCreation(mockSend(`{}`), creationScenarios())
	c.step = stepScenario

	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	assert.Equal(t, 1, c.scenIndex)

	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	assert.Equal(t, 0, c.scenIndex)
}

func TestCreation_StatVimKeys(t *testing.T) {
	c := NewGameCreation(mockSend(`{}`), creationScenarios())
	c.step = stepStats

	// l adds a point.
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	assert.Equal(t, 1, c.statPoints[0])

	// h removes a point.
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	assert.Equal(t, 0, c.statPoints[0])
}

func TestCreation_View(t *testing.T) {
	c := NewGameCreation(mockSend(`{}`), creationScenarios())
	c.width = 80
	c.height = 30

	v := c.View()
	assert.Contains(t, v, "CREATE YOUR CHARACTER")
	assert.Contains(t, v, "Scenario")
	assert.Contains(t, v, "The Dungeon")
}

func TestCreation_ViewClass(t *testing.T) {
	c := NewGameCreation(mockSend(`{}`), creationScenarios())
	c.width = 80
	c.height = 30
	c.step = stepClass

	v := c.View()
	assert.Contains(t, v, "Fighter")
	assert.Contains(t, v, "Mage")
	assert.Contains(t, v, "Priest")
	assert.Contains(t, v, "Thief")
}

func TestCreation_ViewStats(t *testing.T) {
	c := NewGameCreation(mockSend(`{}`), creationScenarios())
	c.width = 80
	c.height = 30
	c.step = stepStats
	c.statPoints = [6]int{4, 2, 2, 0, 0, 0}

	v := c.View()
	assert.Contains(t, v, "STR")
	assert.Contains(t, v, "DEX")
	assert.Contains(t, v, "Remaining: 0")
}
