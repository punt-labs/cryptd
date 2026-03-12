package narrator

import (
	"context"
	"fmt"
	"strings"

	"github.com/punt-labs/cryptd/internal/inference"
	"github.com/punt-labs/cryptd/internal/model"
)

// SLM uses a small language model to produce atmospheric narration for room
// events (moved, looked). All other event types delegate to the fallback
// narrator. On inference failure, room events also fall back.
type SLM struct {
	client   *inference.Client
	fallback model.Narrator
}

// NewSLM creates an SLM narrator that enriches room descriptions via the
// inference client and falls back to the provided narrator on failure or
// for non-room events.
func NewSLM(client *inference.Client, fallback model.Narrator) *SLM {
	return &SLM{client: client, fallback: fallback}
}

// roomEventTypes are the event types that benefit from SLM narration.
var roomEventTypes = map[string]bool{
	"moved":  true,
	"looked": true,
}

// narratorSystemPrompt instructs the SLM to produce atmospheric room descriptions.
const narratorSystemPrompt = `You are a narrator for a text adventure game. Given a room description seed and context, write 2-4 atmospheric sentences describing what the player sees.

Rules:
- Write in second person present tense ("You see...", "The air smells of...").
- Be concise but evocative. Focus on sensory details.
- Do not invent game mechanics, items, or exits not mentioned in the input.
- Do not use markdown, code fences, or any formatting. Plain prose only.
- Do NOT include exits or visible items — those are appended automatically.`

// Narrate produces narration for the given event. Room events (moved, looked)
// are sent to the SLM for atmospheric expansion. All other events use the
// fallback narrator. Context cancellation is propagated, not fallen back.
func (s *SLM) Narrate(ctx context.Context, event model.EngineEvent, state model.GameState) (string, error) {
	if !roomEventTypes[event.Type] {
		return s.fallback.Narrate(ctx, event, state)
	}

	desc, _ := event.Details["description"].(string)
	if desc == "" {
		return s.fallback.Narrate(ctx, event, state)
	}

	if err := ctx.Err(); err != nil {
		return "", err
	}

	prompt := buildRoomPrompt(event)

	temp := 0.7
	resp, err := s.client.ChatCompletion(ctx, []inference.Message{
		{Role: inference.RoleSystem, Content: narratorSystemPrompt},
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

	return trimmed + roomSuffix(event.Details), nil
}

// roomSuffix deterministically appends exits and visible items so gameplay
// affordances are never lost even if the SLM omits them.
func roomSuffix(details map[string]any) string {
	var parts []string
	if exits, ok := details["exits"].([]string); ok && len(exits) > 0 {
		parts = append(parts, fmt.Sprintf("Exits: %s.", strings.Join(exits, ", ")))
	}
	if items, ok := details["items"].([]string); ok && len(items) > 0 {
		parts = append(parts, fmt.Sprintf("You see: %s.", strings.Join(items, ", ")))
	}
	if len(parts) == 0 {
		return ""
	}
	return " " + strings.Join(parts, " ")
}

// buildRoomPrompt constructs the user message from event details.
func buildRoomPrompt(event model.EngineEvent) string {
	var parts []string

	parts = append(parts, "Room: "+event.Room)

	if desc, ok := event.Details["description"].(string); ok && desc != "" {
		parts = append(parts, "Description seed: "+desc)
	}

	if exits, ok := event.Details["exits"].([]string); ok && len(exits) > 0 {
		parts = append(parts, "Exits: "+strings.Join(exits, ", "))
	}

	if items, ok := event.Details["items"].([]string); ok && len(items) > 0 {
		parts = append(parts, "Visible items: "+strings.Join(items, ", "))
	}

	return strings.Join(parts, "\n")
}
