// Package engine implements the deterministic game rules machine.
// It knows nothing about interpreters, narrators, or renderers.
package engine

import (
	"fmt"
	"sort"
	"time"

	"github.com/punt-labs/cryptd/internal/dice"
	"github.com/punt-labs/cryptd/internal/model"
	"github.com/punt-labs/cryptd/internal/scenario"
)

// MoveResult holds the outcome of a successful move action.
type MoveResult struct {
	NewRoom string
	Exits   []string
	Items   []string
	Enemies []string
}

// LookResult holds the outcome of a look action.
type LookResult struct {
	Room        string
	Name        string
	Description string
	Exits       []string
	Items       []string
	Enemies     []string
}

// NoExitError is returned when the player moves in a direction with no connection.
type NoExitError struct {
	Direction string
}

func (e *NoExitError) Error() string {
	return fmt.Sprintf("no exit to the %s", e.Direction)
}

// LockedError is returned when the player tries to move through a locked door.
type LockedError struct {
	Direction string
	Room      string
}

func (e *LockedError) Error() string {
	return fmt.Sprintf("the way %s to %s is locked", e.Direction, e.Room)
}

// PickUpResult holds the outcome of a successful pick-up action.
type PickUpResult struct {
	Item model.Item
}

// DropResult holds the outcome of a successful drop action.
type DropResult struct {
	Item model.Item
}

// EquipResult holds the outcome of a successful equip action.
type EquipResult struct {
	Item model.Item
	Slot string
}

// UnequipResult holds the outcome of a successful unequip action.
type UnequipResult struct {
	Item model.Item
	Slot string
}

// ExamineResult holds item details returned by Examine.
type ExamineResult struct {
	Item model.Item
}

// InventoryResult holds the character's current inventory and equipment.
type InventoryResult struct {
	Items    []model.Item
	Equipped model.Equipment
	Weight   float64
	Capacity float64
}

// ItemNotInRoomError is returned when the player tries to pick up an item not in the room.
type ItemNotInRoomError struct {
	ItemID string
}

func (e *ItemNotInRoomError) Error() string {
	return fmt.Sprintf("no item %q here", e.ItemID)
}

// ItemNotInInventoryError is returned when referencing an item not in inventory.
type ItemNotInInventoryError struct {
	ItemID string
}

func (e *ItemNotInInventoryError) Error() string {
	return fmt.Sprintf("you don't have %q", e.ItemID)
}

// TooHeavyError is returned when picking up an item would exceed carry weight.
type TooHeavyError struct {
	ItemID   string
	Weight   float64
	Capacity float64
	Current  float64
}

func (e *TooHeavyError) Error() string {
	return fmt.Sprintf("picking up %q (%.1f) would exceed carry limit (%.1f/%.1f)", e.ItemID, e.Weight, e.Current, e.Capacity)
}

// NotEquippableError is returned when trying to equip a non-equippable item.
type NotEquippableError struct {
	ItemID   string
	ItemType string
}

func (e *NotEquippableError) Error() string {
	return fmt.Sprintf("cannot equip %q (type %s)", e.ItemID, e.ItemType)
}

// SlotOccupiedError is returned when the equipment slot already holds an item.
type SlotOccupiedError struct {
	Slot       string
	OccupiedBy string
}

func (e *SlotOccupiedError) Error() string {
	return fmt.Sprintf("slot %s already occupied by %q", e.Slot, e.OccupiedBy)
}

// SlotEmptyError is returned when trying to unequip from an empty slot.
type SlotEmptyError struct {
	Slot string
}

func (e *SlotEmptyError) Error() string {
	return fmt.Sprintf("nothing equipped in %s slot", e.Slot)
}

// Engine is the deterministic rules machine. All game state transitions go
// through Engine methods. The Engine holds the scenario but never mutates it.
type Engine struct {
	s       *scenario.Spec
	Now     func() time.Time // injectable clock; defaults to time.Now
	SaveDir string           // injectable save directory; defaults to ".dungeon/saves"
}

// New creates an Engine for the given scenario.
func New(s *scenario.Spec) *Engine {
	return &Engine{s: s, Now: time.Now}
}

// NewGame initialises a fresh GameState for the scenario and character.
// PlayMode is left empty; the composition layer (cmd or game.Loop) sets it.
// Room items are seeded from the scenario into mutable RoomState.
func (e *Engine) NewGame(char model.Character) (model.GameState, error) {
	roomState := make(map[string]model.RoomState, len(e.s.Rooms))
	for id, room := range e.s.Rooms {
		rs := model.RoomState{}
		if len(room.Items) > 0 {
			rs.Items = make([]string, len(room.Items))
			copy(rs.Items, room.Items)
		}
		roomState[id] = rs
	}
	state := model.GameState{
		Scenario: e.s.ID,
		Dungeon: model.DungeonState{
			CurrentRoom:  e.s.StartingRoom,
			VisitedRooms: []string{e.s.StartingRoom},
			RoomState:    roomState,
		},
		Party: []model.Character{char},
	}
	return state, nil
}

// Move executes a move in the given direction, mutating state in place.
// Returns a MoveResult on success, or NoExitError / LockedError on failure.
func (e *Engine) Move(state *model.GameState, direction string) (MoveResult, error) {
	room, ok := e.s.Rooms[state.Dungeon.CurrentRoom]
	if !ok {
		return MoveResult{}, fmt.Errorf("current room %q not found in scenario", state.Dungeon.CurrentRoom)
	}

	conn, ok := room.Connections[direction]
	if !ok {
		return MoveResult{}, &NoExitError{Direction: direction}
	}

	switch conn.Type {
	case "locked":
		return MoveResult{}, &LockedError{Direction: direction, Room: conn.Room}
	case "hidden":
		// Hidden connections are undiscoverable until revealed; treat as no exit.
		return MoveResult{}, &NoExitError{Direction: direction}
	case "open", "stairway":
		// traversable — continue
	default:
		return MoveResult{}, fmt.Errorf("unknown connection type %q to the %s", conn.Type, direction)
	}

	// Validate destination before mutating state so a broken scenario ref
	// cannot corrupt the in-progress game state.
	dest, ok := e.s.Rooms[conn.Room]
	if !ok {
		return MoveResult{}, fmt.Errorf("destination room %q not found in scenario", conn.Room)
	}

	state.Dungeon.CurrentRoom = conn.Room
	state.Dungeon.VisitedRooms = appendUnique(state.Dungeon.VisitedRooms, conn.Room)

	state.AdventureLog = append(state.AdventureLog, model.LogEntry{
		Text:      fmt.Sprintf("You move %s and enter %s.", direction, conn.Room),
		Timestamp: e.Now().UTC().Format(time.RFC3339),
	})

	return MoveResult{
		NewRoom: conn.Room,
		Exits:   exitList(dest),
		Items:   e.ensureRoomState(state, conn.Room).Items,
		Enemies: dest.Enemies,
	}, nil
}

// Look returns information about the current room without mutating state.
func (e *Engine) Look(state *model.GameState) LookResult {
	room, ok := e.s.Rooms[state.Dungeon.CurrentRoom]
	if !ok {
		return LookResult{Room: state.Dungeon.CurrentRoom}
	}
	itemIDs := e.ensureRoomState(state, state.Dungeon.CurrentRoom).Items
	itemNames := make([]string, len(itemIDs))
	for i, id := range itemIDs {
		if it, ok := e.s.Items[id]; ok {
			itemNames[i] = it.Name
		} else {
			itemNames[i] = id
		}
	}
	return LookResult{
		Room:        state.Dungeon.CurrentRoom,
		Name:        room.Name,
		Description: room.DescriptionSeed,
		Exits:       exitList(room),
		Items:       itemNames,
		Enemies:     room.Enemies,
	}
}

func exitList(room *scenario.Room) []string {
	exits := make([]string, 0, len(room.Connections))
	for dir, conn := range room.Connections {
		// Hidden connections are undiscoverable until revealed; omit from exits.
		if conn.Type == "hidden" {
			continue
		}
		exits = append(exits, dir)
	}
	sort.Strings(exits)
	return exits
}

func appendUnique(slice []string, s string) []string {
	for _, v := range slice {
		if v == s {
			return slice
		}
	}
	return append(slice, s)
}

// hero returns a pointer to the first party member (single-player, DES-021).
func hero(state *model.GameState) *model.Character {
	return &state.Party[0]
}

// totalCarryWeight returns the total weight of all carried items (inventory + equipped).
func totalCarryWeight(char *model.Character, s *scenario.Spec) float64 {
	var w float64
	for _, item := range char.Inventory {
		w += item.Weight
	}
	for _, id := range equippedIDs(char) {
		if si, ok := s.Items[id]; ok {
			w += si.Weight
		}
	}
	return w
}

// equippedIDs returns the non-empty item IDs from all equipment slots.
func equippedIDs(char *model.Character) []string {
	var ids []string
	for _, id := range []string{char.Equipped.Weapon, char.Equipped.Armor, char.Equipped.Ring, char.Equipped.Amulet} {
		if id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

// containsItem checks whether itemID is in the items slice.
func containsItem(items []string, itemID string) bool {
	for _, id := range items {
		if id == itemID {
			return true
		}
	}
	return false
}

// ensureRoomState returns the RoomState for the given room, seeding from the
// scenario if no entry exists (handles older save files that lack item state).
func (e *Engine) ensureRoomState(state *model.GameState, roomID string) model.RoomState {
	if rs, ok := state.Dungeon.RoomState[roomID]; ok {
		return rs
	}
	rs := model.RoomState{}
	if room, ok := e.s.Rooms[roomID]; ok && len(room.Items) > 0 {
		rs.Items = make([]string, len(room.Items))
		copy(rs.Items, room.Items)
	}
	if state.Dungeon.RoomState == nil {
		state.Dungeon.RoomState = make(map[string]model.RoomState)
	}
	state.Dungeon.RoomState[roomID] = rs
	return rs
}

// scenarioItem converts a scenario item definition to a model.Item.
func scenarioItem(id string, si *scenario.Item) model.Item {
	return model.Item{
		ID:          id,
		Name:        si.Name,
		Type:        si.Type,
		Damage:      si.Damage,
		Defense:     si.Defense,
		Power:       si.Power,
		Effect:      si.Effect,
		Weight:      si.Weight,
		Value:       si.Value,
		Description: si.Description,
	}
}

// removeItem removes the first occurrence of itemID from a string slice,
// returning the modified slice and whether the item was found.
func removeItem(items []string, itemID string) ([]string, bool) {
	for i, id := range items {
		if id == itemID {
			return append(items[:i], items[i+1:]...), true
		}
	}
	return items, false
}

// PickUp removes an item from the current room and adds it to the character's inventory.
func (e *Engine) PickUp(state *model.GameState, itemID string) (PickUpResult, error) {
	rs := e.ensureRoomState(state, state.Dungeon.CurrentRoom)
	if !containsItem(rs.Items, itemID) {
		return PickUpResult{}, &ItemNotInRoomError{ItemID: itemID}
	}

	si, ok := e.s.Items[itemID]
	if !ok {
		return PickUpResult{}, fmt.Errorf("item %q not defined in scenario", itemID)
	}

	// Check weight before mutating state so a failed pickup leaves the room intact.
	char := hero(state)
	current := totalCarryWeight(char, e.s)
	if current+si.Weight > model.MaxCarryWeight {
		return PickUpResult{}, &TooHeavyError{
			ItemID: itemID, Weight: si.Weight,
			Capacity: model.MaxCarryWeight, Current: current,
		}
	}

	// All checks passed — now mutate state.
	newItems, _ := removeItem(rs.Items, itemID)
	rs.Items = newItems
	state.Dungeon.RoomState[state.Dungeon.CurrentRoom] = rs

	item := scenarioItem(itemID, si)
	char.Inventory = append(char.Inventory, item)

	state.AdventureLog = append(state.AdventureLog, model.LogEntry{
		Text:      fmt.Sprintf("You pick up %s.", si.Name),
		Timestamp: e.Now().UTC().Format(time.RFC3339),
	})

	return PickUpResult{Item: item}, nil
}

// Drop removes an item from inventory and places it in the current room.
func (e *Engine) Drop(state *model.GameState, itemID string) (DropResult, error) {
	char := hero(state)
	idx := -1
	for i, item := range char.Inventory {
		if item.ID == itemID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return DropResult{}, &ItemNotInInventoryError{ItemID: itemID}
	}

	item := char.Inventory[idx]
	char.Inventory = append(char.Inventory[:idx], char.Inventory[idx+1:]...)

	rs := e.ensureRoomState(state, state.Dungeon.CurrentRoom)
	rs.Items = append(rs.Items, itemID)
	state.Dungeon.RoomState[state.Dungeon.CurrentRoom] = rs

	state.AdventureLog = append(state.AdventureLog, model.LogEntry{
		Text:      fmt.Sprintf("You drop %s.", item.Name),
		Timestamp: e.Now().UTC().Format(time.RFC3339),
	})

	return DropResult{Item: item}, nil
}

// slotForType maps item types to equipment slot names.
var slotForType = map[string]string{
	"weapon": "weapon",
	"armor":  "armor",
	"ring":   "ring",
	"amulet": "amulet",
}

// getSlot reads the equipment slot value for the given slot name.
func getSlot(eq *model.Equipment, slot string) string {
	switch slot {
	case "weapon":
		return eq.Weapon
	case "armor":
		return eq.Armor
	case "ring":
		return eq.Ring
	case "amulet":
		return eq.Amulet
	default:
		return ""
	}
}

// setSlot writes an item ID to the named equipment slot.
func setSlot(eq *model.Equipment, slot, itemID string) {
	switch slot {
	case "weapon":
		eq.Weapon = itemID
	case "armor":
		eq.Armor = itemID
	case "ring":
		eq.Ring = itemID
	case "amulet":
		eq.Amulet = itemID
	}
}

// Equip moves an item from inventory to the appropriate equipment slot.
func (e *Engine) Equip(state *model.GameState, itemID string) (EquipResult, error) {
	char := hero(state)
	idx := -1
	for i, item := range char.Inventory {
		if item.ID == itemID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return EquipResult{}, &ItemNotInInventoryError{ItemID: itemID}
	}

	item := char.Inventory[idx]
	slot, ok := slotForType[item.Type]
	if !ok {
		return EquipResult{}, &NotEquippableError{ItemID: itemID, ItemType: item.Type}
	}

	if existing := getSlot(&char.Equipped, slot); existing != "" {
		return EquipResult{}, &SlotOccupiedError{Slot: slot, OccupiedBy: existing}
	}

	char.Inventory = append(char.Inventory[:idx], char.Inventory[idx+1:]...)
	setSlot(&char.Equipped, slot, itemID)

	state.AdventureLog = append(state.AdventureLog, model.LogEntry{
		Text:      fmt.Sprintf("You equip %s.", item.Name),
		Timestamp: e.Now().UTC().Format(time.RFC3339),
	})

	return EquipResult{Item: item, Slot: slot}, nil
}

// Unequip moves an item from an equipment slot back to inventory.
func (e *Engine) Unequip(state *model.GameState, slot string) (UnequipResult, error) {
	char := hero(state)
	itemID := getSlot(&char.Equipped, slot)
	if itemID == "" {
		return UnequipResult{}, &SlotEmptyError{Slot: slot}
	}

	si, ok := e.s.Items[itemID]
	if !ok {
		return UnequipResult{}, fmt.Errorf("equipped item %q not defined in scenario", itemID)
	}

	item := scenarioItem(itemID, si)
	setSlot(&char.Equipped, slot, "")
	char.Inventory = append(char.Inventory, item)

	state.AdventureLog = append(state.AdventureLog, model.LogEntry{
		Text:      fmt.Sprintf("You unequip %s.", item.Name),
		Timestamp: e.Now().UTC().Format(time.RFC3339),
	})

	return UnequipResult{Item: item, Slot: slot}, nil
}

// UseItemResult holds the outcome of using a consumable item.
type UseItemResult struct {
	Item   model.Item
	Effect string // "heal"
	Power  int    // amount healed/damaged
	HeroHP int    // HP after effect
}

// NotConsumableError is returned when trying to use a non-consumable item.
type NotConsumableError struct {
	ItemID   string
	ItemType string
}

func (e *NotConsumableError) Error() string {
	return fmt.Sprintf("cannot use %q (type %s)", e.ItemID, e.ItemType)
}

// UseItem consumes an item from inventory, applying its effect.
func (e *Engine) UseItem(state *model.GameState, itemID string) (UseItemResult, error) {
	char := hero(state)
	idx := -1
	for i, item := range char.Inventory {
		if item.ID == itemID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return UseItemResult{}, &ItemNotInInventoryError{ItemID: itemID}
	}

	item := char.Inventory[idx]
	if item.Type != "consumable" {
		return UseItemResult{}, &NotConsumableError{ItemID: itemID, ItemType: item.Type}
	}

	// Apply effect.
	var power int
	switch item.Effect {
	case "heal":
		if item.Power == "" {
			return UseItemResult{}, fmt.Errorf("consumable %q has no power defined", item.ID)
		}
		d, err := dice.Parse(item.Power)
		if err != nil {
			return UseItemResult{}, fmt.Errorf("invalid power dice %q: %w", item.Power, err)
		}
		power = d.Roll()
		if power < 1 {
			power = 1
		}
		char.HP += power
		if char.HP > char.MaxHP {
			char.HP = char.MaxHP
		}
	default:
		return UseItemResult{}, fmt.Errorf("unknown consumable effect %q", item.Effect)
	}

	// Remove from inventory (consumed).
	char.Inventory = append(char.Inventory[:idx], char.Inventory[idx+1:]...)

	state.AdventureLog = append(state.AdventureLog, model.LogEntry{
		Text:      fmt.Sprintf("You use %s.", item.Name),
		Timestamp: e.Now().UTC().Format(time.RFC3339),
	})

	return UseItemResult{
		Item:   item,
		Effect: item.Effect,
		Power:  power,
		HeroHP: char.HP,
	}, nil
}

// Examine returns details about an item in inventory, equipped, or the current room.
func (e *Engine) Examine(state *model.GameState, itemID string) (ExamineResult, error) {
	char := hero(state)

	// Check inventory.
	for _, item := range char.Inventory {
		if item.ID == itemID {
			return ExamineResult{Item: item}, nil
		}
	}

	// Check equipped items.
	for _, id := range equippedIDs(char) {
		if id == itemID {
			si, ok := e.s.Items[id]
			if !ok {
				return ExamineResult{}, fmt.Errorf("equipped item %q not defined in scenario", id)
			}
			return ExamineResult{Item: scenarioItem(id, si)}, nil
		}
	}

	// Check room.
	rs := e.ensureRoomState(state, state.Dungeon.CurrentRoom)
	for _, id := range rs.Items {
		if id == itemID {
			si, ok := e.s.Items[id]
			if !ok {
				return ExamineResult{}, fmt.Errorf("item %q not defined in scenario", id)
			}
			return ExamineResult{Item: scenarioItem(id, si)}, nil
		}
	}

	return ExamineResult{}, &ItemNotInRoomError{ItemID: itemID}
}

// Inventory returns the hero's current inventory and equipment.
func (e *Engine) Inventory(state *model.GameState) InventoryResult {
	char := hero(state)
	return InventoryResult{
		Items:    char.Inventory,
		Equipped: char.Equipped,
		Weight:   totalCarryWeight(char, e.s),
		Capacity: model.MaxCarryWeight,
	}
}
