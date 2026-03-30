package tui

import (
	"strings"
	"testing"

	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testHero() model.Character {
	return model.Character{
		Name:  "Aldric",
		Class: "fighter",
		Level: 3,
		HP:    15,
		MaxHP: 20,
		MP:    0,
		MaxMP: 0,
		XP:    48,
		Stats: model.Stats{STR: 14, INT: 10, DEX: 12, CON: 12, WIS: 10, CHA: 10},
		Inventory: []model.Item{
			{ID: "sword-1", Name: "Iron Sword", Type: "weapon", Weight: 3.0, Value: 50},
			{ID: "shield-1", Name: "Wooden Shield", Type: "armor", Weight: 5.0, Value: 30},
			{ID: "potion-1", Name: "Heal Potion", Type: "consumable", Weight: 0.5, Value: 10},
		},
		Equipped: model.Equipment{
			Weapon: "sword-1",
			Armor:  "shield-1",
		},
	}
}

func testResp() *protocol.PlayResponse {
	hero := testHero()
	return &protocol.PlayResponse{
		Text:        "You stand in a dark corridor.",
		State:       &model.GameState{Party: []model.Character{hero}},
		Exits:       []string{"north", "east"},
		NextLevelXP: 200,
	}
}

func TestSidebar_NilResponse(t *testing.T) {
	s := NewSidebarPane(SidebarWidth, 40)
	assert.Equal(t, "", s.Render(nil))
}

func TestSidebar_NilState(t *testing.T) {
	s := NewSidebarPane(SidebarWidth, 40)
	assert.Equal(t, "", s.Render(&protocol.PlayResponse{Text: "hello"}))
}

func TestSidebar_EmptyParty(t *testing.T) {
	s := NewSidebarPane(SidebarWidth, 40)
	resp := &protocol.PlayResponse{State: &model.GameState{}}
	assert.Equal(t, "", s.Render(resp))
}

func TestSidebar_HPBar_FullHP(t *testing.T) {
	s := NewSidebarPane(SidebarWidth, 40)
	hero := testHero()
	hero.HP = 20
	hero.MaxHP = 20
	resp := &protocol.PlayResponse{
		State:       &model.GameState{Party: []model.Character{hero}},
		Exits:       []string{},
		NextLevelXP: 200,
	}
	out := s.Render(resp)
	require.NotEmpty(t, out)
	assert.Contains(t, out, "HP 20/20")
	// Full HP should have all filled blocks
	assert.Contains(t, out, "█")
	assert.NotContains(t, out, "HP 20/20 [░") // no leading empty blocks
}

func TestSidebar_HPBar_LowHP(t *testing.T) {
	s := NewSidebarPane(SidebarWidth, 40)
	hero := testHero()
	hero.HP = 3
	hero.MaxHP = 20
	resp := &protocol.PlayResponse{
		State:       &model.GameState{Party: []model.Character{hero}},
		Exits:       []string{},
		NextLevelXP: 200,
	}
	out := s.Render(resp)
	require.NotEmpty(t, out)
	assert.Contains(t, out, "HP 3/20")
	// Low HP bar should have mostly empty blocks
	assert.Contains(t, out, "░")
}

func TestSidebar_XPBar_MaxLevel(t *testing.T) {
	s := NewSidebarPane(SidebarWidth, 40)
	hero := testHero()
	hero.XP = 999
	resp := &protocol.PlayResponse{
		State:       &model.GameState{Party: []model.Character{hero}},
		Exits:       []string{},
		NextLevelXP: 0, // max level
	}
	out := s.Render(resp)
	require.NotEmpty(t, out)
	assert.Contains(t, out, "XP 999 (MAX)")
}

func TestSidebar_Compass(t *testing.T) {
	s := NewSidebarPane(SidebarWidth, 40)
	resp := testResp()
	resp.Exits = []string{"north", "east"}
	out := s.Render(resp)
	require.NotEmpty(t, out)
	// The compass should contain N and E labels
	assert.Contains(t, out, "N")
	assert.Contains(t, out, "E")
}

func TestSidebar_EquippedItems(t *testing.T) {
	s := NewSidebarPane(SidebarWidth, 40)
	resp := testResp()
	out := s.Render(resp)
	require.NotEmpty(t, out)
	// Should resolve IDs to names
	assert.Contains(t, out, "Iron Sword")
	assert.Contains(t, out, "weapon")
	assert.Contains(t, out, "Wooden Shield")
	assert.Contains(t, out, "armor")
}

func TestSidebar_EmptyEquipment(t *testing.T) {
	s := NewSidebarPane(SidebarWidth, 40)
	hero := testHero()
	hero.Equipped = model.Equipment{} // nothing equipped
	resp := &protocol.PlayResponse{
		State:       &model.GameState{Party: []model.Character{hero}},
		Exits:       []string{},
		NextLevelXP: 200,
	}
	out := s.Render(resp)
	require.NotEmpty(t, out)
	// EQUIPPED section should not appear
	assert.NotContains(t, out, "EQUIPPED")
}

func TestSidebar_Stats(t *testing.T) {
	s := NewSidebarPane(SidebarWidth, 40)
	resp := testResp()
	out := s.Render(resp)
	require.NotEmpty(t, out)
	assert.Contains(t, out, "STATS")
	assert.Contains(t, out, "STR")
	assert.Contains(t, out, "INT")
	assert.Contains(t, out, "DEX")
	assert.Contains(t, out, "WIS")
	assert.Contains(t, out, "CON")
	assert.Contains(t, out, "CHA")
}

func TestResolveEquipped_IDsToNames(t *testing.T) {
	hero := testHero()
	entries := resolveEquipped(hero)
	require.Len(t, entries, 2)
	assert.Equal(t, "Iron Sword", entries[0].name)
	assert.Equal(t, "weapon", entries[0].slot)
	assert.Equal(t, "Wooden Shield", entries[1].name)
	assert.Equal(t, "armor", entries[1].slot)
}

func TestResolveEquipped_SkipsEmpty(t *testing.T) {
	hero := testHero()
	hero.Equipped = model.Equipment{Weapon: "sword-1"}
	entries := resolveEquipped(hero)
	require.Len(t, entries, 1)
	assert.Equal(t, "weapon", entries[0].slot)
}

func TestFormatBar(t *testing.T) {
	bar := formatBar("HP", 10, 20, 30, BarStyle(10, 20))
	assert.Contains(t, bar, "HP 10/20")
	assert.Contains(t, bar, "█")
	assert.Contains(t, bar, "░")
	assert.Contains(t, bar, "[")
	assert.Contains(t, bar, "]")
}

func TestRenderCompass_AllDirections(t *testing.T) {
	exits := []string{"north", "south", "east", "west", "ne", "nw", "se", "sw"}
	out := renderCompass(exits)
	lines := strings.Split(out, "\n")
	require.Len(t, lines, 3)
	// Each direction label should appear
	assert.Contains(t, out, "N")
	assert.Contains(t, out, "S")
	assert.Contains(t, out, "E")
	assert.Contains(t, out, "W")
}
