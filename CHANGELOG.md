# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Changed

- **Multiplayer is a standalone capability (DES-034).** Multiplayer — a real party of several human players in one game, with the game as the DM — is a capability of the engine and daemon, defined in terms of game state and the daemon protocol: a real `Party []Character`, `character_id` routing on actions, per-player session-to-character binding, turn ownership, and the DM narrating for the full party. The transport by which players connect is a separate implementation choice.
- **Backfilled three missing ADRs (DES-030, DES-031, DES-032).** DESIGN.md requires every design change to be logged, but three shipped decisions were never recorded. DES-030 corrects the "LLM as DM" framing (DES-009/DES-014): the game engine is the DM, and the LLM enhances it — the engine is always the DM, present at every inference tier, and no client or session is the DM. DES-031 logs the direct JSON-RPC method protocol (`session.init`, `game.*`) with per-session mode selection that replaced the MCP `tools/call` framing and `--passthrough` flag (supersedes that part of DES-025). DES-032 logs the Bubble Tea TUI as the `crypt` client display with a `--plain` readline fallback (extends DES-026/DES-029).
- **JSON-RPC dispatcher typing hardened.** The daemon's error codes are now a named `RPCErrorCode` type instead of bare `int`, `*RPCError` implements the `error` interface (with a nil-receiver guard), every `map[string]any` response body in the dispatcher is replaced by a named response struct shared from `internal/protocol` (so the `crypt` client and server decode one contract), and each dispatcher argument struct carries a `validate() *RPCError` method called right after unmarshal. The on-the-wire JSON is unchanged at the semantic level — identical field names, presence, and null-vs-absent values — verified by the unchanged protocol-contract and integration tests plus round-trip marshal tests; object key ordering shifts from alphabetical (map) to declaration order (struct), which no consumer depends on. (cryptd-7z8, cryptd-dbe, cryptd-8c0, cryptd-64x)
- **Ethos enabled.** Ran `ethos enable`: `CLAUDE.md` imports `@.punt-labs/ethos/CLAUDE.md`, and the `enabled` marker, operating guide, setup guide, and `.vendored-manifest` are tracked under `.punt-labs/ethos/` (matching ethos/quarry/vox). The audit-seal and commit-trailer git hooks are now active with a live session, so the PostToolUse audit writer seals cleanly instead of erroring (`audit entry not written: invalid JSON input`) on every edit as it did while ethos was dormant.
- **`.envrc` loads secrets from Proton Pass.** The developer environment now sources secrets through the canonical `pass-cli` block (one `pass-cli item list --vault-name dev` call, with a keyring/fs key-provider fallback) instead of the platform-specific macOS Keychain / Linux `pass` lookups. Contributors need an authenticated Proton Pass CLI session for the `dev` vault rather than a populated Keychain or `pass` store. `.gitignore` adds `.direnv` to the canonical-managed section.
- **Ethos resolves repo-only.** `.punt-labs/ethos.yaml` sets `resolution: repo-only`; the crypt team's full identity closure (20 identities with personalities, writing styles, talents, roles, and base extensions) is vendored into `.punt-labs/ethos` via `ethos vendor --team crypt --prune`, so `ethos doctor` and agent rendering no longer depend on a host's global `~/.punt-labs/ethos` store. The crypt team now includes every handle its mission-routing table dispatches — the engineering specialists and their evaluators (`rsc`, `rop`, `bcs`, `kth`, `ylc`, `jms`, `jra`, `mcg`, `tdt`) and `gstack-qa` for QA — each with a generated, dispatchable agent definition. Added a QA row to the evaluator-defaults table and corrected the stale "team submodule" / `team: engineering` references in CLAUDE.md to the repo-only `team: crypt` reality.
- **Ethos agent files match the crypt team.** Agent files under `.claude/agents/` for handles not on the crypt team were removed. The infra, Go, security, ML and CLI specialists that CLAUDE.md delegates to (`adb`, `bwk`, `djb`, `kpz`, `mdm`) and the `claude` leader were added to the team, so their agent files stay.
- **unix-megadungeon room descriptions use portable paths.** The generated `description_seed` texts in `scenarios/unix-megadungeon/regions/default.yaml` now show `~/`-relative paths instead of machine-specific absolute ones.
- **Acceptance test tier split out.** `e2e/acceptance_test.go` now uses `//go:build acceptance` (separate from the `e2e` tag). New `make test-acceptance` target and dedicated CI job. `helpers_test.go` uses `e2e || acceptance` so both build. `make check-full` runs both.
- **README badges added.** License, CI status, Go Report Card, and Go Reference.
- **CLAUDE.md M10 status clarified.** M10 session-routing infrastructure (concurrent sessions, per-session isolation) is complete; DM/player privilege gating, `tools/list_changed` notifications, and multi-session integration tests (cryptd-90e.2, .3, .5) remain open.
- **Package map updated.** CLAUDE.md adds `cmd/crypt-admin`; both CLAUDE.md and README.md drop the removed `cmd/dump-mcp-schema`. README's JSON-RPC API surface replaces the stale MCP tool list. `docs/build-plan.md` and `docs/testing.md` drop references to the deleted schema contract.
- **architecture.tex synced to current protocol.** Rewrote MCP tool-surface sections (15 tools, `tools/call` dispatch, `--passthrough` flag) as JSON-RPC method-surface sections (direct named methods, per-session mode via `session.init`). PDF rebuilt.
- **Protocol: direct JSON-RPC methods replace MCP framing.** Daemon protocol refactored from `tools/call` dispatch to direct named methods: `session.init`, `game.new`, `game.move`, `game.look`, `game.play`, `session.quit`, etc. Response results are direct JSON objects instead of `ToolResult{content:[{text:"..."}]}` wrappers. Error responses use standard JSON-RPC error objects instead of `ToolResult{isError:true}`.
- **Removed MCP schema infrastructure.** Deleted `cmd/dump-mcp-schema`, `testdata/mcp-schema.json`, and the `mcp-schema` CI job. The daemon is a game server, not an MCP server.
- **Client updated for new protocol.** `crypt` sends `session.init` instead of `initialize`, `game.new` instead of `tools/call` with `name=new_game`, and `game.play` instead of `play`.

### Added

- **`docs/architecture-as-built.md` — the read-first structural map.** Reconstructed from source: the engine-is-the-DM model, the normal vs passthrough request paths (the distinction the old docs blurred), the layer/dependency ladder, inference tiers and fallback, the concurrency model, the state model, and a "documented vs. built" table of where the record drifted from the code. Linked from `CLAUDE.md`'s Documentation Map and `DESIGN.md`'s header. Corrected stale package docs along the way: `internal/engine/doc.go` ("movement and look" → the full rules surface) and the `internal/daemon/daemon.go` "single connection at a time (M8)" comment (the server is concurrent). Also explains **why two modes exist** — the enhancement layer (interpretation + narration) runs client-side in passthrough because Claude Code already has an LLM, and server-side in normal mode because a plain terminal does not; and warns against confusing the engine (the DM) with the narration layer (the enhancement), which never becomes the DM wherever it runs.
- **Bubble Tea TUI client.** `crypt` now launches a rich terminal UI by default on TTY: scrolling narration viewport, HP/XP bars with text overlays, 3x3 compass grid with styled exits, equipped items, inventory with weights, stat block, and a combat overlay with single-key shortcuts (A/D/F/U). `--plain` flag falls back to the readline REPL. Visual styling matches `docs/crypt-tui-mockup.html`: bordered panes, section headers, gold accents, color-coded narration text (room names, actions, combat, system). Wire change: `PlayResponse` now carries `exits` and `next_level_xp` for sidebar rendering.
- **TUI lobby and game creation.** Launching `crypt` without `--scenario` shows a centered lobby screen with New Game / Resume / Quit menu. New Game enters a 4-step character creation wizard: scenario picker (from `game.list_scenarios` RPC), name input, class selection with stat previews, and point-buy stat allocation (8 points, base 10). Resume shows saved sessions from `game.list_sessions` RPC. `--scenario`/`--session` flags skip to gameplay.
- **`game.list_scenarios` RPC.** Returns available scenarios with id, title, and description. Scans the scenario directory for both single-file and directory-format scenarios.
- **`game.list_sessions` RPC.** Returns active sessions with game metadata (scenario, character name/class, level, room). Session metadata cached at creation time.
- **DM SKILL.md and ethos identities.** `/crypt` skill invokes Claude as Dungeon Master. Two DM personas: The Cryptkeeper (sardonic, theatrical, UNIX-gothic) and The Archivist (clinical, precise, log-like). Ethos personalities, writing styles, and talents define DM behavior. `game.context` command gives the DM full scenario awareness — current room, adjacent rooms with description seeds, visited rooms, hero status.
- **`crypt mcp`: stdio MCP bridge for Claude Code.** Runs as an MCP server on stdio, proxies tool calls to `cryptd serve` over Unix socket/TCP. 16 game tools registered (new_game, move, look, attack, etc.). Thread-safe daemon proxy with mutex. Registration: `claude mcp add -s project crypt -- crypt mcp`.
- **LLM inference tier (M9):** `LLMInterpreter` and `LLMNarrator` implementations that call Claude's API via the OpenAI-compatible `/v1/chat/completions` endpoint. Same two-tier fallback strategy as the SLM tier: Rules-first for deterministic commands, LLM for ambiguous input; Template fallback for tactical events, LLM for atmospheric narration. Claude-optimized system prompts.
- **`inference.Client` auth support:** `NewClientWithOpts` constructor with `WithAPIKey(string)` and `WithTimeout(time.Duration)` functional options. When set, adds `Authorization: Bearer <key>` to every request. Existing `NewClient` is unchanged.
- **`cryptd serve --api-key`** flag (and `CRYPTD_API_KEY` env var fallback) selects the Claude LLM tier. Probe chain: Claude API → ollama → llama.cpp → rules+templates.
- **M10 concurrent sessions with per-session isolation:** daemon accepts concurrent connections. Each session gets an independent game instance. Game is a goroutine that exclusively owns its engine and state — no mutexes on game state, serialization by command channel. Session/Game type separation: sessions persist across reconnects, games are created on `new_game`. Both passthrough and normal mode are fully isolated. Server registry with `sync.RWMutex` for session/game lookup.
- **`crypt --session <id>`** flag for session resumption. Sends the session ID in the `initialize` handshake to reconnect to an existing game. Server automatically re-enters the game loop and renders the current room on reconnect.
- **Session ID in protocol (M10 prep):** `initialize` handshake now accepts an optional `session_id` param and returns a `session_id` in the result. Server assigns a 32-char hex ID via `crypto/rand` when the client doesn't provide one. Foundation for M10 session routing.
- **DM pipeline integration tests:** full game loop and daemon-level tests wiring `FakeLLMInterpreter` + `FakeLLMNarrator` through the interpreter → engine → narrator pipeline. Verifies state transitions are identical to headless mode.
- **EnemyVisitor:** distance-based enemy placement with 5-tier progression (rat→goblin→skeleton→troll→dragon). 40% spawn rate, hub skipping, ±1 tier jitter for variety.
- **ItemVisitor:** distance-based item placement with 5-tier weapons (dagger→sword→longsword→axe→vorpal blade), matching armor tiers (leather→chain→plate→dragon scale), and scaling health potions. Starter weapon at distance 1, starter potion at start, potions at hub rest stops.
- **DefaultSpells():** standard spell catalog (fireball, heal, lightning, blessing) added to generated scenarios.
- `Graph.MaxDistance()` for normalizing distance-to-tier mapping.
- `crypt-admin generate` now runs description→enemy→item visitors and prints room/enemy/XP/item stats.
- Make targets: `generate-dungeon`, `validate-dungeon`, `eval-balance-generated`, `eval-balance-unix`.
- **Armor damage reduction:** equipped armor's `defense` field reduces incoming enemy damage (flat subtraction, floor 1). Defend stance and armor stack: halve first, then subtract defense.
- **Consumable items:** `use`/`drink`/`eat` verbs consume items with `effect`/`power` fields. Health potions heal dice-based HP (e.g. `2d6`). New engine method `UseItem()`, typed error `NotConsumableError`.
- **CON modifier scales HP per level:** HP gain = class base + floor((CON-10)/2). Fighters (CON growth) gain 9-10+ HP/level at higher levels; mages (no CON growth) gain base 4. New `StatModifier()` function.
- **Point-buy stat allocation:** 8 points distributed across 6 stats (base 10 each). `engine.NewCharacter()` centralizes character creation with `ValidateStats()`. Daemon `new_game` RPC accepts optional `stats` field.
- **Interactive character creation:** `cryptd serve -t` (without `--script`) prompts for name, class, and stat allocation before the game begins. Enter for defaults (STR +4, DEX +2, CON +2).
- **Monkey test harness for game balance tuning** (`cmd/eval-balance`): runs N parallel game sessions with weighted-random action selection, collects per-session metrics (moves, XP, kills, damage, survival, flee rate, spells, potions, level-ups), and produces JSON aggregate reports with per-class breakdowns. `make eval-balance` runs 1000 sessions across all classes; `make eval-balance-quick` runs 100 fighter sessions.
- `internal/monkeytest` package: `MonkeyRenderer` (state-aware weighted-random Renderer), `SessionMetrics`, `AggregateReport`, parallel `Run()` with configurable workers, percentile computation. Monkey auto-equips weapons and armor, uses health potions at ≤50% HP, class-optimal stat distributions.
- `docs/gameplay.tex` / `.pdf` — 10-page gameplay mechanics specification: attributes, point-buy, classes, XP tables, combat, spells, inventory, consumables, movement, commands, balance tuning targets.
- **`crypt-admin` binary** — third binary for scenario authoring (alongside `cryptd` and `crypt`). Subcommands: `generate`, `validate`, `export`.
- **Graph-first scenario generation (DES-027):** `crypt-admin generate --topology tree --source <dir>` walks a filesystem tree, builds a connected graph with bidirectional edges, assigns 6-direction compass directions, and exports to YAML directory format. Hub nodes inserted automatically for directories with >5 children.
- **YAML directory format:** scenarios can now be a directory with `scenario.yaml` manifest and `regions/*.yaml` sub-files. Cross-region room references work naturally. `scenariodir.Load()` tries directory format first, falls back to single-file.
- `internal/scengen` package: `Graph`, `Node`, `Edge`, `Direction` types with `Validate()` (bidirectionality, max degree 6, BFS connectivity); `TopologySource` interface with `TreeSource` adapter; `Visitor` interface with `DescriptionVisitor`; `WriteYAMLDir()` exporter; SQLite `Store` for iterative authoring.
- `internal/scenario.LoadDir()` — loads directory-format scenarios with duplicate room ID detection across regions.
- `internal/scenariodir` tests — directory fallback, precedence, path traversal protection.

- **Thin client architecture (DES-025 revised):** `crypt` is now a thin client — connects to `cryptd serve`, sends natural language text via the `play` JSON-RPC method, displays narrated text. No engine, interpreter, or narrator in the client. Auto-starts `cryptd serve` if the socket is not present.
- **Two daemon modes (DES-025):** `cryptd serve` runs in Normal mode (interpreter → engine → narrator → display text for CLI) or Passthrough mode (direct `game.*` methods with structured JSON for Claude Code). Session mode is selected per-session during `session.init` via the `mode` field in `InitializeParams`; both modes coexist on the same socket. Normal mode auto-detects ollama for SLM inference, falls back to Rules + Template.
- `internal/daemon/play.go` — `handlePlay()` processes text input through the full interpreter → engine → narrator pipeline; `handleNewGamePlay()` starts a game and narrates the initial room description.
- `internal/daemon` ServerOption functional options: `WithPassthrough()`, `WithInterpreter()`, `WithNarrator()`.
- `internal/game.Loop.Dispatch()` exported so the daemon can reuse the game loop's orchestration logic (combat, enemy turns, level-ups) without duplicating 300+ lines.
- Nil client guards in `interpreter.SLM` and `narrator.SLM` — graceful fallback when no inference server is available.

### Fixed

- **`.gitignore` now ignores live ethos mission runtime state.** `.punt-labs/ethos/missions/` (the live mission tree) and `.punt-labs/ethos/missions.jsonl` (the index) are per-run runtime churn and were untracked-but-not-ignored, so a `git add -A` would commit them. Added both to `.gitignore`; `.punt-labs/ethos/missions-closed/` (redacted, exported closed missions) stays tracked. The stale comment block inherited from vox's opposite policy (missions/ tracked) was corrected to cryptd's policy.
- `make check` no longer fails on quarry session-transcript captures. `.markdownlint-cli2.jsonc` now ignores `.punt-labs/quarry/captures/**`, which is gitignored runtime state (auto-captured transcripts), not authored docs — the same treatment as `.claude/agents/**` and `.punt-labs/ethos/agents/**`.
- `crypt` REPL now uses `ergochat/readline` for proper terminal line editing, history, and Ctrl-C handling (was broken with bare `bufio.Scanner`).
- `TestSLMLoop_HappyPath` integration test no longer hangs — test inputs changed to phrases the Rules interpreter can't parse, so the SLM is actually invoked.
- CI `staticcheck` pin bumped from v0.7.0 to v0.8.1 — v0.7.0 can't parse the Go 1.27+ stdlib export data format, so `make check` failed on any machine with Go 1.27 as the default local toolchain.

### Removed

- **Lux renderer integration (abandoned).** Deleted the `internal/lux` wire-protocol client, the `renderer.Lux` renderer and its element/transport translation layer, the `FakeLuxServer`/`LuxElementDisplay` test doubles, the Lux integration tests, and the `cryptd serve -t --lux` flag (~3,300 lines). The game's player-facing display is the `crypt` client's Bubble Tea TUI, fed typed state over the twin-renderer wire (DES-026); the engine keeps `CLIRenderer` and `Autoplay`. The Go Lux client had drifted to wire protocol `0.1` against Lux 0.33.1 and no play mode shipped on it. See DES-029; operator ruling 2026-11-08 (bead cryptd-oal). The M5 Lux HUD and M12 Lux HUD Release Pass milestones are abandoned.
- `cmd/crypt/embedded.go` — deleted fat client that embedded the game engine. Engine access is always through the server.
- `crypt solo`, `crypt headless`, `crypt autoplay`, `crypt connect` subcommands — replaced by plain `crypt` thin client.
- `cryptd headless` and `cryptd autoplay` subcommands — replaced by `cryptd serve -t` (testing mode on stdin/stdout).
- E2E acceptance and headless tests — replaced by `cryptd serve -t --script` demos.
- `cryptd validate` prints deprecation warning directing users to `crypt-admin validate`.
- `make build` now builds all three binaries (`cryptd`, `crypt`, `crypt-admin`).
- `cryptd serve -t` (interactive, no `--script`) now probes for ollama and uses SLM interpreter + narrator when available. Scripted mode retains deterministic rules+templates.
- unix-catacombs scenario rebalanced: OOM Killer 25→15 HP and 1d8+2→1d6+1, Segfault Daemon 15→12 HP, new Zombie Process enemy (6 HP), short sword in starting room, alias shield gains defense 2, three health potions placed along the critical path. Survival rate: 0%→65%.
- `ScenarioItem` gains `defense` (int), `power` (dice string), `effect` (string) fields for armor and consumable mechanics.
- `crypt` takes no subcommands. Flags: `--socket`, `--addr`, `--scenario`, `--name`, `--class`.
- Existing daemon tests updated to use `WithPassthrough()` (they exercise the MCP tool surface, which is passthrough mode by definition).
- `internal/scenariodir` package — canonical scenario ID resolution with path-traversal protection, eliminating duplication between CLI and daemon
- `internal/daemon.DefaultSocketPath()` — shared default socket path for both server and client binaries
- `cryptd serve [--socket <path> | --listen <addr>]` — daemon serving 15 MCP tools as JSON-RPC 2.0 over NDJSON; Unix socket (default `~/.crypt/daemon.sock`) or TCP transport; single-connection, signal-handled shutdown, stale socket cleanup
- `internal/daemon` package: protocol types, dispatcher (maps tool names → engine methods with combat auto-processing and level-up checks), JSON-RPC handler (initialize, tools/list, tools/call), and server with Unix socket lifecycle
- Daemon error mapping: engine typed errors → JSON-RPC error codes (-32602 invalid params, -32001 state blocked, -32002 game over, -32003 no active game)
- Daemon unit tests (17 tests): all 15 tools, protocol errors, combat-blocked actions, save/load
- Daemon integration tests (5 tests): socket-level initialize, multi-tool session, cross-connection state persistence, TCP initialize, TCP game session
- Daemon E2E test: subprocess spawn, socket connect, full 6-step game session (initialize → tools/list → new_game → look → pick_up → move with combat)
- Makefile: build, test, demo, and ollama management targets with `make help`; `CRYPT_SCENARIO_DIR` set centrally so demo commands work without env vars; `GIN_MODE=release` on `ollama serve` suppresses verbose HTTP request logs during gameplay
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
- `cryptd headless --scenario <id>` — runs the game in headless mode (no LLM) using `RulesInterpreter`, `TemplateNarrator`, and `CLIRenderer` wired through the new game loop; supports `CRYPT_SCENARIO_DIR` env override
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
