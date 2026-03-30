package tui

import "github.com/charmbracelet/lipgloss"

// Layout constants.
const (
	SidebarWidth  = 30
	MinTermWidth  = 80
	MinTermHeight = 24
)

// Colors — matched to the HTML mockup in docs/crypt-tui-mockup.html.
var (
	ColorGold          = lipgloss.Color("#e0a040")
	ColorGreen         = lipgloss.Color("#4caf50")
	ColorRed           = lipgloss.Color("#f44336")
	ColorDamage        = lipgloss.Color("#e05050")
	ColorXP            = lipgloss.Color("#9b59b6")
	ColorSystem        = lipgloss.Color("#666666")
	ColorDim           = lipgloss.Color("#333333")
	ColorBorder        = lipgloss.Color("#333333")
	ColorCombatBg      = lipgloss.Color("#1a0a0a")
	ColorCombatBorder  = lipgloss.Color("#4a1a1a")
	ColorHPHigh        = lipgloss.Color("#2ecc71")
	ColorHPLow         = lipgloss.Color("#e74c3c")
	ColorMPBar         = lipgloss.Color("#3498db")
	ColorXPBar         = lipgloss.Color("#9b59b6")
	ColorCompass       = lipgloss.Color("#ffd54f")
)

// Text styles.
var (
	StyleRoomName  = lipgloss.NewStyle().Foreground(ColorGold).Bold(true)
	StyleAction    = lipgloss.NewStyle().Foreground(ColorGreen).Italic(true)
	StyleCombat    = lipgloss.NewStyle().Foreground(ColorRed)
	StyleDamage    = lipgloss.NewStyle().Foreground(ColorDamage).Bold(true)
	StyleXPGain    = lipgloss.NewStyle().Foreground(ColorXP)
	StyleSystem    = lipgloss.NewStyle().Foreground(ColorSystem)
	StyleNarration = lipgloss.NewStyle().Foreground(lipgloss.Color("#b0b0b0"))
)

// Layout styles.
var (
	StyleHeader = lipgloss.NewStyle().
			Padding(0, 1).
			Background(lipgloss.Color("#1a1a1a")).
			Foreground(lipgloss.Color("#c0c0c0"))

	StyleCharName  = lipgloss.NewStyle().Foreground(ColorGold).Bold(true)
	StyleCharClass = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	StyleSessionID = lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))

	StyleSidebar = lipgloss.NewStyle().
			BorderLeft(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1)

	StyleSidebarLabel = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#888888")).
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

	StyleCompassActive = lipgloss.NewStyle().
				Foreground(ColorCompass).
				Bold(true)

	StyleCompassInactive = lipgloss.NewStyle().
				Foreground(ColorDim)

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
