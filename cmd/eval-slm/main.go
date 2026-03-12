// Command eval-slm runs the SLM interpreter verb table through a real
// inference server and reports classification accuracy. This is a developer
// tool — not CI — intended for evaluating model quality before shipping.
//
// Usage:
//
//	go run ./cmd/eval-slm [--server <url>] [--model <name>] [--timeout <dur>] [--verbose]
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/punt-labs/cryptd/internal/inference"
	"github.com/punt-labs/cryptd/internal/interpreter"
	"github.com/punt-labs/cryptd/internal/model"
)

// testCase defines a natural-language input and the expected action.
type testCase struct {
	Input    string
	Expected model.EngineAction
}

// verbTable is the comprehensive set of inputs the SLM must classify correctly.
// Each entry maps a natural-language phrase to the expected engine action.
var verbTable = []testCase{
	// Movement — canonical and natural
	{Input: "go north", Expected: model.EngineAction{Type: "move", Direction: "north"}},
	{Input: "go south", Expected: model.EngineAction{Type: "move", Direction: "south"}},
	{Input: "go east", Expected: model.EngineAction{Type: "move", Direction: "east"}},
	{Input: "go west", Expected: model.EngineAction{Type: "move", Direction: "west"}},
	{Input: "north", Expected: model.EngineAction{Type: "move", Direction: "north"}},
	{Input: "n", Expected: model.EngineAction{Type: "move", Direction: "north"}},
	{Input: "s", Expected: model.EngineAction{Type: "move", Direction: "south"}},
	{Input: "e", Expected: model.EngineAction{Type: "move", Direction: "east"}},
	{Input: "w", Expected: model.EngineAction{Type: "move", Direction: "west"}},
	{Input: "walk north", Expected: model.EngineAction{Type: "move", Direction: "north"}},
	{Input: "head south", Expected: model.EngineAction{Type: "move", Direction: "south"}},
	{Input: "go up", Expected: model.EngineAction{Type: "move", Direction: "up"}},
	{Input: "go down", Expected: model.EngineAction{Type: "move", Direction: "down"}},
	{Input: "climb up", Expected: model.EngineAction{Type: "move", Direction: "up"}},
	{Input: "descend", Expected: model.EngineAction{Type: "move", Direction: "down"}},

	// Look
	{Input: "look", Expected: model.EngineAction{Type: "look"}},
	{Input: "look around", Expected: model.EngineAction{Type: "look"}},
	{Input: "l", Expected: model.EngineAction{Type: "look"}},
	{Input: "examine room", Expected: model.EngineAction{Type: "look"}},

	// Take / pick up
	{Input: "take sword", Expected: model.EngineAction{Type: "take", ItemID: "sword"}},
	{Input: "pick up the rusty key", Expected: model.EngineAction{Type: "take", ItemID: "rusty_key"}},
	{Input: "get potion", Expected: model.EngineAction{Type: "take", ItemID: "potion"}},
	{Input: "grab torch", Expected: model.EngineAction{Type: "take", ItemID: "torch"}},

	// Drop
	{Input: "drop sword", Expected: model.EngineAction{Type: "drop", ItemID: "sword"}},
	{Input: "drop the rusty key", Expected: model.EngineAction{Type: "drop", ItemID: "rusty_key"}},

	// Equip / unequip
	{Input: "equip sword", Expected: model.EngineAction{Type: "equip", ItemID: "sword"}},
	{Input: "wield short sword", Expected: model.EngineAction{Type: "equip", ItemID: "short_sword"}},
	{Input: "wear armor", Expected: model.EngineAction{Type: "equip", ItemID: "armor"}},
	{Input: "unequip weapon", Expected: model.EngineAction{Type: "unequip", Target: "weapon"}},
	{Input: "remove armor", Expected: model.EngineAction{Type: "unequip", Target: "armor"}},

	// Examine
	{Input: "examine sword", Expected: model.EngineAction{Type: "examine", ItemID: "sword"}},
	{Input: "x key", Expected: model.EngineAction{Type: "examine", ItemID: "key"}},
	{Input: "look at the potion", Expected: model.EngineAction{Type: "examine", ItemID: "potion"}},
	{Input: "inspect shield", Expected: model.EngineAction{Type: "examine", ItemID: "shield"}},

	// Inventory
	{Input: "inventory", Expected: model.EngineAction{Type: "inventory"}},
	{Input: "i", Expected: model.EngineAction{Type: "inventory"}},
	{Input: "check my bag", Expected: model.EngineAction{Type: "inventory"}},

	// Combat — attack
	{Input: "attack goblin", Expected: model.EngineAction{Type: "attack"}},
	{Input: "hit the skeleton", Expected: model.EngineAction{Type: "attack"}},
	{Input: "strike goblin", Expected: model.EngineAction{Type: "attack"}},
	{Input: "attack", Expected: model.EngineAction{Type: "attack"}},
	{Input: "a", Expected: model.EngineAction{Type: "attack"}},

	// Combat — defend
	{Input: "defend", Expected: model.EngineAction{Type: "defend"}},
	{Input: "block", Expected: model.EngineAction{Type: "defend"}},
	{Input: "guard", Expected: model.EngineAction{Type: "defend"}},

	// Combat — flee
	{Input: "flee", Expected: model.EngineAction{Type: "flee"}},
	{Input: "run away", Expected: model.EngineAction{Type: "flee"}},
	{Input: "escape", Expected: model.EngineAction{Type: "flee"}},

	// Spells
	{Input: "cast fireball", Expected: model.EngineAction{Type: "cast", SpellID: "fireball"}},
	{Input: "cast fireball at goblin", Expected: model.EngineAction{Type: "cast", SpellID: "fireball"}},
	{Input: "cast heal", Expected: model.EngineAction{Type: "cast", SpellID: "heal"}},

	// Save / load
	{Input: "save", Expected: model.EngineAction{Type: "save"}},
	{Input: "save game", Expected: model.EngineAction{Type: "save"}},
	{Input: "save quicksave", Expected: model.EngineAction{Type: "save", Target: "quicksave"}},
	{Input: "load", Expected: model.EngineAction{Type: "load"}},
	{Input: "load quicksave", Expected: model.EngineAction{Type: "load", Target: "quicksave"}},

	// Help / quit
	{Input: "help", Expected: model.EngineAction{Type: "help"}},
	{Input: "?", Expected: model.EngineAction{Type: "help"}},
	{Input: "quit", Expected: model.EngineAction{Type: "quit"}},
	{Input: "exit", Expected: model.EngineAction{Type: "quit"}},

	// Unknown / nonsense — should classify as unknown
	{Input: "dance with the moon", Expected: model.EngineAction{Type: "unknown"}},
	{Input: "xyzzy", Expected: model.EngineAction{Type: "unknown"}},
	{Input: "tell me a joke", Expected: model.EngineAction{Type: "unknown"}},
}

func main() {
	serverURL := flag.String("server", "", "inference server URL (auto-detected if omitted)")
	modelName := flag.String("model", "", "model name (auto-detected if omitted)")
	timeout := flag.Duration("timeout", 30*time.Second, "per-request timeout")
	verbose := flag.Bool("verbose", false, "print each test case result")
	flag.Parse()

	base, mdl := resolveRuntime(*serverURL, *modelName)
	if base == "" || mdl == "" {
		fmt.Fprintln(os.Stderr, "error: no inference server found — provide --server and --model, or start an inference server (e.g., ollama or llama.cpp)")
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "Evaluating %s (model: %s)\n", base, mdl)
	fmt.Fprintf(os.Stderr, "Running %d test cases...\n\n", len(verbTable))

	client := inference.NewClient(base, mdl, *timeout)
	ctx := context.Background()

	var passed, failed, errors int
	var failures []string

	for i, tc := range verbTable {
		temp := 0.0
		resp, err := client.ChatCompletion(ctx, []inference.Message{
			{Role: inference.RoleSystem, Content: interpreter.SystemPrompt},
			{Role: inference.RoleUser, Content: tc.Input},
		}, &inference.Options{Temperature: &temp, MaxTokens: 100})

		if err != nil {
			errors++
			if *verbose {
				fmt.Printf("[%3d] ERROR  %q → %v\n", i+1, tc.Input, err)
			}
			failures = append(failures, fmt.Sprintf("  ERROR  %q → %v", tc.Input, err))
			continue
		}

		got, err := interpreter.ParseSLMResponse(resp)
		if err != nil {
			errors++
			if *verbose {
				fmt.Printf("[%3d] PARSE  %q → %v (raw: %s)\n", i+1, tc.Input, err, truncate(resp, 80))
			}
			failures = append(failures, fmt.Sprintf("  PARSE  %q → %v", tc.Input, err))
			continue
		}

		if matchAction(got, tc.Expected) {
			passed++
			if *verbose {
				fmt.Printf("[%3d] PASS   %q → %s\n", i+1, tc.Input, actionString(got))
			}
		} else {
			failed++
			if *verbose {
				fmt.Printf("[%3d] FAIL   %q → got %s, want %s\n", i+1, tc.Input, actionString(got), actionString(tc.Expected))
			}
			failures = append(failures, fmt.Sprintf("  FAIL   %q → got %s, want %s", tc.Input, actionString(got), actionString(tc.Expected)))
		}
	}

	total := len(verbTable)
	accuracy := float64(passed) / float64(total) * 100

	fmt.Printf("\n── Results ─────────────────────────────────\n")
	fmt.Printf("  Model:    %s\n", mdl)
	fmt.Printf("  Total:    %d\n", total)
	fmt.Printf("  Passed:   %d\n", passed)
	fmt.Printf("  Failed:   %d\n", failed)
	fmt.Printf("  Errors:   %d\n", errors)
	fmt.Printf("  Accuracy: %.1f%%\n", accuracy)
	fmt.Printf("────────────────────────────────────────────\n")

	if len(failures) > 0 {
		fmt.Printf("\nFailures:\n")
		for _, f := range failures {
			fmt.Println(f)
		}
	}

	if errors > 0 {
		os.Exit(2)
	}
	if accuracy < 80 {
		os.Exit(1)
	}
}

// matchAction compares two actions, matching only type and the fields relevant
// to that type. Empty expected fields are treated as "don't care".
func matchAction(got, want model.EngineAction) bool {
	if got.Type != want.Type {
		return false
	}
	if want.Direction != "" && got.Direction != want.Direction {
		return false
	}
	if want.ItemID != "" && got.ItemID != want.ItemID {
		return false
	}
	if want.Target != "" && got.Target != want.Target {
		return false
	}
	if want.SpellID != "" && got.SpellID != want.SpellID {
		return false
	}
	return true
}

func actionString(a model.EngineAction) string {
	parts := []string{a.Type}
	if a.Direction != "" {
		parts = append(parts, "dir="+a.Direction)
	}
	if a.ItemID != "" {
		parts = append(parts, "item="+a.ItemID)
	}
	if a.Target != "" {
		parts = append(parts, "target="+a.Target)
	}
	if a.SpellID != "" {
		parts = append(parts, "spell="+a.SpellID)
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// resolveRuntime finds an inference server via flags or auto-detection.
func resolveRuntime(serverURL, modelName string) (string, string) {
	if serverURL != "" && modelName != "" {
		return serverURL, modelName
	}

	var endpoints []inference.Endpoint
	if serverURL != "" {
		endpoints = []inference.Endpoint{
			{Name: "user-server", BaseURL: serverURL, HealthPath: "/api/tags", ModelExtractor: inference.OllamaModels},
			{Name: "user-server", BaseURL: serverURL, HealthPath: "/v1/models", ModelExtractor: inference.OpenAIModels},
		}
	} else {
		endpoints = inference.DefaultEndpoints()
	}

	rt := inference.Probe(context.Background(), endpoints, time.Second)
	if rt == nil {
		return "", ""
	}

	if serverURL == "" {
		serverURL = rt.BaseURL
	}
	if modelName == "" {
		modelName = rt.Model
	}
	return serverURL, modelName
}
