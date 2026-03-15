package scengen

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// manifestYAML is the top-level scenario.yaml for directory format.
type manifestYAML struct {
	Title        string   `yaml:"title"`
	StartingRoom string   `yaml:"starting_room"`
	Death        string   `yaml:"death"`
	Regions      []string `yaml:"regions"`
	Items        map[string]*itemYAML    `yaml:"items,omitempty"`
	Enemies      map[string]*enemyYAML   `yaml:"enemies,omitempty"`
	Spells       map[string]*spellYAML   `yaml:"spells,omitempty"`
}

type itemYAML struct {
	Name        string  `yaml:"name"`
	Type        string  `yaml:"type"`
	Damage      string  `yaml:"damage,omitempty"`
	Weight      float64 `yaml:"weight"`
	Value       int     `yaml:"value"`
	Description string  `yaml:"description,omitempty"`
}

type enemyYAML struct {
	Name   string `yaml:"name"`
	HP     int    `yaml:"hp"`
	Attack string `yaml:"attack"`
	AI     string `yaml:"ai"`
}

type spellYAML struct {
	Name    string   `yaml:"name"`
	MP      int      `yaml:"mp"`
	Effect  string   `yaml:"effect"`
	Power   string   `yaml:"power"`
	Classes []string `yaml:"classes"`
}

// regionYAML is the structure of a region YAML file.
type regionYAML struct {
	Rooms map[string]*roomYAML `yaml:"rooms"`
}

type roomYAML struct {
	Name            string                    `yaml:"name"`
	DescriptionSeed string                    `yaml:"description_seed,omitempty"`
	Connections     map[string]*connectionYAML `yaml:"connections,omitempty"`
	Items           []string                  `yaml:"items,omitempty"`
	Enemies         []string                  `yaml:"enemies,omitempty"`
}

type connectionYAML struct {
	Room string `yaml:"room"`
	Type string `yaml:"type"`
}

// WriteYAMLDir exports a Graph and ScenarioContent to the YAML directory format.
// Writes to a temporary directory first and renames atomically on success
// to avoid leaving partial output on failure.
//
//	output/
//	  scenario.yaml        ← manifest
//	  regions/
//	    default.yaml       ← rooms grouped by region metadata
func WriteYAMLDir(g *Graph, content *ScenarioContent, outputDir string) error {
	// Write to a temp dir alongside the target, then rename on success.
	// Clean the path to remove trailing slashes before appending suffix.
	outputDir = filepath.Clean(outputDir)
	tmpDir := outputDir + ".tmp"
	if err := os.RemoveAll(tmpDir); err != nil {
		return fmt.Errorf("clean temp directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "regions"), 0o755); err != nil {
		return fmt.Errorf("create temp directory: %w", err)
	}
	cleanup := true
	defer func() {
		if cleanup {
			os.RemoveAll(tmpDir)
		}
	}()

	// Group rooms by region.
	regions := groupByRegion(g)

	// Build manifest.
	manifest := &manifestYAML{
		Title:        content.Title,
		StartingRoom: g.Start,
		Death:        content.Death,
	}

	// Export items.
	if len(content.Items) > 0 {
		manifest.Items = make(map[string]*itemYAML, len(content.Items))
		for id, item := range content.Items {
			manifest.Items[id] = &itemYAML{
				Name: item.Name, Type: item.Type, Damage: item.Damage,
				Weight: item.Weight, Value: item.Value, Description: item.Description,
			}
		}
	}

	// Export enemies.
	if len(content.Enemies) > 0 {
		manifest.Enemies = make(map[string]*enemyYAML, len(content.Enemies))
		for id, enemy := range content.Enemies {
			manifest.Enemies[id] = &enemyYAML{
				Name: enemy.Name, HP: enemy.HP, Attack: enemy.Attack, AI: enemy.AI,
			}
		}
	}

	// Export spells.
	if len(content.Spells) > 0 {
		manifest.Spells = make(map[string]*spellYAML, len(content.Spells))
		for id, spell := range content.Spells {
			manifest.Spells[id] = &spellYAML{
				Name: spell.Name, MP: spell.MP, Effect: spell.Effect,
				Power: spell.Power, Classes: spell.Classes,
			}
		}
	}

	// Build connection map from edges.
	connections := buildConnectionMap(g)

	// Write region files.
	regionNames := sortedKeys(regions)
	for _, regionName := range regionNames {
		roomIDs := regions[regionName]
		sort.Strings(roomIDs)

		region := &regionYAML{Rooms: make(map[string]*roomYAML, len(roomIDs))}
		for _, id := range roomIDs {
			rc := content.Rooms[id]
			room := &roomYAML{}
			if rc != nil {
				room.Name = rc.Name
				room.DescriptionSeed = rc.DescriptionSeed
				room.Items = rc.Items
				room.Enemies = rc.Enemies
			}
			if room.Name == "" {
				room.Name = id
			}
			if conns, ok := connections[id]; ok {
				room.Connections = conns
			}
			region.Rooms[id] = room
		}

		regionPath := filepath.Join("regions", regionName+".yaml")
		manifest.Regions = append(manifest.Regions, regionPath)

		if err := writeYAML(filepath.Join(tmpDir, regionPath), region); err != nil {
			return fmt.Errorf("write region %s: %w", regionName, err)
		}
	}

	sort.Strings(manifest.Regions)

	if err := writeYAML(filepath.Join(tmpDir, "scenario.yaml"), manifest); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	// Atomic swap: remove existing output, rename temp to final.
	if err := os.RemoveAll(outputDir); err != nil {
		return fmt.Errorf("remove existing output: %w", err)
	}
	if err := os.Rename(tmpDir, outputDir); err != nil {
		return fmt.Errorf("rename temp to output: %w", err)
	}
	cleanup = false
	return nil
}

// groupByRegion groups node IDs by their "region" metadata.
// Nodes without a region go into "default".
func groupByRegion(g *Graph) map[string][]string {
	regions := make(map[string][]string)
	for id, node := range g.Nodes {
		region := "default"
		if node.Meta != nil {
			if r, ok := node.Meta["region"]; ok && r != "" {
				region = r
			}
		}
		regions[region] = append(regions[region], id)
	}
	return regions
}

// buildConnectionMap builds direction→connection for every node from the edge list.
func buildConnectionMap(g *Graph) map[string]map[string]*connectionYAML {
	conns := make(map[string]map[string]*connectionYAML)
	for _, e := range g.Edges {
		if conns[e.From] == nil {
			conns[e.From] = make(map[string]*connectionYAML)
		}
		conns[e.From][string(e.FromDir)] = &connectionYAML{Room: e.To, Type: e.Type}

		if conns[e.To] == nil {
			conns[e.To] = make(map[string]*connectionYAML)
		}
		conns[e.To][string(e.ToDir)] = &connectionYAML{Room: e.From, Type: e.Type}
	}
	return conns
}

func writeYAML(path string, v any) error {
	data, err := yaml.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal yaml for %s: %w", filepath.Base(path), err)
	}
	return os.WriteFile(path, data, 0o644)
}

func sortedKeys(m map[string][]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
