package testutil

import (
	"context"
	"sync"

	"github.com/punt-labs/cryptd/internal/model"
)

// FakeLLMNarrator is an in-process test double for an LLM-based Narrator.
// It returns narration strings from a pre-loaded fixture slice, cycling back
// to the start when exhausted.
type FakeLLMNarrator struct {
	mu      sync.Mutex
	lines   []string
	pos     int
}

// NewFakeLLMNarrator creates a FakeLLMNarrator that will return the provided
// strings in order, cycling if called more times than len(lines).
func NewFakeLLMNarrator(lines []string) *FakeLLMNarrator {
	return &FakeLLMNarrator{lines: lines}
}

// Narrate returns the next canned narration string.
func (f *FakeLLMNarrator) Narrate(_ context.Context, _ model.EngineEvent, _ model.GameState) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.lines) == 0 {
		return "", nil
	}
	s := f.lines[f.pos%len(f.lines)]
	f.pos++
	return s, nil
}
