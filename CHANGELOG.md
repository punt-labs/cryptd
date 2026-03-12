# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Added

- **Thin client architecture (DES-025 revised):** `crypt` is now a thin client — connects to `cryptd serve`, sends natural language text via the `play` JSON-RPC method, displays narrated text. No engine, interpreter, or narrator in the client. Auto-starts `cryptd serve` if the socket is not present.
- **Two daemon modes (DES-025):** `cryptd serve` runs in Normal mode (interpreter → engine → narrator → display text for CLI) or `--passthrough` mode (raw MCP tool surface with structured JSON for Claude Code). Normal mode auto-detects ollama for SLM inference, falls back to Rules + Template.
- `internal/daemon/play.go` — `handlePlay()` processes text input through the full interpreter → engine → narrator pipeline; `handleNewGamePlay()` starts a game and narrates the initial room description.
- `internal/daemon` ServerOption functional options: `WithPassthrough()`, `WithInterpreter()`, `WithNarrator()`.
- `internal/game.Loop.Dispatch()` exported so the daemon can reuse the game loop's orchestration logic (combat, enemy turns, level-ups) without duplicating 300+ lines.
- Nil client guards in `interpreter.SLM` and `narrator.SLM` — graceful fallback when no inference server is available.
- `cryptd headless` and `cryptd autoplay` — testing tools moved from client to server binary (engine is server infrastructure).

### Removed

- `cmd/crypt/embedded.go` — deleted fat client that embedded the game engine. Engine access is always through the server.
- `crypt solo`, `crypt headless`, `crypt autoplay`, `crypt connect` subcommands — replaced by plain `crypt` thin client.

### Changed

- `crypt` takes no subcommands. Flags: `--socket`, `--addr`, `--scenario`, `--name`, `--class`.
- Existing daemon tests updated to use `WithPassthrough()` (they exercise the MCP tool surface, which is passthrough mode by definition).
- E2E acceptance and headless tests now use `cryptd` binary for autoplay/headless commands.
- `internal/scenariodir` package — canonical scenario ID resolution with path-traversal protection, eliminating duplication between CLI and daemon
- `internal/daemon.DefaultSocketPath()` — shared default socket path for both server and client binaries
- `cryptd serve [--socket <path> | --listen <addr>]` — daemon serving 15 MCP tools as JSON-RPC 2.0 over NDJSON; Unix socket (default `~/.crypt/daemon.sock`) or TCP transport; single-connection, signal-handled shutdown, stale socket cleanup
- `internal/daemon` package: protocol types, dispatcher (maps tool names → engine methods with combat auto-processing and level-up checks), JSON-RPC handler (initialize, tools/list, tools/call), and server with Unix socket lifecycle
- Daemon error mapping: engine typed errors → JSON-RPC error codes (-32602 invalid params, -32001 state blocked, -32002 game over, -32003 no active game)
- Daemon unit tests (17 tests): all 15 tools, protocol errors, combat-blocked actions, save/load
- Daemon integration tests (5 tests): socket-level initialize, multi-tool session, cross-connection state persistence, TCP initialize, TCP game session
- Daemon E2E test: subprocess spawn, socket connect, full 6-step game session (initialize → tools/list → new_game → look → pick_up → move with combat)

- Makefile: build, test, demo, and ollama management targets with `make help`; `CRYPT_SCENARIO_DIR` set centrally so demo commands work without env vars
- SLM interpreter rules-first routing: aliases and exact verbs bypass SLM entirely (zero latency); SLM called only for natural language and item/enemy/spell ID resolution
- SLM context injection: game state (room, items, exits, enemies, inventory, equipment) injected into SLM user message, grounding output in valid game objects
- `interpreter.BuildContext` and `interpreter.ParseSLMResponse` exported for eval harness reuse
- `needsIDResolution`: SLM resolves item IDs for take/drop/equip/examine, targets for attack/unequip, spell IDs for cast — fuzzy name matching handled by SLM, not engine
- Rules interpreter: article stripping (the, a, an) from multi-word item targets; `look at <item>` parsed as examine
- Eval harness: rules-first routing mirroring runtime behavior; realistic game state for context injection; accuracy improved from 63.5% to 98.4%
- `docs/slm-improvement.md`: strategy document for SLM accuracy improvement (context injection, prompt engineering, fine-tuning, model scaling)
- Rules interpreter autocorrect: typos within edit distance 1 of known verbs are corrected deterministically (e.g. `attacl` → `attack`, `tke` → `take`); only verbs 3+ characters, zero latency, no SLM call needed
- Rules interpreter: `descend` → move down, `ascend` → move up (directional synonyms)
- SLM system prompt: few-shot examples, stronger unknown guidance ("do not guess"), item ID resolution instructions referencing game state context; eval accuracy 98.4% → 100%

### Changed

- Makefile: `GIN_MODE=release` on `ollama serve` suppresses verbose HTTP request logs during gameplay

- `LuxUpdate.Log`: recent adventure log entries included in incremental updates, enabling the frontend to render a scrolling narration panel without waiting for a full scene rebuild; truncated to last 5 entries (same as `LuxScene.Log`)
- `LuxScene.Exits` and `LuxScene.Actions` / `LuxUpdate.Actions`: navigation exits and context-sensitive action buttons in Lux payloads — exploration mode shows directional exits + look/inventory; combat mode shows attack/defend/flee/cast; game loop populates exits via `enrichForDisplay()` transient field on `DungeonState`
- `cryptd solo --lux` — Lux JSON-lines display mode; writes scene/update payloads as newline-delimited JSON to stdout, reads `InputEvent` JSON from stdin; falls back to CLI renderer when `--lux` is omitted
- `renderer.JSONTransport`: `LuxDisplay` implementation over JSON-lines on arbitrary `io.Reader`/`io.Writer` streams — the wire format for Lux frontends
- `renderer.Lux`: Lux ImGui display surface renderer (Wizardry I layout) — `show()` on scene transitions (room change, combat start/end), `update()` for incremental changes (HP tick, narration); `LuxDisplay` interface abstracts MCP transport; FakeLuxServer-backed tests with performance red line guard (no `show()` for incremental updates)
- Lux integration tests: 6 tests wiring full game loop (engine + interpreter + narrator + LuxRenderer) through FakeLuxServer — two-room navigation, show/update regression guards, combat state transitions, synthetic event injection, party data structure verification
- `LuxHero.XP` and `LuxHero.NextLevelXP` fields: XP and next-level threshold included in both `show()` scene and `update()` payloads for stats panel progress bar rendering; game loop enriches display state via `engine.NextLevelXP()` before each render
- `cmd/eval-slm`: model evaluation harness — runs 65+ natural-language inputs through a real SLM and scores classification accuracy against the full verb table; auto-detects ollama/llama.cpp, supports `--server`, `--model`, `--timeout`, `--verbose` flags; exits non-zero below 80% accuracy
- `interpreter.SystemPrompt` and `interpreter.ParseSLMResponse` exported for eval harness reuse
- `narrator.SLM`: atmospheric narration via local SLM for room descriptions (moved/looked), combat moments (combat_started, combat_won, hero_died), item examination, and level-up; tactical events (damage, errors) use deterministic template narrator for speed and precision; all SLM events fall back gracefully on inference failure
- `interpreter.SLM`: sends free-text player input to a local SLM via `inference.Client`, parses JSON response into `EngineAction`, falls back to `interpreter.Rules` on failure
- DES-024: Inference client — generic text interface (returns raw text, callers own JSON parsing)
- `internal/inference`: OpenAI-compatible HTTP client for `/v1/chat/completions` (works with llama.cpp and ollama)
- `cryptd solo --scenario <id> [--model <name>] [--server <url>] [--timeout <dur>]` — plays with SLM interpreter + narrator when an inference server is available; auto-detects ollama/llama.cpp, falls back to rules+templates if none found
- `inference.Probe`: runtime auto-detection of ollama and llama.cpp servers; probes well-known endpoints in priority order (ollama `/api/tags` → llama.cpp `/v1/models`) and returns first responding runtime with model name; prefers medium-tier models (gemma3:1b → llama3.2:3b → smollm2:135m) when multiple models are available
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

