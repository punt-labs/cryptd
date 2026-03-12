package interpreter

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/punt-labs/cryptd/internal/inference"
	"github.com/punt-labs/cryptd/internal/model"
)

// BuildContext constructs the game-state context block injected into the SLM
// user message. Exported for use by the eval harness (cmd/eval-slm).
func BuildContext(state model.GameState) string {
	var b strings.Builder
	b.WriteString("Room: ")
	b.WriteString(state.Dungeon.CurrentRoom)
	b.WriteByte('\n')

	// Items visible in the current room.
	if rs, ok := state.Dungeon.RoomState[state.Dungeon.CurrentRoom]; ok && len(rs.Items) > 0 {
		b.WriteString("Items here: ")
		b.WriteString(strings.Join(rs.Items, ", "))
		b.WriteByte('\n')
	}

	// Available exits.
	if len(state.Dungeon.Exits) > 0 {
		b.WriteString("Exits: ")
		b.WriteString(strings.Join(state.Dungeon.Exits, ", "))
		b.WriteByte('\n')
	}

	// Active enemies.
	if state.Dungeon.Combat.Active {
		var names []string
		for _, e := range state.Dungeon.Combat.Enemies {
			if e.HP > 0 {
				names = append(names, e.ID)
			}
		}
		if len(names) > 0 {
			b.WriteString("Enemies: ")
			b.WriteString(strings.Join(names, ", "))
			b.WriteByte('\n')
		}
	}

	// Hero inventory and equipment.
	if len(state.Party) > 0 {
		hero := state.Party[0]
		if len(hero.Inventory) > 0 {
			var ids []string
			for _, item := range hero.Inventory {
				ids = append(ids, item.ID)
			}
			b.WriteString("Inventory: ")
			b.WriteString(strings.Join(ids, ", "))
			b.WriteByte('\n')
		}
		if hero.Equipped.Weapon != "" || hero.Equipped.Armor != "" {
			var slots []string
			if hero.Equipped.Weapon != "" {
				slots = append(slots, "weapon="+hero.Equipped.Weapon)
			}
			if hero.Equipped.Armor != "" {
				slots = append(slots, "armor="+hero.Equipped.Armor)
			}
			if hero.Equipped.Ring != "" {
				slots = append(slots, "ring="+hero.Equipped.Ring)
			}
			if hero.Equipped.Amulet != "" {
				slots = append(slots, "amulet="+hero.Equipped.Amulet)
			}
			b.WriteString("Equipped: ")
			b.WriteString(strings.Join(slots, ", "))
			b.WriteByte('\n')
		}
	}

	return b.String()
}

// SLM uses a small language model to interpret free-text player input into
// engine actions. On failure (network error, unparseable response, unknown
// action type), it falls back to the Rules interpreter.
type SLM struct {
	client   *inference.Client
	fallback model.CommandInterpreter
}

// NewSLM creates an SLM interpreter that sends player input to the given
// inference client and falls back to the provided interpreter on failure.
func NewSLM(client *inference.Client, fallback model.CommandInterpreter) *SLM {
	return &SLM{client: client, fallback: fallback}
}

// SystemPrompt instructs the SLM to produce structured JSON actions.
// Exported for use by the eval harness (cmd/eval-slm).
const SystemPrompt = `You are a text adventure command parser. Given a player's input, output ONLY a JSON object with the player's intended action.

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
- {"type":"cast","spell_id":"<spell>","target":"<enemy>"}  (target is optional)
- {"type":"save","target":"<slot_name>"}
- {"type":"load","target":"<slot_name>"}
- {"type":"help"}
- {"type":"quit"}
- {"type":"unknown"}

Rules:
- Output ONLY the JSON object, no other text.
- For item_id, use the exact snake_case ID from the "Items here" or "Inventory" context (e.g. if context says "short_sword", use "short_sword" not "sword").
- For target, use the exact enemy ID from the "Enemies" context (e.g. "goblin_0").
- Direction must be one of: north, south, east, west, up, down.
- If the input does not clearly match any supported action, output {"type":"unknown"}. Do not guess.

Examples:
"walk north" → {"type":"move","direction":"north"}
"head south" → {"type":"move","direction":"south"}
"climb up" → {"type":"move","direction":"up"}
"grab the key" → {"type":"take","item_id":"rusty_key"}
"inspect shield" → {"type":"examine","item_id":"shield"}
"check my bag" → {"type":"inventory"}
"xyzzy" → {"type":"unknown"}
"dance with the moon" → {"type":"unknown"}`

// validTypes is the set of action types the engine recognizes.
var validTypes = map[string]bool{
	"move": true, "look": true, "take": true, "drop": true,
	"equip": true, "unequip": true, "examine": true, "inventory": true,
	"attack": true, "defend": true, "flee": true, "cast": true,
	"help": true, "quit": true, "save": true, "load": true,
	"unknown": true,
}

// validDirections is the set of recognized movement directions.
var validDirections = map[string]bool{
	"north": true, "south": true, "east": true, "west": true,
	"up": true, "down": true,
}

// slmResponse is the JSON structure expected from the SLM.
type slmResponse struct {
	Type      string `json:"type"`
	Direction string `json:"direction,omitempty"`
	ItemID    string `json:"item_id,omitempty"`
	Target    string `json:"target,omitempty"`
	SpellID   string `json:"spell_id,omitempty"`
}

// needsIDResolution returns true if the action targets an item, enemy, or spell
// that the SLM might resolve better than rules (which just underscore-joins).
func needsIDResolution(a model.EngineAction) bool {
	switch a.Type {
	case "take", "drop", "equip", "examine":
		return a.ItemID != ""
	case "unequip":
		return a.Target != ""
	case "cast":
		return a.SpellID != ""
	case "attack":
		return a.Target != ""
	}
	return false
}

// Interpret tries the rules interpreter first for deterministic commands
// (aliases, exact verbs without targets). For actions that reference items,
// enemies, or spells by name, the SLM is called to resolve IDs using game
// state context. Context cancellation is always propagated.
func (s *SLM) Interpret(ctx context.Context, input string, state model.GameState) (model.EngineAction, error) {
	if err := ctx.Err(); err != nil {
		return model.EngineAction{}, err
	}

	// Rules-first: handle aliases and exact verbs without SLM latency.
	rulesAction, err := s.fallback.Interpret(ctx, input, state)
	if err != nil {
		return model.EngineAction{}, err
	}

	// If rules produced a definitive action without an ID to resolve,
	// use it directly (no SLM call needed).
	if rulesAction.Type != "unknown" && !needsIDResolution(rulesAction) {
		return rulesAction, nil
	}

	// Either rules returned "unknown" or it returned an action with an ID
	// the SLM might resolve better. Send to SLM with game state context.
	slmAction, err := s.callSLM(ctx, input, state)
	if err != nil {
		return rulesAction, nil
	}

	// If SLM returned a concrete action, prefer it. Otherwise fall back to rules.
	if slmAction.Type != "unknown" {
		return slmAction, nil
	}
	return rulesAction, nil
}

// callSLM sends the input to the SLM with game state context. Returns the
// parsed action or an error. Context cancellation is propagated.
func (s *SLM) callSLM(ctx context.Context, input string, state model.GameState) (model.EngineAction, error) {
	gameCtx := BuildContext(state)
	userMsg := gameCtx + "\nPlayer input: " + input

	temp := 0.0
	resp, err := s.client.ChatCompletion(ctx, []inference.Message{
		{Role: inference.RoleSystem, Content: SystemPrompt},
		{Role: inference.RoleUser, Content: userMsg},
	}, &inference.Options{Temperature: &temp, MaxTokens: 100})
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

// ParseSLMResponse extracts an EngineAction from the SLM's JSON response.
// Exported for use by the eval harness (cmd/eval-slm).
func ParseSLMResponse(resp string) (model.EngineAction, error) {
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

	if err := validateFields(parsed); err != nil {
		return model.EngineAction{}, err
	}

	return model.EngineAction{
		Type:      parsed.Type,
		Direction: parsed.Direction,
		ItemID:    parsed.ItemID,
		Target:    parsed.Target,
		SpellID:   parsed.SpellID,
	}, nil
}

// validateFields checks that required fields are present for each action type.
func validateFields(r slmResponse) error {
	switch r.Type {
	case "move":
		if !validDirections[r.Direction] {
			return fmt.Errorf("move requires valid direction, got %q", r.Direction)
		}
	case "take", "drop", "equip", "examine":
		if r.ItemID == "" {
			return fmt.Errorf("%s requires item_id", r.Type)
		}
	case "cast":
		if r.SpellID == "" {
			return fmt.Errorf("cast requires spell_id")
		}
	case "unequip":
		if r.Target == "" {
			return fmt.Errorf("unequip requires target")
		}
	}
	return nil
}
