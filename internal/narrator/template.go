// Package narrator provides Narrator implementations.
package narrator

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/punt-labs/cryptd/internal/model"
)

// Template produces minimal one-sentence narrations from fixed templates.
// No creativity or external calls required.
type Template struct{}

// NewTemplate returns a new TemplateNarrator.
func NewTemplate() *Template { return &Template{} }

// Narrate returns a short templated string for the given event.
func (t *Template) Narrate(_ context.Context, event model.EngineEvent, _ model.GameState) (string, error) {
	switch event.Type {
	case "moved":
		return roomNarration(fmt.Sprintf("You enter %s.", event.Room), event.Details), nil
	case "looked":
		header := "You look around."
		if event.Room != "" {
			header = fmt.Sprintf("You look around %s.", event.Room)
		}
		return roomNarration(header, event.Details), nil
	case "locked_door":
		return "That way is locked.", nil
	case "no_exit":
		return "You can't go that way.", nil
	case "picked_up":
		if name, ok := event.Details["item_name"].(string); ok {
			return fmt.Sprintf("You pick up the %s.", name), nil
		}
		return "You pick something up.", nil
	case "dropped":
		if name, ok := event.Details["item_name"].(string); ok {
			return fmt.Sprintf("You drop the %s.", name), nil
		}
		return "You drop something.", nil
	case "equipped":
		if name, ok := event.Details["item_name"].(string); ok {
			return fmt.Sprintf("You equip the %s.", name), nil
		}
		return "You equip an item.", nil
	case "unequipped":
		if name, ok := event.Details["item_name"].(string); ok {
			return fmt.Sprintf("You unequip the %s.", name), nil
		}
		return "You unequip an item.", nil
	case "examined":
		if desc, ok := event.Details["description"].(string); ok && desc != "" {
			return desc, nil
		}
		if name, ok := event.Details["item_name"].(string); ok {
			return fmt.Sprintf("You see nothing special about the %s.", name), nil
		}
		return "You examine it closely.", nil
	case "inventory_listed":
		return inventoryNarration(event.Details), nil
	case "item_not_found":
		return "You don't see that here.", nil
	case "not_in_inventory":
		return "You don't have that.", nil
	case "too_heavy":
		return "That's too heavy to carry.", nil
	case "slot_occupied":
		return "You already have something equipped there.", nil
	case "slot_empty":
		return "You have nothing equipped there.", nil
	case "not_equippable":
		return "You can't equip that.", nil
	case "combat_started":
		if names, ok := event.Details["enemy_names"].(string); ok {
			return fmt.Sprintf("Combat begins! You face: %s", names), nil
		}
		return "Combat begins!", nil
	case "attack_hit":
		target, _ := event.Details["target"].(string)
		damage, _ := event.Details["damage"].(int)
		return fmt.Sprintf("You strike the %s for %d damage.", target, damage), nil
	case "attack_kill":
		target, _ := event.Details["target"].(string)
		xp, _ := event.Details["xp"].(int)
		return fmt.Sprintf("You defeat the %s! (+%d XP)", target, xp), nil
	case "enemy_attacks":
		enemy, _ := event.Details["enemy"].(string)
		damage, _ := event.Details["damage"].(int)
		return fmt.Sprintf("The %s attacks you for %d damage.", enemy, damage), nil
	case "enemy_flees":
		enemy, _ := event.Details["enemy"].(string)
		return fmt.Sprintf("The %s flees!", enemy), nil
	case "defend":
		return "You raise your guard.", nil
	case "flee_success":
		return "You flee from combat!", nil
	case "flee_fail":
		return "You fail to escape!", nil
	case "combat_won":
		return "Combat is over. You are victorious.", nil
	case "hero_died":
		return "You have fallen...", nil
	case "not_in_combat":
		return "You are not in combat.", nil
	case "in_combat":
		return "You can't do that during combat!", nil
	case "not_hero_turn":
		return "It is not your turn.", nil
	case "invalid_target":
		return "That is not a valid target.", nil
	case "spell_damage":
		spell, _ := event.Details["spell"].(string)
		target, _ := event.Details["target"].(string)
		damage, _ := event.Details["damage"].(int)
		return fmt.Sprintf("You cast %s on the %s for %d damage.", spell, target, damage), nil
	case "spell_heal":
		spell, _ := event.Details["spell"].(string)
		power, _ := event.Details["power"].(int)
		return fmt.Sprintf("You cast %s and recover %d HP.", spell, power), nil
	case "unknown_spell":
		return "You don't know that spell.", nil
	case "not_caster":
		return "Your class cannot cast that spell.", nil
	case "insufficient_mp":
		return "You don't have enough MP.", nil
	case "game_saved":
		if slot, ok := event.Details["slot"].(string); ok {
			return fmt.Sprintf("Game saved to slot %q.", slot), nil
		}
		return "Game saved.", nil
	case "game_loaded":
		if slot, ok := event.Details["slot"].(string); ok {
			return fmt.Sprintf("Game loaded from slot %q.", slot), nil
		}
		return "Game loaded.", nil
	case "save_error":
		return "Failed to save the game.", nil
	case "load_error":
		return "Failed to load the game.", nil
	case "level_up":
		level, _ := event.Details["level"].(int)
		hpGain, _ := event.Details["hp_gain"].(int)
		return fmt.Sprintf("You have reached level %d! (+%d HP)", level, hpGain), nil
	case "used_item":
		name, _ := event.Details["item_name"].(string)
		effect, _ := event.Details["effect"].(string)
		power, _ := event.Details["power"].(int)
		hp, _ := event.Details["hero_hp"].(int)
		switch effect {
		case "heal":
			return fmt.Sprintf("You use the %s and recover %d HP. (HP: %d)", name, power, hp), nil
		default:
			return fmt.Sprintf("You use the %s.", name), nil
		}
	case "not_consumable":
		return "You can't use that item.", nil
	case "help":
		return "Commands: go <dir> (n/s/e/w/u/d), look (l), take <item>, drop <item>, " +
			"equip <item>, unequip <slot>, use <item>, examine <item> (x), inventory (i), " +
			"attack [target] (a), defend, flee, cast <spell> [at <target>], " +
			"save [slot], load [slot], help (?), quit (q).", nil
	case "quit":
		return "Farewell, adventurer.", nil
	case "unknown_action":
		return "I don't understand that command.", nil
	default:
		return fmt.Sprintf("Something happens: %s.", event.Type), nil
	}
}

// inventoryNarration formats a Zork-style inventory list from event details.
func inventoryNarration(details map[string]any) string {
	items := toItemSlice(details["items"])
	equipped := toEquippedSet(details["equipped"])

	if len(items) == 0 && len(equipped) == 0 {
		return "You are empty-handed."
	}

	var lines []string
	lines = append(lines, "You are carrying:")
	for _, item := range items {
		tag := ""
		if equipped[item.id] {
			tag = " (equipped)"
		}
		lines = append(lines, fmt.Sprintf("  %s%s", item.name, tag))
	}

	// Equipped items not in inventory (shouldn't happen, but defensive).
	// Sorted for deterministic output.
	var orphaned []string
	for id := range equipped {
		found := false
		for _, item := range items {
			if item.id == id {
				found = true
				break
			}
		}
		if !found {
			orphaned = append(orphaned, id)
		}
	}
	sort.Strings(orphaned)
	for _, id := range orphaned {
		lines = append(lines, fmt.Sprintf("  %s (equipped)", id))
	}

	weight, _ := details["weight"].(float64)
	capacity, _ := details["capacity"].(float64)
	if capacity > 0 {
		lines = append(lines, fmt.Sprintf("Weight: %.1f/%.1f lbs", weight, capacity))
	}

	return strings.Join(lines, "\n")
}

// inventoryItem holds the fields we extract from []model.Item or []any.
type inventoryItem struct {
	id   string
	name string
}

// toItemSlice extracts item id/name pairs from the items detail.
// Handles both typed Go slices (from the engine) and []any (from JSON round-trip).
func toItemSlice(v any) []inventoryItem {
	switch items := v.(type) {
	case []model.Item:
		out := make([]inventoryItem, 0, len(items))
		for _, item := range items {
			name := item.Name
			if name == "" {
				name = item.ID
			}
			if item.ID == "" && name == "" {
				continue
			}
			out = append(out, inventoryItem{id: item.ID, name: name})
		}
		return out
	case []any:
		var out []inventoryItem
		for _, elem := range items {
			m, ok := elem.(map[string]any)
			if !ok {
				continue
			}
			id, _ := m["id"].(string)
			name, _ := m["name"].(string)
			if name == "" {
				name = id
			}
			if id == "" && name == "" {
				continue // skip items with no identity
			}
			out = append(out, inventoryItem{id: id, name: name})
		}
		return out
	default:
		if v != nil {
			log.Printf("narrator: toItemSlice: unexpected type %T", v)
		}
		return nil
	}
}

// toEquippedSet returns a set of item IDs that are currently equipped.
// Handles both typed model.Equipment (from the engine) and map[string]any (from JSON).
func toEquippedSet(v any) map[string]bool {
	set := make(map[string]bool)
	switch eq := v.(type) {
	case model.Equipment:
		for _, id := range []string{eq.Weapon, eq.Armor, eq.Ring, eq.Amulet} {
			if id != "" {
				set[id] = true
			}
		}
	case map[string]any:
		for _, val := range eq {
			if id, ok := val.(string); ok && id != "" {
				set[id] = true
			}
		}
	default:
		if v != nil {
			log.Printf("narrator: toEquippedSet: unexpected type %T", v)
		}
	}
	return set
}

// roomNarration appends description, exits, and items from event details to a
// header line (e.g. "You enter X." or "You look around X.").
func roomNarration(header string, details map[string]any) string {
	var parts []string
	parts = append(parts, header)

	if desc, ok := details["description"].(string); ok && desc != "" {
		parts = append(parts, desc)
	}

	parts = append(parts, exitsAndItems(details)...)

	return strings.Join(parts, " ")
}

// exitsAndItems returns formatted "Exits: ..." and "You see: ..." strings
// from event details. Handles both []string (from Go code) and []any
// (from JSON unmarshalling).
func exitsAndItems(details map[string]any) []string {
	var parts []string
	if exits := toStringSlice(details["exits"]); len(exits) > 0 {
		parts = append(parts, fmt.Sprintf("Exits: %s.", strings.Join(exits, ", ")))
	}
	if items := toStringSlice(details["items"]); len(items) > 0 {
		parts = append(parts, fmt.Sprintf("You see: %s.", strings.Join(items, ", ")))
	}
	return parts
}

// toStringSlice coerces a value to []string. Supports []string directly
// and []any with string elements (as produced by JSON unmarshalling).
func toStringSlice(v any) []string {
	switch s := v.(type) {
	case []string:
		return s
	case []any:
		out := make([]string, 0, len(s))
		for _, elem := range s {
			if str, ok := elem.(string); ok {
				out = append(out, str)
			}
		}
		return out
	default:
		return nil
	}
}
