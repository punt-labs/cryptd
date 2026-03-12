package renderer

import (
	"context"
	"strings"

	"github.com/punt-labs/cryptd/internal/model"
)

// LuxDisplay abstracts the Lux MCP transport. Show rebuilds the entire scene
// (expensive — full ImGui element tree). Update patches individual elements
// (cheap — incremental state change). The real implementation will call Lux
// MCP tools; tests use FakeLuxServer.
type LuxDisplay interface {
	RecordShow(payload any)
	RecordUpdate(payload any)
	Events() <-chan model.InputEvent
}

// LuxScene is the full element tree sent via Show on scene transitions.
// Layout follows Wizardry I: room header, party stats, enemy roster (combat),
// narration log, and input prompt — all text-based.
type LuxScene struct {
	Room      string       `json:"room"`
	Party     []LuxHero    `json:"party"`
	Enemies   []LuxEnemy   `json:"enemies,omitempty"`
	Narration string       `json:"narration"`
	InCombat  bool         `json:"in_combat"`
	Log       []string     `json:"log,omitempty"`
}

// LuxHero is the party member display state (Wizardry I showed name, class,
// AC, HP as a compact row).
type LuxHero struct {
	Name  string `json:"name"`
	Class string `json:"class"`
	Level int    `json:"level"`
	HP    int    `json:"hp"`
	MaxHP int    `json:"max_hp"`
	MP    int    `json:"mp"`
	MaxMP int    `json:"max_mp"`
	XP         int `json:"xp"`
	NextLevelXP int `json:"next_level_xp"` // 0 at max level
}

// LuxEnemy is the enemy display state during combat.
type LuxEnemy struct {
	Name  string `json:"name"`
	HP    int    `json:"hp"`
	MaxHP int    `json:"max_hp"`
}

// LuxUpdate is an incremental patch sent via Update for non-scene-changing
// events (damage ticks, log appends, HP changes).
type LuxUpdate struct {
	Type    string `json:"type"`              // "narration" (only variant for now)
	Content string `json:"content,omitempty"` // narration text or log line
	Hero    *LuxHero  `json:"hero,omitempty"`
	Enemies []LuxEnemy `json:"enemies,omitempty"`
}

// Lux renders to the Lux ImGui display surface via MCP tool calls.
// Show is called on scene transitions (room change, combat start/end).
// Update is called for incremental changes (HP tick, log append).
type Lux struct {
	display  LuxDisplay
	lastRoom string
	inCombat bool
}

// NewLux creates a LuxRenderer backed by the given display transport.
func NewLux(display LuxDisplay) *Lux {
	return &Lux{display: display}
}

// Render presents the game state and narration. Calls show() on scene
// transitions (new room, combat state change) and update() otherwise.
// Returns any transport write error from the underlying display.
func (l *Lux) Render(_ context.Context, state model.GameState, narration string) error {
	combatActive := state.Dungeon.Combat.Active
	roomChanged := state.Dungeon.CurrentRoom != l.lastRoom
	combatChanged := combatActive != l.inCombat

	if roomChanged || combatChanged {
		l.lastRoom = state.Dungeon.CurrentRoom
		l.inCombat = combatActive
		l.display.RecordShow(buildScene(state, narration))
		return l.displayErr()
	}

	// Incremental update — narration + current HP/enemy state.
	l.display.RecordUpdate(buildUpdate(state, narration))
	return l.displayErr()
}

// displayErr checks if the display supports error reporting and returns
// the first error, if any. Displays without error reporting (e.g.
// FakeLuxServer) always return nil.
func (l *Lux) displayErr() error {
	type errReporter interface {
		WriteErr() error
	}
	if r, ok := l.display.(errReporter); ok {
		return r.WriteErr()
	}
	return nil
}

// Events returns the channel of InputEvents from the Lux display.
func (l *Lux) Events() <-chan model.InputEvent {
	return l.display.Events()
}

// titleCase returns s with the first letter uppercased. Returns empty string
// for empty input (avoids slice-bounds panic).
func titleCase(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// buildScene constructs the full LuxScene for a show() call.
func buildScene(state model.GameState, narration string) LuxScene {
	scene := LuxScene{
		Room:      state.Dungeon.CurrentRoom,
		Narration: narration,
		InCombat:  state.Dungeon.Combat.Active,
	}

	for _, char := range state.Party {
		scene.Party = append(scene.Party, LuxHero{
			Name:        char.Name,
			Class:       titleCase(char.Class),
			Level:       char.Level,
			HP:          char.HP,
			MaxHP:       char.MaxHP,
			MP:          char.MP,
			MaxMP:       char.MaxMP,
			XP:          char.XP,
			NextLevelXP: char.NextLevelXP,
		})
	}

	if state.Dungeon.Combat.Active {
		for _, enemy := range state.Dungeon.Combat.Enemies {
			if enemy.HP > 0 {
				scene.Enemies = append(scene.Enemies, LuxEnemy{
					Name:  enemy.Name,
					HP:    enemy.HP,
					MaxHP: enemy.MaxHP,
				})
			}
		}
	}

	// Last few log entries for context.
	logLen := len(state.AdventureLog)
	start := logLen - 5
	if start < 0 {
		start = 0
	}
	for _, entry := range state.AdventureLog[start:] {
		scene.Log = append(scene.Log, entry.Text)
	}

	return scene
}

// buildUpdate constructs a LuxUpdate for an update() call.
func buildUpdate(state model.GameState, narration string) LuxUpdate {
	update := LuxUpdate{
		Type:    "narration",
		Content: narration,
	}

	if len(state.Party) > 0 {
		char := state.Party[0]
		hero := LuxHero{
			Name:        char.Name,
			Class:       titleCase(char.Class),
			Level:       char.Level,
			HP:          char.HP,
			MaxHP:       char.MaxHP,
			MP:          char.MP,
			MaxMP:       char.MaxMP,
			XP:          char.XP,
			NextLevelXP: char.NextLevelXP,
		}
		update.Hero = &hero
	}

	if state.Dungeon.Combat.Active {
		for _, enemy := range state.Dungeon.Combat.Enemies {
			if enemy.HP > 0 {
				update.Enemies = append(update.Enemies, LuxEnemy{
					Name:  enemy.Name,
					HP:    enemy.HP,
					MaxHP: enemy.MaxHP,
				})
			}
		}
	}

	return update
}
