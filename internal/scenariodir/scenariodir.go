// Package scenariodir resolves scenario IDs to file paths with path-traversal protection.
package scenariodir

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/punt-labs/cryptd/internal/scenario"
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

// Dir returns the scenario directory from CRYPT_SCENARIO_DIR env or "scenarios".
func Dir() string {
	if d := os.Getenv("CRYPT_SCENARIO_DIR"); d != "" {
		return d
	}
	return "scenarios"
}
