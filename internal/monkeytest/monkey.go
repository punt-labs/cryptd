package monkeytest

import (
	"context"
	"math/rand"

	"github.com/punt-labs/cryptd/internal/model"
)

// weightedCmd pairs a command string with a relative weight.
type weightedCmd struct {
	cmd    string
	weight int
}

// MonkeyRenderer implements model.Renderer. It examines GameState on each
// Render() call, picks a weighted-random command, and feeds it back via Events().
// Metrics are tracked by diffing consecutive states.
type MonkeyRenderer struct {
	rng       *rand.Rand
	maxMoves  int
	moveCount int
	events    chan model.InputEvent
	prev      *model.GameState // nil before first Render
	metrics   SessionMetrics
	lastCmd   string
	done      bool
}

// NewMonkey creates a MonkeyRenderer.
func NewMonkey(rng *rand.Rand, maxMoves int, class string, seed int64) *MonkeyRenderer {
	return &MonkeyRenderer{
		rng:      rng,
		maxMoves: maxMoves,
		events:   make(chan model.InputEvent, 1),
		metrics:  SessionMetrics{Seed: seed, Class: class, Survived: true},
	}
}

// Render receives the current game state and queues the next command.
func (m *MonkeyRenderer) Render(ctx context.Context, state model.GameState, _ string) error {
	if m.done {
		return nil
	}

	m.updateMetrics(&state)

	hero := state.Party[0]

	// End conditions: max moves reached or hero dead.
	if m.moveCount >= m.maxMoves || hero.HP <= 0 {
		m.done = true
		if hero.HP <= 0 {
			m.metrics.Survived = false
		}
		select {
		case m.events <- model.InputEvent{Type: "quit"}:
		case <-ctx.Done():
			return ctx.Err()
		}
		return nil
	}

	cmd := m.chooseCommand(&state)
	m.lastCmd = cmd
	m.moveCount++
	m.metrics.TotalMoves = m.moveCount

	select {
	case m.events <- model.InputEvent{Type: "input", Payload: cmd}:
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}

// Events returns the input event channel.
func (m *MonkeyRenderer) Events() <-chan model.InputEvent {
	return m.events
}

// Metrics returns the collected session metrics. Call after Run() completes.
func (m *MonkeyRenderer) Metrics() SessionMetrics {
	return m.metrics
}

// updateMetrics diffs the current state against the previous to track changes.
func (m *MonkeyRenderer) updateMetrics(state *model.GameState) {
	hero := &state.Party[0]

	// Always-current values.
	m.metrics.RoomsVisited = len(state.Dungeon.VisitedRooms)
	m.metrics.XPGained = hero.XP
	m.metrics.FinalLevel = hero.Level
	m.metrics.FinalHP = hero.HP
	m.metrics.FinalMaxHP = hero.MaxHP

	if m.prev == nil {
		m.prev = copyState(state)
		return
	}

	prevHero := &m.prev.Party[0]

	// Damage taken: HP decreased (ignore healing which increases HP).
	if hero.HP < prevHero.HP {
		m.metrics.DamageTaken += prevHero.HP - hero.HP
	}

	// Damage dealt: sum enemy HP reductions.
	if m.prev.Dungeon.Combat.Active && state.Dungeon.Combat.Active {
		m.metrics.DamageDealt += enemyHPDelta(m.prev.Dungeon.Combat.Enemies, state.Dungeon.Combat.Enemies)
	}

	// Enemies killed: count enemies that went from HP>0 to HP≤0.
	if m.prev.Dungeon.Combat.Active {
		m.metrics.EnemiesKilled += countNewKills(m.prev.Dungeon.Combat.Enemies, state.Dungeon.Combat.Enemies)
		// Count kills that ended combat (enemies array cleared on victory).
		// Only count if room was cleared (victory), not on flee.
		if !state.Dungeon.Combat.Active {
			room := state.Dungeon.CurrentRoom
			rs := state.Dungeon.RoomState[room]
			if rs.Cleared {
				m.metrics.EnemiesKilled += countAliveEnemies(m.prev.Dungeon.Combat.Enemies)
				m.metrics.DamageDealt += sumAliveHP(m.prev.Dungeon.Combat.Enemies)
			}
		}
	}

	// Combat rounds.
	if state.Dungeon.Combat.Active {
		m.metrics.CombatRounds = max(m.metrics.CombatRounds, state.Dungeon.Combat.Round)
	}

	// Items picked up: inventory grew.
	if len(hero.Inventory) > len(prevHero.Inventory) {
		m.metrics.ItemsPickedUp += len(hero.Inventory) - len(prevHero.Inventory)
	}

	// Items equipped: weapon/armor slot changed from empty to filled.
	if hero.Equipped.Weapon != "" && prevHero.Equipped.Weapon == "" {
		m.metrics.ItemsEquipped++
	}
	if hero.Equipped.Armor != "" && prevHero.Equipped.Armor == "" {
		m.metrics.ItemsEquipped++
	}

	// Flee tracking: we sent "flee" and combat changed.
	if m.lastCmd == "flee" && m.prev.Dungeon.Combat.Active {
		m.metrics.FleeAttempts++
		if !state.Dungeon.Combat.Active {
			// Check if room was NOT cleared (true flee, not victory).
			rs := state.Dungeon.RoomState[state.Dungeon.CurrentRoom]
			if !rs.Cleared {
				m.metrics.FleeSuccesses++
			}
		}
	}

	// Spell tracking.
	if len(m.lastCmd) > 5 && m.lastCmd[:5] == "cast " {
		if hero.MP < prevHero.MP {
			m.metrics.SpellsCast++
			if hero.HP > prevHero.HP {
				m.metrics.HealsCast++
			}
		}
	}

	// Potion/consumable tracking.
	if len(m.lastCmd) > 4 && m.lastCmd[:4] == "use " {
		if hero.HP > prevHero.HP || len(hero.Inventory) < len(prevHero.Inventory) {
			m.metrics.PotionsUsed++
		}
	}

	// Level tracking.
	if hero.Level > prevHero.Level {
		m.metrics.LeveledUp = true
		m.metrics.LevelsGained += hero.Level - prevHero.Level
	}

	m.prev = copyState(state)
}

// chooseCommand picks a weighted-random command based on current game state.
func (m *MonkeyRenderer) chooseCommand(state *model.GameState) string {
	hero := state.Party[0]
	combat := state.Dungeon.Combat

	// Priority: equip any unequipped weapon or armor in inventory (always, not random).
	if !combat.Active {
		if hero.Equipped.Weapon == "" {
			for _, item := range hero.Inventory {
				if item.Type == "weapon" {
					return "equip " + item.ID
				}
			}
		}
		if hero.Equipped.Armor == "" {
			for _, item := range hero.Inventory {
				if item.Type == "armor" {
					return "equip " + item.ID
				}
			}
		}
	}

	// Use health potion if low HP and have one.
	if float64(hero.HP)/float64(hero.MaxHP) <= 0.5 {
		for _, item := range hero.Inventory {
			if item.Type == "consumable" && item.Effect == "heal" {
				return "use " + item.ID
			}
		}
	}

	if combat.Active {
		hpRatio := float64(hero.HP) / float64(hero.MaxHP)
		isCaster := hero.Class == "mage" || hero.Class == "priest"
		hasMP := hero.MP >= 2 // min spell cost

		if hpRatio > 0.3 {
			choices := []weightedCmd{
				{"attack", 70},
				{"defend", 15},
				{"flee", 10},
			}
			if isCaster && hasMP {
				choices = append(choices, weightedCmd{"cast heal", 5})
			}
			return m.weightedChoice(choices)
		}
		// Low HP: defensive strategy.
		choices := []weightedCmd{
			{"flee", 40},
			{"defend", 20},
			{"attack", 20},
		}
		if isCaster && hasMP {
			choices = append(choices, weightedCmd{"cast heal", 20})
		}
		return m.weightedChoice(choices)
	}

	// Not in combat: check for items in room.
	rs := state.Dungeon.RoomState[state.Dungeon.CurrentRoom]
	if len(rs.Items) > 0 {
		takeCmd := "take " + rs.Items[0]
		return m.weightedChoice([]weightedCmd{
			{takeCmd, 60},
			{m.randomMove(state), 30},
			{"look", 10},
		})
	}

	// No items, not in combat: explore.
	return m.weightedChoice([]weightedCmd{
		{m.randomMove(state), 80},
		{"look", 10},
		{"inventory", 10},
	})
}

// randomMove picks a random valid exit direction, or "look" if none.
func (m *MonkeyRenderer) randomMove(state *model.GameState) string {
	exits := state.Dungeon.Exits
	if len(exits) == 0 {
		return "look"
	}
	return exits[m.rng.Intn(len(exits))]
}

// weightedChoice selects a command using weighted random selection.
func (m *MonkeyRenderer) weightedChoice(choices []weightedCmd) string {
	total := 0
	for _, c := range choices {
		total += c.weight
	}
	if total == 0 {
		return "look"
	}
	r := m.rng.Intn(total)
	for _, c := range choices {
		r -= c.weight
		if r < 0 {
			return c.cmd
		}
	}
	return choices[len(choices)-1].cmd
}

// --- Helper functions for state diffing ---

func copyState(s *model.GameState) *model.GameState {
	cp := *s
	cp.Party = make([]model.Character, len(s.Party))
	copy(cp.Party, s.Party)
	if s.Party[0].Inventory != nil {
		cp.Party[0].Inventory = make([]model.Item, len(s.Party[0].Inventory))
		copy(cp.Party[0].Inventory, s.Party[0].Inventory)
	}
	cp.Dungeon.Combat.Enemies = make([]model.EnemyInstance, len(s.Dungeon.Combat.Enemies))
	copy(cp.Dungeon.Combat.Enemies, s.Dungeon.Combat.Enemies)
	if s.Dungeon.VisitedRooms != nil {
		cp.Dungeon.VisitedRooms = make([]string, len(s.Dungeon.VisitedRooms))
		copy(cp.Dungeon.VisitedRooms, s.Dungeon.VisitedRooms)
	}
	// Deep copy RoomState map.
	if s.Dungeon.RoomState != nil {
		cp.Dungeon.RoomState = make(map[string]model.RoomState, len(s.Dungeon.RoomState))
		for k, v := range s.Dungeon.RoomState {
			rs := v
			if v.Items != nil {
				rs.Items = make([]string, len(v.Items))
				copy(rs.Items, v.Items)
			}
			cp.Dungeon.RoomState[k] = rs
		}
	}
	return &cp
}

func enemyHPDelta(prev, curr []model.EnemyInstance) int {
	total := 0
	prevMap := make(map[string]int, len(prev))
	for _, e := range prev {
		prevMap[e.ID] = e.HP
	}
	for _, e := range curr {
		if ph, ok := prevMap[e.ID]; ok && e.HP < ph {
			total += ph - e.HP
		}
	}
	return total
}

func countNewKills(prev, curr []model.EnemyInstance) int {
	kills := 0
	currMap := make(map[string]int, len(curr))
	for _, e := range curr {
		currMap[e.ID] = e.HP
	}
	for _, e := range prev {
		if e.HP > 0 {
			if ch, ok := currMap[e.ID]; ok && ch <= 0 {
				kills++
			}
		}
	}
	return kills
}

func countAliveEnemies(enemies []model.EnemyInstance) int {
	n := 0
	for _, e := range enemies {
		if e.HP > 0 {
			n++
		}
	}
	return n
}

func sumAliveHP(enemies []model.EnemyInstance) int {
	total := 0
	for _, e := range enemies {
		if e.HP > 0 {
			total += e.HP
		}
	}
	return total
}

