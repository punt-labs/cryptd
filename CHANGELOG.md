# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Added

- `cryptd headless --scenario <id>` — runs the game in headless mode (no LLM, no Lux) using `RulesInterpreter`, `TemplateNarrator`, and `CLIRenderer` wired through the new game loop; supports `CRYPT_SCENARIO_DIR` env override
- `internal/engine` — deterministic game rules engine: `NewGame`, `Move`, `Look`; typed errors `NoExitError` and `LockedError`
- `internal/interpreter` — `RulesInterpreter`: maps `go <dir>`, `look`, `quit` (and aliases) to `EngineAction` without LLM involvement
- `internal/narrator` — `TemplateNarrator`: fixed-template narrations for `moved`, `looked`, `unknown_action`, `quit`, and fallback events
- `internal/renderer` — `CLIRenderer`: writes room description + narration to stdout, reads one line of input per render cycle
- `internal/game` — `Loop`: wires engine + interpreter + narrator + renderer; drives full game until quit or context cancel
- `e2e/headless_test.go` — E2E test (build tag `e2e`) that compiles the binary and runs a scripted headless session
- `testdata/scripts/minimal-run.yaml` — M2 acceptance script: move south → look → move north → quit
- `cryptd validate <scenario-file>` — validates a scenario YAML file; prints `OK` exits 0 on success, error message exits 1 on failure
- `internal/model` — full data model structs: `GameState`, `Character`, `Stats`, `Equipment`, `Condition`, `Item`, `DungeonState`, `RoomState`, `LogEntry`
- `internal/dice` — dice notation parser supporting `NdS`, `NdS+M`, `NdS-M` forms with `Roll()`, `Min()`, `Max()`
- `internal/scenario` — YAML scenario parser and validator with typed errors for missing fields, broken room references, unknown enemy templates, and invalid dice notation
- `internal/save` — JSON save/load for `GameState`; `schema_version` field; `ErrVersionMismatch` on version mismatch; unknown fields silently ignored
- `testdata/scenarios/minimal.yaml` — 2-room scenario matching DES-016 schema
- `testdata/scenarios/invalid/` — 6 broken scenario fixtures for parser tests
- `testdata/saves/fighter-level-3.json` — known-state fixture for save/load tests

