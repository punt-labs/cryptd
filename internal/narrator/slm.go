package narrator

import (
	"context"
	"fmt"
	"strings"

	"github.com/punt-labs/cryptd/internal/inference"
	"github.com/punt-labs/cryptd/internal/model"
)

// SLM uses a small language model to produce atmospheric narration for key
// game events: room descriptions, combat moments, item examination, and
// level-up. Tactical events (damage numbers, error messages) use the
// fallback narrator for speed and precision. On inference failure, all
// events fall back.
type SLM struct {
	client   *inference.Client
	fallback model.Narrator
}

// NewSLM creates an SLM narrator that enriches event narration via the
// inference client and falls back to the provided narrator on failure or
// for events that don't benefit from atmospheric prose.
func NewSLM(client *inference.Client, fallback model.Narrator) *SLM {
	return &SLM{client: client, fallback: fallback}
}

// slmEventTypes maps event types to their prompt builder and post-processor.
// Events not in this map delegate to the fallback narrator.
var slmEventTypes = map[string]eventHandler{
	"moved":          {prompt: buildRoomPrompt, suffix: roomSuffix, sysPrompt: roomSystemPrompt},
	"looked":         {prompt: buildRoomPrompt, suffix: roomSuffix, sysPrompt: roomSystemPrompt},
	"combat_started": {prompt: buildCombatStartPrompt, suffix: nil, sysPrompt: momentSystemPrompt},
	"combat_won":     {prompt: buildCombatWonPrompt, suffix: nil, sysPrompt: momentSystemPrompt},
	"hero_died":      {prompt: buildHeroDiedPrompt, suffix: nil, sysPrompt: momentSystemPrompt},
	"level_up":       {prompt: buildLevelUpPrompt, suffix: levelUpSuffix, sysPrompt: momentSystemPrompt},
	"examined":       {prompt: buildExaminePrompt, suffix: nil, sysPrompt: examineSystemPrompt},
}

type eventHandler struct {
	prompt    func(model.EngineEvent) string
	suffix    func(map[string]any) string
	sysPrompt string
}

// System prompts for different event categories.
const roomSystemPrompt = `You are a narrator for a text adventure game. Given a room description seed and context, write 2-4 atmospheric sentences describing what the player sees.

Rules:
- Write in second person present tense ("You see...", "The air smells of...").
- Be concise but evocative. Focus on sensory details.
- Do not invent game mechanics, items, or exits not mentioned in the input.
- Do not use markdown, code fences, or any formatting. Plain prose only.
- Do NOT include exits or visible items — those are appended automatically.`

const momentSystemPrompt = `You are a narrator for a text adventure game. Write 1-2 dramatic sentences for this game moment.

Rules:
- Write in second person present tense.
- Be concise and impactful. One or two sentences maximum.
- Do not invent details not mentioned in the input.
- Do not use markdown, code fences, or any formatting. Plain prose only.`

const examineSystemPrompt = `You are a narrator for a text adventure game. Describe an item the player is examining.

Rules:
- Write in second person present tense.
- Write 1-3 sentences describing the item's appearance and feel.
- Stay faithful to the item description seed — do not invent properties.
- Do not use markdown, code fences, or any formatting. Plain prose only.`

// Narrate produces narration for the given event. Atmospheric events are sent
// to the SLM; tactical and error events use the fallback narrator. Context
// cancellation is propagated, not fallen back.
func (s *SLM) Narrate(ctx context.Context, event model.EngineEvent, state model.GameState) (string, error) {
	handler, ok := slmEventTypes[event.Type]
	if !ok {
		return s.fallback.Narrate(ctx, event, state)
	}

	// Room events require a description seed; examined events require a description.
	// Other atmospheric events always have enough context from their details.
	if (event.Type == "moved" || event.Type == "looked") && descriptionSeed(event) == "" {
		return s.fallback.Narrate(ctx, event, state)
	}
	if event.Type == "examined" && descriptionSeed(event) == "" {
		return s.fallback.Narrate(ctx, event, state)
	}

	if err := ctx.Err(); err != nil {
		return "", err
	}

	if s.client == nil {
		return s.fallback.Narrate(ctx, event, state)
	}

	prompt := handler.prompt(event)

	temp := 0.7
	resp, err := s.client.ChatCompletion(ctx, []inference.Message{
		{Role: inference.RoleSystem, Content: handler.sysPrompt},
		{Role: inference.RoleUser, Content: prompt},
	}, &inference.Options{Temperature: &temp, MaxTokens: 200})
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return s.fallback.Narrate(ctx, event, state)
	}

	trimmed := strings.TrimSpace(resp)
	if trimmed == "" {
		return s.fallback.Narrate(ctx, event, state)
	}

	if handler.suffix != nil {
		trimmed += handler.suffix(event.Details)
	}

	return trimmed, nil
}

func descriptionSeed(event model.EngineEvent) string {
	desc, _ := event.Details["description"].(string)
	return desc
}

// roomSuffix deterministically appends exits and visible items so gameplay
// affordances are never lost even if the SLM omits them.
func roomSuffix(details map[string]any) string {
	parts := exitsAndItems(details)
	if len(parts) == 0 {
		return ""
	}
	return " " + strings.Join(parts, " ")
}

// levelUpSuffix appends the mechanical level and HP gain.
func levelUpSuffix(details map[string]any) string {
	level, _ := details["level"].(int)
	hpGain, _ := details["hp_gain"].(int)
	if level > 0 {
		return fmt.Sprintf(" (Level %d, +%d HP)", level, hpGain)
	}
	return ""
}

// buildRoomPrompt constructs the user message from event details.
func buildRoomPrompt(event model.EngineEvent) string {
	var parts []string

	parts = append(parts, "Room: "+event.Room)

	if desc, ok := event.Details["description"].(string); ok && desc != "" {
		parts = append(parts, "Description seed: "+desc)
	}

	if exits := toStringSlice(event.Details["exits"]); len(exits) > 0 {
		parts = append(parts, "Exits: "+strings.Join(exits, ", "))
	}

	if items := toStringSlice(event.Details["items"]); len(items) > 0 {
		parts = append(parts, "Visible items: "+strings.Join(items, ", "))
	}

	return strings.Join(parts, "\n")
}

func buildCombatStartPrompt(event model.EngineEvent) string {
	names, _ := event.Details["enemy_names"].(string)
	if names == "" {
		return "Combat begins against unknown foes."
	}
	return fmt.Sprintf("Combat begins! The player faces: %s.", names)
}

func buildCombatWonPrompt(event model.EngineEvent) string {
	return "The player has defeated all enemies and won the battle."
}

func buildHeroDiedPrompt(event model.EngineEvent) string {
	return "The player has been slain in combat. Their adventure ends here."
}

func buildLevelUpPrompt(event model.EngineEvent) string {
	level, _ := event.Details["level"].(int)
	return fmt.Sprintf("The player has reached level %d! Describe this moment of growth.", level)
}

func buildExaminePrompt(event model.EngineEvent) string {
	name, _ := event.Details["item_name"].(string)
	desc, _ := event.Details["description"].(string)
	if name != "" && desc != "" {
		return fmt.Sprintf("Item: %s\nDescription seed: %s", name, desc)
	}
	if name != "" {
		return fmt.Sprintf("Item: %s", name)
	}
	return "An item of unknown origin."
}
