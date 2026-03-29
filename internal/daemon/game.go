package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/punt-labs/cryptd/internal/engine"
	"github.com/punt-labs/cryptd/internal/game"
	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/scenariodir"
)

// CommandResult holds the result of a command executed inside the game goroutine.
type CommandResult struct {
	Result any
	Err    *RPCError
}

// Command represents a request sent to the game goroutine via its command channel.
type Command struct {
	// Type is "tool", "run_loop", or "inspect".
	Type string

	// Tool command fields.
	Name string
	Args json.RawMessage

	// RunLoop command fields (normal mode).
	LoopReq *RunLoopRequest

	// Inspect command field (test-only).
	InspectFn func(*engine.Engine, *model.GameState)

	// Reply receives the command result. Buffered 1.
	Reply chan<- CommandResult
}

// RunLoopRequest carries the dependencies for running the game loop inside the
// game goroutine (normal mode).
type RunLoopRequest struct {
	Scanner            *bufio.Scanner
	Writer             io.Writer
	Interp             model.CommandInterpreter
	Narr               model.Narrator
	SkipInitialRender  bool // true when caller already sent the initial room (new_game path)
}

// Game represents a running game instance. Each Game is a goroutine that
// exclusively owns its *engine.Engine and *model.GameState. No mutex needed —
// serialization by construction.
type Game struct {
	id       string
	commands chan Command
	done     chan struct{} // closed when the game goroutine exits
}

// newGame creates a Game and starts its goroutine. The goroutine runs until ctx
// is cancelled or the commands channel is closed.
func newGame(ctx context.Context, id, scenarioDir, defaultScenario string) *Game {
	g := &Game{
		id:       id,
		commands: make(chan Command, 1),
		done:     make(chan struct{}),
	}
	go g.run(ctx, scenarioDir, defaultScenario)
	return g
}

// run is the game goroutine. It exclusively owns eng and state.
// No other goroutine may touch these.
func (g *Game) run(ctx context.Context, scenarioDir, defaultScenario string) {
	var eng *engine.Engine
	var state *model.GameState
	defer close(g.done)
	defer func() {
		if r := recover(); r != nil {
			log.Printf("daemon: game %s panicked: %v", g.id, r)
		}
	}()

	for {
		select {
		case cmd, ok := <-g.commands:
			if !ok {
				return // channel closed, game over
			}

			switch cmd.Type {
			case "tool":
				if cmd.Name == "new_game" {
					newEng, newState, result, rpcErr := dispatchNewGame(cmd.Args, scenarioDir, defaultScenario)
					if rpcErr == nil {
						eng = newEng
						state = newState
					}
					cmd.Reply <- CommandResult{Result: result, Err: rpcErr}
					continue
				}
				if eng == nil || state == nil {
					cmd.Reply <- CommandResult{Err: &RPCError{Code: CodeNoActiveGame, Message: "no active game — call new_game first"}}
					continue
				}
				result, rpcErr := dispatch(eng, state, cmd.Name, cmd.Args)
				cmd.Reply <- CommandResult{Result: result, Err: rpcErr}

			case "run_loop":
				if eng == nil || state == nil {
					cmd.Reply <- CommandResult{Err: &RPCError{Code: CodeNoActiveGame, Message: "no active game — call new_game first"}}
					continue
				}
				lr := cmd.LoopReq
				loopCtx, loopCancel := context.WithCancel(ctx)
				rend := NewRPCRenderer(lr.Scanner, lr.Writer)
				rend.skipInitialRender = lr.SkipInitialRender

				loop := game.NewLoop(eng, lr.Interp, lr.Narr, rend)
				rend.StartReader(loopCtx)

				var loopErr *RPCError
				if err := loop.Run(loopCtx, state); err != nil {
					log.Printf("daemon: game loop error: %v", err)
					loopErr = &RPCError{Code: CodeInternalError, Message: err.Error()}
				}
				loopCancel() // stop readLoop goroutine

				// Send the final PlayResponse with terminal flags.
				dead := len(state.Party) > 0 && state.Party[0].HP <= 0
				finalResp := Response{
					JSONRPC: "2.0",
					ID:      rend.getLastID(),
					Result: PlayResponse{
						Quit: !dead,
						Dead: dead,
					},
				}
				data, err := json.Marshal(finalResp)
				if err != nil {
					log.Printf("daemon: marshal final response: %v", err)
				} else {
					data = append(data, '\n')
					if _, err := lr.Writer.Write(data); err != nil {
						log.Printf("daemon: game %s: write final response: %v", g.id, err)
					}
				}

				cmd.Reply <- CommandResult{Err: loopErr}

			case "inspect":
				// eng and state may be nil if new_game hasn't been called yet.
				// InspectFn callers must handle nil values or ensure new_game preceded Inspect.
				cmd.InspectFn(eng, state)
				cmd.Reply <- CommandResult{}

			default:
				cmd.Reply <- CommandResult{Err: &RPCError{Code: CodeInternalError, Message: fmt.Sprintf("unknown command type %q", cmd.Type)}}
			}

		case <-ctx.Done():
			return
		}
	}
}

// Stop signals the game goroutine to exit by closing the commands channel,
// then waits for it to finish.
func (g *Game) Stop() {
	close(g.commands)
	<-g.done
}

// Send sends a tool command to the game goroutine and waits for the result.
// Safe to call from any goroutine.
func (g *Game) Send(ctx context.Context, name string, args json.RawMessage) (any, *RPCError) {
	reply := make(chan CommandResult, 1)
	cmd := Command{Type: "tool", Name: name, Args: args, Reply: reply}
	select {
	case g.commands <- cmd:
	case <-ctx.Done():
		return nil, &RPCError{Code: CodeInternalError, Message: "context cancelled"}
	case <-g.done:
		return nil, &RPCError{Code: CodeInternalError, Message: "game terminated"}
	}
	select {
	case res := <-reply:
		return res.Result, res.Err
	case <-ctx.Done():
		return nil, &RPCError{Code: CodeInternalError, Message: "context cancelled"}
	case <-g.done:
		return nil, &RPCError{Code: CodeInternalError, Message: "game terminated"}
	}
}

// RunLoop sends a run_loop command that makes the game goroutine run the full
// game loop internally (normal mode). Blocks until the loop exits.
func (g *Game) RunLoop(ctx context.Context, req *RunLoopRequest) *RPCError {
	reply := make(chan CommandResult, 1)
	cmd := Command{Type: "run_loop", LoopReq: req, Reply: reply}
	select {
	case g.commands <- cmd:
	case <-ctx.Done():
		return &RPCError{Code: CodeInternalError, Message: "context cancelled"}
	case <-g.done:
		return &RPCError{Code: CodeInternalError, Message: "game terminated"}
	}
	select {
	case res := <-reply:
		return res.Err
	case <-ctx.Done():
		return &RPCError{Code: CodeInternalError, Message: "context cancelled"}
	case <-g.done:
		return &RPCError{Code: CodeInternalError, Message: "game terminated"}
	}
}

// Inspect runs fn inside the game goroutine with exclusive access to eng and
// state. The function executes synchronously — the game goroutine blocks until
// fn returns. Used for safe state inspection from handler code and tests.
func (g *Game) Inspect(ctx context.Context, fn func(*engine.Engine, *model.GameState)) error {
	reply := make(chan CommandResult, 1)
	cmd := Command{Type: "inspect", InspectFn: fn, Reply: reply}
	select {
	case g.commands <- cmd:
	case <-ctx.Done():
		return ctx.Err()
	case <-g.done:
		return fmt.Errorf("game terminated")
	}
	select {
	case <-reply:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-g.done:
		return fmt.Errorf("game terminated")
	}
}

// --- dispatch (free functions, called only from inside the game goroutine) ---

// dispatch routes a tool call to the appropriate engine method.
func dispatch(eng *engine.Engine, state *model.GameState, name string, args json.RawMessage) (any, *RPCError) {
	switch name {
	case "move":
		return dispatchMove(eng, state, args)
	case "look":
		return dispatchLook(eng, state)
	case "pick_up":
		return dispatchPickUp(eng, state, args)
	case "drop":
		return dispatchDrop(eng, state, args)
	case "equip":
		return dispatchEquip(eng, state, args)
	case "unequip":
		return dispatchUnequip(eng, state, args)
	case "examine":
		return dispatchExamine(eng, state, args)
	case "inventory":
		return dispatchInventory(eng, state)
	case "attack":
		return dispatchAttack(eng, state, args)
	case "defend":
		return dispatchDefend(eng, state)
	case "flee":
		return dispatchFlee(eng, state)
	case "cast_spell":
		return dispatchCastSpell(eng, state, args)
	case "use_item":
		return dispatchUseItem(eng, state, args)
	case "save_game":
		return dispatchSaveGame(eng, state, args)
	case "load_game":
		return dispatchLoadGame(eng, state, args)
	default:
		return nil, &RPCError{Code: CodeInvalidParams, Message: fmt.Sprintf("unknown command %q", name)}
	}
}

// --- new_game ---

type newGameArgs struct {
	ScenarioID     string       `json:"scenario_id"`
	CharacterName  string       `json:"character_name"`
	CharacterClass string       `json:"character_class"`
	Stats          *model.Stats `json:"stats,omitempty"`
}

// dispatchNewGame creates a new engine and state. Returns them alongside the
// result so the game goroutine can replace its locals.
func dispatchNewGame(raw json.RawMessage, scenarioDir, defaultScenario string) (*engine.Engine, *model.GameState, any, *RPCError) {
	var a newGameArgs
	if err := json.Unmarshal(raw, &a); err != nil {
		return nil, nil, nil, &RPCError{Code: CodeInvalidParams, Message: "invalid arguments: " + err.Error()}
	}
	if a.ScenarioID == "" {
		a.ScenarioID = defaultScenario
	}
	if a.ScenarioID == "" || a.CharacterName == "" || a.CharacterClass == "" {
		return nil, nil, nil, &RPCError{Code: CodeInvalidParams, Message: "scenario_id, character_name, and character_class are required"}
	}

	sc, err := scenariodir.Load(scenarioDir, a.ScenarioID)
	if err != nil {
		return nil, nil, nil, &RPCError{Code: CodeInvalidParams, Message: err.Error()}
	}

	hero, err := engine.NewCharacter(a.CharacterName, a.CharacterClass, a.Stats)
	if err != nil {
		return nil, nil, nil, &RPCError{Code: CodeInvalidParams, Message: err.Error()}
	}

	eng := engine.New(sc)
	gs, err := eng.NewGame(hero)
	if err != nil {
		return nil, nil, nil, &RPCError{Code: CodeInternalError, Message: err.Error()}
	}

	state := &gs
	look := eng.Look(state)
	hs := heroSummary(state)

	result := map[string]any{
		"room":        look.Room,
		"name":        look.Name,
		"description": look.Description,
		"exits":       look.Exits,
		"items":       look.Items,
		"hero":        hs,
	}

	return eng, state, result, nil
}

// --- move ---

type moveArgs struct {
	Direction string `json:"direction"`
}

func dispatchMove(eng *engine.Engine, state *model.GameState, raw json.RawMessage) (any, *RPCError) {
	if state.Dungeon.Combat.Active {
		return nil, &RPCError{Code: CodeStateBlocked, Message: "cannot move during combat"}
	}

	var a moveArgs
	if err := json.Unmarshal(raw, &a); err != nil {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid arguments: " + err.Error()}
	}
	if a.Direction == "" {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "direction is required"}
	}

	result, err := eng.Move(state, a.Direction)
	if err != nil {
		return nil, engineError(err)
	}

	look := eng.Look(state)

	response := map[string]any{
		"room":        result.NewRoom,
		"name":        look.Name,
		"description": look.Description,
		"exits":       look.Exits,
		"items":       look.Items,
	}

	combatResult, combatErr := eng.StartCombat(state)
	switch {
	case combatErr == nil:
		response["combat"] = combatSummary(combatResult, state)
		if state.Dungeon.Combat.Active && !isHeroTurn(state) {
			response["enemy_turns"] = processEnemyTurns(eng, state)
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

func dispatchLook(eng *engine.Engine, state *model.GameState) (any, *RPCError) {
	look := eng.Look(state)
	result := map[string]any{
		"room":        look.Room,
		"name":        look.Name,
		"description": look.Description,
		"exits":       look.Exits,
		"items":       look.Items,
	}
	if state.Dungeon.Combat.Active {
		result["combat"] = currentCombatState(state)
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

func dispatchPickUp(eng *engine.Engine, state *model.GameState, raw json.RawMessage) (any, *RPCError) {
	if state.Dungeon.Combat.Active {
		return nil, &RPCError{Code: CodeStateBlocked, Message: "cannot pick up items during combat"}
	}
	var a itemArgs
	if err := json.Unmarshal(raw, &a); err != nil {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid arguments: " + err.Error()}
	}
	if a.ItemID == "" {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "item_id is required"}
	}
	result, err := eng.PickUp(state, a.ItemID)
	if err != nil {
		return nil, engineError(err)
	}
	return map[string]any{
		"item": result.Item,
		"hero": heroSummary(state),
	}, nil
}

func dispatchDrop(eng *engine.Engine, state *model.GameState, raw json.RawMessage) (any, *RPCError) {
	if state.Dungeon.Combat.Active {
		return nil, &RPCError{Code: CodeStateBlocked, Message: "cannot drop items during combat"}
	}
	var a itemArgs
	if err := json.Unmarshal(raw, &a); err != nil {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid arguments: " + err.Error()}
	}
	if a.ItemID == "" {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "item_id is required"}
	}
	result, err := eng.Drop(state, a.ItemID)
	if err != nil {
		return nil, engineError(err)
	}
	return map[string]any{
		"item": result.Item,
		"hero": heroSummary(state),
	}, nil
}

func dispatchEquip(eng *engine.Engine, state *model.GameState, raw json.RawMessage) (any, *RPCError) {
	if state.Dungeon.Combat.Active {
		return nil, &RPCError{Code: CodeStateBlocked, Message: "cannot equip items during combat"}
	}
	var a itemArgs
	if err := json.Unmarshal(raw, &a); err != nil {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid arguments: " + err.Error()}
	}
	if a.ItemID == "" {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "item_id is required"}
	}
	result, err := eng.Equip(state, a.ItemID)
	if err != nil {
		return nil, engineError(err)
	}
	return map[string]any{
		"item": result.Item,
		"slot": result.Slot,
		"hero": heroSummary(state),
	}, nil
}

func dispatchUnequip(eng *engine.Engine, state *model.GameState, raw json.RawMessage) (any, *RPCError) {
	if state.Dungeon.Combat.Active {
		return nil, &RPCError{Code: CodeStateBlocked, Message: "cannot unequip items during combat"}
	}
	var a slotArgs
	if err := json.Unmarshal(raw, &a); err != nil {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid arguments: " + err.Error()}
	}
	if a.Slot == "" {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "slot is required"}
	}
	result, err := eng.Unequip(state, a.Slot)
	if err != nil {
		return nil, engineError(err)
	}
	return map[string]any{
		"item": result.Item,
		"slot": result.Slot,
		"hero": heroSummary(state),
	}, nil
}

func dispatchExamine(eng *engine.Engine, state *model.GameState, raw json.RawMessage) (any, *RPCError) {
	if state.Dungeon.Combat.Active {
		return nil, &RPCError{Code: CodeStateBlocked, Message: "cannot examine items during combat"}
	}
	var a itemArgs
	if err := json.Unmarshal(raw, &a); err != nil {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid arguments: " + err.Error()}
	}
	if a.ItemID == "" {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "item_id is required"}
	}
	result, err := eng.Examine(state, a.ItemID)
	if err != nil {
		return nil, engineError(err)
	}
	return map[string]any{"item": result.Item}, nil
}

func dispatchInventory(eng *engine.Engine, state *model.GameState) (any, *RPCError) {
	result := eng.Inventory(state)
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

func dispatchAttack(eng *engine.Engine, state *model.GameState, raw json.RawMessage) (any, *RPCError) {
	if !state.Dungeon.Combat.Active {
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
		targetID = eng.FirstAliveEnemy(state)
		if targetID == "" {
			return nil, &RPCError{Code: CodeStateBlocked, Message: "no alive enemies to attack"}
		}
	}

	result, err := eng.Attack(state, targetID)
	if err != nil {
		return nil, engineError(err)
	}

	response := map[string]any{
		"target": result.Target,
		"damage": result.Damage,
		"killed": result.Killed,
		"hero":   heroSummary(state),
	}

	if result.Killed {
		response["xp_awarded"] = result.XPAwarded
	}

	if result.CombatOver {
		response["combat_over"] = true
		response["level_up"] = checkLevelUp(eng, state)
	} else if state.Dungeon.Combat.Active && !isHeroTurn(state) {
		response["enemy_turns"] = processEnemyTurns(eng, state)
	}

	return response, nil
}

func dispatchDefend(eng *engine.Engine, state *model.GameState) (any, *RPCError) {
	if !state.Dungeon.Combat.Active {
		return nil, &RPCError{Code: CodeStateBlocked, Message: "not in combat"}
	}

	_, err := eng.Defend(state)
	if err != nil {
		return nil, engineError(err)
	}

	response := map[string]any{
		"defending": true,
		"hero":      heroSummary(state),
	}

	if state.Dungeon.Combat.Active && !isHeroTurn(state) {
		response["enemy_turns"] = processEnemyTurns(eng, state)
	}

	return response, nil
}

func dispatchFlee(eng *engine.Engine, state *model.GameState) (any, *RPCError) {
	if !state.Dungeon.Combat.Active {
		return nil, &RPCError{Code: CodeStateBlocked, Message: "not in combat"}
	}

	result, err := eng.Flee(state)
	if err != nil {
		return nil, engineError(err)
	}

	response := map[string]any{
		"success": result.Success,
		"hero":    heroSummary(state),
	}

	if !result.Success && state.Dungeon.Combat.Active && !isHeroTurn(state) {
		response["enemy_turns"] = processEnemyTurns(eng, state)
	}

	return response, nil
}

type castSpellArgs struct {
	SpellID  string `json:"spell_id"`
	TargetID string `json:"target_id"`
}

func dispatchCastSpell(eng *engine.Engine, state *model.GameState, raw json.RawMessage) (any, *RPCError) {
	var a castSpellArgs
	if err := json.Unmarshal(raw, &a); err != nil {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid arguments: " + err.Error()}
	}
	if a.SpellID == "" {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "spell_id is required"}
	}

	targetID := a.TargetID
	if targetID == "" && state.Dungeon.Combat.Active {
		targetID = eng.FirstAliveEnemy(state)
	}

	result, err := eng.CastSpell(state, a.SpellID, targetID)
	if err != nil {
		return nil, engineError(err)
	}

	response := map[string]any{
		"spell":   result.SpellName,
		"effect":  result.Effect,
		"power":   result.Power,
		"mp_cost": result.MPCost,
		"hero":    heroSummary(state),
	}

	if result.Effect == "damage" {
		response["target"] = result.TargetID
	}

	if state.Dungeon.Combat.Active && !isHeroTurn(state) {
		response["enemy_turns"] = processEnemyTurns(eng, state)
	} else if result.Effect == "damage" && !state.Dungeon.Combat.Active {
		response["combat_over"] = true
		response["level_up"] = checkLevelUp(eng, state)
	}

	return response, nil
}

// --- use_item ---

func dispatchUseItem(eng *engine.Engine, state *model.GameState, raw json.RawMessage) (any, *RPCError) {
	var a itemArgs
	if err := json.Unmarshal(raw, &a); err != nil {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid arguments: " + err.Error()}
	}
	if a.ItemID == "" {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "item_id is required"}
	}
	result, err := eng.UseItem(state, a.ItemID)
	if err != nil {
		return nil, engineError(err)
	}
	return map[string]any{
		"item":    result.Item.Name,
		"effect":  result.Effect,
		"power":   result.Power,
		"hero_hp": result.HeroHP,
	}, nil
}

// --- save/load ---

type saveLoadArgs struct {
	Slot string `json:"slot"`
}

func dispatchSaveGame(eng *engine.Engine, state *model.GameState, raw json.RawMessage) (any, *RPCError) {
	var a saveLoadArgs
	if raw != nil {
		if err := json.Unmarshal(raw, &a); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid arguments: " + err.Error()}
		}
	}
	result, err := eng.SaveGame(state, a.Slot)
	if err != nil {
		return nil, engineError(err)
	}
	return map[string]any{"slot": result.Slot, "path": result.Path}, nil
}

func dispatchLoadGame(eng *engine.Engine, state *model.GameState, raw json.RawMessage) (any, *RPCError) {
	var a saveLoadArgs
	if raw != nil {
		if err := json.Unmarshal(raw, &a); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid arguments: " + err.Error()}
		}
	}
	loaded, result, err := eng.LoadGame(a.Slot)
	if err != nil {
		return nil, engineError(err)
	}
	*state = loaded
	look := eng.Look(state)
	return map[string]any{
		"slot":        result.Slot,
		"room":        look.Room,
		"name":        look.Name,
		"description": look.Description,
		"exits":       look.Exits,
		"items":       look.Items,
		"hero":        heroSummary(state),
	}, nil
}

// --- helpers ---

// processEnemyTurns runs all enemy turns until it's the hero's turn (or combat ends).
func processEnemyTurns(eng *engine.Engine, state *model.GameState) []map[string]any {
	var turns []map[string]any
	maxIter := len(state.Dungeon.Combat.TurnOrder) * 2
	if maxIter < 1 {
		maxIter = 1
	}
	for i := 0; i < maxIter && state.Dungeon.Combat.Active && !isHeroTurn(state); i++ {
		result, err := eng.ProcessEnemyTurn(state)
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
func checkLevelUp(eng *engine.Engine, state *model.GameState) any {
	result := eng.CheckLevelUp(state)
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
		"name":   h.Name,
		"class":  h.Class,
		"level":  h.Level,
		"hp":     h.HP,
		"max_hp": h.MaxHP,
		"mp":     h.MP,
		"max_mp": h.MaxMP,
		"xp":     h.XP,
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

// isHeroTurn safely checks whether it is the hero's turn.
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
	var noExit *engine.NoExitError
	var notInRoom *engine.ItemNotInRoomError
	var notInInv *engine.ItemNotInInventoryError
	var tooHeavy *engine.TooHeavyError
	var notEquip *engine.NotEquippableError
	var notConsumable *engine.NotConsumableError
	var occupied *engine.SlotOccupiedError
	var slotEmpty *engine.SlotEmptyError
	var invalidTarget *engine.InvalidTargetError
	var unknownSpell *engine.UnknownSpellError
	var notCaster *engine.NotCasterError
	var insuffMP *engine.InsufficientMPError
	var invalidSlot *engine.InvalidSlotError
	var scenarioMismatch *engine.ScenarioMismatchError

	var locked *engine.LockedError
	var notInCombat *engine.NotInCombatError
	var notHeroTurn *engine.NotHeroTurnError
	var alreadyCombat *engine.AlreadyInCombatError
	var noEnemies *engine.NoEnemiesError

	var heroDead *engine.HeroDeadError

	switch {
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
	case errors.As(err, &notConsumable):
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

	case errors.As(err, &heroDead):
		return &RPCError{Code: CodeGameOver, Message: err.Error()}

	default:
		return &RPCError{Code: CodeInternalError, Message: err.Error()}
	}
}
