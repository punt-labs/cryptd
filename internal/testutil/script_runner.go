package testutil

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ScriptStep is a single command in a game script.
type ScriptStep struct {
	Input  string `yaml:"input"`
	Expect string `yaml:"expect,omitempty"`
}

// Script is a sequence of steps loaded from a testdata/scripts/*.yaml file.
type Script struct {
	Name  string       `yaml:"name"`
	Steps []ScriptStep `yaml:"steps"`
}

// LoadScript reads a YAML game script from path.
func LoadScript(path string) (*Script, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load script %s: %w", path, err)
	}
	var s Script
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse script %s: %w", path, err)
	}
	return &s, nil
}

