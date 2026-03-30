// Package scenariodir resolves scenario IDs to file paths with path-traversal protection.
package scenariodir

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/punt-labs/cryptd/internal/scenario"
	"gopkg.in/yaml.v3"
)

// Load resolves a scenario ID to a file path within scenarioDir and loads it.
// The ID must be a bare filename stem — slashes, "..", and volume names are rejected.
//
// Resolution order:
//  1. dir/id/scenario.yaml (directory format)
//  2. dir/id.yaml (legacy single-file format)
func Load(scenarioDir, id string) (*scenario.Spec, error) {
	if strings.ContainsAny(id, `/\`) || strings.Contains(id, "..") || filepath.VolumeName(id) != "" {
		return nil, fmt.Errorf("invalid scenario ID")
	}
	absDir, err := filepath.Abs(scenarioDir)
	if err != nil {
		return nil, fmt.Errorf("resolving scenario directory: %w", err)
	}

	// Try directory format first: dir/id/scenario.yaml
	dirPath := filepath.Join(absDir, id)
	manifestPath := filepath.Join(dirPath, "scenario.yaml")
	info, statErr := os.Stat(manifestPath)
	if statErr != nil && !os.IsNotExist(statErr) {
		return nil, fmt.Errorf("checking scenario directory %s: %w", manifestPath, statErr)
	}
	if statErr == nil && !info.IsDir() {
		return scenario.LoadDir(dirPath)
	}

	// Fall back to single-file format: dir/id.yaml
	absPath, err := filepath.Abs(filepath.Join(scenarioDir, id+".yaml"))
	if err != nil {
		return nil, fmt.Errorf("resolving scenario path: %w", err)
	}
	rel, err := filepath.Rel(absDir, absPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
		return nil, fmt.Errorf("invalid scenario ID")
	}
	return scenario.Load(absPath)
}

// ScenarioInfo holds the metadata returned by ListScenarios.
type ScenarioInfo struct {
	ID          string
	Title       string
	Description string
}

// scenarioMeta is a lightweight struct for reading only the title and description
// from a scenario YAML without full parsing or validation.
type scenarioMeta struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
}

// ListScenarios reads scenarioDir and returns metadata for each valid scenario.
// It skips entries that cannot be read or parsed (e.g. test fixtures in invalid/).
func ListScenarios(scenarioDir string) ([]ScenarioInfo, error) {
	entries, err := os.ReadDir(scenarioDir)
	if err != nil {
		return nil, fmt.Errorf("reading scenario directory %s: %w", scenarioDir, err)
	}

	var infos []ScenarioInfo
	for _, e := range entries {
		name := e.Name()

		if e.IsDir() {
			// Skip known test-fixture directories.
			if name == "invalid" {
				continue
			}
			meta, ok := readDirMeta(filepath.Join(scenarioDir, name))
			if !ok {
				continue
			}
			infos = append(infos, ScenarioInfo{
				ID:          name,
				Title:       meta.Title,
				Description: meta.Description,
			})
			continue
		}

		if filepath.Ext(name) != ".yaml" {
			continue
		}
		id := strings.TrimSuffix(name, ".yaml")
		meta, ok := readFileMeta(filepath.Join(scenarioDir, name))
		if !ok {
			continue
		}
		infos = append(infos, ScenarioInfo{
			ID:          id,
			Title:       meta.Title,
			Description: meta.Description,
		})
	}
	return infos, nil
}

// readFileMeta reads title and description from a single-file scenario.
func readFileMeta(path string) (scenarioMeta, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return scenarioMeta{}, false
	}
	var m scenarioMeta
	if err := yaml.Unmarshal(data, &m); err != nil {
		return scenarioMeta{}, false
	}
	return m, true
}

// readDirMeta reads title and description from a directory-format scenario's manifest.
func readDirMeta(dir string) (scenarioMeta, bool) {
	return readFileMeta(filepath.Join(dir, "scenario.yaml"))
}

// Dir returns the scenario directory from CRYPT_SCENARIO_DIR env or "scenarios".
func Dir() string {
	if d := os.Getenv("CRYPT_SCENARIO_DIR"); d != "" {
		return d
	}
	return "scenarios"
}
