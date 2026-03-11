# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Added

- Turn-based combat system: `StartCombat`, `Attack`, `Defend`, `Flee`, `ProcessEnemyTurn` engine methods with initiative rolls, damage resolution, XP awards, and room clearing on victory
- Combat model types: `CombatState` and `EnemyInstance` on `DungeonState` for tracking active encounters with turn order, rounds, and per-enemy mutable state
- Combat error types: `NotInCombatError`, `NotHeroTurnError`, `InvalidTargetError`, `HeroDeadError`, `AlreadyInCombatError`, `NoEnemiesError`
- Interpreter verbs: `attack`/`a`/`hit`/`strike`/`kill`, `defend`/`block`/`guard`, `flee`/`run`/`escape`
- Narrator templates for combat events: `combat_started`, `attack_hit`, `attack_kill`, `enemy_attacks`, `enemy_flees`, `defend`, `flee_success`, `flee_fail`, `combat_won`, `hero_died`, `not_in_combat`, `in_combat`
- Game loop combat dispatch: auto-starts combat on room entry, blocks non-combat actions during combat, processes enemy turns after hero acts, handles hero death
- AI patterns: aggressive (always attacks), cautious (flees at ≤30% HP), scripted (attacks every turn)
- `cryptd autoplay --scenario <id> --script <file> [--json]` — reads a command file (one command per line, `#` comments), feeds commands one at a time through the game loop, and writes a request-response transcript; `--json` outputs structured `[{command, room, response}]` array
- `renderer.Autoplay` — `Renderer` implementation that pops commands from a queue and collects transcript entries, reusing the game loop without modification
- `testdata/demos/combat-walkthrough.txt` — demo script: pick up sword, equip, fight goblin
- Default hero stats: `STR:14 DEX:12 CON:12 INT:10 WIS:10 CHA:10`, HP 20 (was 10) — enables meaningful combat and flee checks
- Extracted `loadScenario()` and `defaultHero()` helpers in CLI, eliminating scenario-loading duplication between `headless` and `autoplay`
- Inventory system: `pick_up`, `drop`, `equip`, `unequip`, `examine`, `inventory` engine methods with typed errors (`ItemNotInRoomError`, `ItemNotInInventoryError`, `TooHeavyError`, `NotEquippableError`, `SlotOccupiedError`, `SlotEmptyError`)
- Mutable room item state: `RoomState.Items` seeded from scenario on `NewGame`; items flow between rooms and character inventory
- Weight limit enforcement: `MaxCarryWeight` (50.0) checked on pickup
- Equipment slot management: weapon, armor, ring, amulet slots with conflict detection
- Interpreter verbs: `take`/`get`/`grab`/`pick up`, `drop`, `equip`/`wear`/`wield`, `unequip`/`remove`, `examine`/`x`, `inventory`/`i`
- Narrator templates for all inventory events: picked_up, dropped, equipped, unequipped, examined, inventory_listed, plus error events
- Game loop dispatch for all inventory action types
- `short_sword` weapon added to minimal scenario entrance room
- `cryptd headless --scenario <id>` — runs the game in headless mode (no LLM, no Lux) using `RulesInterpreter`, `TemplateNarrator`, and `CLIRenderer` wired through the new game loop; supports `CRYPT_SCENARIO_DIR` env override
- `internal/engine` — deterministic game rules engine: `NewGame`, `Move`, `Look`; typed errors `NoExitError` and `LockedError`
- `internal/interpreter` — `RulesInterpreter`: maps `go <dir>`, `look`, `quit` (and aliases) to `EngineAction` without LLM involvement
- `internal/narrator` — `TemplateNarrator`: fixed-template narrations for `moved`, `looked`, `unknown_action`, `quit`, and fallback events
- `internal/renderer` — `CLIRenderer`: writes room ID header and narration to stdout, reads one line of input per render cycle
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

