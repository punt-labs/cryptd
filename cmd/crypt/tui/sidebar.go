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
	// Inner width available for content (sidebar border + padding eat ~4 chars).
	innerW := s.width - 4
	if innerW < 10 {
		innerW = 10
	}

	var sections []string

	// --- HIT POINTS ---
	sections = append(sections, SectionHeader("Hit Points"))
	hpColor := ColorHPHigh
	if hero.MaxHP > 0 {
		pct := float64(hero.HP) / float64(hero.MaxHP)
		if pct <= 0.3 {
			hpColor = ColorHPLow
		} else if pct <= 0.6 {
			hpColor = ColorGold
		}
	}
	sections = append(sections, RenderBar(hero.HP, hero.MaxHP, innerW, hpColor))

	// --- EXPERIENCE ---
	sections = append(sections, SectionHeader("Experience"))
	if resp.NextLevelXP == 0 {
		sections = append(sections, fmt.Sprintf("XP %d (MAX)", hero.XP))
	} else {
		sections = append(sections, RenderBar(hero.XP, resp.NextLevelXP, innerW, ColorXPBar))
	}

	sections = append(sections, SectionDivider(innerW))

	// --- EXITS ---
	sections = append(sections, SectionHeader("Exits"))
	sections = append(sections, renderCompass(resp.Exits))

	sections = append(sections, SectionDivider(innerW))

	// --- EQUIPPED ---
	equipped := resolveEquipped(hero)
	if len(equipped) > 0 {
		sections = append(sections, SectionHeader("Equipped"))
		for _, e := range equipped {
			name := StyleEquippedItem.Render(e.name)
			slot := StyleEquippedSlot.Render(e.slot)
			// Right-align the slot label.
			nameLen := lipgloss.Width(name)
			slotLen := lipgloss.Width(slot)
			gap := innerW - nameLen - slotLen
			if gap < 1 {
				gap = 1
			}
			sections = append(sections, name+strings.Repeat(" ", gap)+slot)
		}
		sections = append(sections, SectionDivider(innerW))
	}

	// --- INVENTORY (non-equipped items only) ---
	invItems := nonEquippedItems(hero)
	if len(invItems) > 0 {
		sections = append(sections, renderInventory(hero, invItems, innerW, s.height))
		sections = append(sections, SectionDivider(innerW))
	}

	// --- STATS ---
	sections = append(sections, renderStats(hero.Stats, innerW))

	return strings.Join(sections, "\n")
}

// formatBar renders a bar like: HP 15/20 [████████░░]
// Kept for backward compatibility with combat overlay and tests.
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
	"north":     " N ",
	"south":     " S ",
	"east":      " E ",
	"west":      " W ",
	"northeast": "NE ",
	"ne":        "NE ",
	"northwest": "NW ",
	"nw":        "NW ",
	"southeast": "SE ",
	"se":        "SE ",
	"southwest": "SW ",
	"sw":        "SW ",
}

func renderCompass(exits []string) string {
	// Cell width for each compass cell.
	cellW := 5
	cellH := 1

	// Mark active exits.
	active := make(map[[2]int]string)
	for _, exit := range exits {
		lower := strings.ToLower(exit)
		if pos, ok := compassDir[lower]; ok {
			active[pos] = compassLabel[lower]
		}
	}

	// Render each cell.
	var grid [3][3]string
	for r := 0; r < 3; r++ {
		for c := 0; c < 3; c++ {
			pos := [2]int{r, c}
			if label, ok := active[pos]; ok {
				grid[r][c] = StyleExitActive.
					Width(cellW).
					Height(cellH).
					Render(label)
			} else if r == 1 && c == 1 {
				// Center dot.
				grid[r][c] = StyleExitInactive.
					Width(cellW).
					Height(cellH).
					Render(" * ")
			} else {
				// Empty cell — show dim direction hint.
				hints := map[[2]int]string{
					{0, 0}: "   ", {0, 1}: " N ", {0, 2}: "   ",
					{1, 0}: " W ", {1, 2}: " E ",
					{2, 0}: "   ", {2, 1}: " S ", {2, 2}: "   ",
				}
				hint := hints[pos]
				grid[r][c] = StyleExitInactive.
					Width(cellW).
					Height(cellH).
					Render(hint)
			}
		}
	}

	var rows []string
	for r := 0; r < 3; r++ {
		rows = append(rows,
			lipgloss.JoinHorizontal(lipgloss.Center, grid[r][0], grid[r][1], grid[r][2]))
	}
	return lipgloss.JoinVertical(lipgloss.Center, rows...)
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

// nonEquippedItems returns inventory items that are not currently equipped.
func nonEquippedItems(hero model.Character) []model.Item {
	equippedIDs := map[string]bool{
		hero.Equipped.Weapon: true,
		hero.Equipped.Armor:  true,
		hero.Equipped.Ring:   true,
		hero.Equipped.Amulet: true,
	}
	var items []model.Item
	for _, item := range hero.Inventory {
		if !equippedIDs[item.ID] {
			items = append(items, item)
		}
	}
	return items
}

func renderInventory(hero model.Character, items []model.Item, innerW, maxHeight int) string {
	lines := []string{SectionHeader("Inventory")}

	// Reserve lines for weight summary + possible truncation indicator
	available := maxHeight - 10 // rough budget for other sections
	if available < 3 {
		available = 3
	}

	var totalWeight float64
	for _, item := range hero.Inventory {
		totalWeight += item.Weight
	}

	shown := items
	truncated := 0
	if len(shown) > available {
		truncated = len(shown) - available
		shown = shown[:available]
	}

	for _, item := range shown {
		name := StyleInventoryItem.Render(item.Name)
		weight := StyleEquippedSlot.Render(fmt.Sprintf("%.1f lb", item.Weight))
		nameLen := lipgloss.Width(name)
		weightLen := lipgloss.Width(weight)
		gap := innerW - nameLen - weightLen
		if gap < 1 {
			gap = 1
		}
		lines = append(lines, name+strings.Repeat(" ", gap)+weight)
	}
	if truncated > 0 {
		lines = append(lines, StyleSystem.Render(fmt.Sprintf("...+%d more", truncated)))
	}
	lines = append(lines,
		StyleSystem.Render(fmt.Sprintf("Weight: %.1f / %.1f lbs", totalWeight, model.MaxCarryWeight)))

	return strings.Join(lines, "\n")
}

func renderStats(stats model.Stats, innerW int) string {
	lines := []string{SectionHeader("Stats")}
	pairs := [][2]struct {
		label string
		val   int
	}{
		{{"STR", stats.STR}, {"INT", stats.INT}},
		{{"DEX", stats.DEX}, {"WIS", stats.WIS}},
		{{"CON", stats.CON}, {"CHA", stats.CHA}},
	}
	colW := innerW / 2
	if colW < 8 {
		colW = 8
	}
	for _, row := range pairs {
		left := StyleStatLabel.Render(row[0].label) + " " + StyleStatValue.Render(fmt.Sprintf("%d", row[0].val))
		right := StyleStatLabel.Render(row[1].label) + " " + StyleStatValue.Render(fmt.Sprintf("%d", row[1].val))
		leftPad := colW - lipgloss.Width(left)
		if leftPad < 1 {
			leftPad = 1
		}
		lines = append(lines, left+strings.Repeat(" ", leftPad)+right)
	}
	return strings.Join(lines, "\n")
}
