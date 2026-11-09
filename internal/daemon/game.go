package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"sort"

	"github.com/punt-labs/cryptd/internal/engine"
	"github.com/punt-labs/cryptd/internal/game"
	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/protocol"
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
	case "context":
		return dispatchContext(eng, state)
	case "save_game":
		return dispatchSaveGame(eng, state, args)
	case "load_game":
		return dispatchLoadGame(eng, state, args)
	default:
		return nil, &RPCError{Code: CodeMethodNotFound, Message: fmt.Sprintf("unknown command %q", name)}
	}
}

// --- new_game ---

type newGameArgs struct {
	ScenarioID     string       `json:"scenario_id"`
	CharacterName  string       `json:"character_name"`
	CharacterClass string       `json:"character_class"`
	Stats          *model.Stats `json:"stats,omitempty"`
}

// validate checks the required new-game fields. The caller applies the
// default scenario before calling this, so an empty ScenarioID here means
// no default was configured either.
func (a newGameArgs) validate() *RPCError {
	if a.ScenarioID == "" || a.CharacterName == "" || a.CharacterClass == "" {
		return &RPCError{Code: CodeInvalidParams, Message: "scenario_id, character_name, and character_class are required"}
	}
	return nil
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
	if rpcErr := a.validate(); rpcErr != nil {
		return nil, nil, nil, rpcErr
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

	result := protocol.NewGameResult{
		Room:        look.Room,
		Name:        look.Name,
		Description: look.Description,
		Exits:       look.Exits,
		Items:       look.Items,
		Hero:        heroSummary(state),
	}

	return eng, state, result, nil
}

// --- move ---

type moveArgs struct {
	Direction string `json:"direction"`
}

func (a moveArgs) validate() *RPCError {
	if a.Direction == "" {
		return &RPCError{Code: CodeInvalidParams, Message: "direction is required"}
	}
	return nil
}

func dispatchMove(eng *engine.Engine, state *model.GameState, raw json.RawMessage) (any, *RPCError) {
	if state.Dungeon.Combat.Active {
		return nil, &RPCError{Code: CodeStateBlocked, Message: "cannot move during combat"}
	}

	var a moveArgs
	if err := json.Unmarshal(raw, &a); err != nil {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid arguments: " + err.Error()}
	}
	if rpcErr := a.validate(); rpcErr != nil {
		return nil, rpcErr
	}

	result, err := eng.Move(state, a.Direction)
	if err != nil {
		return nil, engineError(err)
	}

	look := eng.Look(state)

	response := protocol.MoveResult{
		Room:        result.NewRoom,
		Name:        look.Name,
		Description: look.Description,
		Exits:       look.Exits,
		Items:       look.Items,
	}

	combatResult, combatErr := eng.StartCombat(state)
	switch {
	case combatErr == nil:
		response.Combat = combatSummary(combatResult, state)
		if state.Dungeon.Combat.Active && !isHeroTurn(state) {
			turns := processEnemyTurns(eng, state)
			response.EnemyTurns = &turns
		}
	case errors.As(combatErr, new(*engine.NoEnemiesError)),
		errors.As(combatErr, new(*engine.AlreadyInCombatError)):
		// Expected — room has no enemies or combat already active.
	default:
		log.Printf("daemon: dispatchMove: StartCombat error: %v", combatErr)
		response.Warning = "combat failed to start due to an internal error"
	}

	return response, nil
}

// --- look ---

func dispatchLook(eng *engine.Engine, state *model.GameState) (any, *RPCError) {
	look := eng.Look(state)
	result := protocol.LookResult{
		Room:        look.Room,
		Name:        look.Name,
		Description: look.Description,
		Exits:       look.Exits,
		Items:       look.Items,
	}
	if state.Dungeon.Combat.Active {
		result.Combat = currentCombatState(state)
	}
	return result, nil
}

// --- inventory tools ---

type itemArgs struct {
	ItemID string `json:"item_id"`
}

func (a itemArgs) validate() *RPCError {
	if a.ItemID == "" {
		return &RPCError{Code: CodeInvalidParams, Message: "item_id is required"}
	}
	return nil
}

type slotArgs struct {
	Slot string `json:"slot"`
}

func (a slotArgs) validate() *RPCError {
	if a.Slot == "" {
		return &RPCError{Code: CodeInvalidParams, Message: "slot is required"}
	}
	return nil
}

func dispatchPickUp(eng *engine.Engine, state *model.GameState, raw json.RawMessage) (any, *RPCError) {
	if state.Dungeon.Combat.Active {
		return nil, &RPCError{Code: CodeStateBlocked, Message: "cannot pick up items during combat"}
	}
	var a itemArgs
	if err := json.Unmarshal(raw, &a); err != nil {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid arguments: " + err.Error()}
	}
	if rpcErr := a.validate(); rpcErr != nil {
		return nil, rpcErr
	}
	result, err := eng.PickUp(state, a.ItemID)
	if err != nil {
		return nil, engineError(err)
	}
	return protocol.PickUpResult{
		Item: result.Item,
		Hero: heroSummary(state),
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
	if rpcErr := a.validate(); rpcErr != nil {
		return nil, rpcErr
	}
	result, err := eng.Drop(state, a.ItemID)
	if err != nil {
		return nil, engineError(err)
	}
	return protocol.PickUpResult{
		Item: result.Item,
		Hero: heroSummary(state),
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
	if rpcErr := a.validate(); rpcErr != nil {
		return nil, rpcErr
	}
	result, err := eng.Equip(state, a.ItemID)
	if err != nil {
		return nil, engineError(err)
	}
	return protocol.EquipResult{
		Item: result.Item,
		Slot: result.Slot,
		Hero: heroSummary(state),
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
	if rpcErr := a.validate(); rpcErr != nil {
		return nil, rpcErr
	}
	result, err := eng.Unequip(state, a.Slot)
	if err != nil {
		return nil, engineError(err)
	}
	return protocol.EquipResult{
		Item: result.Item,
		Slot: result.Slot,
		Hero: heroSummary(state),
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
	if rpcErr := a.validate(); rpcErr != nil {
		return nil, rpcErr
	}
	result, err := eng.Examine(state, a.ItemID)
	if err != nil {
		return nil, engineError(err)
	}
	return protocol.ExamineResult{Item: result.Item}, nil
}

func dispatchInventory(eng *engine.Engine, state *model.GameState) (any, *RPCError) {
	result := eng.Inventory(state)
	return protocol.InventoryResult{
		Items:    result.Items,
		Equipped: result.Equipped,
		Weight:   result.Weight,
		Capacity: result.Capacity,
	}, nil
}

// --- combat tools ---

type attackArgs struct {
	TargetID string `json:"target_id"`
}

// validate has no required fields: target_id is optional and defaults to the
// first alive enemy. It exists so every dispatcher arg struct validates the
// same way.
func (a attackArgs) validate() *RPCError { return nil }

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
	if rpcErr := a.validate(); rpcErr != nil {
		return nil, rpcErr
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

	response := protocol.AttackResult{
		Target: result.Target,
		Damage: result.Damage,
		Killed: result.Killed,
		Hero:   heroSummary(state),
	}

	if result.Killed {
		xp := result.XPAwarded
		response.XPAwarded = &xp
	}

	if result.CombatOver {
		response.CombatOutcome = &protocol.CombatOutcome{
			CombatOver: true,
			LevelUp:    checkLevelUp(eng, state),
		}
	} else if state.Dungeon.Combat.Active && !isHeroTurn(state) {
		turns := processEnemyTurns(eng, state)
		response.EnemyTurns = &turns
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

	response := protocol.DefendResult{
		Defending: true,
		Hero:      heroSummary(state),
	}

	if state.Dungeon.Combat.Active && !isHeroTurn(state) {
		turns := processEnemyTurns(eng, state)
		response.EnemyTurns = &turns
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

	response := protocol.FleeResult{
		Success: result.Success,
		Hero:    heroSummary(state),
	}

	if !result.Success && state.Dungeon.Combat.Active && !isHeroTurn(state) {
		turns := processEnemyTurns(eng, state)
		response.EnemyTurns = &turns
	}

	return response, nil
}

type castSpellArgs struct {
	SpellID  string `json:"spell_id"`
	TargetID string `json:"target_id"`
}

func (a castSpellArgs) validate() *RPCError {
	if a.SpellID == "" {
		return &RPCError{Code: CodeInvalidParams, Message: "spell_id is required"}
	}
	return nil
}

func dispatchCastSpell(eng *engine.Engine, state *model.GameState, raw json.RawMessage) (any, *RPCError) {
	var a castSpellArgs
	if err := json.Unmarshal(raw, &a); err != nil {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid arguments: " + err.Error()}
	}
	if rpcErr := a.validate(); rpcErr != nil {
		return nil, rpcErr
	}

	targetID := a.TargetID
	if targetID == "" && state.Dungeon.Combat.Active {
		targetID = eng.FirstAliveEnemy(state)
	}

	result, err := eng.CastSpell(state, a.SpellID, targetID)
	if err != nil {
		return nil, engineError(err)
	}

	response := protocol.CastSpellResult{
		Spell:  result.SpellName,
		Effect: result.Effect,
		Power:  result.Power,
		MPCost: result.MPCost,
		Hero:   heroSummary(state),
	}

	if result.Effect == "damage" {
		target := result.TargetID
		response.Target = &target
	}

	if state.Dungeon.Combat.Active && !isHeroTurn(state) {
		turns := processEnemyTurns(eng, state)
		response.EnemyTurns = &turns
	} else if result.Effect == "damage" && !state.Dungeon.Combat.Active {
		response.CombatOutcome = &protocol.CombatOutcome{
			CombatOver: true,
			LevelUp:    checkLevelUp(eng, state),
		}
	}

	return response, nil
}

// --- use_item ---

func dispatchUseItem(eng *engine.Engine, state *model.GameState, raw json.RawMessage) (any, *RPCError) {
	var a itemArgs
	if err := json.Unmarshal(raw, &a); err != nil {
		return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid arguments: " + err.Error()}
	}
	if rpcErr := a.validate(); rpcErr != nil {
		return nil, rpcErr
	}
	result, err := eng.UseItem(state, a.ItemID)
	if err != nil {
		return nil, engineError(err)
	}
	return protocol.UseItemResult{
		Item:   result.Item.Name,
		Effect: result.Effect,
		Power:  result.Power,
		HeroHP: result.HeroHP,
	}, nil
}

// --- context ---

func dispatchContext(eng *engine.Engine, state *model.GameState) (any, *RPCError) {
	sc := eng.Scenario()

	// Current room info.
	currentRoomID := state.Dungeon.CurrentRoom
	currentRoom := protocol.CurrentRoomContext{ID: currentRoomID}
	if room, ok := sc.Rooms[currentRoomID]; ok {
		detail := &protocol.RoomContextDetail{
			Name:            room.Name,
			DescriptionSeed: room.DescriptionSeed,
		}

		exits := make([]string, 0, len(room.Connections))
		for dir, conn := range room.Connections {
			if conn.Type != "hidden" {
				exits = append(exits, dir)
			}
		}
		sort.Strings(exits)
		detail.Exits = exits

		rs := eng.EnsureRoomState(state, currentRoomID)
		detail.Items = rs.Items

		// Active enemies in combat, or scenario enemies for the room.
		if state.Dungeon.Combat.Active {
			var enemies []string
			for _, e := range state.Dungeon.Combat.Enemies {
				if e.HP > 0 {
					enemies = append(enemies, e.ID)
				}
			}
			detail.Enemies = enemies
		} else if rs.Cleared {
			detail.Enemies = []string{}
		} else {
			detail.Enemies = room.Enemies
		}
		currentRoom.RoomContextDetail = detail
	}

	// Adjacent rooms.
	var adjacentRooms []protocol.AdjacentRoom
	if room, ok := sc.Rooms[currentRoomID]; ok {
		for dir, conn := range room.Connections {
			if conn.Type == "hidden" {
				continue
			}
			adj := protocol.AdjacentRoom{
				ID:        conn.Room,
				Direction: dir,
				Visited:   containsString(state.Dungeon.VisitedRooms, conn.Room),
			}
			if adjRoom, ok := sc.Rooms[conn.Room]; ok {
				adj.AdjacentRoomDetail = &protocol.AdjacentRoomDetail{
					Name:            adjRoom.Name,
					DescriptionSeed: adjRoom.DescriptionSeed,
				}
			}
			adjacentRooms = append(adjacentRooms, adj)
		}
		// Sort by direction for deterministic output.
		sort.Slice(adjacentRooms, func(i, j int) bool {
			return adjacentRooms[i].Direction < adjacentRooms[j].Direction
		})
	}

	// Hero summary with context-only inventory and equipped details.
	hero := protocol.HeroContext{HeroSummary: heroSummary(state)}
	if len(state.Party) > 0 {
		h := &state.Party[0]
		var invNames []string
		for _, item := range h.Inventory {
			invNames = append(invNames, item.ID)
		}
		hero.Inventory = invNames
		hero.Equipped = h.Equipped
	}

	return protocol.ContextResult{
		Scenario: protocol.ScenarioContext{
			Title:        sc.Title,
			StartingRoom: sc.StartingRoom,
			DeathMode:    sc.Death,
		},
		CurrentRoom:   currentRoom,
		AdjacentRooms: adjacentRooms,
		Hero:          hero,
		VisitedRooms:  state.Dungeon.VisitedRooms,
		RoomsTotal:    len(sc.Rooms),
		CombatActive:  state.Dungeon.Combat.Active,
	}, nil
}

// containsString checks whether s is in the slice.
func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// --- save/load ---

type saveLoadArgs struct {
	Slot string `json:"slot"`
}

// validate has no required fields: an empty slot is normalised to the
// quicksave slot by the engine. It exists so every dispatcher arg struct
// validates the same way.
func (a saveLoadArgs) validate() *RPCError { return nil }

func dispatchSaveGame(eng *engine.Engine, state *model.GameState, raw json.RawMessage) (any, *RPCError) {
	var a saveLoadArgs
	if raw != nil {
		if err := json.Unmarshal(raw, &a); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid arguments: " + err.Error()}
		}
	}
	if rpcErr := a.validate(); rpcErr != nil {
		return nil, rpcErr
	}
	result, err := eng.SaveGame(state, a.Slot)
	if err != nil {
		return nil, engineError(err)
	}
	return protocol.SaveResult{Slot: result.Slot, Path: result.Path}, nil
}

func dispatchLoadGame(eng *engine.Engine, state *model.GameState, raw json.RawMessage) (any, *RPCError) {
	var a saveLoadArgs
	if raw != nil {
		if err := json.Unmarshal(raw, &a); err != nil {
			return nil, &RPCError{Code: CodeInvalidParams, Message: "invalid arguments: " + err.Error()}
		}
	}
	if rpcErr := a.validate(); rpcErr != nil {
		return nil, rpcErr
	}
	loaded, result, err := eng.LoadGame(a.Slot)
	if err != nil {
		return nil, engineError(err)
	}
	// A valid game always has exactly one hero. A save with an empty party
	// is corrupt; reject it loudly rather than install a heroless state that
	// would panic on the first gameplay command.
	if len(loaded.Party) == 0 {
		return nil, &RPCError{Code: CodeInternalError, Message: "loaded save has no hero"}
	}
	*state = loaded
	look := eng.Look(state)
	return protocol.LoadResult{
		Slot:        result.Slot,
		Room:        look.Room,
		Name:        look.Name,
		Description: look.Description,
		Exits:       look.Exits,
		Items:       look.Items,
		Hero:        heroSummary(state),
	}, nil
}

// --- helpers ---

// processEnemyTurns runs all enemy turns until it's the hero's turn (or combat ends).
func processEnemyTurns(eng *engine.Engine, state *model.GameState) []protocol.EnemyTurn {
	var turns []protocol.EnemyTurn
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
		turn := protocol.EnemyTurn{
			Enemy:  result.EnemyName,
			Action: result.Action,
		}
		if result.Action == "attack" {
			dmg, hp := result.Damage, result.HeroHP
			turn.Damage = &dmg
			turn.HeroHP = &hp
			turn.HeroDead = result.HeroDead
		}
		turns = append(turns, turn)
	}
	return turns
}

// checkLevelUp checks and applies a level-up, returning the summary or nil.
func checkLevelUp(eng *engine.Engine, state *model.GameState) *protocol.LevelUpSummary {
	result := eng.CheckLevelUp(state)
	if !result.Leveled {
		return nil
	}
	return &protocol.LevelUpSummary{
		NewLevel: result.NewLevel,
		HPGain:   result.HPGain,
		MPGain:   result.MPGain,
	}
}

// heroSummary returns a snapshot of the hero for inclusion in tool results.
// Every active game has exactly one hero (engine.NewGame seeds it; load
// rejects a heroless save), so an empty party is a programmer bug, not a
// runtime condition.
func heroSummary(state *model.GameState) protocol.HeroSummary {
	if len(state.Party) == 0 {
		panic("daemon: heroSummary called with empty party")
	}
	h := &state.Party[0]
	return protocol.HeroSummary{
		Name:  h.Name,
		Class: h.Class,
		Level: h.Level,
		HP:    h.HP,
		MaxHP: h.MaxHP,
		MP:    h.MP,
		MaxMP: h.MaxMP,
		XP:    h.XP,
	}
}

// combatSummary builds a combat summary from a StartCombat result.
func combatSummary(result engine.CombatStartResult, state *model.GameState) *protocol.CombatSummary {
	var enemies []protocol.CombatEnemy
	for _, e := range result.Enemies {
		enemies = append(enemies, protocol.CombatEnemy{ID: e.ID, Name: e.Name, HP: e.HP, MaxHP: e.MaxHP})
	}
	return &protocol.CombatSummary{
		Enemies:   enemies,
		TurnOrder: result.TurnOrder,
		HeroTurn:  isHeroTurn(state),
	}
}

// currentCombatState returns the current combat state for look results.
func currentCombatState(state *model.GameState) *protocol.CombatView {
	var enemies []protocol.CombatEnemy
	for _, e := range state.Dungeon.Combat.Enemies {
		if e.HP > 0 {
			enemies = append(enemies, protocol.CombatEnemy{ID: e.ID, Name: e.Name, HP: e.HP, MaxHP: e.MaxHP})
		}
	}
	return &protocol.CombatView{
		Enemies:  enemies,
		Round:    state.Dungeon.Combat.Round,
		HeroTurn: isHeroTurn(state),
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
