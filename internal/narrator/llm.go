package narrator

import (
	"context"
	"log"
	"strings"

	"github.com/punt-labs/cryptd/internal/inference"
	"github.com/punt-labs/cryptd/internal/model"
)

// LLM uses a large language model (e.g. Claude) to produce atmospheric narration
// for key game events. Tactical events use the fallback narrator. On inference
// failure, all events fall back.
type LLM struct {
	client   *inference.Client
	fallback model.Narrator
}

// NewLLM creates an LLM narrator that enriches event narration via the
// inference client and falls back to the provided narrator on failure or
// for events that don't benefit from atmospheric prose.
func NewLLM(client *inference.Client, fallback model.Narrator) *LLM {
	return &LLM{client: client, fallback: fallback}
}

// llmEventTypes maps event types to their prompt builder and post-processor.
var llmEventTypes = map[string]eventHandler{
	"moved":          {prompt: buildRoomPrompt, suffix: roomSuffix, sysPrompt: llmRoomSystemPrompt},
	"looked":         {prompt: buildRoomPrompt, suffix: roomSuffix, sysPrompt: llmRoomSystemPrompt},
	"combat_started": {prompt: buildCombatStartPrompt, suffix: nil, sysPrompt: llmMomentSystemPrompt},
	"combat_won":     {prompt: buildCombatWonPrompt, suffix: nil, sysPrompt: llmMomentSystemPrompt},
	"hero_died":      {prompt: buildHeroDiedPrompt, suffix: nil, sysPrompt: llmMomentSystemPrompt},
	"level_up":       {prompt: buildLevelUpPrompt, suffix: levelUpSuffix, sysPrompt: llmMomentSystemPrompt},
	"examined":       {prompt: buildExaminePrompt, suffix: nil, sysPrompt: llmExamineSystemPrompt},
}

// System prompts for LLM narration — more concise than SLM prompts.
const llmRoomSystemPrompt = `Text adventure narrator. Write 2-4 atmospheric sentences in second person present tense describing what the player sees. Plain prose only — no markdown, no exits or items list (appended automatically). Stay faithful to the description seed.`

const llmMomentSystemPrompt = `Text adventure narrator. Write 1-2 dramatic sentences in second person present tense. Plain prose only — no markdown. Do not invent details.`

const llmExamineSystemPrompt = `Text adventure narrator. Describe an item in 1-3 sentences, second person present tense. Stay faithful to the description seed. Plain prose only.`

// Narrate produces narration for the given event. Atmospheric events are sent
// to the LLM; tactical and error events use the fallback narrator. Context
// cancellation is propagated, not fallen back.
func (l *LLM) Narrate(ctx context.Context, event model.EngineEvent, state model.GameState) (string, error) {
	handler, ok := llmEventTypes[event.Type]
	if !ok {
		return l.fallback.Narrate(ctx, event, state)
	}

	// Room events require a description seed; examined events require a description.
	if (event.Type == "moved" || event.Type == "looked") && descriptionSeed(event) == "" {
		return l.fallback.Narrate(ctx, event, state)
	}
	if event.Type == "examined" && descriptionSeed(event) == "" {
		return l.fallback.Narrate(ctx, event, state)
	}

	if err := ctx.Err(); err != nil {
		return "", err
	}

	if l.client == nil {
		return l.fallback.Narrate(ctx, event, state)
	}

	prompt := handler.prompt(event)

	temp := 0.7
	resp, err := l.client.ChatCompletion(ctx, []inference.Message{
		{Role: inference.RoleSystem, Content: handler.sysPrompt},
		{Role: inference.RoleUser, Content: prompt},
	}, &inference.Options{Temperature: &temp, MaxTokens: 300})
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		log.Printf("cryptd: LLM narrate failed, falling back to template: %v", err)
		return l.fallback.Narrate(ctx, event, state)
	}

	trimmed := strings.TrimSpace(resp)
	if trimmed == "" {
		log.Println("cryptd: LLM returned empty response, falling back to template")
		return l.fallback.Narrate(ctx, event, state)
	}

	if handler.suffix != nil {
		trimmed += handler.suffix(event.Details)
	}

	return trimmed, nil
}

