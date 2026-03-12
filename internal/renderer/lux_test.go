package renderer_test

import (
	"context"
	"strings"
	"testing"

	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/renderer"
	"github.com/punt-labs/cryptd/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newLuxRenderer() (*renderer.Lux, *testutil.FakeLuxServer) {
	fake := testutil.NewFakeLuxServer()
	return renderer.NewLux(fake), fake
}

func baseState(room string) model.GameState {
	return model.GameState{
		Party: []model.Character{{
			Name: "Adventurer", Class: "fighter", Level: 1,
			HP: 20, MaxHP: 20,
		}},
		Dungeon: model.DungeonState{CurrentRoom: room},
	}
}

func TestLux_InitialRenderCallsShow(t *testing.T) {
	lux, fake := newLuxRenderer()

	state := baseState("entrance")
	err := lux.Render(context.Background(), state, "You stand in a dark corridor.")
	require.NoError(t, err)

	calls := fake.Calls()
	require.Len(t, calls, 1)
	assert.Equal(t, "show", calls[0].Method)

	scene := calls[0].Payload.(renderer.LuxScene)
	assert.Equal(t, "entrance", scene.Room)
	assert.Equal(t, "You stand in a dark corridor.", scene.Narration)
	assert.False(t, scene.InCombat)
	require.Len(t, scene.Party, 1)
	assert.Equal(t, "Adventurer", scene.Party[0].Name)
	assert.Equal(t, "Fighter", scene.Party[0].Class)
	assert.Equal(t, 20, scene.Party[0].HP)
}

func TestLux_RoomChangeCallsShow(t *testing.T) {
	lux, fake := newLuxRenderer()

	state := baseState("entrance")
	_ = lux.Render(context.Background(), state, "You enter the entrance.")

	state.Dungeon.CurrentRoom = "goblin_lair"
	_ = lux.Render(context.Background(), state, "A foul stench fills the air.")

	calls := fake.Calls()
	require.Len(t, calls, 2)
	assert.Equal(t, "show", calls[0].Method)
	assert.Equal(t, "show", calls[1].Method)

	scene := calls[1].Payload.(renderer.LuxScene)
	assert.Equal(t, "goblin_lair", scene.Room)
}

func TestLux_SameRoomCallsUpdate(t *testing.T) {
	lux, fake := newLuxRenderer()

	state := baseState("entrance")
	_ = lux.Render(context.Background(), state, "You enter the entrance.")
	_ = lux.Render(context.Background(), state, "You look around.")

	calls := fake.Calls()
	require.Len(t, calls, 2)
	assert.Equal(t, "show", calls[0].Method)
	assert.Equal(t, "update", calls[1].Method, "same room must use update, not show")

	update := calls[1].Payload.(renderer.LuxUpdate)
	assert.Equal(t, "narration", update.Type)
	assert.Equal(t, "You look around.", update.Content)
}

func TestLux_CombatStartCallsShow(t *testing.T) {
	lux, fake := newLuxRenderer()

	state := baseState("goblin_lair")
	_ = lux.Render(context.Background(), state, "You enter the lair.")

	// Combat starts — same room but combat state changed.
	state.Dungeon.Combat = model.CombatState{
		Active: true,
		Enemies: []model.EnemyInstance{
			{Name: "Goblin", HP: 8, MaxHP: 8},
		},
	}
	_ = lux.Render(context.Background(), state, "A goblin attacks!")

	calls := fake.Calls()
	require.Len(t, calls, 2)
	assert.Equal(t, "show", calls[0].Method)
	assert.Equal(t, "show", calls[1].Method, "combat state change must trigger show")

	scene := calls[1].Payload.(renderer.LuxScene)
	assert.True(t, scene.InCombat)
	require.Len(t, scene.Enemies, 1)
	assert.Equal(t, "Goblin", scene.Enemies[0].Name)
	assert.Equal(t, 8, scene.Enemies[0].HP)
}

func TestLux_CombatEndCallsShow(t *testing.T) {
	lux, fake := newLuxRenderer()

	state := baseState("goblin_lair")
	state.Dungeon.Combat = model.CombatState{Active: true}
	_ = lux.Render(context.Background(), state, "Combat begins!")

	// Combat ends.
	state.Dungeon.Combat = model.CombatState{Active: false}
	_ = lux.Render(context.Background(), state, "Victory!")

	calls := fake.Calls()
	require.Len(t, calls, 2)
	assert.Equal(t, "show", calls[0].Method)
	assert.Equal(t, "show", calls[1].Method, "combat end must trigger show")
}

func TestLux_DamageDuringCombatCallsUpdate(t *testing.T) {
	lux, fake := newLuxRenderer()

	state := baseState("goblin_lair")
	state.Dungeon.Combat = model.CombatState{
		Active: true,
		Enemies: []model.EnemyInstance{
			{Name: "Goblin", HP: 8, MaxHP: 8},
		},
	}
	_ = lux.Render(context.Background(), state, "Combat begins!")

	// Damage tick — same room, same combat state.
	state.Party[0].HP = 15
	state.Dungeon.Combat.Enemies[0].HP = 5
	_ = lux.Render(context.Background(), state, "You strike the goblin for 3 damage.")

	calls := fake.Calls()
	require.Len(t, calls, 2)
	assert.Equal(t, "show", calls[0].Method)
	assert.Equal(t, "update", calls[1].Method, "damage tick must use update, not show (performance red line)")

	update := calls[1].Payload.(renderer.LuxUpdate)
	require.NotNil(t, update.Hero)
	assert.Equal(t, 15, update.Hero.HP)
	require.Len(t, update.Enemies, 1)
	assert.Equal(t, 5, update.Enemies[0].HP)
}

func TestLux_DeadEnemiesHidden(t *testing.T) {
	lux, fake := newLuxRenderer()

	state := baseState("goblin_lair")
	state.Dungeon.Combat = model.CombatState{
		Active: true,
		Enemies: []model.EnemyInstance{
			{Name: "Goblin", HP: 0, MaxHP: 8},
			{Name: "Skeleton", HP: 6, MaxHP: 6},
		},
	}
	_ = lux.Render(context.Background(), state, "The goblin falls.")

	scene := fake.Calls()[0].Payload.(renderer.LuxScene)
	require.Len(t, scene.Enemies, 1)
	assert.Equal(t, "Skeleton", scene.Enemies[0].Name)
}

func TestLux_MPShownForCasters(t *testing.T) {
	lux, fake := newLuxRenderer()

	state := baseState("entrance")
	state.Party[0].Class = "mage"
	state.Party[0].MP = 5
	state.Party[0].MaxMP = 10
	_ = lux.Render(context.Background(), state, "You arrive.")

	scene := fake.Calls()[0].Payload.(renderer.LuxScene)
	assert.Equal(t, 5, scene.Party[0].MP)
	assert.Equal(t, 10, scene.Party[0].MaxMP)
}

func TestLux_EventsFromDisplay(t *testing.T) {
	lux, fake := newLuxRenderer()

	fake.InjectEvent(model.InputEvent{Type: "input", Payload: "go north"})
	fake.InjectEvent(model.InputEvent{Type: "quit"})

	events := lux.Events()
	ev1 := <-events
	assert.Equal(t, "input", ev1.Type)
	assert.Equal(t, "go north", ev1.Payload)

	ev2 := <-events
	assert.Equal(t, "quit", ev2.Type)
}

func TestLux_LogEntriesInScene(t *testing.T) {
	lux, fake := newLuxRenderer()

	state := baseState("entrance")
	state.AdventureLog = []model.LogEntry{
		{Text: "Game started."},
		{Text: "Moved to entrance."},
		{Text: "Picked up key."},
	}
	_ = lux.Render(context.Background(), state, "Welcome.")

	scene := fake.Calls()[0].Payload.(renderer.LuxScene)
	require.Len(t, scene.Log, 3)
	assert.Equal(t, "Game started.", scene.Log[0])
}

func TestLux_LogTruncatedToFive(t *testing.T) {
	lux, fake := newLuxRenderer()

	state := baseState("entrance")
	for i := range 8 {
		state.AdventureLog = append(state.AdventureLog, model.LogEntry{
			Text: strings.Repeat("x", i+1),
		})
	}
	_ = lux.Render(context.Background(), state, "Welcome.")

	scene := fake.Calls()[0].Payload.(renderer.LuxScene)
	require.Len(t, scene.Log, 5, "log should be truncated to last 5 entries")
	assert.Equal(t, "xxxx", scene.Log[0]) // entries 4-8 (0-indexed 3-7)
}

// Performance red line: multiple renders in the same room without combat
// changes must NEVER call show(). This test guards against regressions.
func TestLux_PerformanceRedLine_NoShowForIncrementalUpdates(t *testing.T) {
	lux, fake := newLuxRenderer()

	state := baseState("entrance")
	_ = lux.Render(context.Background(), state, "Initial.")

	// 10 consecutive renders in the same room.
	for i := range 10 {
		state.Party[0].HP = 20 - i
		_ = lux.Render(context.Background(), state, "Something happens.")
	}

	calls := fake.Calls()
	require.Len(t, calls, 11)

	showCount := 0
	for _, c := range calls {
		if c.Method == "show" {
			showCount++
		}
	}
	assert.Equal(t, 1, showCount, "only the initial render should call show; all others must be update")
}
