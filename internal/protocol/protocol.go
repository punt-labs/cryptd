// Package protocol defines the JSON-RPC 2.0 wire types shared between
// the cryptd server and crypt client.
package protocol

import (
	"encoding/json"
	"fmt"

	"github.com/punt-labs/cryptd/internal/model"
)

// JSON-RPC 2.0 wire types.

// Request is a JSON-RPC 2.0 request.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response is a JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCErrorCode is a JSON-RPC 2.0 error code. Standard codes are negative
// integers in the -32768..-32000 range; application codes use -32001..-32099.
type RPCErrorCode int

// RPCError is a JSON-RPC 2.0 error object.
type RPCError struct {
	Code    RPCErrorCode `json:"code"`
	Message string       `json:"message"`
	Data    any          `json:"data,omitempty"`
}

// Error reports the code and message, so *RPCError satisfies the error
// interface. A nil receiver reads as "<nil>" rather than panicking.
func (e *RPCError) Error() string {
	if e == nil {
		return "<nil>"
	}
	return fmt.Sprintf("rpc error %d: %s", e.Code, e.Message)
}

// Standard JSON-RPC error codes.
const (
	CodeParseError     RPCErrorCode = -32700
	CodeInvalidRequest RPCErrorCode = -32600
	CodeMethodNotFound RPCErrorCode = -32601
	CodeInvalidParams  RPCErrorCode = -32602
	CodeInternalError  RPCErrorCode = -32603
)

// Application-specific error codes.
const (
	CodeStateBlocked RPCErrorCode = -32001 // combat/turn/lock constraints
	CodeGameOver     RPCErrorCode = -32002 // hero is dead
	CodeNoActiveGame RPCErrorCode = -32003 // tool called before new_game
)

// PlayRequest is the JSON-RPC params for the "play" method.
type PlayRequest struct {
	Text string `json:"text"`
}

// PlayResponse is the JSON-RPC result for the "play" method.
type PlayResponse struct {
	Text        string           `json:"text"`
	State       *model.GameState `json:"state,omitempty"`
	Dead        bool             `json:"dead,omitempty"`
	Quit        bool             `json:"quit,omitempty"`
	Exits       []string         `json:"exits,omitempty"`         // transient: available exits from current room
	NextLevelXP int              `json:"next_level_xp,omitempty"` // transient: XP needed for next level
}

// InitializeParams is the optional params for the "session.init" method.
type InitializeParams struct {
	SessionID string `json:"session_id,omitempty"`
	Mode      string `json:"mode,omitempty"` // "passthrough" or "" (normal)
}

// InitializeResult is the result of the initialize method.
type InitializeResult struct {
	ProtocolVersion string         `json:"protocolVersion"`
	ServerInfo      ServerInfo     `json:"serverInfo"`
	Capabilities    map[string]any `json:"capabilities"`
	SessionID       string         `json:"session_id"`
	HasGame         bool           `json:"has_game,omitempty"`
}

// ServerInfo identifies the server in the initialize handshake.
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ScenarioInfo describes a single available scenario for the client.
type ScenarioInfo struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

// ListScenariosResult is the result of the game.list_scenarios method.
type ListScenariosResult struct {
	Scenarios []ScenarioInfo `json:"scenarios"`
}

// SessionInfo describes one active game session for the lobby screen.
type SessionInfo struct {
	ID             string `json:"id"`
	ScenarioID     string `json:"scenario_id,omitempty"`
	CharacterName  string `json:"character_name,omitempty"`
	CharacterClass string `json:"character_class,omitempty"`
	Level          int    `json:"level,omitempty"`
	RoomID         string `json:"room_id,omitempty"`
}

// ListSessionsResult is the result of the game.list_sessions method.
type ListSessionsResult struct {
	Sessions []SessionInfo `json:"sessions"`
}

// Tool response bodies. These are the JSON results returned by the
// passthrough dispatcher for each game.* method, shared so the crypt
// client can decode them. Field presence mirrors the former
// map[string]any construction: a plain field is always emitted; a
// pointer or omitempty field appears only when the dispatcher set it.
// An embedded pointer struct is a group of keys that appear together.

// HeroSummary is the hero snapshot embedded in most tool results.
type HeroSummary struct {
	Name  string `json:"name"`
	Class string `json:"class"`
	Level int    `json:"level"`
	HP    int    `json:"hp"`
	MaxHP int    `json:"max_hp"`
	MP    int    `json:"mp"`
	MaxMP int    `json:"max_mp"`
	XP    int    `json:"xp"`
}

// HeroContext is the hero snapshot in a context result: the base summary
// plus the current inventory (item IDs) and equipped slots, which the
// context tool always emits even when empty.
type HeroContext struct {
	HeroSummary
	Inventory []string        `json:"inventory"`
	Equipped  model.Equipment `json:"equipped"`
}

// CombatEnemy is one enemy entry in a combat summary.
type CombatEnemy struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	HP    int    `json:"hp"`
	MaxHP int    `json:"max_hp"`
}

// CombatSummary describes combat as it begins (move result).
type CombatSummary struct {
	Enemies   []CombatEnemy `json:"enemies"`
	TurnOrder []string      `json:"turn_order"`
	HeroTurn  bool          `json:"hero_turn"`
}

// CombatView describes ongoing combat (look result).
type CombatView struct {
	Enemies  []CombatEnemy `json:"enemies"`
	Round    int           `json:"round"`
	HeroTurn bool          `json:"hero_turn"`
}

// EnemyTurn is one enemy's action during the enemy phase. Damage and
// HeroHP are set only for an attack action.
type EnemyTurn struct {
	Enemy    string `json:"enemy"`
	Action   string `json:"action"`
	Damage   *int   `json:"damage,omitempty"`
	HeroHP   *int   `json:"hero_hp,omitempty"`
	HeroDead bool   `json:"hero_dead,omitempty"`
}

// LevelUpSummary reports a level-up after combat.
type LevelUpSummary struct {
	NewLevel int `json:"new_level"`
	HPGain   int `json:"hp_gain"`
	MPGain   int `json:"mp_gain"`
}

// CombatOutcome is the combat_over/level_up pair, emitted together when
// combat ends. LevelUp is null when no level was gained.
type CombatOutcome struct {
	CombatOver bool            `json:"combat_over"`
	LevelUp    *LevelUpSummary `json:"level_up"`
}

// NewGameResult is the game.new result.
type NewGameResult struct {
	Room        string      `json:"room"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Exits       []string    `json:"exits"`
	Items       []string    `json:"items"`
	Hero        HeroSummary `json:"hero"`
}

// MoveResult is the game.move result.
type MoveResult struct {
	Room        string         `json:"room"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Exits       []string       `json:"exits"`
	Items       []string       `json:"items"`
	Combat      *CombatSummary `json:"combat,omitempty"`
	EnemyTurns  *[]EnemyTurn   `json:"enemy_turns,omitempty"`
	Warning     string         `json:"warning,omitempty"`
}

// LookResult is the game.look result.
type LookResult struct {
	Room        string      `json:"room"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Exits       []string    `json:"exits"`
	Items       []string    `json:"items"`
	Combat      *CombatView `json:"combat,omitempty"`
}

// PickUpResult is the game.pick_up and game.drop result.
type PickUpResult struct {
	Item model.Item  `json:"item"`
	Hero HeroSummary `json:"hero"`
}

// EquipResult is the game.equip and game.unequip result.
type EquipResult struct {
	Item model.Item  `json:"item"`
	Slot string      `json:"slot"`
	Hero HeroSummary `json:"hero"`
}

// ExamineResult is the game.examine result.
type ExamineResult struct {
	Item model.Item `json:"item"`
}

// InventoryResult is the game.inventory result.
type InventoryResult struct {
	Items    []model.Item    `json:"items"`
	Equipped model.Equipment `json:"equipped"`
	Weight   float64         `json:"weight"`
	Capacity float64         `json:"capacity"`
}

// AttackResult is the game.attack result.
type AttackResult struct {
	Target    string      `json:"target"`
	Damage    int         `json:"damage"`
	Killed    bool        `json:"killed"`
	Hero      HeroSummary `json:"hero"`
	XPAwarded *int        `json:"xp_awarded,omitempty"`
	*CombatOutcome
	EnemyTurns *[]EnemyTurn `json:"enemy_turns,omitempty"`
	Warning    string       `json:"warning,omitempty"`
}

// DefendResult is the game.defend result.
type DefendResult struct {
	Defending  bool         `json:"defending"`
	Hero       HeroSummary  `json:"hero"`
	EnemyTurns *[]EnemyTurn `json:"enemy_turns,omitempty"`
	Warning    string       `json:"warning,omitempty"`
}

// FleeResult is the game.flee result.
type FleeResult struct {
	Success    bool         `json:"success"`
	Hero       HeroSummary  `json:"hero"`
	EnemyTurns *[]EnemyTurn `json:"enemy_turns,omitempty"`
	Warning    string       `json:"warning,omitempty"`
}

// CastSpellResult is the game.cast_spell result. Target is set only for a
// damage spell.
type CastSpellResult struct {
	Spell  string      `json:"spell"`
	Effect string      `json:"effect"`
	Power  int         `json:"power"`
	MPCost int         `json:"mp_cost"`
	Hero   HeroSummary `json:"hero"`
	Target *string     `json:"target,omitempty"`
	*CombatOutcome
	EnemyTurns *[]EnemyTurn `json:"enemy_turns,omitempty"`
	Warning    string       `json:"warning,omitempty"`
}

// UseItemResult is the game.use_item result. Item is the item's display name.
type UseItemResult struct {
	Item   string `json:"item"`
	Effect string `json:"effect"`
	Power  int    `json:"power"`
	HeroHP int    `json:"hero_hp"`
}

// SaveResult is the game.save_game result.
type SaveResult struct {
	Slot string `json:"slot"`
	Path string `json:"path"`
}

// LoadResult is the game.load_game result.
type LoadResult struct {
	Slot        string      `json:"slot"`
	Room        string      `json:"room"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Exits       []string    `json:"exits"`
	Items       []string    `json:"items"`
	Hero        HeroSummary `json:"hero"`
}

// ScenarioContext is the scenario-level info in a context result.
type ScenarioContext struct {
	Title        string `json:"title"`
	StartingRoom string `json:"starting_room"`
	DeathMode    string `json:"death_mode"`
}

// RoomContextDetail holds the current-room fields present when the room is
// found in the scenario; absent for the id-only fallback.
type RoomContextDetail struct {
	Name            string   `json:"name"`
	DescriptionSeed string   `json:"description_seed"`
	Exits           []string `json:"exits"`
	Items           []string `json:"items"`
	Enemies         []string `json:"enemies"`
}

// CurrentRoomContext is the current_room field of a context result.
type CurrentRoomContext struct {
	ID string `json:"id"`
	*RoomContextDetail
}

// AdjacentRoomDetail holds the adjacent-room fields present when that room
// is found in the scenario.
type AdjacentRoomDetail struct {
	Name            string `json:"name"`
	DescriptionSeed string `json:"description_seed"`
}

// AdjacentRoom is one entry in a context result's adjacent_rooms list.
type AdjacentRoom struct {
	ID        string `json:"id"`
	Direction string `json:"direction"`
	Visited   bool   `json:"visited"`
	*AdjacentRoomDetail
}

// ContextResult is the game.context result.
type ContextResult struct {
	Scenario      ScenarioContext    `json:"scenario"`
	CurrentRoom   CurrentRoomContext `json:"current_room"`
	AdjacentRooms []AdjacentRoom     `json:"adjacent_rooms"`
	Hero          HeroContext        `json:"hero"`
	VisitedRooms  []string           `json:"visited_rooms"`
	RoomsTotal    int                `json:"rooms_total"`
	CombatActive  bool               `json:"combat_active"`
}
