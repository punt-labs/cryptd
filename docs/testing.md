# Dungeon — Test and Verification Architecture

## Philosophy

The single most important property of the test suite is that it tells you
immediately and unambiguously whether a change broke something. That means:

- **Fast feedback at the unit level.** The engine is pure Go with no external
  dependencies. Unit tests run in milliseconds and cover every state transition.
- **Determinism is the invariant.** The engine must always produce the same
  output for the same input. Every test that touches engine logic asserts a
  deterministic outcome. No probabilistic assertions, no "roughly correct."
- **Interface seams are the injection points.** `CommandInterpreter`, `Narrator`,
  and `Renderer` are the only coupling points to external systems (LLM, SLM, Lux).
  All tests above the unit level use test doubles at these seams. Real external
  services are never required to run CI.
- **Headless mode is the CI workhorse.** `headless` uses `RulesInterpreter` +
  `TemplateNarrator` + `CLIRenderer`. It has no external dependencies and runs
  complete game sessions programmatically. It is the primary vehicle for
  integration and end-to-end tests.
- **Scripts over manual setup.** Game sequences are expressed as fixture files
  (`testdata/scripts/*.yaml`), not as bespoke test code. Scripts are readable,
  reusable, and reviewable.

---

## The Testing Pyramid

```text
               ╔════════════════╗
               ║  Acceptance    ║  ~5   Full adventure runs (headless)
               ║  (E2E scripts) ║
              ╔╩════════════════╩╗
             ║  End-to-End       ║  ~20  CLI subprocess + MCP wire smoke tests
             ║  (subprocess)     ║
            ╔╩═══════════════════╩╗
           ║  Integration          ║  ~80  Interface compositions, daemon, MCP dispatch
           ║  (in-process, fakes)  ║
          ╔╩═══════════════════════╩╗
         ║  Unit                     ║  ~200  Pure engine functions, parsers, templates
         ║  (go test, no I/O)        ║
         ╚═══════════════════════════╝
```

| Layer | Trigger | Target time | Requires |
|---|---|---|---|
| Unit | Every commit | < 5 s | Nothing |
| Integration | Every commit | < 30 s | Nothing (fakes only) |
| End-to-End | Every PR | < 2 min | Nothing (embedded fakes) |
| Acceptance | Release branch | < 10 min | Nothing (headless mode) |

No layer requires a real LLM, real SLM, or running Lux instance.

---

## Layer 1 — Unit Tests

**Location:** `internal/engine/*_test.go`, `internal/interpreter/*_test.go`,
`internal/narrator/*_test.go`

**Tooling:** `go test ./...`, table-driven tests, `testify/assert`

**Style:** Pure functions only. No goroutines, no sockets, no filesystem (except
`testdata/`). Each test provides inputs and asserts outputs — no setup/teardown
beyond struct initialization.

### Engine Core

The engine is the most critical unit test target. Every function that touches
`GameState` must have exhaustive table-driven coverage.

| Package | What to test | Key cases |
|---|---|---|
| `engine/character` | Stat bonuses, HP/MP calculation, condition apply/expire | All six stat values; boundary cases (8, 18); conditions that stack |
| `engine/combat` | Initiative order, attack roll, damage, AC reduction, flee DEX check | Hit/miss boundary (roll == AC); fumble/crit if added; multi-actor initiative |
| `engine/inventory` | Pick up/drop weight check, equip slot conflict, use-item effect | Full inventory rejection; slot already occupied; consumable removes itself |
| `engine/map` | Move through open/locked/hidden doors; fog reveal; stairway traversal | Locked door without key; hidden door pre/post search; one-way return block |
| `engine/leveling` | XP threshold crossing, stat delta per class | Level 1→2, 9→10 (boundary); class-specific thresholds |
| `engine/combat/ai` | Enemy AI pattern execution | `aggressive` always attacks; `cautious` flees at 30% HP; `scripted` follows sequence |
| `engine/save` | JSON round-trip fidelity | Full `GameState` → marshal → unmarshal → deep equal; version field present |

**Target coverage:** `internal/engine` ≥ 90 % statement coverage.

**Target coverage:** `internal/daemon` ≥ 80% statement coverage. The daemon is
the most architecturally complex package: game-as-goroutine dispatch, session
registry, concurrent accept loop, graceful shutdown. `-race` is mandatory for
all daemon tests.

### Parsers and Formats

| Package | What to test |
|---|---|
| `scenario/parser` | Valid minimal YAML loads cleanly; all required fields; missing required field → typed error; unknown field → warning not error; room with no exits; enemy template reference resolves |
| `dice/parser` | `1d6`, `2d6+3`, `1d20-1`, boundary rolls (min, max), invalid notation → error |
| `save/loader` | Load well-formed save; load save with unknown fields (forward compat); load save from wrong `version` → error |

### RulesInterpreter

Exhaustive verb/noun coverage. This interpreter never calls any model.

```text
"go north"        → {action: move, direction: N}
"n"               → {action: move, direction: N}
"attack goblin"   → {action: attack, target: "goblin"}
"a"               → {action: attack}  (default target)
"pick up sword"   → {action: pick_up, item: "sword"}
"search"          → {action: search}
"search walls"    → {action: search, target: "walls"}
"i" / "inventory" → {action: get_inventory}
"rest"            → {action: rest}
unknown input     → {action: unknown, raw: "..."}  (not an error)
```

### TemplateNarrator

Table-driven: event type + room seed → expected substring in output.

```text
{event: moved, room: "goblin_lair", seed: "reeks of smoke"}
  → output contains "goblin_lair" name AND seed text
{event: combat_start, enemies: ["goblin x3"]}
  → output contains enemy count
{event: item_found, item: "rusty_key"}
  → output contains item name
```

### Client (`cmd/crypt`)

| Test | What it proves |
|---|---|
| `TestSession_ResumeReadsInitialRoom` | Client reads and displays server's initial room on `--session` resume |
| `TestFormatBar`, `TestFormatHUD`, etc. | Display formatting (HP bars, enemy lines) |

---

## Layer 2 — Integration Tests

**Location:** `internal/**/integration_test.go`, `cmd/**/integration_test.go`

**Tooling:** `go test -tags integration ./...`

**Style:** Real implementations wired together; external services replaced with
in-process fakes. May use goroutines and in-memory channels. No subprocess
spawning; no network sockets outside `net/http/httptest`.

### Engine State Machine Sequences

Test multi-step action sequences against real engine state. This is the most
valuable integration layer — it catches regressions that unit tests miss because
they only test individual functions.

```text
Sequence: new_game → move(S) → move(E) [locked] → pick_up(key) → move(E) [unlocked]
  Assert: room progression correct; locked door blocks without key; key enables pass

Sequence: new_game → move(S) → combat begins → attack × N → end_combat
  Assert: HP decrements correctly; XP awarded; loot present in inventory

Sequence: level-up crossing
  new_game → accumulate XP via several combats → assert level increased and
  max_HP increased per class table

Sequence: save → mutate state → load → assert pre-mutation state restored
  (save is authoritative; subsequent engine calls do not bleed through)
```

### Interface Composition: Headless Mode

Wire `RulesInterpreter + TemplateNarrator + CLIRenderer` together with a real
engine. Drive input via a `bytes.Buffer`; capture output to a `bytes.Buffer`.
Assert that narration is non-empty and state is correct after each step.

This is the closest integration test to a real play session that requires zero
external services.

### MCP Tool Dispatch

For each MCP tool, assert:

1. The correct engine method is called with the correct arguments.
2. The JSON response matches the expected schema.
3. Error paths (invalid args, out-of-range values, calling `attack` outside
   combat) return structured MCP errors, not panics.

Use an in-process MCP handler — no subprocess, no stdio.

### Mock ollama HTTP Server

Use `net/http/httptest.NewServer` to serve canned responses. Test that:

- `SLMInterpreter` correctly parses a JSON action from a canned response.
- `SLMNarrator` returns the model's prose string verbatim.
- HTTP timeout triggers fallback to `RulesInterpreter`.
- Non-200 response triggers fallback.
- Malformed JSON response triggers fallback (not panic).

```go
// testdata/fixtures/ollama-move-response.json
{"response": "{\"action\": \"move\", \"direction\": \"north\"}"}
```

### Mock Lux MCP Server

An in-process fake that records all `show()` and `update()` calls and can inject
synthetic interaction events (button presses) via a channel.

Test that `LuxRenderer`:

- Calls `show()` on scene transitions (room entry, combat start/end).
- Calls `update()` (not `show()`) for incremental HP, MP, and log changes.
- Routes `recv()` events correctly to the engine as `InputEvent`s.
- Does not call `show()` for every HP tick (regression guard against performance
  anti-pattern).

### Daemon: Game-as-Goroutine Architecture

Each Game is a goroutine that owns its engine and state exclusively. Commands
flow via channel; no mutexes on game state. Tests exercise:

| Test | What it proves |
|---|---|
| `TestResumeGameLoop_NormalMode` | Disconnect → reconnect with session ID → room state preserved |
| `TestResumeGameLoop_NoGame` | Initialize without game → no RunLoop entered |
| `TestResumeGameLoop_SkipInitialRender` | new\_game path doesn't double-render initial room |
| `TestGame_Stop` | Stop() terminates game goroutine cleanly |
| `TestGame_StopDuringRunLoop` | Stop() works mid-RunLoop, no deadlock |
| `TestGame_SendDuringRunLoop` | Send() to game in RunLoop times out, doesn't hang |
| `TestGame_PanicRecovery` | Game goroutine panic doesn't crash server |
| `TestRepeatedNewGame_CleansUpOldGame` | Second new\_game on same session stops the old game goroutine |

### Daemon: Concurrent Sessions

| Test | What it proves |
|---|---|
| `TestIntegration_ConcurrentSessionIsolation` | Two concurrent TCP goroutines, independent games, race detector active |
| `TestIntegration_SessionReconnect_StatePreserved` | Real TCP reconnect preserves room and inventory |
| `TestIntegration_GracefulShutdown` | Server exits without deadlock when connections are active |

---

## Layer 3 — End-to-End Tests

**Location:** `e2e/`

**Tooling:** `go test -tags e2e ./e2e/...`; spawns real `cryptd` and `crypt`
subprocesses via `os/exec`. Requires both binaries to be built first
(`go build ./cmd/cryptd` and `go build ./cmd/crypt`).

**Style:** Black-box. Talks to the binary over stdin/stdout (`cryptd serve -t`)
or over a real socket. Asserts observable outputs only — exit codes, JSON
responses, file side effects (save files created).

### CLI Smoke Tests

| Test | Command | Assert |
|---|---|---|
| Help exits clean | `cryptd --help` | Exit 0, usage text present |
| Validate scenario | `cryptd validate testdata/scenarios/minimal.yaml` | Exit 0, valid |
| Test mode new game | `cryptd serve -t` with new\_game on stdin | Exit 0, initial state JSON |
| Test mode scripted run | `cryptd serve -t` with scripted commands on stdin | All steps pass, final state matches expected |
| Save round-trip | new\_game → save → kill → load → assert state | Save file created, state restored |

### MCP Wire Smoke

Spawn `cryptd serve` as a subprocess. Connect a minimal MCP client over stdio.
Call each tool once with valid arguments. Assert valid JSON responses and no
unexpected stderr output. This is the "does the binary actually work" check that
integration tests (in-process) cannot catch.

### Session Reconnect E2E

| Test | What it proves |
|---|---|
| `TestE2E_SessionReconnect` | Compiled binary, real socket: play → disconnect → reconnect with session ID → room preserved. Full-stack proof that session resume works. |

### Cross-Mode Save Compatibility

```text
1. Run crypt headless to a known save point.
2. Verify save file exists.
3. Load the save with crypt solo --dry-run (no ollama needed).
4. Assert GameState matches expected values.
```

---

## Layer 4 — Acceptance Tests (Game Scripts)

**Location:** `e2e/acceptance/`

**Format:** YAML game scripts that describe a complete adventure session as a
sequence of steps. The runner executes each step via `crypt headless`, asserts
the expected state, and fails on any deviation.

**Why headless?** Acceptance tests must be deterministic. `headless` uses
`RulesInterpreter` (no model) and `TemplateNarrator` (no model). The engine
outcome is always identical for the same scenario and input sequence.

### Script Format

```yaml
# testdata/scripts/complete-run.yaml
scenario: minimal
character:
  name: TestHero
  class: fighter
steps:
  - input: "go south"
    expect:
      current_room: goblin_lair
      narration_contains: "goblin"

  - input: "attack"
    expect:
      combat_active: true
      enemy_hp_reduced: true

  - input: "attack"
    repeat_until:
      combat_active: false
    max_iterations: 20
    expect:
      xp_gained: true

  - input: "go north"
    expect:
      current_room: entrance

  - input: "save"
    expect:
      save_file_exists: true
```

### Bundled Acceptance Scenarios

| Script | What it covers |
|---|---|
| `minimal-run.yaml` | Two rooms, one combat, one item, save/load |
| `combat-full.yaml` | All combat actions (attack, defend, use item, flee, cast spell) |
| `leveling.yaml` | Enough XP to cross level threshold; assert stat increase |
| `locked-door.yaml` | Find key, unlock door, traverse; search reveals hidden door |
| `permadeath.yaml` | Character death in permadeath scenario; assert game over state |

---

## Test Fixtures and Testdata Layout

```text
testdata/
├── scenarios/
│   ├── minimal.yaml          # 2 rooms, 1 enemy — used by unit and integration tests
│   ├── combat-heavy.yaml     # Many enemy types, all AI patterns
│   └── invalid/
│       ├── missing-id.yaml
│       ├── broken-exit-ref.yaml
│       └── unknown-class.yaml
├── saves/
│   ├── fighter-level-3.json  # Known mid-adventure state for load tests
│   └── near-levelup.json     # One kill away from level-up
├── scripts/
│   ├── minimal-run.yaml
│   ├── combat-full.yaml
│   ├── leveling.yaml
│   ├── locked-door.yaml
│   └── permadeath.yaml
└── fixtures/
    ├── ollama-move-north.json      # Canned ollama response: move north
    ├── ollama-attack-goblin.json   # Canned ollama response: attack
    └── ollama-narrator-move.json   # Canned ollama narration response
```

---

## Test Doubles Reference

| External System | Test Double | Used in |
|---|---|---|
| LLM (Claude) | `FakeLLMInterpreter` — returns canned action from fixture | Integration: MCP dispatch, Renderer |
| LLM (Claude) | `FakeLLMNarrator` — returns fixture string | Integration: full-loop sequences |
| LLM (Claude API) | `FakeSLMServer` with `WithAPIKey` — same server, auth header verified | Integration: LLM interpreter/narrator timeout + fallback |
| SLM (ollama) | `FakeSLMServer` — `httptest.Server` with auth header recording, configurable delay | Integration: SLM timeout fallback, auth verification |
| Lux MCP | `FakeLuxServer` — records calls, injects events via channel | Integration: LuxRenderer |
| Game goroutine | `Game.Inspect()` — runs function inside game goroutine for safe state access | Unit: game\_test.go, resume\_test.go |
| Daemon socket | In-process fake transport | Integration: mcp-proxy routing |
| CLI stdin | `bytes.Buffer` | Integration: headless mode |
| CLI stdout | `bytes.Buffer` | Integration: CLIRenderer assertions |

All fakes live in `internal/testutil/`. They implement the same Go interfaces as
the real implementations and are importable by any test package.

---

## CI Configuration

```yaml
# .github/workflows/test.yml (target layout)
jobs:
  unit-and-integration:
    runs-on: ubuntu-latest
    steps:
      - go test -race -count=1 ./...
      - go test -race -tags integration -count=1 ./...
      - go test -cover -coverprofile=coverage.out ./internal/engine/...
      - go tool cover -func=coverage.out  # fail if engine < 90%

  e2e:
    runs-on: ubuntu-latest
    steps:
      - go build -o cryptd ./cmd/cryptd
      - go build -o crypt ./cmd/crypt
      - go test -tags e2e ./e2e/...

  acceptance:
    runs-on: ubuntu-latest
    if: github.ref == 'refs/heads/main' || startsWith(github.ref, 'refs/heads/release/')
    steps:
      - go build -o cryptd ./cmd/cryptd
      - go build -o crypt ./cmd/crypt
      - go test -tags acceptance -timeout 10m ./e2e/acceptance/...
```

`-race` runs on every unit/integration pass to detect data races in the daemon's
session routing code. Acceptance tests run only on main and release branches to
keep PR feedback fast.

---

## Verification Beyond Tests

### Scenario Validation

A `crypt validate <file.yaml>` command (not a test) validates a scenario YAML
against the engine's schema before committing it. This is the safeguard for DM-authored
content — catch broken room references, unknown enemy templates, and invalid dice
notation before a player encounters them at runtime.

Run in CI as a pre-check on any changed `scenarios/*.yaml`.

### Race Detection

`go test -race` is mandatory on CI for all packages that touch the daemon's
goroutine model (session routing, push delivery). This is non-negotiable — the
daemon handles concurrent MCP connections and a data race here produces
silent state corruption.

### Save Format Forward Compatibility

When the save format changes, a migration test loads every fixture in
`testdata/saves/` (which are pinned to old versions) and asserts clean
migration to the current format. Old saves must never silently corrupt.

---

## What We Do Not Test

| Thing | Why not |
|---|---|
| LLM narration quality | Non-deterministic; evaluated by human playtest |
| SLM command accuracy against real ollama | Requires model; evaluated by model eval harness, not CI |
| Lux visual rendering | ImGui pixel output; evaluated by human inspection |
| mcp-proxy (pre-ship) | Does not exist yet; contract test stubs only |
| Permadeath roguelike balance | Game design, not correctness; playtesting |

---

## Iteration Speed Targets

The feedback loop is the product. These are the targets to preserve as the
codebase grows:

| Action | Target |
|---|---|
| `go test ./internal/engine/...` | < 2 s |
| `go test ./...` (unit + integration) | < 30 s |
| Full E2E suite | < 2 min |
| Full acceptance suite | < 10 min |

If any target is breached, investigate before adding more tests. The usual
culprits are: test that sleeps, test that dials a real socket, test that
reads more testdata than necessary. Slow tests get skipped; skipped tests
catch nothing.
