// Package testutil provides in-process test doubles for all external
// dependencies. No test in CI should require a real LLM, SLM, Lux, or
// daemon socket — use these fakes instead.
package testutil
