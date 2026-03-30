package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/protocol"
)

// SidebarPane renders the right sidebar panel. Pure rendering — no input handling.
type SidebarPane struct {
	width  int
	height int
}

// NewSidebarPane creates a sidebar with the given dimensions.
func NewSidebarPane(width, height int) SidebarPane {
	return SidebarPane{width: width, height: height}
}

// Render returns the full sidebar content. Returns empty string if resp is nil
// or has no State or no Party.
func (s SidebarPane) Render(resp *protocol.PlayResponse) string {
	if resp == nil || resp.State == nil || len(resp.State.Party) == 0 {
		return ""
	}
	hero := resp.State.Party[0]
	barWidth := s.width - 4 // padding for label + spacing

	var sections []string

	// HP bar
	sections = append(sections, formatBar("HP", hero.HP, hero.MaxHP, barWidth,
		BarStyle(hero.HP, hero.MaxHP)))

	// MP bar (only if MaxMP > 0)
	if hero.MaxMP > 0 {
		sections = append(sections, formatBar("MP", hero.MP, hero.MaxMP, barWidth,
			lipgloss.NewStyle().Foreground(ColorMPBar)))
	}

	// XP bar
	if resp.NextLevelXP == 0 {
		sections = append(sections, fmt.Sprintf("XP %d (MAX)", hero.XP))
	} else {
		sections = append(sections, formatBar("XP", hero.XP, resp.NextLevelXP, barWidth,
			lipgloss.NewStyle().Foreground(ColorXPBar)))
	}

	// Compass
	sections = append(sections, renderCompass(resp.Exits))

	// Equipped items
	equipped := resolveEquipped(hero)
	if len(equipped) > 0 {
		lines := []string{StyleSidebarLabel.Render("EQUIPPED")}
		for _, e := range equipped {
			lines = append(lines, StyleEquippedItem.Render(e.name)+"  "+StyleEquippedSlot.Render(e.slot))
		}
		sections = append(sections, strings.Join(lines, "\n"))
	}

	// Inventory
	if len(hero.Inventory) > 0 {
		sections = append(sections, renderInventory(hero, s.height))
	}

	// Stats
	sections = append(sections, renderStats(hero.Stats))

	return strings.Join(sections, "\n\n")
}

// formatBar renders a bar like: HP 15/20 [████████░░]
func formatBar(label string, cur, max, width int, style lipgloss.Style) string {
	prefix := fmt.Sprintf("%s %d/%d ", label, cur, max)
	// Bar width = total width minus prefix and brackets
	bw := width - len(prefix) - 2 // 2 for [ ]
	if bw < 2 {
		bw = 2
	}
	filled := 0
	if max > 0 {
		filled = bw * cur / max
		if filled > bw {
			filled = bw
		}
		if filled < 0 {
			filled = 0
		}
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", bw-filled)
	return prefix + style.Render("["+bar+"]")
}

// compassDir maps direction strings to (row, col) in the 3x3 grid.
var compassDir = map[string][2]int{
	"north":     {0, 1},
	"south":     {2, 1},
	"east":      {1, 2},
	"west":      {1, 0},
	"northeast": {0, 2},
	"ne":        {0, 2},
	"northwest": {0, 0},
	"nw":        {0, 0},
	"southeast": {2, 2},
	"se":        {2, 2},
	"southwest": {2, 0},
	"sw":        {2, 0},
}

// compassLabel maps direction strings to their display abbreviation.
var compassLabel = map[string]string{
	"north":     "N",
	"south":     "S",
	"east":      "E",
	"west":      "W",
	"northeast": "NE",
	"ne":        "NE",
	"northwest": "NW",
	"nw":        "NW",
	"southeast": "SE",
	"se":        "SE",
	"southwest": "SW",
	"sw":        "SW",
}

func renderCompass(exits []string) string {
	// Build a 3x3 grid, center is always dot
	var grid [3][3]string
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			grid[r][c] = StyleCompassInactive.Render(fmt.Sprintf("%-4s", "·"))
		}
	}
	grid[1][1] = StyleCompassInactive.Render(fmt.Sprintf("%-4s", "·"))

	// Mark active exits
	active := make(map[[2]int]string)
	for _, exit := range exits {
		lower := strings.ToLower(exit)
		if pos, ok := compassDir[lower]; ok {
			active[pos] = compassLabel[lower]
		}
	}
	for pos, label := range active {
		grid[pos[0]][pos[1]] = StyleCompassActive.Render(fmt.Sprintf("%-4s", label))
	}

	var rows []string
	for r := 0; r < 3; r++ {
		rows = append(rows, grid[r][0]+" "+grid[r][1]+" "+grid[r][2])
	}
	return strings.Join(rows, "\n")
}

type equippedEntry struct {
	name string
	slot string
}

func resolveEquipped(hero model.Character) []equippedEntry {
	// Build ID -> name lookup from inventory
	nameByID := make(map[string]string, len(hero.Inventory))
	for _, item := range hero.Inventory {
		nameByID[item.ID] = item.Name
	}

	type slotDef struct {
		id   string
		slot string
	}
	slots := []slotDef{
		{hero.Equipped.Weapon, "weapon"},
		{hero.Equipped.Armor, "armor"},
		{hero.Equipped.Ring, "ring"},
		{hero.Equipped.Amulet, "amulet"},
	}

	var entries []equippedEntry
	for _, s := range slots {
		if s.id == "" {
			continue
		}
		name := s.id // fallback to ID if not in inventory
		if n, ok := nameByID[s.id]; ok {
			name = n
		}
		entries = append(entries, equippedEntry{name: name, slot: s.slot})
	}
	return entries
}

func renderInventory(hero model.Character, maxHeight int) string {
	lines := []string{StyleSidebarLabel.Render("INVENTORY")}

	// Reserve lines for weight summary + possible truncation indicator
	available := maxHeight - 10 // rough budget for other sections
	if available < 3 {
		available = 3
	}

	var totalWeight float64
	for _, item := range hero.Inventory {
		totalWeight += item.Weight
	}

	shown := hero.Inventory
	truncated := 0
	if len(shown) > available {
		truncated = len(shown) - available
		shown = shown[:available]
	}

	for _, item := range shown {
		lines = append(lines,
			StyleInventoryItem.Render(fmt.Sprintf("%-16s %4.1f", item.Name, item.Weight)))
	}
	if truncated > 0 {
		lines = append(lines, StyleSystem.Render(fmt.Sprintf("...+%d more", truncated)))
	}
	lines = append(lines,
		StyleSystem.Render(fmt.Sprintf("Weight: %.1f / %.1f lbs", totalWeight, model.MaxCarryWeight)))

	return strings.Join(lines, "\n")
}

func renderStats(stats model.Stats) string {
	lines := []string{StyleSidebarLabel.Render("STATS")}
	pairs := [][2]struct {
		label string
		val   int
	}{
		{{" STR", stats.STR}, {"  INT", stats.INT}},
		{{" DEX", stats.DEX}, {"  WIS", stats.WIS}},
		{{" CON", stats.CON}, {"  CHA", stats.CHA}},
	}
	for _, row := range pairs {
		left := StyleStatLabel.Render(row[0].label) + " " + StyleStatValue.Render(fmt.Sprintf("%-4d", row[0].val))
		right := StyleStatLabel.Render(row[1].label) + " " + StyleStatValue.Render(fmt.Sprintf("%d", row[1].val))
		lines = append(lines, left+right)
	}
	return strings.Join(lines, "\n")
}
