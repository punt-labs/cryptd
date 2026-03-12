# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Added

- `renderer.Lux`: Lux ImGui display surface renderer (Wizardry I layout) — `show()` on scene transitions (room change, combat start/end), `update()` for incremental changes (HP tick, narration); `LuxDisplay` interface abstracts MCP transport; FakeLuxServer-backed tests with performance red line guard (no `show()` for incremental updates)
- `cmd/eval-slm`: model evaluation harness — runs 65+ natural-language inputs through a real SLM and scores classification accuracy against the full verb table; auto-detects ollama/llama.cpp, supports `--server`, `--model`, `--timeout`, `--verbose` flags; exits non-zero below 80% accuracy
- `interpreter.SystemPrompt` and `interpreter.ParseSLMResponse` exported for eval harness reuse
- `narrator.SLM`: atmospheric narration via local SLM for room descriptions (moved/looked), combat moments (combat_started, combat_won, hero_died), item examination, and level-up; tactical events (damage, errors) use deterministic template narrator for speed and precision; all SLM events fall back gracefully on inference failure
- `interpreter.SLM`: sends free-text player input to a local SLM via `inference.Client`, parses JSON response into `EngineAction`, falls back to `interpreter.Rules` on failure
- DES-024: Inference client — generic text interface (returns raw text, callers own JSON parsing)
- `internal/inference`: OpenAI-compatible HTTP client for `/v1/chat/completions` (works with llama.cpp and ollama)
- `cryptd solo --scenario <id> [--model <name>] [--server <url>] [--timeout <dur>]` — plays with SLM interpreter + narrator when an inference server is available; auto-detects ollama/llama.cpp, falls back to rules+templates if none found
- `inference.Probe`: runtime auto-detection of ollama and llama.cpp servers; probes well-known endpoints in priority order (ollama `/api/tags` → llama.cpp `/v1/models`) and returns first responding runtime with model name
- `FakeSLMServer` upgraded to OpenAI-compatible API with call recording, `/v1/models` endpoint, and configurable response delay (`SetDelay`)
- SLM fallback integration tests: game loop with SLM interpreter + narrator, timeout→fallback, partial failure (SLM degrades mid-session), server-down→fallback
- DES-023: Four-tier inference architecture (tiny/small/medium/large) with graceful failover chain
- Room descriptions: `description_seed` wired through event system, displayed on move/look with exits and visible items
- `unix-catacombs.yaml` scenario: 9-room UNIX-themed dungeon with 3 enemies, 7 items
- CLIRenderer readline support: line editing and command history when stdin is a terminal (ergochat/readline)
- Look resolves item IDs to human-readable names via scenario data

- 5 acceptance scripts: `full-run`, `combat-walkthrough`, `pick-up-item`, `combat`, `save-and-reload` — all passing via `cryptd autoplay --json`
- E2E acceptance test suite (`e2e/acceptance_test.go`) with structured JSON transcript assertions
- MCP schema contract: `cmd/dump-mcp-schema` generates `testdata/mcp-schema.json`; CI diffs generated vs committed to catch unintentional API changes
- 15 MCP tools defined: new_game, move, look, pick_up, drop, equip, unequip, examine, inventory, attack, defend, flee, cast_spell, save_game, load_game
- CLIRenderer HUD: HP/MP progress bars with filled/empty block characters, auto-hidden MP for non-caster classes
- CLIRenderer combat display: live enemy list with per-enemy HP bars, dead enemies hidden
- Interpreter verb: `help`/`?` — lists all available commands
- Narrator template: `help` event with full command reference
- Game loop dispatch for `help` action
- In-game save/load: `SaveGame` and `LoadGame` engine methods with named slots (default: `quicksave`)
- Interpreter verbs: `save [slot]`, `load [slot]`
- Narrator templates: `game_saved`, `game_loaded`, `save_error`, `load_error`
- Game loop dispatch for save/load actions; load replaces current state entirely
- Leveling system: `CheckLevelUp` engine method with per-class XP thresholds (Wizardry-inspired), HP/MP gains, and stat deltas on level-up
- Per-class XP tables: fighter (cheapest), thief, priest, mage (most expensive), max level 10
- Per-class stat gains on level-up: fighter (+STR/+CON), mage (+INT/+WIS), priest (+WIS/+CHA), thief (+DEX/+CHA)
- Game loop level-up narration after combat victory (attack kill or spell damage kill)
- Narrator template: `level_up` event with level and HP gain
- Spell system: `CastSpell` engine method with MP cost deduction, class gates (mage/priest), damage and heal effects
- Spell error types: `UnknownSpellError`, `NotCasterError`, `InsufficientMPError`
- `MP` and `MaxMP` fields on `Character` model
- `SpellTemplate` in scenario YAML: name, MP cost, effect (damage/heal), power (dice notation), allowed classes
- Interpreter verb: `cast <spell>`, `cast <spell> at <target>`
- Narrator templates for spell events: `spell_damage`, `spell_heal`, `unknown_spell`, `not_caster`, `insufficient_mp`
- Game loop spell dispatch: damage spells require active combat, heal spells work in or out of combat, both consume hero turn in combat
- `fireball` (2d6 damage, 3 MP, mage/priest) and `heal` (1d6+2, 2 MP, priest/mage) added to minimal scenario
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

