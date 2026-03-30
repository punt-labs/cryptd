package tui

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/protocol"
)

// TestRenderLobbyHarness renders the lobby screen for visual inspection.
func TestRenderLobbyHarness(t *testing.T) {
	app := NewApp(nil, "", "", "", "", nil)
	app.width = 120
	app.height = 40
	app.lobby.width = 120
	app.lobby.height = 40
	app.lobby.loadingScen = false
	app.lobby.loadingSess = false
	app.lobby.scenarios = []protocol.ScenarioInfo{
		{ID: "dungeon", Title: "The Dungeon of Doom", Description: "A classic dungeon crawl through the depths."},
		{ID: "forest", Title: "The Darkwood", Description: "Lost in a cursed forest, find your way out alive."},
		{ID: "castle", Title: "Castle Cryptborne", Description: "Siege the undead lord's fortress."},
	}
	app.lobby.sessions = []protocol.SessionInfo{
		{ID: "s1", CharacterName: "Thorn", CharacterClass: "fighter", Level: 3, RoomName: "Entry Hall"},
		{ID: "s2", CharacterName: "Wisp", CharacterClass: "mage", Level: 1, RoomName: "Clearing"},
	}

	output := app.View()
	fmt.Println("\n--- LOBBY ---\n" + output + "\n--- END ---")
}

// TestRenderCreationHarness renders the creation screen for visual inspection.
func TestRenderCreationHarness(t *testing.T) {
	scenarios := []protocol.ScenarioInfo{
		{ID: "dungeon", Title: "The Dungeon of Doom", Description: "A classic dungeon crawl."},
		{ID: "forest", Title: "The Darkwood", Description: "Lost in a cursed forest."},
	}
	creation := NewGameCreation(nil, scenarios)
	creation.width = 120
	creation.height = 40

	// Scenario step.
	fmt.Println("\n--- CREATION: SCENARIO ---")
	fmt.Println(creation.View())

	// Class step.
	creation.step = stepClass
	creation.classIndex = 1
	fmt.Println("\n--- CREATION: CLASS ---")
	fmt.Println(creation.View())

	// Stats step.
	creation.step = stepStats
	creation.statPoints = [6]int{4, 2, 2, 0, 0, 0}
	creation.statIndex = 2
	fmt.Println("\n--- CREATION: STATS ---")
	fmt.Println(creation.View())
	fmt.Println("--- END ---")
}

// TestRenderHarness instantiates the App with rich mock game state and prints
// View() output so we can visually iterate on TUI layout.
func TestRenderHarness(t *testing.T) {
	resp := &protocol.PlayResponse{
		Text: "The secrets of the system lie bare before you in this forbidden vault of encrypted credentials. Password hashes line the walls like ancient runes, each one a cryptographic incantation protecting a user's identity. The air crackles with the residual energy of failed authentication attempts, and somewhere in the shadows, corrupted processes stir restlessly.",
		State: &model.GameState{
			Party: []model.Character{
				{
					Name:  "Claude",
					Class: "thief",
					Level: 2,
					HP:    20, MaxHP: 21,
					MP: 0, MaxMP: 0,
					XP: 48,
					Stats: model.Stats{
						STR: 14, INT: 10,
						DEX: 12, WIS: 10,
						CON: 12, CHA: 10,
					},
					Inventory: []model.Item{
						{ID: "kill-nine", Name: "Kill Nine", Type: "weapon", Weight: 2.5, Value: 100},
						{ID: "alias-shield", Name: "Alias Shield", Type: "armor", Weight: 4.0, Value: 80},
						{ID: "grep-tool", Name: "Grep Tool", Type: "amulet", Weight: 0.3, Value: 60},
						{ID: "man-page", Name: "Man Page", Type: "misc", Weight: 0.1, Value: 5},
						{ID: "health-potion", Name: "Health Potion", Type: "consumable", Weight: 0.5, Value: 15},
						{ID: "shadow-file", Name: "Shadow File", Type: "misc", Weight: 0.2, Value: 20},
					},
					Equipped: model.Equipment{
						Weapon: "kill-nine",
						Armor:  "alias-shield",
						Amulet: "grep-tool",
					},
				},
			},
			Dungeon: model.DungeonState{
				CurrentRoom: "/etc/shadow",
			},
		},
		Exits:       []string{"north"},
		NextLevelXP: 200,
	}

	app := NewApp(nil, "e697dc2d", "", "Claude", "thief", resp)

	// Process GameStartMsg to populate lastResp.
	result, _ := app.Update(GameStartMsg{Response: *resp})
	appPtr := result.(*App)

	// Set window size.
	result2, _ := appPtr.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	appPtr2 := result2.(*App)

	output := appPtr2.View()
	fmt.Println("\n" + output + "\n")
}
