package testutil

import (
	"context"
	"sync"

	"github.com/punt-labs/cryptd/internal/model"
)

// FakeLLMInterpreter is an in-process test double for an LLM-based
// CommandInterpreter. It returns EngineActions from a pre-loaded fixture
// slice, cycling back to the start when exhausted.
type FakeLLMInterpreter struct {
	mu      sync.Mutex
	actions []model.EngineAction
	pos     int
}

// NewFakeLLMInterpreter creates a FakeLLMInterpreter that will return the
// provided actions in order, cycling if called more times than len(actions).
func NewFakeLLMInterpreter(actions []model.EngineAction) *FakeLLMInterpreter {
	return &FakeLLMInterpreter{actions: actions}
}

// Interpret returns the next canned EngineAction.
func (f *FakeLLMInterpreter) Interpret(_ context.Context, _ string, _ model.GameState) (model.EngineAction, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.actions) == 0 {
		return model.EngineAction{}, nil
	}
	a := f.actions[f.pos%len(f.actions)]
	f.pos++
	return a, nil
}
