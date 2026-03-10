# Copilot Instructions — cryptd

## No "Pre-existing" Excuse

There is no such thing as a "pre-existing" issue. If you see a problem — in code
you wrote, code a reviewer flagged, or code you happen to be reading — you fix it.
Do not classify issues as "pre-existing" to justify ignoring them. Do not suggest
that something is "outside the scope of this change." If it is broken and you can
see it, it is your problem now.

## Project State

**M0 and M1 are complete.** M2 (thin E2E slice) is the next milestone.
See `bd ready` for the current unblocked issue (`cryptd-7uw`).

The binary is named `cryptd` (repo name). The CLI subcommands are `crypt dm`,
`crypt solo`, `crypt headless`, and `crypt serve`.

## Workflow

**Always follow the punt-kit workflow standards (`../punt-kit/standards/workflow.md`):**

- **Feature branches** — never commit directly to `main`. Branch naming:
  `feat/m2-thin-e2e`, `fix/combat-initiative`, `refactor/interpreter`, etc.
- **Beads** (`bd`) for issue tracking. Before starting any milestone:
  1. `bd ready` — confirm the issue is unblocked
  2. `bd update <id> --status in_progress` — claim it
  3. `git checkout -b feat/m<N>-short-desc main` — work on a branch
  4. Commit with conventional commit format: `feat(engine): add move command`
  5. `bd close <id>` after merge
- **Conventional commits**: `feat:`, `fix:`, `refactor:`, `test:`, `docs:`, `chore:`
- **Quality gates before every commit**: `go vet ./...`, `go test -race ./...`, markdownlint 0 errors
- **Never commit directly to `main`** — this includes chore/docs changes

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
- **Daemon** (`dm`, future multi-player): `crypt serve` on a Unix domain socket
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
go build -o cryptd ./cmd/crypt
go test -tags e2e ./e2e/...

# Acceptance (main and release branches only)
go test -tags acceptance -timeout 10m ./e2e/acceptance/...

# MCP schema contract check
go run ./cmd/dump-mcp-schema > /tmp/schema.json && diff testdata/mcp-schema.json /tmp/schema.json

# Scenario validation
go run ./cmd/crypt validate scenarios/minimal.yaml

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
`e2e/acceptance/`, executed via `crypt headless`.

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

## Documentation Maintenance

Three documents must stay current — updated **in the same commit that changes
behavior**, not retroactively:

| Document | What It Tracks | When to Update |
|---|---|---|
| `CHANGELOG.md` | User-visible changes (features, fixes, breaking changes) | Every PR that changes behavior. Entry goes under `## [Unreleased]`. |
| `README.md` | User-facing docs (commands, flags, config, examples) | Every PR that changes user-facing behavior — new flags, renamed commands, changed defaults. |

**CHANGELOG is mandatory for every behavior-changing PR.** A PR that changes
user-facing behavior without a CHANGELOG entry is not ready to merge.

## Standards Authority

**`../punt-kit/`** is the Punt Labs standards repo. The following standards apply
to this project:

- [`punt-kit/standards/github.md`](../punt-kit/standards/github.md) — branch
  protection, PR workflow, Copilot review
- [`punt-kit/standards/workflow.md`](../punt-kit/standards/workflow.md) — beads,
  branch discipline, micro-commits, session close protocol

Go-specific standards do not yet exist in punt-kit. Until they do, follow the
quality gates in the **Build and Test Commands** section above. When there is a
conflict between a child repo decision and punt-kit standards, the child repo
decision wins (it may have project-specific overrides).

## Quality Gates

Every PR must pass all of these before merge. No exceptions for "minor" changes.

```bash
go vet ./...
staticcheck ./...
go test -race -count=1 ./...
go test -race -tags integration -count=1 ./...
go test -cover -coverprofile=coverage.out ./internal/engine/...
go tool cover -func=coverage.out           # must be ≥ 90%
npx markdownlint-cli2 "**/*.md" "#node_modules"
# Once .sh files exist:
shellcheck -x install.sh hooks/*.sh scripts/*.sh
```

## Issue Tracking (Beads)

```bash
bd ready          # see what's next in this project
bd done <id>      # close a bead
```

Org-wide issues (touching 2+ repos or changing a punt-kit standard) go in
`../punt-kit/`. Project-specific bugs, features, and tech debt go here.

## Workspace Conventions

- **`.tmp/`** — use for scratch files, diffs, analysis output, or any throwaway
  data during a session. Contents are gitignored. Always use `.tmp/` instead of
  `/tmp` to keep temp files visible and workspace-local.
- **`../.bin/`** — cross-repo scripts for repeated operations. Write durable
  scripts there for things you'd otherwise repeat across sessions.
- **Quarry** — semantic search is available via MCP tools (`quarry-find`,
  `quarry-list`, `quarry-show`, etc.), connected to the `punt-labs` database
  (903+ documents across all org repos). This repo is indexed as the `cryptd`
  collection. Search this repo's docs with `collection: "cryptd"`; search
  org-wide (standards, other repos) without a collection filter.

## Documentation Map

| File | Contents |
|---|---|
| `DESIGN.md` | Authoritative decision log (DES-001–022). Read before any design work. |
| `docs/build-plan.md` | 14-milestone implementation roadmap with guiding principles and red lines |
| `docs/plan.md` | Architecture evolution plan: interfaces, engine design, MCP tool surface |
| `docs/testing.md` | Full test architecture: pyramid, fixture layout, fakes reference, CI config |
| `docs/architecture.tex` / `.pdf` | Technical architecture specification (LaTeX) |
| `docs/review.md` | Compliance review of the predecessor project; gap list still relevant |
