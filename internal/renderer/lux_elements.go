package renderer

import (
	"fmt"
	"strings"

	"github.com/punt-labs/cryptd/internal/model"
)

// SceneToElements converts a LuxScene into Lux-native element dicts suitable
// for a show() call. Layout: room header, party HP bars, separator, narration
// markdown, separator, quick-action buttons (horizontal), text input + send.
func SceneToElements(scene LuxScene) []map[string]any {
	var elements []map[string]any

	// Room header.
	elements = append(elements, map[string]any{
		"kind":    "text",
		"id":      "room_header",
		"content": scene.Room,
		"style":   "heading",
	})

	// Party HP — grouped in columns.
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

	// Separator before narration.
	elements = append(elements, map[string]any{
		"kind": "separator",
		"id":   "sep_narration",
	})

	// Narration.
	elements = append(elements, map[string]any{
		"kind":    "markdown",
		"id":      "narration",
		"content": scene.Narration,
	})

	// Separator before controls.
	elements = append(elements, map[string]any{
		"kind": "separator",
		"id":   "sep_controls",
	})

	// Quick-action buttons in a horizontal row.
	if len(scene.Actions) > 0 {
		var buttons []map[string]any
		for _, action := range scene.Actions {
			buttons = append(buttons, map[string]any{
				"kind":  "button",
				"id":    "act_" + action,
				"label": action,
			})
		}
		elements = append(elements, map[string]any{
			"kind":     "group",
			"id":       "actions",
			"layout":   "columns",
			"children": buttons,
		})
	}

	// Free-form text input + send button.
	elements = append(elements, map[string]any{
		"kind":     "group",
		"id":       "input_row",
		"layout":   "columns",
		"children": []map[string]any{
			{
				"kind":  "input_text",
				"id":    "cmd_input",
				"label": "",
				"hint":  "Type a command...",
				"value": "",
			},
			{
				"kind":  "button",
				"id":    "act_send",
				"label": "Send",
			},
		},
	})

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

	// Enemy HP patches — use name+index IDs to match scene elements.
	for i, enemy := range update.Enemies {
		fraction := hpFraction(enemy.HP, enemy.MaxHP)
		patches = append(patches, map[string]any{
			"id": enemyElementID(enemy.Name, i),
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
// TranslateLuxEvent converts a Lux interaction event map into an InputEvent.
// Accepts button clicks on act_<command> elements. Lux sends button
// interactions with action set to the element ID or "clicked"; both are
// accepted. Other actions (hover, focus) are rejected.
func TranslateLuxEvent(luxEvent map[string]any) (model.InputEvent, bool) {
	elementID, _ := luxEvent["element_id"].(string)
	action, _ := luxEvent["action"].(string)

	if !strings.HasPrefix(elementID, "act_") {
		return model.InputEvent{}, false
	}
	if action != "clicked" && action != elementID {
		return model.InputEvent{}, false
	}

	command := strings.TrimPrefix(elementID, "act_")
	if command == "" {
		return model.InputEvent{}, false
	}
	return model.InputEvent{Type: "input", Payload: command}, true
}

// TranslateLuxTextInput extracts the text value from a cmd_input "changed"
// interaction. Returns the text and true if this is a text input event.
func TranslateLuxTextInput(luxEvent map[string]any) (string, bool) {
	elementID, _ := luxEvent["element_id"].(string)
	action, _ := luxEvent["action"].(string)
	if elementID == "cmd_input" && action == "changed" {
		val, ok := luxEvent["value"].(string)
		return val, ok
	}
	return "", false
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
// Uses name+index IDs so that (a) same-named enemies get distinct IDs and
// (b) IDs are stable across show/update cycles within the same combat
// encounter (both buildScene and buildUpdate iterate the same filtered list).
func enemyProgressElement(index int, enemy LuxEnemy) map[string]any {
	return map[string]any{
		"kind":     "progress",
		"id":       enemyElementID(enemy.Name, index),
		"fraction": hpFraction(enemy.HP, enemy.MaxHP),
		"label":    hpLabel(enemy.Name, enemy.HP, enemy.MaxHP),
	}
}

// enemyElementID returns a stable element ID for an enemy based on name + index.
func enemyElementID(name string, index int) string {
	return fmt.Sprintf("enemy_%s_%d_hp", strings.ToLower(strings.ReplaceAll(name, " ", "_")), index)
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
