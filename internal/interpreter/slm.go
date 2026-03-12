package interpreter

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/punt-labs/cryptd/internal/inference"
	"github.com/punt-labs/cryptd/internal/model"
)

// SLM uses a small language model to interpret free-text player input into
// engine actions. On failure (network error, unparseable response, unknown
// action type), it falls back to the Rules interpreter.
type SLM struct {
	client   *inference.Client
	fallback *Rules
}

// NewSLM creates an SLM interpreter that sends player input to the given
// inference client and falls back to the Rules interpreter on failure.
func NewSLM(client *inference.Client, fallback *Rules) *SLM {
	return &SLM{client: client, fallback: fallback}
}

// systemPrompt instructs the SLM to produce structured JSON actions.
const systemPrompt = `You are a text adventure command parser. Given a player's input, output ONLY a JSON object with the player's intended action.

Supported action types and their fields:
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
- {"type":"help"}
- {"type":"quit"}
- {"type":"unknown"}

Rules:
- Output ONLY the JSON object, no other text.
- Use snake_case for item and enemy IDs (e.g. "short_sword", "goblin_0").
- If the input is ambiguous or unrecognizable, use {"type":"unknown"}.
- Direction must be one of: north, south, east, west, up, down.`

// validTypes is the set of action types the engine recognizes.
var validTypes = map[string]bool{
	"move": true, "look": true, "take": true, "drop": true,
	"equip": true, "unequip": true, "examine": true, "inventory": true,
	"attack": true, "defend": true, "flee": true, "cast": true,
	"help": true, "quit": true, "save": true, "load": true,
	"unknown": true,
}

// slmResponse is the JSON structure expected from the SLM.
type slmResponse struct {
	Type      string `json:"type"`
	Direction string `json:"direction,omitempty"`
	ItemID    string `json:"item_id,omitempty"`
	Target    string `json:"target,omitempty"`
	SpellID   string `json:"spell_id,omitempty"`
}

// Interpret sends the player's input to the SLM and parses the response
// into an EngineAction. Falls back to the Rules interpreter on any failure.
func (s *SLM) Interpret(ctx context.Context, input string, state model.GameState) (model.EngineAction, error) {
	temp := 0.0
	resp, err := s.client.ChatCompletion(ctx, []inference.Message{
		{Role: inference.RoleSystem, Content: systemPrompt},
		{Role: inference.RoleUser, Content: input},
	}, &inference.Options{Temperature: &temp, MaxTokens: 100})
	if err != nil {
		log.Printf("slm interpreter: inference error, falling back to rules: %v", err)
		return s.fallback.Interpret(ctx, input, state)
	}

	action, err := parseResponse(resp)
	if err != nil {
		log.Printf("slm interpreter: parse error, falling back to rules: %v", err)
		return s.fallback.Interpret(ctx, input, state)
	}

	return action, nil
}

// parseResponse extracts an EngineAction from the SLM's JSON response.
func parseResponse(resp string) (model.EngineAction, error) {
	// Strip markdown code fences if present (some models wrap JSON).
	resp = strings.TrimSpace(resp)
	if strings.HasPrefix(resp, "```") {
		resp = strings.TrimPrefix(resp, "```json")
		resp = strings.TrimPrefix(resp, "```")
		resp = strings.TrimSuffix(resp, "```")
		resp = strings.TrimSpace(resp)
	}

	var parsed slmResponse
	if err := json.Unmarshal([]byte(resp), &parsed); err != nil {
		return model.EngineAction{}, fmt.Errorf("unmarshal SLM response: %w", err)
	}

	if !validTypes[parsed.Type] {
		return model.EngineAction{}, fmt.Errorf("unknown action type from SLM: %q", parsed.Type)
	}

	return model.EngineAction{
		Type:      parsed.Type,
		Direction: parsed.Direction,
		ItemID:    parsed.ItemID,
		Target:    parsed.Target,
		SpellID:   parsed.SpellID,
	}, nil
}
