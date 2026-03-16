package monkeytest

import (
	"math/rand"
	"testing"

	"github.com/punt-labs/cryptd/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestWeightedChoice_Distribution(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	m := &MonkeyRenderer{rng: rng}

	choices := []weightedCmd{
		{"attack", 70},
		{"defend", 20},
		{"flee", 10},
	}

	counts := map[string]int{}
	n := 10000
	for i := 0; i < n; i++ {
		cmd := m.weightedChoice(choices)
		counts[cmd]++
	}

	// Verify attack is ~70% (within 5% tolerance).
	attackPct := float64(counts["attack"]) / float64(n)
	assert.InDelta(t, 0.70, attackPct, 0.05, "attack should be ~70%%, got %.1f%%", attackPct*100)

	// Verify defend is ~20%.
	defendPct := float64(counts["defend"]) / float64(n)
	assert.InDelta(t, 0.20, defendPct, 0.05, "defend should be ~20%%, got %.1f%%", defendPct*100)

	// Verify flee is ~10%.
	fleePct := float64(counts["flee"]) / float64(n)
	assert.InDelta(t, 0.10, fleePct, 0.05, "flee should be ~10%%, got %.1f%%", fleePct*100)
}

func TestChooseCommand_InCombat_Healthy(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	m := &MonkeyRenderer{rng: rng}

	state := &model.GameState{
		Party: []model.Character{{
			HP: 20, MaxHP: 20, Class: "fighter",
			Stats: model.Stats{DEX: 12},
		}},
		Dungeon: model.DungeonState{
			Combat: model.CombatState{
				Active:  true,
				Enemies: []model.EnemyInstance{{ID: "goblin_0", HP: 8}},
			},
		},
	}

	counts := map[string]int{}
	n := 1000
	for i := 0; i < n; i++ {
		cmd := m.chooseCommand(state)
		counts[cmd]++
	}

	// Attack should dominate (~70%).
	assert.Greater(t, counts["attack"], n/2, "attack should be the most common action")
	// Flee should exist.
	assert.Greater(t, counts["flee"], 0, "flee should appear sometimes")
}

func TestChooseCommand_InCombat_LowHP(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	m := &MonkeyRenderer{rng: rng}

	state := &model.GameState{
		Party: []model.Character{{
			HP: 4, MaxHP: 20, Class: "fighter",
			Stats: model.Stats{DEX: 12},
		}},
		Dungeon: model.DungeonState{
			Combat: model.CombatState{
				Active:  true,
				Enemies: []model.EnemyInstance{{ID: "goblin_0", HP: 8}},
			},
		},
	}

	counts := map[string]int{}
	n := 1000
	for i := 0; i < n; i++ {
		cmd := m.chooseCommand(state)
		counts[cmd]++
	}

	// Flee should dominate (~40%).
	assert.Greater(t, counts["flee"], counts["attack"], "flee should be more common than attack when low HP")
}

func TestChooseCommand_EquipWeapon(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	m := &MonkeyRenderer{rng: rng}

	state := &model.GameState{
		Party: []model.Character{{
			HP: 20, MaxHP: 20, Class: "fighter",
			Inventory: []model.Item{{ID: "short_sword", Type: "weapon"}},
			Equipped:  model.Equipment{},
		}},
		Dungeon: model.DungeonState{
			CurrentRoom: "entrance",
			RoomState:   map[string]model.RoomState{"entrance": {}},
		},
	}

	cmd := m.chooseCommand(state)
	assert.Equal(t, "equip short_sword", cmd, "should immediately equip unequipped weapon")
}

func TestChooseCommand_ItemsInRoom(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	m := &MonkeyRenderer{rng: rng}

	state := &model.GameState{
		Party: []model.Character{{
			HP: 20, MaxHP: 20, Class: "fighter",
			Equipped: model.Equipment{Weapon: "short_sword"},
		}},
		Dungeon: model.DungeonState{
			CurrentRoom: "entrance",
			Exits:       []string{"south", "north"},
			RoomState: map[string]model.RoomState{
				"entrance": {Items: []string{"rusty_key"}},
			},
		},
	}

	counts := map[string]int{}
	n := 1000
	for i := 0; i < n; i++ {
		cmd := m.chooseCommand(state)
		counts[cmd]++
	}

	assert.Greater(t, counts["take rusty_key"], n/3, "take should be dominant when items are present")
}

func TestChooseCommand_CasterHeals(t *testing.T) {
	rng := rand.New(rand.NewSource(99))
	m := &MonkeyRenderer{rng: rng}

	state := &model.GameState{
		Party: []model.Character{{
			HP: 4, MaxHP: 20, MP: 10, MaxMP: 10, Class: "priest",
			Stats: model.Stats{DEX: 12},
		}},
		Dungeon: model.DungeonState{
			Combat: model.CombatState{
				Active:  true,
				Enemies: []model.EnemyInstance{{ID: "goblin_0", HP: 8}},
			},
		},
	}

	counts := map[string]int{}
	n := 1000
	for i := 0; i < n; i++ {
		cmd := m.chooseCommand(state)
		counts[cmd]++
	}

	assert.Greater(t, counts["cast heal"], 0, "caster with low HP and MP should sometimes heal")
}

func TestChooseCommand_ExploreWhenEmpty(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	m := &MonkeyRenderer{rng: rng}

	state := &model.GameState{
		Party: []model.Character{{
			HP: 20, MaxHP: 20, Class: "fighter",
			Equipped: model.Equipment{Weapon: "short_sword"},
		}},
		Dungeon: model.DungeonState{
			CurrentRoom: "entrance",
			Exits:       []string{"south"},
			RoomState:   map[string]model.RoomState{"entrance": {}},
		},
	}

	counts := map[string]int{}
	n := 1000
	for i := 0; i < n; i++ {
		cmd := m.chooseCommand(state)
		counts[cmd]++
	}

	// Movement should dominate (~80%).
	assert.Greater(t, counts["south"], n/2, "movement should dominate in empty rooms")
}
