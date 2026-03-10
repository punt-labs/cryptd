# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Added

- `crypt validate <file>` — validates a scenario YAML file; prints `OK` exits 0 on success, error message exits 1 on failure
- `internal/model` — full data model structs: `GameState`, `Character`, `Stats`, `Equipment`, `Condition`, `Item`, `DungeonState`, `RoomState`, `LogEntry`
- `internal/dice` — dice notation parser supporting `NdS`, `NdS+M`, `NdS-M` forms with `Roll()`, `Min()`, `Max()`
- `internal/scenario` — YAML scenario parser and validator with typed errors for missing fields, broken room references, unknown enemy templates, and invalid dice notation
- `internal/save` — JSON save/load for `GameState`; `schema_version` field; `ErrVersionMismatch` on version mismatch; unknown fields silently ignored
- `testdata/scenarios/minimal.yaml` — 2-room scenario matching DES-016 schema
- `testdata/scenarios/invalid/` — 6 broken scenario fixtures for parser tests
- `testdata/saves/fighter-level-3.json` — known-state fixture for save/load tests

