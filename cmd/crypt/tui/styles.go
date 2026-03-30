package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Layout constants.
const (
	SidebarWidth  = 32
	MinTermWidth  = 80
	MinTermHeight = 24
)

// Colors — matched to the HTML mockup in docs/crypt-tui-mockup.html.
var (
	ColorGold         = lipgloss.Color("#e0a040")
	ColorGreen        = lipgloss.Color("#4caf50")
	ColorActionGreen  = lipgloss.Color("#7a9a7a")
	ColorRed          = lipgloss.Color("#f44336")
	ColorDamage       = lipgloss.Color("#e05050")
	ColorXP           = lipgloss.Color("#9b59b6")
	ColorSystem       = lipgloss.Color("#666666")
	ColorDim          = lipgloss.Color("#333333")
	ColorBorder       = lipgloss.Color("#333333")
	ColorCombatBg     = lipgloss.Color("#1a0a0a")
	ColorCombatBorder = lipgloss.Color("#4a1a1a")
	ColorCombatText   = lipgloss.Color("#c07050")
	ColorHPHigh       = lipgloss.Color("#2ecc71")
	ColorHPLow        = lipgloss.Color("#e74c3c")
	ColorMPBar        = lipgloss.Color("#3498db")
	ColorXPBar        = lipgloss.Color("#9b59b6")
	ColorExitActive   = lipgloss.Color("#4a9a4a")
	ColorExitBorder   = lipgloss.Color("#2d5a2d")
	ColorExitBg       = lipgloss.Color("#1a2a1a")
	ColorExitInactive = lipgloss.Color("#333333")
	ColorExitDimBg    = lipgloss.Color("#1a1a1a")
	ColorBarBg        = lipgloss.Color("#222222")
	ColorBarText      = lipgloss.Color("#ffffff")
	ColorSectionLabel = lipgloss.Color("#888888")
	ColorDivider      = lipgloss.Color("#2a2a2a")
	ColorNarrBg       = lipgloss.Color("#0f0f0f")
	ColorSidebarBg    = lipgloss.Color("#111111")
	ColorHeaderBg     = lipgloss.Color("#1a1a1a")
)

// Text styles.
var (
	StyleRoomName  = lipgloss.NewStyle().Foreground(ColorGold).Bold(true)
	StyleAction    = lipgloss.NewStyle().Foreground(ColorActionGreen).Italic(true)
	StyleCombat    = lipgloss.NewStyle().Foreground(ColorCombatText)
	StyleDamage    = lipgloss.NewStyle().Foreground(ColorDamage).Bold(true)
	StyleXPGain    = lipgloss.NewStyle().Foreground(ColorXP)
	StyleSystem    = lipgloss.NewStyle().Foreground(ColorSystem)
	StyleNarration = lipgloss.NewStyle().Foreground(lipgloss.Color("#b0b0b0"))
)

// Layout styles.
var (
	StyleHeader = lipgloss.NewStyle().
			Padding(0, 1).
			Background(ColorHeaderBg).
			Foreground(lipgloss.Color("#c0c0c0"))

	StyleCharName  = lipgloss.NewStyle().Foreground(ColorGold).Bold(true)
	StyleCharClass = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	StyleLevelXP   = lipgloss.NewStyle().Foreground(lipgloss.Color("#7a7a7a"))
	StyleSessionID = lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))

	StyleNarrationPane = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorBorder).
				Background(ColorNarrBg).
				Padding(0, 1)

	StyleSidebar = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Background(ColorSidebarBg).
			Padding(0, 1)

	StyleSidebarLabel = lipgloss.NewStyle().
				Foreground(ColorSectionLabel).
				Bold(true)

	StyleCombatOverlay = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorCombatBorder).
				Background(ColorCombatBg).
				Padding(1, 2)

	StyleCombatTitle = lipgloss.NewStyle().
				Foreground(ColorRed).
				Bold(true)

	StylePrompt = lipgloss.NewStyle().Foreground(ColorGold).Bold(true)

	StyleExitActive = lipgloss.NewStyle().
			Foreground(ColorExitActive).
			Background(ColorExitBg).
			Border(lipgloss.NormalBorder()).
			BorderForeground(ColorExitBorder).
			Bold(true).
			Align(lipgloss.Center)

	StyleExitInactive = lipgloss.NewStyle().
				Foreground(ColorExitInactive).
				Background(ColorExitDimBg).
				Border(lipgloss.NormalBorder()).
				BorderForeground(ColorExitDimBg).
				Align(lipgloss.Center)

	StyleEquippedItem  = lipgloss.NewStyle().Foreground(ColorGold)
	StyleEquippedSlot  = lipgloss.NewStyle().Foreground(lipgloss.Color("#666666"))
	StyleInventoryItem = lipgloss.NewStyle().Foreground(lipgloss.Color("#c0c0c0"))
	StyleStatLabel     = lipgloss.NewStyle().Foreground(lipgloss.Color("#777777"))
	StyleStatValue     = lipgloss.NewStyle().Foreground(lipgloss.Color("#aaaaaa"))
)

// BarStyle returns the appropriate HP bar color based on percentage.
func BarStyle(cur, max int) lipgloss.Style {
	if max <= 0 {
		return lipgloss.NewStyle().Foreground(ColorHPLow)
	}
	pct := float64(cur) / float64(max)
	switch {
	case pct > 0.6:
		return lipgloss.NewStyle().Foreground(ColorHPHigh)
	case pct > 0.3:
		return lipgloss.NewStyle().Foreground(ColorGold)
	default:
		return lipgloss.NewStyle().Foreground(ColorHPLow)
	}
}

// RenderBar renders a progress bar with a centered text overlay.
// The bar is `width` characters wide. The filled portion uses `fillColor`
// as background, the empty portion uses ColorBarBg. The text label is
// centered across the entire bar.
func RenderBar(label string, cur, max, width int, fillColor lipgloss.Color) string {
	if width < 4 {
		width = 4
	}

	// Calculate fill width.
	filled := 0
	if max > 0 {
		filled = width * cur / max
		if filled > width {
			filled = width
		}
		if filled < 0 {
			filled = 0
		}
	}

	// Build the text overlay, e.g. "20 / 21"
	text := fmt.Sprintf("%d / %d", cur, max)

	// Pad or truncate text to exactly `width` characters, centered.
	if len(text) > width {
		text = text[:width]
	}
	padTotal := width - len(text)
	padLeft := padTotal / 2
	padRight := padTotal - padLeft
	paddedText := strings.Repeat(" ", padLeft) + text + strings.Repeat(" ", padRight)

	// Render each character with the appropriate background.
	fillStyle := lipgloss.NewStyle().
		Foreground(ColorBarText).
		Background(fillColor).
		Bold(true)
	emptyStyle := lipgloss.NewStyle().
		Foreground(ColorBarText).
		Background(ColorBarBg)

	filledText := paddedText[:filled]
	emptyText := paddedText[filled:]

	return fillStyle.Render(filledText) + emptyStyle.Render(emptyText)
}

// SectionDivider returns a horizontal divider line for the sidebar.
func SectionDivider(width int) string {
	if width < 1 {
		width = 1
	}
	return lipgloss.NewStyle().Foreground(ColorDivider).Render(strings.Repeat("─", width))
}

// SectionHeader renders an uppercase section label.
func SectionHeader(label string) string {
	return StyleSidebarLabel.Render(strings.ToUpper(label))
}

// WrapText wraps text to fit within the given width, breaking on word
// boundaries. This ensures narration text never overflows the viewport.
func WrapText(text string, width int) string {
	if width <= 0 {
		return text
	}
	var result strings.Builder
	for _, line := range strings.Split(text, "\n") {
		if result.Len() > 0 {
			result.WriteByte('\n')
		}
		wrapLine(&result, line, width)
	}
	return result.String()
}

func wrapLine(b *strings.Builder, line string, width int) {
	words := strings.Fields(line)
	if len(words) == 0 {
		return
	}
	col := 0
	for i, word := range words {
		wLen := len(word)
		if i == 0 {
			b.WriteString(word)
			col = wLen
			continue
		}
		if col+1+wLen > width {
			b.WriteByte('\n')
			b.WriteString(word)
			col = wLen
		} else {
			b.WriteByte(' ')
			b.WriteString(word)
			col += 1 + wLen
		}
	}
}
