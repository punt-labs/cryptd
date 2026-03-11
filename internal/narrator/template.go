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
	case "quit":
		return "Farewell, adventurer.", nil
	case "unknown_action":
		return "I don't understand that command.", nil
	default:
		return fmt.Sprintf("Something happens: %s.", event.Type), nil
	}
}
