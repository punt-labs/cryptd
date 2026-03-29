package scenario

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// dirManifest is the top-level scenario.yaml for directory-format scenarios.
type dirManifest struct {
	Title        string                       `yaml:"title"`
	StartingRoom string                       `yaml:"starting_room"`
	Death        string                       `yaml:"death"`
	Regions      []string                     `yaml:"regions"`
	Items        map[string]*Item     `yaml:"items"`
	Enemies      map[string]*EnemyTemplate    `yaml:"enemies"`
	Spells       map[string]*SpellTemplate    `yaml:"spells"`
}

// dirRegion is the structure of a region YAML file within a scenario directory.
type dirRegion struct {
	Rooms map[string]*Room `yaml:"rooms"`
}

// LoadDir loads a scenario from a directory containing a manifest (scenario.yaml)
// and region sub-files. The directory ID is derived from the directory name.
func LoadDir(dir string) (*Spec, error) {
	manifestPath := filepath.Join(dir, "scenario.yaml")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("read manifest %s: %w", manifestPath, err)
	}

	var m dirManifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest %s: %w", manifestPath, err)
	}

	s := &Spec{
		ID:           filepath.Base(dir),
		Title:        m.Title,
		StartingRoom: m.StartingRoom,
		Death:        m.Death,
		Rooms:        make(map[string]*Room),
		Items:        m.Items,
		Enemies:      m.Enemies,
		Spells:       m.Spells,
	}

	if s.Items == nil {
		s.Items = make(map[string]*Item)
	}
	if s.Enemies == nil {
		s.Enemies = make(map[string]*EnemyTemplate)
	}
	if s.Spells == nil {
		s.Spells = make(map[string]*SpellTemplate)
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("resolve scenario directory: %w", err)
	}

	for _, regionPath := range m.Regions {
		fullPath := filepath.Join(dir, regionPath)

		// Path traversal guard: region files must stay within the scenario directory.
		absRegion, err := filepath.Abs(fullPath)
		if err != nil {
			return nil, fmt.Errorf("resolve region path %s: %w", regionPath, err)
		}
		rel, err := filepath.Rel(absDir, absRegion)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
			return nil, fmt.Errorf("region path %q escapes scenario directory", regionPath)
		}

		regionData, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("read region %s: %w", fullPath, err)
		}

		var region dirRegion
		if err := yaml.Unmarshal(regionData, &region); err != nil {
			return nil, fmt.Errorf("parse region %s: %w", fullPath, err)
		}

		for roomID, room := range region.Rooms {
			if _, exists := s.Rooms[roomID]; exists {
				return nil, &DuplicateRoomError{Room: roomID, Region: regionPath}
			}
			s.Rooms[roomID] = room
		}
	}

	if err := Validate(s); err != nil {
		return nil, err
	}
	return s, nil
}

// DuplicateRoomError is returned when a room ID appears in multiple region files.
type DuplicateRoomError struct {
	Room   string
	Region string
}

func (e *DuplicateRoomError) Error() string {
	return fmt.Sprintf("duplicate room ID %q in region %s", e.Room, e.Region)
}
