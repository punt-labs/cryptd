# Copilot Instructions — cryptd

## Project State

This repo is at **design stage** — no Go source code exists yet. All content is
architecture documentation. The intended first implementation step is Milestone 0
in `docs/build-plan.md` (test infrastructure scaffold).

The binary is named `cryptd` (repo name). The CLI subcommands are `dungeon dm`,
`dungeon solo`, `dungeon headless`, and `dungeon serve`.

## Architecture

### The L4 / L1 Split

The core architectural principle (DES-009): the LLM is the Dungeon Master
(narrator, semantic parser, scenario author) and the Go engine is the
deterministic rules machine. **No game rule or state transition is left to
probabilistic LLM output.**

```
L4 — Claude (SKILL.md)      Narrates, interprets free text, generates scenario YAML
         │ MCP tools
L1 — Go engine              State transitions, combat, inventory, fog-of-war, save/load
L1 — Lux (ImGui)            Display surface (receives element trees via MCP)
```

### Play Modes

A play mode is a named triple of interface implementations. All modes share the
same engine, the same save files (`schema_version` JSON), and the same scenario
YAML format. Switching modes mid-adventure is valid; `play_mode` in the save is
advisory and `--mode` overrides it.

| Mode       | CommandInterpreter  | Narrator          | Renderer           | Engine   |
|------------|---------------------|-------------------|--------------------|----------|
| `dm`       | LLMInterpreter      | LLMNarrator       | LuxRenderer        | daemon   |
| `solo`     | SLMInterpreter      | SLMNarrator       | LuxRenderer or CLI | embedded |
| `headless` | RulesInterpreter    | TemplateNarrator  | CLIRenderer        | embedded |

### The Three Go Interfaces

The engine never implements these — it calls them:

```go
type CommandInterpreter interface {
    Interpret(ctx context.Context, input string, state GameState) (EngineAction, error)
}
type Narrator interface {
    Narrate(ctx context.Context, event EngineEvent, state GameState) (string, error)
}
type Renderer interface {
    Render(ctx context.Context, state GameState, narration string) error
    Events() <-chan InputEvent
}
```

### Dependency Direction

Dependencies flow strictly downward. This is a build-order red line:

```
CLI / Commands
    └── Play Mode Composition (wires Interpreter + Narrator + Renderer)
            └── Interpreters / Narrators / Renderers
                    └── Engine (character, combat, inventory, map, leveling, save)
                            └── Data Contracts (model structs, scenario YAML, save JSON)
```

The engine knows nothing about interpreters. Interpreters know nothing about
narrators. Narrators know nothing about renderers.

### Engine Deployment

- **Embedded** (`solo`, `headless`): engine runs in-process, no socket.
- **Daemon** (`dm`, future multi-player): `dungeon serve` on a Unix domain socket
  (NDJSON, `net.Listen("unix", ...)`). Daemon scope is exactly two things: game
  logic resolution and session-aware push routing. No LLM calls, no orchestration.
- `mcp-proxy` (design-stage, not yet built): per-session Go shim that bridges
  Claude Code stdio MCP to the shared daemon and injects session identity.

### Data Formats

- **Scenarios:** YAML in `scenarios/`, parsed at startup. `gopkg.in/yaml.v3`.
  `description_seed` is a brief seed string; the DM/SLM/template expands it.
- **Save files:** `.dungeon/saves/<slot>.json`, `encoding/json`, `schema_version`
  field. `party` is always a `[]Character` (len 1 for single-player). Gitignored.
- **MCP schema contract:** `testdata/mcp-schema.json` is committed and CI-diffed
  against `go run ./cmd/dump-mcp-schema` output on every build.

### SLM Integration

`SLMInterpreter` and `SLMNarrator` POST to `http://localhost:11434/api/generate`
(ollama). Default model: `phi3`. Timeouts: 5s (interpreter), 10s (narrator).
If ollama is unavailable, both fall back silently — interpreter → `RulesInterpreter`,
narrator → `TemplateNarrator`.

## Build and Test Commands

No commands exist yet. The intended commands once Milestone 0 is scaffolded:

```bash
# Unit + integration (target < 30s)
go test -race -count=1 ./...
go test -race -tags integration -count=1 ./...

# Single package
go test -race ./internal/engine/combat/...

# Single test
go test -race -run TestCombatInitiative ./internal/engine/combat/...

# Engine coverage check (must be ≥ 90%)
go test -cover -coverprofile=coverage.out ./internal/engine/...
go tool cover -func=coverage.out

# E2E (requires built binary)
go build -o dungeon ./cmd/dungeon
go test -tags e2e ./e2e/...

# Acceptance (main and release branches only)
go test -tags acceptance -timeout 10m ./e2e/acceptance/...

# MCP schema contract check
go run ./cmd/dump-mcp-schema > /tmp/schema.json && diff testdata/mcp-schema.json /tmp/schema.json

# Scenario validation
go run ./cmd/dungeon validate scenarios/minimal.yaml

# Lint
go vet ./...
staticcheck ./...
npx markdownlint-cli2 "**/*.md" "#node_modules"
```

Build tags: `integration`, `e2e`, `acceptance`. Unit tests have no tag — they are
the default `go test ./...` target.

## Key Conventions

### Design Decisions Log

**Before proposing any design change, read `DESIGN.md`.** It contains 22 settled
decisions (DES-001–022) with alternatives considered and outcomes. Do not revisit
a settled decision without new evidence. Log any new decision there before
implementing it.

### TDD and Milestone Order

Follow the milestone order in `docs/build-plan.md`. Tests are written before the
code they cover within each milestone. The critical gates are **M2** (thin E2E
slice validates architecture before real mechanics) and **M9** (LLM in loop before
heavy engine investment). `go test ./...` must always be green on `main`.

### Test Doubles

All external dependencies have in-process fakes that live in `internal/testutil/`:

| External System | Fake |
|---|---|
| Claude (LLM) | `FakeLLMInterpreter`, `FakeLLMNarrator` |
| ollama (SLM) | `httptest.NewServer` serving fixture JSON |
| Lux MCP | `FakeLuxServer` (records calls, injects events via channel) |
| Daemon transport | In-process fake transport |

**No real LLM, SLM, or Lux instance is ever required to run CI.** Any test that
dials a real socket is an integration test (build tag `integration`), not a unit
test.

### Race Detection

`go test -race` is mandatory for all packages that touch the daemon's goroutine
model. The daemon handles concurrent MCP connections; a data race produces silent
state corruption.

### Headless Mode is the CI Workhorse

`headless` uses `RulesInterpreter + TemplateNarrator + CLIRenderer` — zero external
dependencies. It is the primary vehicle for integration and acceptance tests.
Acceptance tests are YAML game scripts in `testdata/scripts/` and
`e2e/acceptance/`, executed via `dungeon headless`.

### Party-Ready from Day One

`GameState.Party` is always `[]Character` (length 1 for single-player). `move`,
`attack`, `flee`, `defend` all accept an optional `character_id`. This costs
almost nothing and avoids an engine redesign when Biff multi-player (Milestone 13)
is added.

### Lux Renderer Update Strategy

- Call `lux.show()` on scene transitions (room entry, combat start/end).
- Call `lux.update()` for incremental changes (HP/MP tick, narration append, fog reveal).
- Never call `show()` for every HP tick — this is a performance regression the
  `FakeLuxServer` integration test explicitly guards against.
- Navigation and combat button presses route directly to the engine via `lux.recv()`
  with no LLM round-trip (~50ms total).

## Documentation Map

| File | Contents |
|---|---|
| `DESIGN.md` | Authoritative decision log (DES-001–022). Read before any design work. |
| `docs/build-plan.md` | 14-milestone implementation roadmap with guiding principles and red lines |
| `docs/plan.md` | Architecture evolution plan: interfaces, engine design, MCP tool surface |
| `docs/testing.md` | Full test architecture: pyramid, fixture layout, fakes reference, CI config |
| `docs/architecture.tex` / `.pdf` | Technical architecture specification (LaTeX) |
| `docs/review.md` | Compliance review of the predecessor project; gap list still relevant |
