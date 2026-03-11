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
	case "quit":
		return "Farewell, adventurer.", nil
	case "unknown_action":
		return "I don't understand that command.", nil
	default:
		return fmt.Sprintf("Something happens: %s.", event.Type), nil
	}
}
