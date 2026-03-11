// Command dump-mcp-schema writes the MCP tool schema to stdout as JSON.
// CI diffs this output against testdata/mcp-schema.json to catch
// unintentional API changes.
package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// ToolParam describes one parameter of an MCP tool.
type ToolParam struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// Tool describes one MCP tool in the schema.
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Params      []ToolParam `json:"params"`
}

// Schema is the top-level MCP schema document.
type Schema struct {
	Version string `json:"version"`
	Tools   []Tool `json:"tools"`
}

func main() {
	schema := Schema{
		Version: "0.1.0",
		Tools: []Tool{
			{
				Name:        "new_game",
				Description: "Start a new game with the given scenario and character.",
				Params: []ToolParam{
					{Name: "scenario_id", Type: "string", Description: "Scenario identifier", Required: true},
					{Name: "character_name", Type: "string", Description: "Hero name", Required: true},
					{Name: "character_class", Type: "string", Description: "Hero class: fighter, mage, priest, thief", Required: true},
				},
			},
			{
				Name:        "move",
				Description: "Move the hero in a direction.",
				Params: []ToolParam{
					{Name: "direction", Type: "string", Description: "Direction: north, south, east, west, up, down", Required: true},
				},
			},
			{
				Name:        "look",
				Description: "Describe the current room.",
				Params:      []ToolParam{},
			},
			{
				Name:        "pick_up",
				Description: "Pick up an item from the current room.",
				Params: []ToolParam{
					{Name: "item_id", Type: "string", Description: "Item identifier", Required: true},
				},
			},
			{
				Name:        "drop",
				Description: "Drop an item from inventory into the current room.",
				Params: []ToolParam{
					{Name: "item_id", Type: "string", Description: "Item identifier", Required: true},
				},
			},
			{
				Name:        "equip",
				Description: "Equip an item from inventory.",
				Params: []ToolParam{
					{Name: "item_id", Type: "string", Description: "Item identifier", Required: true},
				},
			},
			{
				Name:        "unequip",
				Description: "Unequip an item from an equipment slot.",
				Params: []ToolParam{
					{Name: "slot", Type: "string", Description: "Equipment slot: weapon, armor, ring, amulet", Required: true},
				},
			},
			{
				Name:        "examine",
				Description: "Examine an item in inventory, equipped, or in the room.",
				Params: []ToolParam{
					{Name: "item_id", Type: "string", Description: "Item identifier", Required: true},
				},
			},
			{
				Name:        "inventory",
				Description: "List the hero's inventory and equipment.",
				Params:      []ToolParam{},
			},
			{
				Name:        "attack",
				Description: "Attack an enemy in combat.",
				Params: []ToolParam{
					{Name: "target_id", Type: "string", Description: "Enemy instance ID (default: first alive)", Required: false},
				},
			},
			{
				Name:        "defend",
				Description: "Raise guard to halve incoming damage for one round.",
				Params:      []ToolParam{},
			},
			{
				Name:        "flee",
				Description: "Attempt to flee from combat (DEX check).",
				Params:      []ToolParam{},
			},
			{
				Name:        "cast_spell",
				Description: "Cast a spell. Damage spells require combat; heal works anytime.",
				Params: []ToolParam{
					{Name: "spell_id", Type: "string", Description: "Spell identifier", Required: true},
					{Name: "target_id", Type: "string", Description: "Target enemy ID (for damage spells)", Required: false},
				},
			},
			{
				Name:        "save_game",
				Description: "Save the current game state to a named slot.",
				Params: []ToolParam{
					{Name: "slot", Type: "string", Description: "Save slot name (default: quicksave)", Required: false},
				},
			},
			{
				Name:        "load_game",
				Description: "Load a saved game state from a named slot.",
				Params: []ToolParam{
					{Name: "slot", Type: "string", Description: "Save slot name (default: quicksave)", Required: false},
				},
			},
		},
	}

	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal schema: %v\n", err)
		os.Exit(1)
	}
	data = append(data, '\n')
	if _, err := os.Stdout.Write(data); err != nil {
		fmt.Fprintf(os.Stderr, "write schema: %v\n", err)
		os.Exit(1)
	}
}
