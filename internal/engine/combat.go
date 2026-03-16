package engine

import (
	"fmt"
	"math/rand"
	"sort"

	"github.com/punt-labs/cryptd/internal/dice"
	"github.com/punt-labs/cryptd/internal/model"
)

// CombatStartResult holds the outcome of starting combat.
type CombatStartResult struct {
	Enemies   []model.EnemyInstance
	TurnOrder []string
}

// AttackResult holds the outcome of a hero attack.
type AttackResult struct {
	Target     string
	Damage     int
	TargetHP   int
	Killed     bool
	XPAwarded  int
	CombatOver bool
}

// DefendResult holds the outcome of a defend action.
type DefendResult struct{}

// FleeResult holds the outcome of a flee attempt.
type FleeResult struct {
	Success bool
}

// EnemyTurnResult holds the outcome of one enemy's turn.
type EnemyTurnResult struct {
	EnemyID   string
	EnemyName string
	Action    string // "attack" or "flee"
	Damage    int
	HeroHP    int
	HeroDead  bool
}

// NotInCombatError is returned when a combat action is attempted outside combat.
type NotInCombatError struct{}

func (e *NotInCombatError) Error() string { return "not in combat" }

// NotHeroTurnError is returned when the hero acts out of turn.
type NotHeroTurnError struct{}

func (e *NotHeroTurnError) Error() string { return "it is not your turn" }

// InvalidTargetError is returned when attacking a nonexistent or dead target.
type InvalidTargetError struct{ TargetID string }

func (e *InvalidTargetError) Error() string {
	return fmt.Sprintf("invalid target %q", e.TargetID)
}

// HeroDeadError is returned when the hero is dead.
type HeroDeadError struct{}

func (e *HeroDeadError) Error() string { return "you are dead" }

// AlreadyInCombatError is returned when combat is started while already active.
type AlreadyInCombatError struct{}

func (e *AlreadyInCombatError) Error() string { return "already in combat" }

// NoEnemiesError is returned when starting combat in a room with no enemies.
type NoEnemiesError struct{}

func (e *NoEnemiesError) Error() string { return "no enemies here" }

// StartCombat initialises combat in the current room.
func (e *Engine) StartCombat(state *model.GameState) (CombatStartResult, error) {
	if state.Dungeon.Combat.Active {
		return CombatStartResult{}, &AlreadyInCombatError{}
	}

	roomID := state.Dungeon.CurrentRoom
	room, ok := e.s.Rooms[roomID]
	if !ok {
		return CombatStartResult{}, fmt.Errorf("current room %q not found in scenario", roomID)
	}

	// Check if room is already cleared.
	rs := e.ensureRoomState(state, roomID)
	if rs.Cleared {
		return CombatStartResult{}, &NoEnemiesError{}
	}

	if len(room.Enemies) == 0 {
		return CombatStartResult{}, &NoEnemiesError{}
	}

	// Instantiate enemies from templates.
	var enemies []model.EnemyInstance
	for i, templateID := range room.Enemies {
		tmpl, ok := e.s.Enemies[templateID]
		if !ok {
			return CombatStartResult{}, fmt.Errorf("enemy template %q not found in scenario", templateID)
		}
		enemies = append(enemies, model.EnemyInstance{
			ID:         fmt.Sprintf("%s_%d", templateID, i),
			TemplateID: templateID,
			Name:       tmpl.Name,
			HP:         tmpl.HP,
			MaxHP:      tmpl.HP,
			Attack:     tmpl.Attack,
			AI:         tmpl.AI,
		})
	}

	// Roll initiative: hero DEX + 1d20 vs each enemy (fixed 10 + 1d20).
	h := hero(state)
	heroInit := h.Stats.DEX + rand.Intn(20) + 1

	type initEntry struct {
		id   string
		roll int
	}
	entries := []initEntry{{id: "hero", roll: heroInit}}
	for _, enemy := range enemies {
		enemyInit := 10 + rand.Intn(20) + 1
		entries = append(entries, initEntry{id: enemy.ID, roll: enemyInit})
	}

	// Sort descending by roll; ties broken by hero first.
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].roll != entries[j].roll {
			return entries[i].roll > entries[j].roll
		}
		// Hero wins ties.
		return entries[i].id == "hero"
	})

	turnOrder := make([]string, len(entries))
	for i, entry := range entries {
		turnOrder[i] = entry.id
	}

	state.Dungeon.Combat = model.CombatState{
		Active:    true,
		Enemies:   enemies,
		TurnOrder: turnOrder,
		Round:     1,
	}

	return CombatStartResult{
		Enemies:   enemies,
		TurnOrder: turnOrder,
	}, nil
}

// Attack executes a hero attack against the named target.
func (e *Engine) Attack(state *model.GameState, targetID string) (AttackResult, error) {
	combat := &state.Dungeon.Combat
	if !combat.Active {
		return AttackResult{}, &NotInCombatError{}
	}
	h := hero(state)
	if h.HP <= 0 {
		return AttackResult{}, &HeroDeadError{}
	}
	if !e.isHeroTurn(combat) {
		return AttackResult{}, &NotHeroTurnError{}
	}

	// Find target.
	target := e.findEnemy(combat, targetID)
	if target == nil || target.HP <= 0 {
		return AttackResult{}, &InvalidTargetError{TargetID: targetID}
	}

	// Roll damage: equipped weapon dice, or 1d2 unarmed.
	damage := e.rollHeroDamage(state)
	target.HP -= damage
	if target.HP < 0 {
		target.HP = 0
	}

	result := AttackResult{
		Target:   targetID,
		Damage:   damage,
		TargetHP: target.HP,
	}

	if target.HP <= 0 {
		result.Killed = true
		result.XPAwarded = target.MaxHP
		h.XP += target.MaxHP
	}

	// Check if all enemies dead.
	if e.allEnemiesDead(combat) {
		e.endCombat(state)
		result.CombatOver = true
	} else {
		combat.HeroDefending = false
		e.advanceTurn(combat)
	}

	return result, nil
}

// Defend sets the hero to a defensive stance.
func (e *Engine) Defend(state *model.GameState) (DefendResult, error) {
	combat := &state.Dungeon.Combat
	if !combat.Active {
		return DefendResult{}, &NotInCombatError{}
	}
	h := hero(state)
	if h.HP <= 0 {
		return DefendResult{}, &HeroDeadError{}
	}
	if !e.isHeroTurn(combat) {
		return DefendResult{}, &NotHeroTurnError{}
	}

	combat.HeroDefending = true
	e.advanceTurn(combat)
	return DefendResult{}, nil
}

// Flee attempts to escape combat.
func (e *Engine) Flee(state *model.GameState) (FleeResult, error) {
	combat := &state.Dungeon.Combat
	if !combat.Active {
		return FleeResult{}, &NotInCombatError{}
	}
	h := hero(state)
	if h.HP <= 0 {
		return FleeResult{}, &HeroDeadError{}
	}
	if !e.isHeroTurn(combat) {
		return FleeResult{}, &NotHeroTurnError{}
	}

	// DEX check: roll 1d20, succeed if <= hero DEX.
	roll := rand.Intn(20) + 1
	if roll <= h.Stats.DEX {
		e.endCombatNoVictory(state)
		return FleeResult{Success: true}, nil
	}

	// Failed — lose turn.
	combat.HeroDefending = false
	e.advanceTurn(combat)
	return FleeResult{Success: false}, nil
}

// ProcessEnemyTurn executes one enemy's turn and returns the outcome of that
// enemy's action, or an error if the combat state is invalid.
func (e *Engine) ProcessEnemyTurn(state *model.GameState) (EnemyTurnResult, error) {
	combat := &state.Dungeon.Combat
	if !combat.Active {
		return EnemyTurnResult{}, &NotInCombatError{}
	}
	if e.isHeroTurn(combat) {
		return EnemyTurnResult{}, &NotHeroTurnError{}
	}

	currentID := combat.TurnOrder[combat.CurrentTurn]
	enemy := e.findEnemy(combat, currentID)
	if enemy == nil || enemy.HP <= 0 {
		// Dead enemy — skip.
		e.advanceTurn(combat)
		return EnemyTurnResult{EnemyID: currentID, Action: "skip"}, nil
	}

	h := hero(state)

	switch enemy.AI {
	case "cautious":
		// Flee if HP <= 30% of MaxHP.
		if enemy.HP*100 <= enemy.MaxHP*30 {
			enemy.HP = 0 // Remove from combat.
			e.advanceTurn(combat)
			if e.allEnemiesDead(combat) {
				e.endCombat(state)
			}
			return EnemyTurnResult{
				EnemyID:   enemy.ID,
				EnemyName: enemy.Name,
				Action:    "flee",
				HeroHP:    h.HP,
			}, nil
		}
		// Otherwise fall through to attack.
		fallthrough
	case "aggressive", "scripted":
		damage := e.applyDefenses(state, e.rollEnemyDamage(enemy), combat.HeroDefending)
		h.HP -= damage
		heroDead := h.HP <= 0

		e.advanceTurn(combat)

		return EnemyTurnResult{
			EnemyID:   enemy.ID,
			EnemyName: enemy.Name,
			Action:    "attack",
			Damage:    damage,
			HeroHP:    h.HP,
			HeroDead:  heroDead,
		}, nil
	default:
		// Unknown AI — treat as aggressive.
		damage := e.applyDefenses(state, e.rollEnemyDamage(enemy), combat.HeroDefending)
		h.HP -= damage
		e.advanceTurn(combat)
		return EnemyTurnResult{
			EnemyID:   enemy.ID,
			EnemyName: enemy.Name,
			Action:    "attack",
			Damage:    damage,
			HeroHP:    h.HP,
			HeroDead:  h.HP <= 0,
		}, nil
	}
}

// IsHeroTurn returns whether it is currently the hero's turn in combat.
func (e *Engine) IsHeroTurn(state *model.GameState) bool {
	return e.isHeroTurn(&state.Dungeon.Combat)
}

// FirstAliveEnemy returns the ID of the first alive enemy, or "" if none.
func (e *Engine) FirstAliveEnemy(state *model.GameState) string {
	for i := range state.Dungeon.Combat.Enemies {
		if state.Dungeon.Combat.Enemies[i].HP > 0 {
			return state.Dungeon.Combat.Enemies[i].ID
		}
	}
	return ""
}

func (e *Engine) isHeroTurn(combat *model.CombatState) bool {
	if len(combat.TurnOrder) == 0 {
		return false
	}
	return combat.TurnOrder[combat.CurrentTurn] == "hero"
}

func (e *Engine) findEnemy(combat *model.CombatState, id string) *model.EnemyInstance {
	for i := range combat.Enemies {
		if combat.Enemies[i].ID == id {
			return &combat.Enemies[i]
		}
	}
	return nil
}

func (e *Engine) allEnemiesDead(combat *model.CombatState) bool {
	for i := range combat.Enemies {
		if combat.Enemies[i].HP > 0 {
			return false
		}
	}
	return true
}

func (e *Engine) rollHeroDamage(state *model.GameState) int {
	h := hero(state)
	diceNotation := "1d2" // unarmed
	if h.Equipped.Weapon != "" {
		if si, ok := e.s.Items[h.Equipped.Weapon]; ok && si.Damage != "" {
			diceNotation = si.Damage
		}
	}
	d, err := dice.Parse(diceNotation)
	if err != nil {
		return 1
	}
	result := d.Roll()
	if result < 1 {
		return 1
	}
	return result
}

// applyDefenses reduces raw damage by defend stance and equipped armor defense.
// Order: halve for defend first, then subtract armor defense. This order
// makes armor slightly less effective when defending (intentional — defend
// is already powerful). Final damage is floored at 1.
func (e *Engine) applyDefenses(state *model.GameState, damage int, defending bool) int {
	if defending {
		damage /= 2
	}
	// Armor damage reduction.
	h := hero(state)
	if h.Equipped.Armor != "" {
		if si, ok := e.s.Items[h.Equipped.Armor]; ok && si.Defense > 0 {
			damage -= si.Defense
		}
	}
	if damage < 1 {
		damage = 1
	}
	return damage
}

func (e *Engine) rollEnemyDamage(enemy *model.EnemyInstance) int {
	d, err := dice.Parse(enemy.Attack)
	if err != nil {
		return 1
	}
	result := d.Roll()
	if result < 1 {
		return 1
	}
	return result
}

func (e *Engine) advanceTurn(combat *model.CombatState) {
	if len(combat.TurnOrder) == 0 {
		return
	}
	start := combat.CurrentTurn
	for {
		combat.CurrentTurn = (combat.CurrentTurn + 1) % len(combat.TurnOrder)
		if combat.CurrentTurn == 0 {
			combat.Round++
		}

		id := combat.TurnOrder[combat.CurrentTurn]
		if id == "hero" {
			combat.HeroDefending = false
			return
		}

		// Skip dead enemies.
		enemy := e.findEnemy(combat, id)
		if enemy != nil && enemy.HP > 0 {
			return
		}

		// Safety: if we've gone all the way around, stop.
		if combat.CurrentTurn == start {
			return
		}
	}
}

// endCombat ends combat with a victory: marks room cleared.
func (e *Engine) endCombat(state *model.GameState) {
	state.Dungeon.Combat = model.CombatState{}
	rs := e.ensureRoomState(state, state.Dungeon.CurrentRoom)
	rs.Cleared = true
	state.Dungeon.RoomState[state.Dungeon.CurrentRoom] = rs
}

// endCombatNoVictory ends combat without marking the room cleared (e.g. flee).
func (e *Engine) endCombatNoVictory(state *model.GameState) {
	state.Dungeon.Combat = model.CombatState{}
}
