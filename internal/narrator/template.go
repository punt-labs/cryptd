// Package narrator provides Narrator implementations.
package narrator

import (
	"context"
	"fmt"

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
		return fmt.Sprintf("You enter %s.", event.Room), nil
	case "looked":
		if event.Room != "" {
			return fmt.Sprintf("You look around %s.", event.Room), nil
		}
		return "You look around.", nil
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
		return "You check your belongings.", nil
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
	case "quit":
		return "Farewell, adventurer.", nil
	case "unknown_action":
		return "I don't understand that command.", nil
	default:
		return fmt.Sprintf("Something happens: %s.", event.Type), nil
	}
}
