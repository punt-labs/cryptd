# cryptd

[![License](https://img.shields.io/github/license/punt-labs/cryptd)](LICENSE)
[![CI](https://img.shields.io/github/actions/workflow/status/punt-labs/cryptd/test.yml?label=CI)](https://github.com/punt-labs/cryptd/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/punt-labs/cryptd)](https://goreportcard.com/report/github.com/punt-labs/cryptd)
[![Go Reference](https://pkg.go.dev/badge/github.com/punt-labs/cryptd.svg)](https://pkg.go.dev/github.com/punt-labs/cryptd)

Game engine and server for [crypt](https://github.com/punt-labs/crypt) — a text
adventure game playable via Claude Code, CLI, or (future) web client.

## Three Binaries

| Binary | Role | Description |
|--------|------|-------------|
| `cryptd` | **Server** | Game engine, interpreter, narrator as a JSON-RPC 2.0 service |
| `crypt` | **Client** | Player-facing CLI with Bubble Tea TUI (readline fallback via `--plain`) |
| `crypt-admin` | **Author** | Graph-first scenario generation and validation |

### cryptd (server)

```bash
cryptd serve                           # daemonize, default Unix socket (~/.crypt/daemon.sock)
cryptd serve -f --listen :9000         # foreground, TCP
cryptd serve -t --scenario minimal     # testing mode (stdin/stdout, no network)
cryptd serve --api-key sk-...          # enable Claude LLM tier (or CRYPTD_API_KEY env)
cryptd serve -t --scenario minimal --script demo.txt  # scripted playthrough
```

Two modes:

| Mode | Interpreter | Narrator | Client |
|------|-------------|----------|--------|
| **Normal** | LLM/SLM → Rules fallback | LLM/SLM → Template fallback | `crypt` (CLI) |
| **Passthrough** | None (MCP tool names) | None (structured JSON) | Claude Code plugin |

Three inference tiers: Claude API (`CRYPTD_API_KEY` or `--api-key`), local SLM
via [ollama](https://ollama.com), or deterministic Rules + Templates. The server
supports concurrent sessions with per-session game isolation — each connection
gets its own game instance. Sessions persist across reconnects.

### crypt (client)

```bash
crypt                                  # connect to local server (auto-starts if needed)
crypt --addr host:9000                 # connect to remote server
crypt --scenario unix-catacombs        # auto-start with specific scenario
crypt --name Gandalf --class mage      # set character name and class
crypt --session <id>                   # resume a previous session
crypt --plain                          # readline mode (no TUI)
```

Connects to `cryptd serve` and sends natural language text via the `play`
JSON-RPC method. On a TTY, launches a Bubble Tea terminal UI with scrolling
narration, HP/XP/MP bars, compass exit grid, inventory sidebar, and combat
overlay with keyboard shortcuts (`a`/`d`/`f`/`u`). Use `--plain` for the
original readline interface. Auto-starts the server if the local socket is
absent. Use `--session` to reconnect to a previous game — the session ID is
printed to stderr on each connection.

### crypt-admin (author)

```bash
# Generate a scenario from a filesystem tree
crypt-admin generate \
    --topology tree \
    --source /usr/local \
    --title "UNIX Catacombs" \
    --output scenarios/unix-catacombs/

# Generate with SQLite working copy for iterative editing
crypt-admin generate \
    --topology tree \
    --source ~/project \
    --title "Project Dungeon" \
    --output scenarios/project/ \
    --db .tmp/project.db

# Validate any scenario format
crypt-admin validate scenarios/unix-catacombs/   # directory format
crypt-admin validate scenario.yaml               # single-file format

# Export SQLite working copy to YAML directory
crypt-admin export --db .tmp/project.db --output scenarios/project/
```

Generates scenarios graph-first ([DES-027](DESIGN.md)): a topology source
produces nodes and edges, BFS direction assignment creates valid bidirectional
6-direction connections, and visitors decorate rooms with content. Output is
YAML directory format that `cryptd` loads directly.

The [crypt plugin](https://github.com/punt-labs/crypt) for Claude Code connects
via `crypt mcp`, a stdio MCP bridge that proxies tool calls to `cryptd serve`.
Session mode (normal vs passthrough) is set per-session during `session.init`.

## Architecture

### The L4 / L1 Split

Core principle ([DES-009](DESIGN.md)): the LLM is the Dungeon Master (narrator,
semantic parser, scenario author) and the Go engine is the deterministic rules
machine. No game rule or state transition is left to probabilistic LLM output.

```text
L4 — Claude (SKILL.md)      Narrates, interprets free text, generates scenario YAML
         │ MCP tools
L1 — Go engine              State transitions, combat, inventory, fog-of-war, save/load
L1 — Lux (ImGui)            Display surface (receives element trees via MCP)
```

### Dependency Direction

```text
CLI / Commands  (cmd/cryptd/, cmd/crypt/, cmd/crypt-admin/)
    ├── Game Loop  (internal/game/)  — wires Interpreter + Narrator + Renderer
    │       ├── Interpreters  (internal/interpreter/)
    │       ├── Narrators     (internal/narrator/)
    │       └── Renderers     (internal/renderer/)
    │               └── Engine  (internal/engine/)
    │                       └── Data Contracts  (internal/model/, internal/scenario/, internal/save/, internal/dice/)
    └── Scenario Gen  (internal/scengen/)  — graph, topology, visitors, export, SQLite store
            └── Data Contracts  (internal/scenario/)
```

The engine knows nothing about interpreters. Interpreters know nothing about
narrators. `crypt-admin` shares data contracts with the server but never imports
the engine, game loop, or daemon packages. SQLite (`modernc.org/sqlite`) is
linked only into `crypt-admin` — never into `cryptd` or `crypt`.

### Package Map

| Package | Purpose |
|---------|---------|
| `cmd/cryptd` | Server binary: `serve` (with `-f`, `-t`, `--api-key`), `validate` |
| `cmd/crypt` | Thin client binary: connects to server, Bubble Tea TUI + readline fallback |
| `cmd/crypt-admin` | Author binary: `generate`, `validate`, `export` |
| `cmd/eval-slm` | SLM accuracy evaluation harness (65+ inputs, needs ollama) |
| `cmd/eval-balance` | Monkey test harness for game balance tuning (parallel sessions, JSON reports) |
| `internal/monkeytest` | MonkeyRenderer, SessionMetrics, AggregateReport, parallel runner |
| `internal/daemon` | JSON-RPC 2.0 handler, game-as-goroutine dispatcher, session registry, Unix socket/TCP listener |
| `internal/engine` | Deterministic game rules: movement, combat, inventory, spells, leveling, save/load |
| `internal/game` | Game loop: drives engine + interpreter + narrator + renderer |
| `internal/inference` | OpenAI-compatible HTTP client for `/v1/chat/completions` |
| `internal/interpreter` | `RulesInterpreter` (keyword/regex), `SLM` (ollama), `LLM` (Claude API) |
| `internal/narrator` | `TemplateNarrator` (fixed), `SLM` (ollama), `LLM` (Claude atmospheric narration) |
| `internal/renderer` | `CLIRenderer` (stdout/stdin), `Lux` (ImGui via MCP), `JSONTransport` |
| `internal/model` | All data types: `GameState`, `Character`, `EngineAction`, `EngineEvent`, interfaces |
| `internal/scenario` | YAML parser, validator, directory-format loader (`LoadDir`) |
| `internal/scenariodir` | Scenario ID resolution with path-traversal protection |
| `internal/scengen` | Graph types, topology sources, visitors, YAML exporter, SQLite store |
| `internal/save` | JSON save/load with `schema_version` and forward compatibility |
| `internal/dice` | Dice notation parser: `NdS`, `NdS+M`, `NdS-M` |
| `internal/protocol` | JSON-RPC 2.0 protocol types shared by daemon and client |
| `internal/testutil` | Test doubles: `FakeLLMInterpreter`, `FakeLLMNarrator`, `FakeSLMServer`, `FakeLuxServer` |
| `e2e` | End-to-end tests (build tag `e2e`): compiled binary, scripted sessions |

### Scenario Formats

**Single-file** (legacy): one `.yaml` file per scenario.

```yaml
# scenarios/minimal.yaml
title: "Minimal Dungeon"
starting_room: entrance
death: respawn
rooms:
  entrance:
    name: "Entrance Hall"
    connections:
      south: { room: goblin_lair, type: open }
```

**Directory format** (new, for large scenarios): a directory with a manifest
and region sub-files. Cross-region room references work — room IDs are globally
unique.

```text
scenarios/unix-catacombs/
    scenario.yaml              ← manifest: title, starting_room, death, region list, catalogs
    regions/
        root.yaml              ← rooms in / region
        home.yaml              ← rooms in /home
```

`scenariodir.Load()` tries directory format first (`id/scenario.yaml`), then
falls back to single-file (`id.yaml`).

### JSON-RPC API Surface

The daemon exposes direct JSON-RPC 2.0 methods over NDJSON (Unix socket or TCP).
Methods (all fully-qualified): `session.init`, `session.quit`, `game.new`,
`game.list_scenarios`, `game.list_sessions`, `game.move`, `game.look`,
`game.play`, `game.context`, `game.pick_up`, `game.drop`, `game.equip`,
`game.unequip`, `game.examine`, `game.inventory`, `game.attack`,
`game.defend`, `game.flee`, `game.cast_spell`, `game.use_item`,
`game.save_game`, `game.load_game`. Session mode (normal vs passthrough) is
set during `session.init` via the `mode` field in `InitializeParams`.

The `crypt mcp` client binary provides a stdio MCP bridge for Claude Code,
translating MCP `tools/call` into daemon RPCs.

### Game Systems

| System | Implementation |
|--------|---------------|
| **Character Creation** | Point-buy stat allocation: 8 points across 6 stats (base 10 each). Interactive prompt in `-t` mode |
| **Movement** | 6-direction compass (N/S/E/W/Up/Down), locked doors, hidden exits |
| **Combat** | Turn-based: initiative (DEX+1d20), attack/defend/flee/cast, enemy AI (aggressive/cautious/scripted), XP = enemy MaxHP |
| **Armor** | Equipped armor reduces incoming damage by flat `defense` value (floor 1). Stacks with defend stance |
| **Consumables** | `use` command consumes items with effect/power (e.g. health potions heal 2d6 HP) |
| **Inventory** | Pick up/drop/equip/unequip/examine/use, weight limit (50.0), equipment slots (weapon/armor/ring/amulet) |
| **Spells** | MP cost, class gates (mage/priest), damage and heal effects, dice-based power |
| **Leveling** | Per-class XP tables (fighter cheapest → mage most expensive), max level 10, CON modifier scales HP/level, stat gains on level-up |
| **Save/Load** | Named slots (default: `quicksave`), JSON with `schema_version`, forward compat |

## Building

Requires Go 1.25+ and Node.js (for markdownlint).

```bash
make build          # build all three binaries: cryptd, crypt, crypt-admin
make build-server   # build cryptd only
make build-client   # build crypt only
make build-admin    # build crypt-admin only
make clean          # remove build artifacts
```

## Testing

```bash
make check          # quick gate: vet + test + lint + markdownlint
make check-full     # full gate: + integration + coverage + build + e2e
make test           # unit tests with race detector
make test-integration  # integration tests
make test-e2e       # end-to-end tests (builds binary first)
make coverage       # engine coverage (fails below 90%)
make lint           # staticcheck
make markdownlint   # markdown lint
```

### Test Pyramid

| Layer | Tag | Target Time | What |
|-------|-----|-------------|------|
| Unit | (none) | < 5s | Pure functions, table-driven, no I/O |
| Integration | `integration` | < 30s | Real implementations wired together, fakes for external |
| E2E | `e2e` | < 2min | Compiled binary, black-box stdin/stdout |

No real LLM, SLM, or Lux instance is ever required for CI. Every external
dependency has a fake in `internal/testutil/`.

## Demos

```bash
make demo                    # run all demos
make demo-exploration        # navigate, take items, help, quit
make demo-inventory          # take, examine, equip, unequip, drop
make demo-combat             # equip sword, fight goblin, gain XP
make demo-save-load          # save to slot, reload
make demo-unix-catacombs     # 9-room UNIX-themed dungeon crawl
make demo-solo               # interactive solo mode (rules+templates)
```

### Balance Testing

```bash
make eval-balance        # 1000 sessions, all classes, 200 max moves
make eval-balance-quick  # 100 sessions, fighter only, 50 max moves
eval-balance --scenario unix-catacombs --players 5000 --class all --workers 8
```

### SLM Setup (optional)

```bash
make ollama-setup   # install ollama, start server, pull gemma3:1b
make eval-slm       # run 65+ input accuracy eval (needs ollama)
```

## Status

M0–M2 (foundation, data contracts, thin E2E), M8 (server thin slice), M9 (DM
thin slice with Claude LLM tier), and M10 session routing (concurrent sessions,
per-session isolation) complete. See [docs/build-plan.md](docs/build-plan.md)
for the 14-milestone roadmap.

## Documentation

| File | Contents |
|------|---------|
| [DESIGN.md](DESIGN.md) | Authoritative decision log (DES-001–028) |
| [docs/architecture.pdf](docs/architecture.pdf) | Technical architecture specification (LaTeX, v0.4) |
| [docs/gameplay.pdf](docs/gameplay.pdf) | Gameplay mechanics: combat, leveling, items, balance |
| [docs/build-plan.md](docs/build-plan.md) | 13-milestone roadmap with guiding principles and red lines |
| [docs/testing.md](docs/testing.md) | Test architecture: pyramid, fixtures, fakes, CI config |
| [docs/distribution.md](docs/distribution.md) | Binary distribution: GitHub Releases, GoReleaser, Homebrew tap |
| [docs/slm-improvement.md](docs/slm-improvement.md) | SLM accuracy improvement strategy for ollama tier |

## Distribution

`cryptd` distributes as pre-built platform binaries via GitHub Releases, with
Homebrew tap and `go install` as secondary channels. The `crypt` plugin repo
owns the `install.sh` that downloads the binary; this repo owns the build
artifacts and GoReleaser config. See [docs/distribution.md](docs/distribution.md)
for the full specification.

## License

See [LICENSE](LICENSE).
