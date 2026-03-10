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

// ScriptRunner executes a Script against the game engine.
// Full execution is implemented in Milestone 2; this skeleton only loads
// the script and returns the step list.
type ScriptRunner struct {
	Script *Script
}

// NewScriptRunner loads a script file and returns a ScriptRunner.
func NewScriptRunner(path string) (*ScriptRunner, error) {
	s, err := LoadScript(path)
	if err != nil {
		return nil, err
	}
	return &ScriptRunner{Script: s}, nil
}

// Steps returns the loaded steps (no execution yet — M2).
func (r *ScriptRunner) Steps() []ScriptStep {
	return r.Script.Steps
}
