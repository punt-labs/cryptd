//go:build integration

package game_test

import (
	"context"
	"testing"

	"github.com/punt-labs/cryptd/internal/engine"
	"github.com/punt-labs/cryptd/internal/game"
	"github.com/punt-labs/cryptd/internal/interpreter"
	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/narrator"
	"github.com/punt-labs/cryptd/internal/renderer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newLuxElementLoop(t *testing.T, display *renderer.LuxElementDisplay) (*game.Loop, model.GameState) {
	t.Helper()
	s := loadScenario(t)
	eng := engine.New(s)
	state := newState(t, eng)

	lux := renderer.NewLux(display)
	loop := game.NewLoop(eng, interpreter.NewRules(), narrator.NewTemplate(), lux)
	return loop, state
}

// TestLuxElement_BidirectionalRoundTrip proves the full Lux round trip:
// game state → element tree (outbound) and Lux interaction → game input (inbound).
func TestLuxElement_BidirectionalRoundTrip(t *testing.T) {
	display := renderer.NewLuxElementDisplay()

	// Inject a Lux button click: "act_south" → InputEvent{Type: "input", Payload: "south"}.
	display.InjectInteraction(map[string]any{
		"element_id": "act_south",
		"action":     "clicked",
	})
	// Clear the goblin in goblin_lair (8 attacks is enough).
	for range 8 {
		display.InjectEvent(model.InputEvent{Type: "input", Payload: "attack"})
	}
	display.InjectEvent(model.InputEvent{Type: "quit"})

	loop, state := newLuxElementLoop(t, display)
	require.NoError(t, loop.Run(context.Background(), &state))

	shows := display.Shows()
	require.NotEmpty(t, shows, "must have at least one show call")

	// --- Outbound assertions: first show has correct element structure ---
	first := shows[0]
	assert.Equal(t, "entrance", first.SceneID)

	// Find elements by ID.
	byID := elementsByID(first.Elements)

	header, ok := byID["room_header"]
	require.True(t, ok, "room_header element must exist")
	assert.Equal(t, "entrance", header["content"])

	heroHP, ok := byID["hero_0_hp"]
	require.True(t, ok, "hero_0_hp element must exist")
	assert.Equal(t, 1.0, heroHP["fraction"])

	_, hasActSouth := byID["act_south"]
	assert.True(t, hasActSouth, "act_south button must exist")

	narration, ok := byID["narration"]
	require.True(t, ok, "narration element must exist")
	assert.NotEmpty(t, narration["content"], "narration must have content")

	// --- Inbound assertion: button click caused movement ---
	var foundGoblinLair bool
	for _, show := range shows {
		if show.SceneID == "goblin_lair" {
			foundGoblinLair = true
			break
		}
	}
	assert.True(t, foundGoblinLair, "Lux button click should have moved player to goblin_lair")
}

// TestLuxElement_UpdatePatchStructure verifies that same-room actions produce
// update patches (not full show calls) targeting correct element IDs.
func TestLuxElement_UpdatePatchStructure(t *testing.T) {
	display := renderer.NewLuxElementDisplay()

	display.InjectEvent(model.InputEvent{Type: "input", Payload: "look"})
	display.InjectEvent(model.InputEvent{Type: "input", Payload: "look"})
	display.InjectEvent(model.InputEvent{Type: "quit"})

	loop, state := newLuxElementLoop(t, display)
	require.NoError(t, loop.Run(context.Background(), &state))

	shows := display.Shows()
	updates := display.Updates()

	// First call is show (initial render), subsequent same-room renders are updates.
	assert.Len(t, shows, 1, "only initial render should produce a show")
	require.NotEmpty(t, updates, "look commands should produce updates")

	// Verify first update patch targets narration.
	firstUpdate := updates[0]
	var hasNarrationPatch bool
	for _, patch := range firstUpdate {
		if patch["id"] == "narration" {
			hasNarrationPatch = true
			set, ok := patch["set"].(map[string]any)
			require.True(t, ok, "narration patch 'set' field must be map[string]any, got %T", patch["set"])
			assert.NotEmpty(t, set["content"], "narration patch must have content")
		}
	}
	assert.True(t, hasNarrationPatch, "update must contain narration patch")
}

// elementsByID indexes a flat element list by "id" field. Recurses into
// group children.
func elementsByID(elements []map[string]any) map[string]map[string]any {
	result := make(map[string]map[string]any)
	for _, el := range elements {
		if id, ok := el["id"].(string); ok {
			result[id] = el
		}
		if children, ok := el["children"].([]map[string]any); ok {
			for _, child := range children {
				if id, ok := child["id"].(string); ok {
					result[id] = child
				}
			}
		}
	}
	return result
}
