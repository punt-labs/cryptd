package interpreter

import (
	"context"
	"fmt"
	"log"

	"github.com/punt-labs/cryptd/internal/inference"
	"github.com/punt-labs/cryptd/internal/model"
)

// LLMSystemPrompt instructs a large language model to produce structured JSON
// actions. More concise than the SLM prompt — Claude needs less hand-holding.
const LLMSystemPrompt = `You are a text adventure command parser. Given player input and game state context, output ONLY a JSON object.

Action types:
- {"type":"move","direction":"<north|south|east|west|up|down>"}
- {"type":"look"}
- {"type":"take","item_id":"<item>"}
- {"type":"drop","item_id":"<item>"}
- {"type":"equip","item_id":"<item>"}
- {"type":"unequip","target":"<slot>"}
- {"type":"examine","item_id":"<item>"}
- {"type":"inventory"}
- {"type":"attack","target":"<enemy>"}
- {"type":"defend"}
- {"type":"flee"}
- {"type":"cast","spell_id":"<spell>","target":"<enemy>"}
- {"type":"save","target":"<slot_name>"}
- {"type":"load","target":"<slot_name>"}
- {"type":"help"}
- {"type":"quit"}
- {"type":"unknown"}

Use exact IDs from the game state context. Resolve ambiguous references using context. Output {"type":"unknown"} if unclear.`

// LLM uses a large language model (e.g. Claude) to interpret free-text player
// input into engine actions. On failure (network error, unparseable response,
// unknown action type), it falls back to the Rules interpreter.
type LLM struct {
	client   *inference.Client
	fallback model.CommandInterpreter
}

// NewLLM creates an LLM interpreter that sends player input to the given
// inference client and falls back to the provided interpreter on failure.
func NewLLM(client *inference.Client, fallback model.CommandInterpreter) *LLM {
	return &LLM{client: client, fallback: fallback}
}

// Interpret tries the rules interpreter first for deterministic commands.
// For actions that reference items, enemies, or spells by name, the LLM is
// called to resolve IDs using game state context. Context cancellation is
// always propagated.
func (l *LLM) Interpret(ctx context.Context, input string, state model.GameState) (model.EngineAction, error) {
	if err := ctx.Err(); err != nil {
		return model.EngineAction{}, err
	}

	// Rules-first: handle aliases and exact verbs without LLM latency.
	rulesAction, err := l.fallback.Interpret(ctx, input, state)
	if err != nil {
		return model.EngineAction{}, err
	}

	// If rules produced a definitive action without an ID to resolve,
	// use it directly (no LLM call needed).
	if rulesAction.Type != "unknown" && !needsIDResolution(rulesAction) {
		return rulesAction, nil
	}

	// Either rules returned "unknown" or it returned an action with an ID
	// the LLM might resolve better. Send to LLM with game state context.
	llmAction, err := l.callLLM(ctx, input, state)
	if err != nil {
		if ctx.Err() != nil {
			return model.EngineAction{}, ctx.Err()
		}
		log.Printf("cryptd: LLM interpret failed, falling back to rules: %v", err)
		return rulesAction, nil
	}

	if llmAction.Type != "unknown" {
		return llmAction, nil
	}
	return rulesAction, nil
}

// callLLM sends the input to the LLM with game state context.
func (l *LLM) callLLM(ctx context.Context, input string, state model.GameState) (model.EngineAction, error) {
	if l.client == nil {
		return model.EngineAction{Type: "unknown"}, fmt.Errorf("no inference client")
	}

	gameCtx := BuildContext(state)
	userMsg := gameCtx + "\nPlayer input: " + input

	temp := 0.0
	resp, err := l.client.ChatCompletion(ctx, []inference.Message{
		{Role: inference.RoleSystem, Content: LLMSystemPrompt},
		{Role: inference.RoleUser, Content: userMsg},
	}, &inference.Options{Temperature: &temp, MaxTokens: 150})
	if err != nil {
		if ctx.Err() != nil {
			return model.EngineAction{}, ctx.Err()
		}
		return model.EngineAction{}, err
	}

	action, err := ParseSLMResponse(resp)
	if err != nil {
		if ctx.Err() != nil {
			return model.EngineAction{}, ctx.Err()
		}
		return model.EngineAction{}, err
	}
	return action, nil
}
