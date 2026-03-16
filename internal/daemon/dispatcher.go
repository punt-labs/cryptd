package daemon

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/punt-labs/cryptd/internal/engine"
	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/scenariodir"
)

// dispatch routes a tool call to the appropriate engine method and returns
// the result as a JSON-serialisable value. Errors are returned as *RPCError
// with appropriate JSON-RPC codes.
func (s *Server) dispatch(name string, args json.RawMessage) (any, *RPCError) {
	if name == "new_game" {
		return s.dispatchNewGame(args)
	}

	// All other tools require an active game.
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.eng == nil || s.state == nil {
		return nil, &RPCError{Code: CodeNoActiveGame, Message: "no active game — call new_game first"}
	}

	switch name {
	case "move":
		return s.dispatchMove(args)
	case "look":
		return s.dispatchLook()
	case "pick_up":
		return s.dispatchPickUp(args)
	case "drop":
		return s.dispatchDrop(args)
	case "equip":
		return s.dispatchEquip(args)
	case "unequip":
		return s.dispatchUnequip(args)
	case "examine":
		return s.dispatchExamine(args)
	case "inventory":
		return s.dispatchInventory()
	case "attack":
		return s.dispatchAttack(args)
	case "defend":
		return s.dispatchDefend()
	case "flee":
		return s.dispatchFlee()
	case "cast_spell":
		return s.dispatchCastSpell(args)
	case "save_game":
		return s.dispatchSaveGame(args)
	case "load_game":
		return s.dispatchLoadGame(args)
	default:
		return nil, &RPCError{Code: CodeInvalidParams, Message: fmt.Sprintf("unknown tool %q", name)}
	}
}

// --- new_game ---

type newGameArgs struct {
	ScenarioID     string       `json:"scenario_id"`
	CharacterName  string       `json:"character_name"`
	CharacterClass string       `json:"character_class"`
	Stats          *model.Stats `json:"stats,omitempty"`
}

func (s *Server) dispatchNewGame(raw json.RawMessage) (any, *RPCError) {
	var a newGameArgs
	if err := json.Unmarshal(raw, &a); err != nil {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid arguments: " + err.Error()}
	}
	if a.ScenarioID == "" || a.CharacterName == "" || a.CharacterClass == "" {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "scenario_id, character_name, and character_class are required"}
	}

	sc, err := scenariodir.Load(s.scenarioDir, a.ScenarioID)
	if err != nil {
		return nil, &RPCError{Code: CodeInvalidParams, Message: err.Error()}
	}

	hero, err := engine.NewCharacter(a.CharacterName, a.CharacterClass, a.Stats)
	if err != nil {
		return nil, &RPCError{Code: CodeInvalidParams, Message: err.Error()}
	}

	eng := engine.New(sc)

	state, err := eng.NewGame(hero)
	if err != nil {
		return nil, &RPCError{Code: CodeInternalError, Message: err.Error()}
	}
	// PlayMode is a client-side concept (DES-025). The server does not set it.

	s.mu.Lock()
	s.eng = eng
	s.state = &state
	look := eng.Look(s.state)
	hs := heroSummary(s.state)
	s.mu.Unlock()

	return map[string]any{
		"room":        look.Room,
		"name":        look.Name,
		"description": look.Description,
		"exits":       look.Exits,
		"items":       look.Items,
		"hero":        hs,
	}, nil
}

// --- move ---

type moveArgs struct {
	Direction string `json:"direction"`
}

func (s *Server) dispatchMove(raw json.RawMessage) (any, *RPCError) {
	if s.state.Dungeon.Combat.Active {
		return nil, &RPCError{Code: CodeStateBlocked, Message: "cannot move during combat"}
	}

	var a moveArgs
	if err := json.Unmarshal(raw, &a); err != nil {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid arguments: " + err.Error()}
	}
	if a.Direction == "" {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "direction is required"}
	}

	result, err := s.eng.Move(s.state, a.Direction)
	if err != nil {
		return nil, engineError(err)
	}

	look := s.eng.Look(s.state)

	response := map[string]any{
		"room":        result.NewRoom,
		"name":        look.Name,
		"description": look.Description,
		"exits":       look.Exits,
		"items":       look.Items,
	}

	// Auto-start combat if the room has enemies.
	combatResult, combatErr := s.eng.StartCombat(s.state)
	switch {
	case combatErr == nil:
		response["combat"] = combatSummary(combatResult, s.state)
		// If enemies go first, process their turns.
		if s.state.Dungeon.Combat.Active && !isHeroTurn(s.state) {
			response["enemy_turns"] = s.processEnemyTurns()
		}
	case errors.As(combatErr, new(*engine.NoEnemiesError)),
		errors.As(combatErr, new(*engine.AlreadyInCombatError)):
		// Expected — room has no enemies or combat already active.
	default:
		log.Printf("daemon: dispatchMove: StartCombat error: %v", combatErr)
		response["warning"] = "combat failed to start due to an internal error"
	}

	return response, nil
}

// --- look ---

func (s *Server) dispatchLook() (any, *RPCError) {
	look := s.eng.Look(s.state)
	result := map[string]any{
		"room":        look.Room,
		"name":        look.Name,
		"description": look.Description,
		"exits":       look.Exits,
		"items":       look.Items,
	}
	if s.state.Dungeon.Combat.Active {
		result["combat"] = currentCombatState(s.state)
	}
	return result, nil
}

// --- inventory tools ---

type itemArgs struct {
	ItemID string `json:"item_id"`
}

type slotArgs struct {
	Slot string `json:"slot"`
}

func (s *Server) dispatchPickUp(raw json.RawMessage) (any, *RPCError) {
	if s.state.Dungeon.Combat.Active {
		return nil, &RPCError{Code: CodeStateBlocked, Message: "cannot pick up items during combat"}
	}
	var a itemArgs
	if err := json.Unmarshal(raw, &a); err != nil {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid arguments: " + err.Error()}
	}
	if a.ItemID == "" {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "item_id is required"}
	}
	result, err := s.eng.PickUp(s.state, a.ItemID)
	if err != nil {
		return nil, engineError(err)
	}
	return map[string]any{
		"item": result.Item,
		"hero": heroSummary(s.state),
	}, nil
}

func (s *Server) dispatchDrop(raw json.RawMessage) (any, *RPCError) {
	if s.state.Dungeon.Combat.Active {
		return nil, &RPCError{Code: CodeStateBlocked, Message: "cannot drop items during combat"}
	}
	var a itemArgs
	if err := json.Unmarshal(raw, &a); err != nil {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid arguments: " + err.Error()}
	}
	if a.ItemID == "" {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "item_id is required"}
	}
	result, err := s.eng.Drop(s.state, a.ItemID)
	if err != nil {
		return nil, engineError(err)
	}
	return map[string]any{
		"item": result.Item,
		"hero": heroSummary(s.state),
	}, nil
}

func (s *Server) dispatchEquip(raw json.RawMessage) (any, *RPCError) {
	if s.state.Dungeon.Combat.Active {
		return nil, &RPCError{Code: CodeStateBlocked, Message: "cannot equip items during combat"}
	}
	var a itemArgs
	if err := json.Unmarshal(raw, &a); err != nil {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid arguments: " + err.Error()}
	}
	if a.ItemID == "" {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "item_id is required"}
	}
	result, err := s.eng.Equip(s.state, a.ItemID)
	if err != nil {
		return nil, engineError(err)
	}
	return map[string]any{
		"item": result.Item,
		"slot": result.Slot,
		"hero": heroSummary(s.state),
	}, nil
}

func (s *Server) dispatchUnequip(raw json.RawMessage) (any, *RPCError) {
	if s.state.Dungeon.Combat.Active {
		return nil, &RPCError{Code: CodeStateBlocked, Message: "cannot unequip items during combat"}
	}
	var a slotArgs
	if err := json.Unmarshal(raw, &a); err != nil {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid arguments: " + err.Error()}
	}
	if a.Slot == "" {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "slot is required"}
	}
	result, err := s.eng.Unequip(s.state, a.Slot)
	if err != nil {
		return nil, engineError(err)
	}
	return map[string]any{
		"item": result.Item,
		"slot": result.Slot,
		"hero": heroSummary(s.state),
	}, nil
}

func (s *Server) dispatchExamine(raw json.RawMessage) (any, *RPCError) {
	if s.state.Dungeon.Combat.Active {
		return nil, &RPCError{Code: CodeStateBlocked, Message: "cannot examine items during combat"}
	}
	var a itemArgs
	if err := json.Unmarshal(raw, &a); err != nil {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid arguments: " + err.Error()}
	}
	if a.ItemID == "" {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "item_id is required"}
	}
	result, err := s.eng.Examine(s.state, a.ItemID)
	if err != nil {
		return nil, engineError(err)
	}
	return map[string]any{"item": result.Item}, nil
}

func (s *Server) dispatchInventory() (any, *RPCError) {
	result := s.eng.Inventory(s.state)
	return map[string]any{
		"items":    result.Items,
		"equipped": result.Equipped,
		"weight":   result.Weight,
		"capacity": result.Capacity,
	}, nil
}

// --- combat tools ---

type attackArgs struct {
	TargetID string `json:"target_id"`
}

func (s *Server) dispatchAttack(raw json.RawMessage) (any, *RPCError) {
	if !s.state.Dungeon.Combat.Active {
		return nil, &RPCError{Code: CodeStateBlocked, Message: "not in combat"}
	}

	var a attackArgs
	if raw != nil {
		if err := json.Unmarshal(raw, &a); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid arguments: " + err.Error()}
		}
	}
	targetID := a.TargetID
	if targetID == "" {
		targetID = s.eng.FirstAliveEnemy(s.state)
		if targetID == "" {
			return nil, &RPCError{Code: CodeStateBlocked, Message: "no alive enemies to attack"}
		}
	}

	result, err := s.eng.Attack(s.state, targetID)
	if err != nil {
		return nil, engineError(err)
	}

	response := map[string]any{
		"target":   result.Target,
		"damage":   result.Damage,
		"killed":   result.Killed,
		"hero":     heroSummary(s.state),
	}

	if result.Killed {
		response["xp_awarded"] = result.XPAwarded
	}

	if result.CombatOver {
		response["combat_over"] = true
		response["level_up"] = s.checkLevelUp()
	} else if s.state.Dungeon.Combat.Active && !isHeroTurn(s.state) {
		response["enemy_turns"] = s.processEnemyTurns()
	}

	return response, nil
}

func (s *Server) dispatchDefend() (any, *RPCError) {
	if !s.state.Dungeon.Combat.Active {
		return nil, &RPCError{Code: CodeStateBlocked, Message: "not in combat"}
	}

	_, err := s.eng.Defend(s.state)
	if err != nil {
		return nil, engineError(err)
	}

	response := map[string]any{
		"defending": true,
		"hero":      heroSummary(s.state),
	}

	if s.state.Dungeon.Combat.Active && !isHeroTurn(s.state) {
		response["enemy_turns"] = s.processEnemyTurns()
	}

	return response, nil
}

func (s *Server) dispatchFlee() (any, *RPCError) {
	if !s.state.Dungeon.Combat.Active {
		return nil, &RPCError{Code: CodeStateBlocked, Message: "not in combat"}
	}

	result, err := s.eng.Flee(s.state)
	if err != nil {
		return nil, engineError(err)
	}

	response := map[string]any{
		"success": result.Success,
		"hero":    heroSummary(s.state),
	}

	if !result.Success && s.state.Dungeon.Combat.Active && !isHeroTurn(s.state) {
		response["enemy_turns"] = s.processEnemyTurns()
	}

	return response, nil
}

type castSpellArgs struct {
	SpellID  string `json:"spell_id"`
	TargetID string `json:"target_id"`
}

func (s *Server) dispatchCastSpell(raw json.RawMessage) (any, *RPCError) {
	var a castSpellArgs
	if err := json.Unmarshal(raw, &a); err != nil {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid arguments: " + err.Error()}
	}
	if a.SpellID == "" {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "spell_id is required"}
	}

	targetID := a.TargetID
	if targetID == "" && s.state.Dungeon.Combat.Active {
		targetID = s.eng.FirstAliveEnemy(s.state)
	}

	result, err := s.eng.CastSpell(s.state, a.SpellID, targetID)
	if err != nil {
		return nil, engineError(err)
	}

	response := map[string]any{
		"spell":   result.SpellName,
		"effect":  result.Effect,
		"power":   result.Power,
		"mp_cost": result.MPCost,
		"hero":    heroSummary(s.state),
	}

	if result.Effect == "damage" {
		response["target"] = result.TargetID
	}

	// After any spell (damage or heal), check combat state and process enemy turns.
	if s.state.Dungeon.Combat.Active && !isHeroTurn(s.state) {
		response["enemy_turns"] = s.processEnemyTurns()
	} else if result.Effect == "damage" && !s.state.Dungeon.Combat.Active {
		// Combat ended — spell killed last enemy.
		response["combat_over"] = true
		response["level_up"] = s.checkLevelUp()
	}

	return response, nil
}

// --- save/load ---

type saveLoadArgs struct {
	Slot string `json:"slot"`
}

func (s *Server) dispatchSaveGame(raw json.RawMessage) (any, *RPCError) {
	var a saveLoadArgs
	if raw != nil {
		if err := json.Unmarshal(raw, &a); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid arguments: " + err.Error()}
		}
	}
	result, err := s.eng.SaveGame(s.state, a.Slot)
	if err != nil {
		return nil, engineError(err)
	}
	return map[string]any{"slot": result.Slot, "path": result.Path}, nil
}

func (s *Server) dispatchLoadGame(raw json.RawMessage) (any, *RPCError) {
	var a saveLoadArgs
	if raw != nil {
		if err := json.Unmarshal(raw, &a); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid arguments: " + err.Error()}
		}
	}
	loaded, result, err := s.eng.LoadGame(a.Slot)
	if err != nil {
		return nil, engineError(err)
	}
	*s.state = loaded
	look := s.eng.Look(s.state)
	return map[string]any{
		"slot":        result.Slot,
		"room":        look.Room,
		"name":        look.Name,
		"description": look.Description,
		"exits":       look.Exits,
		"items":       look.Items,
		"hero":        heroSummary(s.state),
	}, nil
}

// --- helpers ---

// processEnemyTurns runs all enemy turns until it's the hero's turn (or combat ends).
// Returns a slice of per-enemy action summaries.
func (s *Server) processEnemyTurns() []map[string]any {
	var turns []map[string]any
	// Budget allows for all enemies + skips without premature exit.
	// Skips don't count against the budget since they don't represent real turns.
	maxIter := len(s.state.Dungeon.Combat.TurnOrder) * 2
	if maxIter < 1 {
		maxIter = 1
	}
	for i := 0; i < maxIter && s.state.Dungeon.Combat.Active && !isHeroTurn(s.state); i++ {
		result, err := s.eng.ProcessEnemyTurn(s.state)
		if err != nil {
			log.Printf("daemon: processEnemyTurns: unexpected error: %v", err)
			break
		}
		if result.Action == "skip" {
			continue
		}
		turn := map[string]any{
			"enemy":  result.EnemyName,
			"action": result.Action,
		}
		if result.Action == "attack" {
			turn["damage"] = result.Damage
			turn["hero_hp"] = result.HeroHP
			if result.HeroDead {
				turn["hero_dead"] = true
			}
		}
		turns = append(turns, turn)
	}
	return turns
}

// checkLevelUp checks and applies a level-up, returning the result or nil.
func (s *Server) checkLevelUp() any {
	result := s.eng.CheckLevelUp(s.state)
	if !result.Leveled {
		return nil
	}
	return map[string]any{
		"new_level": result.NewLevel,
		"hp_gain":   result.HPGain,
		"mp_gain":   result.MPGain,
	}
}

// heroSummary returns a snapshot of the hero for inclusion in tool results.
func heroSummary(state *model.GameState) map[string]any {
	if len(state.Party) == 0 {
		return map[string]any{}
	}
	h := &state.Party[0]
	return map[string]any{
		"name":  h.Name,
		"class": h.Class,
		"level": h.Level,
		"hp":    h.HP,
		"max_hp": h.MaxHP,
		"mp":    h.MP,
		"max_mp": h.MaxMP,
		"xp":    h.XP,
	}
}

// combatSummary builds a combat summary from a StartCombat result.
func combatSummary(result engine.CombatStartResult, state *model.GameState) map[string]any {
	var enemies []map[string]any
	for _, e := range result.Enemies {
		enemies = append(enemies, map[string]any{
			"id": e.ID, "name": e.Name, "hp": e.HP, "max_hp": e.MaxHP,
		})
	}
	return map[string]any{
		"enemies":    enemies,
		"turn_order": result.TurnOrder,
		"hero_turn":  isHeroTurn(state),
	}
}

// currentCombatState returns the current combat state for look results.
func currentCombatState(state *model.GameState) map[string]any {
	var enemies []map[string]any
	for _, e := range state.Dungeon.Combat.Enemies {
		if e.HP > 0 {
			enemies = append(enemies, map[string]any{
				"id": e.ID, "name": e.Name, "hp": e.HP, "max_hp": e.MaxHP,
			})
		}
	}
	return map[string]any{
		"enemies":   enemies,
		"round":     state.Dungeon.Combat.Round,
		"hero_turn": isHeroTurn(state),
	}
}

// isHeroTurn safely checks whether it is the hero's turn, guarding against empty TurnOrder.
func isHeroTurn(state *model.GameState) bool {
	to := state.Dungeon.Combat.TurnOrder
	idx := state.Dungeon.Combat.CurrentTurn
	if len(to) == 0 || idx < 0 || idx >= len(to) {
		return false
	}
	return to[idx] == "hero"
}

// engineError maps typed engine errors to JSON-RPC error codes.
func engineError(err error) *RPCError {
	// Invalid params errors (user gave bad input).
	var noExit *engine.NoExitError
	var notInRoom *engine.ItemNotInRoomError
	var notInInv *engine.ItemNotInInventoryError
	var tooHeavy *engine.TooHeavyError
	var notEquip *engine.NotEquippableError
	var occupied *engine.SlotOccupiedError
	var slotEmpty *engine.SlotEmptyError
	var invalidTarget *engine.InvalidTargetError
	var unknownSpell *engine.UnknownSpellError
	var notCaster *engine.NotCasterError
	var insuffMP *engine.InsufficientMPError
	var invalidSlot *engine.InvalidSlotError
	var scenarioMismatch *engine.ScenarioMismatchError

	// State-blocked errors.
	var locked *engine.LockedError
	var notInCombat *engine.NotInCombatError
	var notHeroTurn *engine.NotHeroTurnError
	var alreadyCombat *engine.AlreadyInCombatError
	var noEnemies *engine.NoEnemiesError

	// Game over.
	var heroDead *engine.HeroDeadError

	switch {
	// Invalid params.
	case errors.As(err, &noExit):
		return &RPCError{Code: CodeInvalidParams, Message: err.Error()}
	case errors.As(err, &notInRoom):
		return &RPCError{Code: CodeInvalidParams, Message: err.Error()}
	case errors.As(err, &notInInv):
		return &RPCError{Code: CodeInvalidParams, Message: err.Error()}
	case errors.As(err, &tooHeavy):
		return &RPCError{Code: CodeInvalidParams, Message: err.Error()}
	case errors.As(err, &notEquip):
		return &RPCError{Code: CodeInvalidParams, Message: err.Error()}
	case errors.As(err, &occupied):
		return &RPCError{Code: CodeInvalidParams, Message: err.Error()}
	case errors.As(err, &slotEmpty):
		return &RPCError{Code: CodeInvalidParams, Message: err.Error()}
	case errors.As(err, &invalidTarget):
		return &RPCError{Code: CodeInvalidParams, Message: err.Error()}
	case errors.As(err, &unknownSpell):
		return &RPCError{Code: CodeInvalidParams, Message: err.Error()}
	case errors.As(err, &notCaster):
		return &RPCError{Code: CodeInvalidParams, Message: err.Error()}
	case errors.As(err, &insuffMP):
		return &RPCError{Code: CodeInvalidParams, Message: err.Error()}
	case errors.As(err, &invalidSlot):
		return &RPCError{Code: CodeInvalidParams, Message: err.Error()}
	case errors.As(err, &scenarioMismatch):
		return &RPCError{Code: CodeInvalidParams, Message: err.Error()}

	// State blocked.
	case errors.As(err, &locked):
		return &RPCError{Code: CodeStateBlocked, Message: err.Error()}
	case errors.As(err, &notInCombat):
		return &RPCError{Code: CodeStateBlocked, Message: err.Error()}
	case errors.As(err, &notHeroTurn):
		return &RPCError{Code: CodeStateBlocked, Message: err.Error()}
	case errors.As(err, &alreadyCombat):
		return &RPCError{Code: CodeStateBlocked, Message: err.Error()}
	case errors.As(err, &noEnemies):
		return &RPCError{Code: CodeStateBlocked, Message: err.Error()}

	// Game over.
	case errors.As(err, &heroDead):
		return &RPCError{Code: CodeGameOver, Message: err.Error()}

	default:
		return &RPCError{Code: CodeInternalError, Message: err.Error()}
	}
}

