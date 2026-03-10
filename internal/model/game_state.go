package model

// GameState is the full, serialisable state of a game session.
// It is written to .dungeon/saves/<slot>.json by internal/save.
type GameState struct {
	SchemaVersion string       `json:"schema_version"`
	PlayMode      string       `json:"play_mode"`
	Scenario      string       `json:"scenario"`
	Timestamp     string       `json:"timestamp"`
	Party         []Character  `json:"party"`
	Dungeon       DungeonState `json:"dungeon"`
	AdventureLog  []LogEntry   `json:"adventure_log"`
}

// DungeonState tracks the player's progress through the dungeon map.
type DungeonState struct {
	CurrentRoom  string               `json:"current_room"`
	VisitedRooms []string             `json:"visited_rooms"`
	RoomState    map[string]RoomState `json:"room_state"`
}

// RoomState holds per-room mutable state (e.g. whether it has been cleared).
type RoomState struct {
	Cleared bool `json:"cleared"`
}

// Character represents one member of the party.
// Party is always []Character, length 1 for single-player (DES-021).
type Character struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Class      string      `json:"class"` // fighter|mage|thief|priest
	Level      int         `json:"level"`
	HP         int         `json:"hp"`
	MaxHP      int         `json:"max_hp"`
	XP         int         `json:"xp"`
	Gold       int         `json:"gold"`
	Stats      Stats       `json:"stats"`
	Inventory  []Item      `json:"inventory"`
	Equipped   Equipment   `json:"equipped"`
	Conditions []Condition `json:"conditions"`
}

// Stats holds the six Wizardry-inspired character attributes (DES-022).
type Stats struct {
	STR int `json:"str"`
	INT int `json:"int"`
	DEX int `json:"dex"`
	CON int `json:"con"`
	WIS int `json:"wis"`
	CHA int `json:"cha"`
}

// Equipment holds the IDs of items occupying each gear slot (DES-022).
type Equipment struct {
	Weapon string `json:"weapon"`
	Armor  string `json:"armor"`
	Ring   string `json:"ring"`
	Amulet string `json:"amulet"`
}

// Condition is a status effect applied to a character.
type Condition struct {
	Name           string `json:"name"` // poisoned|asleep|paralyzed|confused
	TurnsRemaining int    `json:"turns_remaining"`
}

// Item is a single object in a character's inventory or in a room.
type Item struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Type        string  `json:"type"` // weapon|armor|ring|amulet|consumable|key|misc
	Damage      string  `json:"damage,omitempty"`
	Weight      float64 `json:"weight"`
	Value       int     `json:"value"`
	Description string  `json:"description,omitempty"`
}

// LogEntry is one line in the adventure log shown in the narration panel.
type LogEntry struct {
	Text      string `json:"text"`
	Timestamp string `json:"timestamp"`
}
