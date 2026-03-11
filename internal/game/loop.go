// Package game implements the main game loop that wires together the engine,
// interpreter, narrator, and renderer.
package game

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/punt-labs/cryptd/internal/engine"
	"github.com/punt-labs/cryptd/internal/model"
)

// inventoryErrorEvent converts a typed inventory error into a narration-friendly
// EngineEvent. Returns (event, nil) for expected user errors, or (zero, err)
// for unexpected internal errors that should propagate to the caller.
func inventoryErrorEvent(err error) (model.EngineEvent, error) {
	var notInRoom *engine.ItemNotInRoomError
	var notInInv *engine.ItemNotInInventoryError
	var tooHeavy *engine.TooHeavyError
	var notEquippable *engine.NotEquippableError
	var occupied *engine.SlotOccupiedError
	var slotEmpty *engine.SlotEmptyError

	switch {
	case errors.As(err, &notInRoom):
		return model.EngineEvent{Type: "item_not_found"}, nil
	case errors.As(err, &notInInv):
		return model.EngineEvent{Type: "not_in_inventory"}, nil
	case errors.As(err, &tooHeavy):
		return model.EngineEvent{Type: "too_heavy"}, nil
	case errors.As(err, &notEquippable):
		return model.EngineEvent{Type: "not_equippable"}, nil
	case errors.As(err, &occupied):
		return model.EngineEvent{Type: "slot_occupied"}, nil
	case errors.As(err, &slotEmpty):
		return model.EngineEvent{Type: "slot_empty"}, nil
	default:
		return model.EngineEvent{}, err
	}
}

// lookEvent builds an EngineEvent from a LookResult, carrying the room
// description, exits, and items so the narrator can display them.
func lookEvent(look engine.LookResult) model.EngineEvent {
	name := look.Name
	if name == "" {
		name = look.Room
	}
	return model.EngineEvent{
		Type: "looked",
		Room: name,
		Details: map[string]any{
			"description": look.Description,
			"exits":       look.Exits,
			"items":       look.Items,
		},
	}
}

// combatBlockedActions are action types that cannot be performed during combat.
var combatBlockedActions = map[string]bool{
	"move": true, "take": true, "drop": true,
	"equip": true, "unequip": true, "examine": true,
}

// Loop runs the game by pulling input from the renderer, routing it through
// the interpreter and engine, and pushing narrated output back to the renderer.
type Loop struct {
	eng    *engine.Engine
	interp model.CommandInterpreter
	narr   model.Narrator
	rend   model.Renderer
}

// NewLoop creates a game Loop.
func NewLoop(eng *engine.Engine, interp model.CommandInterpreter, narr model.Narrator, rend model.Renderer) *Loop {
	return &Loop{eng: eng, interp: interp, narr: narr, rend: rend}
}

// Run drives the game loop until the player quits or the context is cancelled.
func (l *Loop) Run(ctx context.Context, state *model.GameState) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	// Render initial state.
	look := l.eng.Look(state)
	narration, err := l.narr.Narrate(ctx, lookEvent(look), *state)
	if err != nil {
		return err
	}
	if err := l.rend.Render(ctx, *state, narration); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-l.rend.Events():
			if !ok {
				return nil
			}

			// Route all events — including renderer-emitted quits — through
			// dispatch so the quit narration is rendered before returning.
			var action model.EngineAction
			switch ev.Type {
			case "quit":
				action = model.EngineAction{Type: "quit"}
			case "error":
				return fmt.Errorf("renderer error: %s", ev.Payload)
			default:
				action, err = l.interp.Interpret(ctx, ev.Payload, *state)
				if err != nil {
					return err
				}
			}

			events, narration, err := l.dispatch(ctx, state, action)
			if err != nil {
				return err
			}

			if err := l.rend.Render(ctx, *state, narration); err != nil {
				return err
			}

			for _, event := range events {
				if event.Type == "quit" {
					return nil
				}
				if event.Type == "hero_died" {
					return nil
				}
			}
		}
	}
}

func (l *Loop) dispatch(ctx context.Context, state *model.GameState, action model.EngineAction) ([]model.EngineEvent, string, error) {
	// Block non-combat actions during combat.
	if state.Dungeon.Combat.Active && combatBlockedActions[action.Type] {
		event := model.EngineEvent{Type: "in_combat"}
		narration, err := l.narr.Narrate(ctx, event, *state)
		if err != nil {
			return nil, "", err
		}
		return []model.EngineEvent{event}, narration, nil
	}

	var event model.EngineEvent

	switch action.Type {
	case "move":
		result, err := l.eng.Move(state, action.Direction)
		if err != nil {
			var locked *engine.LockedError
			var noExit *engine.NoExitError
			switch {
			case errors.As(err, &locked):
				event = model.EngineEvent{Type: "locked_door", Details: map[string]any{"direction": locked.Direction}}
			case errors.As(err, &noExit):
				event = model.EngineEvent{Type: "no_exit"}
			default:
				return nil, "", err
			}
		} else {
			look := l.eng.Look(state)
			roomName := look.Name
			if roomName == "" {
				roomName = result.NewRoom
			}
			event = model.EngineEvent{Type: "moved", Room: roomName, Details: map[string]any{
				"description": look.Description,
				"exits":       look.Exits,
				"items":       look.Items,
			}}

			// Auto-start combat if room has enemies and is not cleared.
			narration, err := l.narr.Narrate(ctx, event, *state)
			if err != nil {
				return nil, "", err
			}
			combatResult, combatErr := l.eng.StartCombat(state)
			if combatErr != nil {
				// Only ignore expected "no combat" errors; propagate real errors.
				var noEnemies *engine.NoEnemiesError
				var already *engine.AlreadyInCombatError
				if !errors.As(combatErr, &noEnemies) && !errors.As(combatErr, &already) {
					return nil, "", combatErr
				}
				// No combat to start — return just the move event.
				return []model.EngineEvent{event}, narration, nil
			}
			// Combat started — append combat narration.
			return l.narrateCombatStart(ctx, state, event, narration, combatResult)
		}

	case "look":
		look := l.eng.Look(state)
		event = lookEvent(look)

	case "take":
		result, err := l.eng.PickUp(state, action.ItemID)
		if err != nil {
			event, err = inventoryErrorEvent(err)
			if err != nil {
				return nil, "", err
			}
		} else {
			event = model.EngineEvent{Type: "picked_up", Details: map[string]any{
				"item_name": result.Item.Name, "item_id": result.Item.ID,
			}}
		}

	case "drop":
		result, err := l.eng.Drop(state, action.ItemID)
		if err != nil {
			event, err = inventoryErrorEvent(err)
			if err != nil {
				return nil, "", err
			}
		} else {
			event = model.EngineEvent{Type: "dropped", Details: map[string]any{
				"item_name": result.Item.Name, "item_id": result.Item.ID,
			}}
		}

	case "equip":
		result, err := l.eng.Equip(state, action.ItemID)
		if err != nil {
			event, err = inventoryErrorEvent(err)
			if err != nil {
				return nil, "", err
			}
		} else {
			event = model.EngineEvent{Type: "equipped", Details: map[string]any{
				"item_name": result.Item.Name, "slot": result.Slot,
			}}
		}

	case "unequip":
		result, err := l.eng.Unequip(state, action.Target)
		if err != nil {
			event, err = inventoryErrorEvent(err)
			if err != nil {
				return nil, "", err
			}
		} else {
			event = model.EngineEvent{Type: "unequipped", Details: map[string]any{
				"item_name": result.Item.Name, "slot": result.Slot,
			}}
		}

	case "examine":
		result, err := l.eng.Examine(state, action.ItemID)
		if err != nil {
			event, err = inventoryErrorEvent(err)
			if err != nil {
				return nil, "", err
			}
		} else {
			event = model.EngineEvent{Type: "examined", Details: map[string]any{
				"item_name": result.Item.Name, "description": result.Item.Description,
			}}
		}

	case "inventory":
		result := l.eng.Inventory(state)
		event = model.EngineEvent{Type: "inventory_listed", Details: map[string]any{
			"items":    result.Items,
			"equipped": result.Equipped,
			"weight":   result.Weight,
			"capacity": result.Capacity,
		}}

	case "attack":
		return l.dispatchAttack(ctx, state, action)

	case "defend":
		return l.dispatchDefend(ctx, state)

	case "flee":
		return l.dispatchFlee(ctx, state)

	case "cast":
		return l.dispatchCast(ctx, state, action)

	case "save":
		result, err := l.eng.SaveGame(state, action.Target)
		if err != nil {
			event = model.EngineEvent{Type: "save_error"}
		} else {
			event = model.EngineEvent{Type: "game_saved", Details: map[string]any{
				"slot": result.Slot,
			}}
		}

	case "load":
		loaded, result, err := l.eng.LoadGame(action.Target)
		if err != nil {
			event = model.EngineEvent{Type: "load_error"}
		} else {
			*state = loaded
			event = model.EngineEvent{Type: "game_loaded", Details: map[string]any{
				"slot": result.Slot,
			}}
		}

	case "help":
		event = model.EngineEvent{Type: "help"}

	case "quit":
		event = model.EngineEvent{Type: "quit"}

	default:
		event = model.EngineEvent{Type: "unknown_action"}
	}

	narration, err := l.narr.Narrate(ctx, event, *state)
	if err != nil {
		return nil, "", err
	}
	return []model.EngineEvent{event}, narration, nil
}

func (l *Loop) dispatchAttack(ctx context.Context, state *model.GameState, action model.EngineAction) ([]model.EngineEvent, string, error) {
	if !state.Dungeon.Combat.Active {
		event := model.EngineEvent{Type: "not_in_combat"}
		narration, err := l.narr.Narrate(ctx, event, *state)
		if err != nil {
			return nil, "", err
		}
		return []model.EngineEvent{event}, narration, nil
	}

	targetID := action.Target
	if targetID == "" {
		targetID = l.eng.FirstAliveEnemy(state)
	}

	// Capture enemy name before Attack() — endCombat zeroes the slice on kill.
	enemyName := targetID
	for _, e := range state.Dungeon.Combat.Enemies {
		if e.ID == targetID {
			enemyName = e.Name
			break
		}
	}

	result, err := l.eng.Attack(state, targetID)
	if err != nil {
		return l.narrateCombatError(ctx, state, err)
	}

	var events []model.EngineEvent
	var parts []string

	if result.Killed {
		event := model.EngineEvent{Type: "attack_kill", Details: map[string]any{
			"target": enemyName, "xp": result.XPAwarded,
		}}
		events = append(events, event)
		narr, err := l.narr.Narrate(ctx, event, *state)
		if err != nil {
			return nil, "", err
		}
		parts = append(parts, narr)
	} else {
		event := model.EngineEvent{Type: "attack_hit", Details: map[string]any{
			"target": enemyName, "damage": result.Damage,
		}}
		events = append(events, event)
		narr, err := l.narr.Narrate(ctx, event, *state)
		if err != nil {
			return nil, "", err
		}
		parts = append(parts, narr)
	}

	if result.CombatOver {
		event := model.EngineEvent{Type: "combat_won"}
		events = append(events, event)
		narr, err := l.narr.Narrate(ctx, event, *state)
		if err != nil {
			return nil, "", err
		}
		parts = append(parts, narr)

		// Check for level-up after combat victory.
		lvlEvents, lvlNarr, err := l.narrateLevelUp(ctx, state)
		if err != nil {
			return nil, "", err
		}
		events = append(events, lvlEvents...)
		if lvlNarr != "" {
			parts = append(parts, lvlNarr)
		}

		return events, strings.Join(parts, " "), nil
	}

	// Process enemy turns.
	enemyEvents, enemyNarr, err := l.processEnemyTurns(ctx, state)
	if err != nil {
		return nil, "", err
	}
	events = append(events, enemyEvents...)
	if enemyNarr != "" {
		parts = append(parts, enemyNarr)
	}

	return events, strings.Join(parts, " "), nil
}

func (l *Loop) dispatchDefend(ctx context.Context, state *model.GameState) ([]model.EngineEvent, string, error) {
	if !state.Dungeon.Combat.Active {
		event := model.EngineEvent{Type: "not_in_combat"}
		narration, err := l.narr.Narrate(ctx, event, *state)
		if err != nil {
			return nil, "", err
		}
		return []model.EngineEvent{event}, narration, nil
	}

	_, err := l.eng.Defend(state)
	if err != nil {
		return l.narrateCombatError(ctx, state, err)
	}

	event := model.EngineEvent{Type: "defend"}
	var events []model.EngineEvent
	var parts []string

	events = append(events, event)
	narr, err := l.narr.Narrate(ctx, event, *state)
	if err != nil {
		return nil, "", err
	}
	parts = append(parts, narr)

	// Process enemy turns.
	enemyEvents, enemyNarr, err := l.processEnemyTurns(ctx, state)
	if err != nil {
		return nil, "", err
	}
	events = append(events, enemyEvents...)
	if enemyNarr != "" {
		parts = append(parts, enemyNarr)
	}

	return events, strings.Join(parts, " "), nil
}

func (l *Loop) dispatchFlee(ctx context.Context, state *model.GameState) ([]model.EngineEvent, string, error) {
	if !state.Dungeon.Combat.Active {
		event := model.EngineEvent{Type: "not_in_combat"}
		narration, err := l.narr.Narrate(ctx, event, *state)
		if err != nil {
			return nil, "", err
		}
		return []model.EngineEvent{event}, narration, nil
	}

	result, err := l.eng.Flee(state)
	if err != nil {
		return l.narrateCombatError(ctx, state, err)
	}

	var events []model.EngineEvent
	var parts []string

	if result.Success {
		event := model.EngineEvent{Type: "flee_success"}
		events = append(events, event)
		narr, err := l.narr.Narrate(ctx, event, *state)
		if err != nil {
			return nil, "", err
		}
		parts = append(parts, narr)
	} else {
		event := model.EngineEvent{Type: "flee_fail"}
		events = append(events, event)
		narr, err := l.narr.Narrate(ctx, event, *state)
		if err != nil {
			return nil, "", err
		}
		parts = append(parts, narr)

		// Process enemy turns after failed flee.
		enemyEvents, enemyNarr, err := l.processEnemyTurns(ctx, state)
		if err != nil {
			return nil, "", err
		}
		events = append(events, enemyEvents...)
		if enemyNarr != "" {
			parts = append(parts, enemyNarr)
		}
	}

	return events, strings.Join(parts, " "), nil
}

// processEnemyTurns processes all enemy turns until it's the hero's turn again.
// Returns the accumulated events and narration text.
func (l *Loop) processEnemyTurns(ctx context.Context, state *model.GameState) ([]model.EngineEvent, string, error) {
	var events []model.EngineEvent
	var parts []string

	// Safety cap derived from turn order length — each enemy gets at most one
	// action per call, so len(TurnOrder) is the maximum iterations needed.
	maxIter := len(state.Dungeon.Combat.TurnOrder)
	if maxIter < 1 {
		maxIter = 1
	}

	for i := 0; i < maxIter && state.Dungeon.Combat.Active && !l.eng.IsHeroTurn(state); i++ {
		result, err := l.eng.ProcessEnemyTurn(state)
		if err != nil {
			return nil, "", err
		}

		switch result.Action {
		case "attack":
			event := model.EngineEvent{Type: "enemy_attacks", Details: map[string]any{
				"enemy": result.EnemyName, "damage": result.Damage,
			}}
			events = append(events, event)
			narr, err := l.narr.Narrate(ctx, event, *state)
			if err != nil {
				return nil, "", err
			}
			parts = append(parts, narr)

			if result.HeroDead {
				event := model.EngineEvent{Type: "hero_died"}
				events = append(events, event)
				narr, err := l.narr.Narrate(ctx, event, *state)
				if err != nil {
					return nil, "", err
				}
				parts = append(parts, narr)
				return events, strings.Join(parts, " "), nil
			}
		case "flee":
			event := model.EngineEvent{Type: "enemy_flees", Details: map[string]any{
				"enemy": result.EnemyName,
			}}
			events = append(events, event)
			narr, err := l.narr.Narrate(ctx, event, *state)
			if err != nil {
				return nil, "", err
			}
			parts = append(parts, narr)

			// If all enemies fled/dead, combat may have ended.
			if !state.Dungeon.Combat.Active {
				wonEvent := model.EngineEvent{Type: "combat_won"}
				events = append(events, wonEvent)
				narr, err := l.narr.Narrate(ctx, wonEvent, *state)
				if err != nil {
					return nil, "", err
				}
				parts = append(parts, narr)

				// Check for level-up after combat victory (enemy flee path).
				lvlEvents, lvlNarr, err := l.narrateLevelUp(ctx, state)
				if err != nil {
					return nil, "", err
				}
				events = append(events, lvlEvents...)
				if lvlNarr != "" {
					parts = append(parts, lvlNarr)
				}
			}
		case "skip":
			// Dead enemy — no narration needed.
		}
	}

	return events, strings.Join(parts, " "), nil
}

func (l *Loop) dispatchCast(ctx context.Context, state *model.GameState, action model.EngineAction) ([]model.EngineEvent, string, error) {
	// Resolve empty target to first alive enemy (same pattern as dispatchAttack).
	targetID := action.Target
	if targetID == "" && state.Dungeon.Combat.Active {
		targetID = l.eng.FirstAliveEnemy(state)
	}

	// Capture enemy name before cast — endCombat zeroes the slice on kill.
	enemyName := targetID
	if state.Dungeon.Combat.Active {
		for _, e := range state.Dungeon.Combat.Enemies {
			if e.ID == targetID {
				enemyName = e.Name
				break
			}
		}
	}

	result, err := l.eng.CastSpell(state, action.SpellID, targetID)
	if err != nil {
		return l.narrateSpellError(ctx, state, err)
	}

	var events []model.EngineEvent
	var parts []string

	switch result.Effect {
	case "damage":
		event := model.EngineEvent{Type: "spell_damage", Details: map[string]any{
			"spell": result.SpellName, "target": enemyName,
			"damage": result.Power, "mp_cost": result.MPCost,
		}}
		events = append(events, event)
		narr, err := l.narr.Narrate(ctx, event, *state)
		if err != nil {
			return nil, "", err
		}
		parts = append(parts, narr)

		// Check if combat ended.
		if !state.Dungeon.Combat.Active {
			wonEvent := model.EngineEvent{Type: "combat_won"}
			events = append(events, wonEvent)
			narr, err := l.narr.Narrate(ctx, wonEvent, *state)
			if err != nil {
				return nil, "", err
			}
			parts = append(parts, narr)

			// Check for level-up after combat victory.
			lvlEvents, lvlNarr, err := l.narrateLevelUp(ctx, state)
			if err != nil {
				return nil, "", err
			}
			events = append(events, lvlEvents...)
			if lvlNarr != "" {
				parts = append(parts, lvlNarr)
			}

			return events, strings.Join(parts, " "), nil
		}

		// Process enemy turns.
		enemyEvents, enemyNarr, err := l.processEnemyTurns(ctx, state)
		if err != nil {
			return nil, "", err
		}
		events = append(events, enemyEvents...)
		if enemyNarr != "" {
			parts = append(parts, enemyNarr)
		}

	case "heal":
		event := model.EngineEvent{Type: "spell_heal", Details: map[string]any{
			"spell": result.SpellName, "power": result.Power,
			"mp_cost": result.MPCost, "hero_hp": result.HeroHP,
		}}
		events = append(events, event)
		narr, err := l.narr.Narrate(ctx, event, *state)
		if err != nil {
			return nil, "", err
		}
		parts = append(parts, narr)

		// If in combat, process enemy turns.
		if state.Dungeon.Combat.Active && !l.eng.IsHeroTurn(state) {
			enemyEvents, enemyNarr, err := l.processEnemyTurns(ctx, state)
			if err != nil {
				return nil, "", err
			}
			events = append(events, enemyEvents...)
			if enemyNarr != "" {
				parts = append(parts, enemyNarr)
			}
		}
	}

	return events, strings.Join(parts, " "), nil
}

func (l *Loop) narrateSpellError(ctx context.Context, state *model.GameState, err error) ([]model.EngineEvent, string, error) {
	var unknownSpell *engine.UnknownSpellError
	var notCaster *engine.NotCasterError
	var insufficientMP *engine.InsufficientMPError
	var notInCombat *engine.NotInCombatError
	var notHeroTurn *engine.NotHeroTurnError
	var invalidTarget *engine.InvalidTargetError
	var heroDead *engine.HeroDeadError

	var event model.EngineEvent
	switch {
	case errors.As(err, &unknownSpell):
		event = model.EngineEvent{Type: "unknown_spell"}
	case errors.As(err, &notCaster):
		event = model.EngineEvent{Type: "not_caster"}
	case errors.As(err, &insufficientMP):
		event = model.EngineEvent{Type: "insufficient_mp"}
	case errors.As(err, &notInCombat):
		event = model.EngineEvent{Type: "not_in_combat"}
	case errors.As(err, &notHeroTurn):
		event = model.EngineEvent{Type: "not_hero_turn"}
	case errors.As(err, &invalidTarget):
		event = model.EngineEvent{Type: "invalid_target"}
	case errors.As(err, &heroDead):
		event = model.EngineEvent{Type: "hero_died"}
	default:
		return nil, "", err
	}

	narration, err := l.narr.Narrate(ctx, event, *state)
	if err != nil {
		return nil, "", err
	}
	return []model.EngineEvent{event}, narration, nil
}

// narrateLevelUp checks for a level-up and returns the narration event if one occurred.
func (l *Loop) narrateLevelUp(ctx context.Context, state *model.GameState) ([]model.EngineEvent, string, error) {
	result := l.eng.CheckLevelUp(state)
	if !result.Leveled {
		return nil, "", nil
	}

	event := model.EngineEvent{Type: "level_up", Details: map[string]any{
		"level":   result.NewLevel,
		"hp_gain": result.HPGain,
		"mp_gain": result.MPGain,
	}}
	narration, err := l.narr.Narrate(ctx, event, *state)
	if err != nil {
		return nil, "", err
	}
	return []model.EngineEvent{event}, narration, nil
}

func (l *Loop) narrateCombatStart(ctx context.Context, state *model.GameState, moveEvent model.EngineEvent, moveNarr string, result engine.CombatStartResult) ([]model.EngineEvent, string, error) {
	var names []string
	for _, e := range result.Enemies {
		names = append(names, e.Name)
	}
	combatEvent := model.EngineEvent{Type: "combat_started", Details: map[string]any{
		"enemy_names": strings.Join(names, ", "),
	}}
	combatNarr, err := l.narr.Narrate(ctx, combatEvent, *state)
	if err != nil {
		return nil, "", err
	}

	events := []model.EngineEvent{moveEvent, combatEvent}
	narration := moveNarr + " " + combatNarr

	// If enemies go first, process their turns.
	if state.Dungeon.Combat.Active && !l.eng.IsHeroTurn(state) {
		enemyEvents, enemyNarr, err := l.processEnemyTurns(ctx, state)
		if err != nil {
			return nil, "", err
		}
		events = append(events, enemyEvents...)
		if enemyNarr != "" {
			narration += " " + enemyNarr
		}
	}

	return events, narration, nil
}

func (l *Loop) narrateCombatError(ctx context.Context, state *model.GameState, err error) ([]model.EngineEvent, string, error) {
	var notInCombat *engine.NotInCombatError
	var notHeroTurn *engine.NotHeroTurnError
	var invalidTarget *engine.InvalidTargetError
	var heroDead *engine.HeroDeadError

	var event model.EngineEvent
	switch {
	case errors.As(err, &notInCombat):
		event = model.EngineEvent{Type: "not_in_combat"}
	case errors.As(err, &notHeroTurn):
		event = model.EngineEvent{Type: "not_hero_turn"}
	case errors.As(err, &invalidTarget):
		event = model.EngineEvent{Type: "invalid_target"}
	case errors.As(err, &heroDead):
		event = model.EngineEvent{Type: "hero_died"}
	default:
		return nil, "", err
	}

	narration, err := l.narr.Narrate(ctx, event, *state)
	if err != nil {
		return nil, "", err
	}
	return []model.EngineEvent{event}, narration, nil
}
