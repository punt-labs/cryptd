package renderer

import (
	"fmt"
	"strings"

	"github.com/punt-labs/cryptd/internal/model"
)

// SceneToElements converts a LuxScene into Lux-native element dicts suitable
// for a show() call. The element tree follows a Wizardry I layout: room header,
// party HP bars, enemy HP bars (combat), narration markdown, action buttons.
func SceneToElements(scene LuxScene) []map[string]any {
	var elements []map[string]any

	// Room header.
	elements = append(elements, map[string]any{
		"kind":    "text",
		"id":      "room_header",
		"content": scene.Room,
		"style":   "heading",
	})

	// Party HP — grouped in columns. Note: SceneToElements renders all party
	// members, but UpdateToPatches only patches hero_0 because LuxUpdate carries
	// a single Hero pointer. Multi-hero updates require a LuxUpdate schema change.
	if len(scene.Party) > 0 {
		var children []map[string]any
		for i, hero := range scene.Party {
			children = append(children, heroProgressElement(i, hero))
		}
		elements = append(elements, map[string]any{
			"kind":     "group",
			"id":       "party",
			"layout":   "columns",
			"children": children,
		})
	}

	// Enemy HP bars (combat only).
	if scene.InCombat {
		for i, enemy := range scene.Enemies {
			elements = append(elements, enemyProgressElement(i, enemy))
		}
	}

	// Narration.
	elements = append(elements, map[string]any{
		"kind":    "markdown",
		"id":      "narration",
		"content": scene.Narration,
	})

	// Action buttons.
	for _, action := range scene.Actions {
		elements = append(elements, map[string]any{
			"kind":  "button",
			"id":    "act_" + action,
			"label": action,
		})
	}

	return elements
}

// UpdateToPatches converts a LuxUpdate into Lux-native patch dicts suitable
// for an update() call. Each patch targets an element by ID and sets changed
// properties.
func UpdateToPatches(update LuxUpdate) []map[string]any {
	var patches []map[string]any

	// Narration content patch (skip if empty to avoid clearing the panel).
	if update.Content != "" {
		patches = append(patches, map[string]any{
			"id":  "narration",
			"set": map[string]any{"content": update.Content},
		})
	}

	// Hero HP patch (first party member only — matches buildUpdate behavior).
	if update.Hero != nil {
		fraction := hpFraction(update.Hero.HP, update.Hero.MaxHP)
		patches = append(patches, map[string]any{
			"id": "hero_0_hp",
			"set": map[string]any{
				"fraction": fraction,
				"label":    hpLabel(update.Hero.Name, update.Hero.HP, update.Hero.MaxHP),
			},
		})
	}

	// Enemy HP patches — use name-based IDs to match scene elements.
	for _, enemy := range update.Enemies {
		fraction := hpFraction(enemy.HP, enemy.MaxHP)
		patches = append(patches, map[string]any{
			"id": enemyElementID(enemy.Name),
			"set": map[string]any{
				"fraction": fraction,
				"label":    hpLabel(enemy.Name, enemy.HP, enemy.MaxHP),
			},
		})
	}

	// Action button patches.
	for _, action := range update.Actions {
		patches = append(patches, map[string]any{
			"id":  "act_" + action,
			"set": map[string]any{"label": action},
		})
	}

	return patches
}

// TranslateLuxEvent converts a Lux interaction event map into an InputEvent.
// Button IDs follow the "act_<action>" convention; unknown IDs return false.
func TranslateLuxEvent(luxEvent map[string]any) (model.InputEvent, bool) {
	elementID, _ := luxEvent["element_id"].(string)
	action, _ := luxEvent["action"].(string)

	if action != "clicked" || !strings.HasPrefix(elementID, "act_") {
		return model.InputEvent{}, false
	}

	command := strings.TrimPrefix(elementID, "act_")
	if command == "" {
		return model.InputEvent{}, false
	}
	return model.InputEvent{Type: "input", Payload: command}, true
}

// heroProgressElement builds a progress bar element for a party member.
func heroProgressElement(index int, hero LuxHero) map[string]any {
	return map[string]any{
		"kind":     "progress",
		"id":       fmt.Sprintf("hero_%d_hp", index),
		"fraction": hpFraction(hero.HP, hero.MaxHP),
		"label":    hpLabel(hero.Name, hero.HP, hero.MaxHP),
	}
}

// enemyProgressElement builds a progress bar element for an enemy.
// Uses name-based IDs (not positional indices) so patches target the correct
// element even after enemies die and the live-enemy list shrinks.
func enemyProgressElement(_ int, enemy LuxEnemy) map[string]any {
	return map[string]any{
		"kind":     "progress",
		"id":       enemyElementID(enemy.Name),
		"fraction": hpFraction(enemy.HP, enemy.MaxHP),
		"label":    hpLabel(enemy.Name, enemy.HP, enemy.MaxHP),
	}
}

// enemyElementID returns a stable element ID for an enemy based on its name.
func enemyElementID(name string) string {
	return "enemy_" + strings.ToLower(strings.ReplaceAll(name, " ", "_")) + "_hp"
}

// hpFraction returns HP/MaxHP as a float64, clamped to [0, 1].
func hpFraction(hp, maxHP int) float64 {
	if maxHP <= 0 {
		return 0
	}
	f := float64(hp) / float64(maxHP)
	if f > 1 {
		return 1
	}
	if f < 0 {
		return 0
	}
	return f
}

// hpLabel formats "Name HP hp/maxHP".
func hpLabel(name string, hp, maxHP int) string {
	return fmt.Sprintf("%s HP %d/%d", name, hp, maxHP)
}
