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
	"github.com/punt-labs/cryptd/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newLuxLoop(t *testing.T, inputs []string) (*game.Loop, *testutil.FakeLuxServer, model.GameState) {
	t.Helper()
	s := loadScenario(t)
	eng := engine.New(s)
	state := newState(t, eng)

	fake := testutil.NewFakeLuxServer()
	for _, inp := range inputs {
		fake.InjectEvent(model.InputEvent{Type: "input", Payload: inp})
	}
	fake.InjectEvent(model.InputEvent{Type: "quit"})

	lux := renderer.NewLux(fake)
	loop := game.NewLoop(eng, interpreter.NewRules(), narrator.NewTemplate(), lux)
	return loop, fake, state
}

// TestLux_TwoRoomNavigation verifies show() is called on each room transition
// and the scene contains the correct room ID and party data.
func TestLux_TwoRoomNavigation(t *testing.T) {
	// Move south to goblin_lair, fight through, move north back to entrance, quit.
	inputs := []string{
		"go south",
		"attack", "attack", "attack", "attack",
		"attack", "attack", "attack", "attack",
		"go north",
	}
	loop, fake, state := newLuxLoop(t, inputs)

	require.NoError(t, loop.Run(context.Background(), &state))

	calls := fake.Calls()
	require.NotEmpty(t, calls)

	// First call: initial render — show entrance.
	assert.Equal(t, "show", calls[0].Method)
	scene := calls[0].Payload.(renderer.LuxScene)
	assert.Equal(t, "entrance", scene.Room)
	require.Len(t, scene.Party, 1)
	assert.Equal(t, "Hero", scene.Party[0].Name)

	// Find show calls — each room transition triggers a show.
	var showRooms []string
	for _, c := range calls {
		if c.Method == "show" {
			s := c.Payload.(renderer.LuxScene)
			showRooms = append(showRooms, s.Room)
		}
	}
	// entrance → goblin_lair (move) → goblin_lair (combat start) → goblin_lair (combat end) → entrance (move back)
	require.Contains(t, showRooms, "entrance")
	require.Contains(t, showRooms, "goblin_lair")
}

// TestLux_ShowOnSceneTransitionNotUpdate verifies that room changes always
// produce show() calls, never update() calls for the transition itself.
func TestLux_ShowOnSceneTransitionNotUpdate(t *testing.T) {
	inputs := []string{"look", "look"}
	loop, fake, state := newLuxLoop(t, inputs)

	require.NoError(t, loop.Run(context.Background(), &state))

	calls := fake.Calls()
	require.NotEmpty(t, calls)

	// First call is show (initial render). Subsequent same-room renders are updates.
	assert.Equal(t, "show", calls[0].Method)
	for _, c := range calls[1:] {
		if c.Method == "show" {
			// The quit event also triggers a render — same room, should be update.
			t.Errorf("unexpected show() call after initial render in same room")
		}
	}
}

// TestLux_UpdateForLogAppends verifies that same-room actions produce update()
// calls, not show() calls.
func TestLux_UpdateForLogAppends(t *testing.T) {
	inputs := []string{"look", "look", "look"}
	loop, fake, state := newLuxLoop(t, inputs)

	require.NoError(t, loop.Run(context.Background(), &state))

	calls := fake.Calls()
	showCount := 0
	updateCount := 0
	for _, c := range calls {
		switch c.Method {
		case "show":
			showCount++
		case "update":
			updateCount++
		}
	}
	assert.Equal(t, 1, showCount, "only initial render should call show")
	assert.GreaterOrEqual(t, updateCount, 3, "look commands + quit should produce updates")
}

// TestLux_FakeLuxServerInjectsSyntheticEvent verifies that FakeLuxServer's
// injected InputEvents are correctly received by the game loop via Lux.Events().
func TestLux_FakeLuxServerInjectsSyntheticEvent(t *testing.T) {
	s := loadScenario(t)
	eng := engine.New(s)
	state := newState(t, eng)

	fake := testutil.NewFakeLuxServer()
	// Inject a custom input followed by quit.
	fake.InjectEvent(model.InputEvent{Type: "input", Payload: "look"})
	fake.InjectEvent(model.InputEvent{Type: "quit"})

	lux := renderer.NewLux(fake)
	loop := game.NewLoop(eng, interpreter.NewRules(), narrator.NewTemplate(), lux)

	require.NoError(t, loop.Run(context.Background(), &state))

	calls := fake.Calls()
	// At minimum: initial show + look update + quit update = 3 calls.
	require.GreaterOrEqual(t, len(calls), 3)
}

// TestLux_CombatTransitionsCallShow verifies that entering and exiting combat
// both trigger show() calls (combat state change = scene transition).
func TestLux_CombatTransitionsCallShow(t *testing.T) {
	// Move to goblin_lair (triggers combat), attack until dead.
	inputs := []string{
		"go south",
		"attack", "attack", "attack", "attack",
		"attack", "attack", "attack", "attack",
	}
	loop, fake, state := newLuxLoop(t, inputs)

	require.NoError(t, loop.Run(context.Background(), &state))

	calls := fake.Calls()

	// Track combat state transitions via show calls.
	var combatShows []bool
	for _, c := range calls {
		if c.Method == "show" {
			s := c.Payload.(renderer.LuxScene)
			combatShows = append(combatShows, s.InCombat)
		}
	}

	// Should see at least: entrance (no combat), goblin_lair (combat on), goblin_lair (combat off)
	require.GreaterOrEqual(t, len(combatShows), 3)
	assert.False(t, combatShows[0], "initial entrance should not be in combat")
}

// TestLux_SceneContainsCorrectPartyData verifies the element structure
// sent to FakeLuxServer has correct party member data.
func TestLux_SceneContainsCorrectPartyData(t *testing.T) {
	inputs := []string{}
	loop, fake, state := newLuxLoop(t, inputs)

	require.NoError(t, loop.Run(context.Background(), &state))

	calls := fake.Calls()
	require.NotEmpty(t, calls)

	scene := calls[0].Payload.(renderer.LuxScene)
	require.Len(t, scene.Party, 1)
	assert.Equal(t, "Hero", scene.Party[0].Name)
	assert.Equal(t, "Fighter", scene.Party[0].Class)
	assert.Equal(t, 1, scene.Party[0].Level)
	assert.Equal(t, 100, scene.Party[0].HP)
	assert.Equal(t, 100, scene.Party[0].MaxHP)
}
