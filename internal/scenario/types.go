package scenario

// Scenario is the top-level structure of a parsed scenario YAML file.
// Field names match the DES-016 YAML schema.
type Scenario struct {
	// ID is derived from the filename (without extension).
	ID    string
	Title string `yaml:"title"`
	// StartingRoom is the key of the room where the adventure begins.
	StartingRoom string                       `yaml:"starting_room"`
	Death        string                       `yaml:"death"` // permadeath|respawn
	Rooms        map[string]*Room             `yaml:"rooms"`
	Items        map[string]*ScenarioItem     `yaml:"items"`
	Enemies      map[string]*EnemyTemplate    `yaml:"enemies"`
	Spells       map[string]*SpellTemplate    `yaml:"spells"`
}

// Room is a single location in the dungeon.
type Room struct {
	Name            string                 `yaml:"name"`
	DescriptionSeed string                 `yaml:"description_seed"`
	Connections     map[string]*Connection `yaml:"connections"`
	// Items lists item IDs available in this room (must exist in Scenario.Items).
	Items []string `yaml:"items"`
	// Enemies lists enemy template IDs present in this room.
	Enemies []string `yaml:"enemies"`
}

// Connection describes a directed exit from one room to another.
type Connection struct {
	Room string `yaml:"room"`
	Type string `yaml:"type"` // open|locked|stairway|hidden
}

// ScenarioItem defines a type of item that can appear in the scenario.
type ScenarioItem struct {
	Name        string  `yaml:"name"`
	Type        string  `yaml:"type"` // weapon|armor|ring|amulet|consumable|key|misc
	Damage      string  `yaml:"damage,omitempty"`
	Weight      float64 `yaml:"weight"`
	Value       int     `yaml:"value"`
	Description string  `yaml:"description,omitempty"`
}

// EnemyTemplate defines a type of enemy that can appear in the scenario.
type EnemyTemplate struct {
	Name   string `yaml:"name"`
	HP     int    `yaml:"hp"`
	Attack string `yaml:"attack"` // dice notation, e.g. "1d4"
	AI     string `yaml:"ai"`     // aggressive|cautious|scripted
}

// SpellTemplate defines a spell that can be cast in the scenario.
type SpellTemplate struct {
	Name    string   `yaml:"name"`
	MP      int      `yaml:"mp"`       // mana cost
	Effect  string   `yaml:"effect"`   // damage|heal
	Power   string   `yaml:"power"`    // dice notation for effect magnitude
	Classes []string `yaml:"classes"`  // which classes can cast this spell
}
