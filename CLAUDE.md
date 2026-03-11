# cryptd

Go game engine daemon for [crypt](https://github.com/punt-labs/crypt), a text adventure game for Claude Code.

## Principal Engineer Mindset

There is no such thing as a "pre-existing" issue. If you see a problem — in code you wrote, code a reviewer flagged, or code you happen to be reading — you fix it. Do not classify issues as "pre-existing" to justify ignoring them. Do not suggest that something is "outside the scope of this change." If it is broken and you can see it, it is your problem now.

## Project State

**M0 (foundation) and M1 (data contracts) are complete. M2 (thin E2E slice) is substantially complete on `feat/m2-thin-e2e`.**

The binary is `cryptd`. CLI subcommands: `crypt headless`, `crypt validate`. Future: `crypt dm`, `crypt solo`, `crypt serve`.

Check `bd ready` for current unblocked work.

## Architecture

### The L4 / L1 Split

Core principle (DES-009): the LLM is the Dungeon Master (narrator, semantic parser, scenario author) and the Go engine is the deterministic rules machine. **No game rule or state transition is left to probabilistic LLM output.**

```text
L4 — Claude (SKILL.md)      Narrates, interprets free text, generates scenario YAML
         │ MCP tools
L1 — Go engine              State transitions, combat, inventory, fog-of-war, save/load
L1 — Lux (ImGui)            Display surface (receives element trees via MCP)
```

### Play Modes

A play mode is a named triple of interface implementations. All modes share the same engine, save format, and scenario YAML. Switching modes mid-adventure is valid.

| Mode       | CommandInterpreter | Narrator         | Renderer    | Engine   |
|------------|-------------------|------------------|-------------|----------|
| `dm`       | LLMInterpreter    | LLMNarrator      | LuxRenderer | daemon   |
| `solo`     | SLMInterpreter    | SLMNarrator      | Lux or CLI  | embedded |
| `headless` | RulesInterpreter  | TemplateNarrator | CLIRenderer | embedded |

### The Three Interfaces

Defined in `internal/model/interfaces.go`. The engine calls these; it never implements them.

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

```text
CLI / Commands  (cmd/crypt/)
    └── Game Loop  (internal/game/)  — wires Interpreter + Narrator + Renderer
            ├── Interpreters  (internal/interpreter/)
            ├── Narrators     (internal/narrator/)
            └── Renderers     (internal/renderer/)
                    └── Engine  (internal/engine/)
                            └── Data Contracts  (internal/model/, internal/scenario/, internal/save/, internal/dice/)
```

The engine knows nothing about interpreters. Interpreters know nothing about narrators. Narrators know nothing about renderers.

### Package Map

| Package | What It Does |
|---------|-------------|
| `cmd/crypt` | CLI entry point; wires play modes |
| `cmd/dump-mcp-schema` | Generates MCP schema JSON for CI contract check |
| `internal/engine` | Deterministic game rules: `NewGame`, `Move`, `Look` |
| `internal/game` | Game loop: drives engine + interpreter + narrator + renderer |
| `internal/interpreter` | `RulesInterpreter` — keyword/regex command parsing |
| `internal/narrator` | `TemplateNarrator` — fixed-template narration |
| `internal/renderer` | `CLIRenderer` — stdout/stdin text interface |
| `internal/model` | All data types: `GameState`, `Character`, `EngineAction`, `EngineEvent`, interfaces |
| `internal/scenario` | YAML scenario parser and validator |
| `internal/save` | JSON save/load with `schema_version` and forward compat |
| `internal/dice` | Dice notation parser: `NdS`, `NdS+M`, `NdS-M` |
| `internal/testutil` | Test doubles: `FakeLLMInterpreter`, `FakeLLMNarrator`, `FakeSLMServer`, `FakeLuxServer`, `ScriptRunner` |
| `e2e` | End-to-end tests (build tag `e2e`); compiles binary and runs scripted sessions |

### Data Formats

- **Scenarios:** YAML in `testdata/scenarios/`, parsed by `internal/scenario`. `gopkg.in/yaml.v3`.
- **Save files:** JSON with `schema_version` field. `internal/save` handles marshal/unmarshal. Unknown fields silently ignored (forward compat).
- **MCP schema contract:** `testdata/mcp-schema.json` committed; CI diffs against `go run ./cmd/dump-mcp-schema`.

## Go Standards

Go-specific standards do not yet exist in punt-kit. These are the project conventions:

- **Go 1.25+**. Module path: `github.com/punt-labs/cryptd`.
- **No external dependencies beyond `testify` and `yaml.v3`** unless there is a strong reason. The engine is intentionally lightweight.
- **Table-driven tests** with `testify/assert` and `testify/require`.
- **No `interface{}` or `any` in public API** unless unavoidable (e.g., `EngineEvent.Details`).
- **Errors are values, not strings.** Use typed errors (`NoExitError`, `LockedError`, `ErrVersionMismatch`) for conditions callers need to distinguish. Wrap with `fmt.Errorf("context: %w", err)` for everything else.
- **No panics in library code.** Panics are reserved for programmer bugs (unreachable cases in exhaustive switches), never for runtime conditions.
- **`internal/` for everything.** Nothing is exported outside the module. The public API is the CLI and (future) MCP tool surface, not Go packages.
- **Build tags** for test tiers: `integration`, `e2e`, `acceptance`. Unit tests have no tag — they run with plain `go test ./...`.

## Quality Gates

Run before every commit. No exceptions for "minor" changes.

```bash
go vet ./...
go test -race -count=1 ./...
npx markdownlint-cli2 "**/*.md" "#node_modules"
```

Full gate (before PR):

```bash
go vet ./...
go test -race -count=1 ./...
go test -race -tags integration -count=1 ./...
go test -cover -coverprofile=coverage.out ./internal/engine/...
go tool cover -func=coverage.out   # engine must be >= 90%
go build -o cryptd ./cmd/crypt && go test -tags e2e ./e2e/...
npx markdownlint-cli2 "**/*.md" "#node_modules"
```

## Testing

### Test Pyramid

| Layer | Tag | Target Time | What |
|-------|-----|-------------|------|
| Unit | (none) | < 5s | Pure functions, table-driven, no I/O |
| Integration | `integration` | < 30s | Real implementations wired together, fakes for external systems |
| E2E | `e2e` | < 2min | Compiled binary, black-box stdin/stdout |
| Acceptance | `acceptance` | < 10min | YAML game scripts via `crypt headless` |

**No real LLM, SLM, or Lux instance is ever required for CI.** Every external dependency has a fake in `internal/testutil/`.

### Test Doubles

| External System | Fake | Location |
|----------------|------|----------|
| Claude (LLM) | `FakeLLMInterpreter`, `FakeLLMNarrator` | `internal/testutil/` |
| ollama (SLM) | `FakeSLMServer` (`httptest.NewServer`) | `internal/testutil/` |
| Lux MCP | `FakeLuxServer` (records calls, injects events) | `internal/testutil/` |

### Headless Mode is the CI Workhorse

`headless` uses `RulesInterpreter + TemplateNarrator + CLIRenderer` — zero external dependencies. It is the primary vehicle for integration and acceptance tests. Acceptance scripts live in `testdata/scripts/`.

### Race Detection

`-race` is mandatory for all test runs. The daemon will handle concurrent MCP connections; a data race produces silent state corruption.

## Workflow

### Branch Discipline

- **Never commit directly to `main`.** All code through PRs.
- Branch naming: `feat/m2-thin-e2e`, `fix/combat-initiative`, `refactor/interpreter`
- Conventional commits: `feat(engine):`, `fix(renderer):`, `refactor(interpreter):`, `test:`, `docs:`, `chore:`

### Beads Issue Tracking

```bash
bd ready                              # what's next
bd update <id> --status in_progress   # claim it
bd close <id>                         # done
bd sync                               # push to remote
```

Org-wide issues (touching 2+ repos or changing a punt-kit standard) go in `../punt-kit/`. Project-specific work goes here.

### Milestone Order

Follow the build plan in `docs/build-plan.md`. The critical integration gates are **M2** (architecture validated end-to-end before real mechanics) and **M9** (LLM in the loop before heavy engine investment). `go test ./...` must always be green on `main`.

### Session Close Protocol

```bash
git status
git add <files>
bd sync
git commit -m "..."
bd sync
git push
```

## Design Decisions

**Before proposing any design change, read `DESIGN.md`.** It contains 22 settled decisions (DES-001–022) with alternatives considered and outcomes. Do not revisit a settled decision without new evidence. Log any new decision there before implementing.

## Documentation Maintenance

Updated **in the same PR that changes behavior**, not retroactively:

| Document | When to Update |
|----------|---------------|
| `CHANGELOG.md` | Every PR that changes behavior. Entry under `## [Unreleased]`. **Mandatory.** |
| `README.md` | Every PR that changes user-facing behavior (flags, commands, defaults). |
| `DESIGN.md` | Every design decision, before implementation. |

## Distribution

`cryptd` distributes as pre-built platform binaries via GitHub Releases, with
Homebrew tap and `go install` as secondary channels. The `crypt` plugin repo
owns the `install.sh` that downloads the binary; this repo owns the build
artifacts and GoReleaser config. See [docs/distribution.md](docs/distribution.md)
for the full specification.

## Documentation Map

| File | Contents |
|------|---------|
| `DESIGN.md` | Authoritative decision log (DES-001–022) |
| `docs/build-plan.md` | 14-milestone roadmap with guiding principles and red lines |
| `docs/plan.md` | Architecture evolution plan: interfaces, engine design, MCP tool surface |
| `docs/testing.md` | Full test architecture: pyramid, fixture layout, fakes reference, CI config |
| `docs/distribution.md` | Binary distribution: GitHub Releases, GoReleaser, Homebrew tap, trust tiers |
| `docs/architecture.tex` / `.pdf` | Technical architecture specification (LaTeX) |
| `docs/review.md` | Compliance review of predecessor project; gap list still relevant |

## Standards Authority

**`../punt-kit/`** is the Punt Labs standards repo. Applicable standards:

- [`punt-kit/standards/github.md`](../punt-kit/standards/github.md) — branch protection, PR workflow
- [`punt-kit/standards/workflow.md`](../punt-kit/standards/workflow.md) — beads, branch discipline, micro-commits

When this file conflicts with punt-kit standards, this file wins (project-specific overrides).

## Workspace Conventions

- **`.tmp/`** — scratch files, diffs, throwaway data. Gitignored. Use instead of `/tmp`.
- **`../.bin/`** — cross-repo scripts for repeated operations.
- **Quarry** — semantic search via MCP tools, connected to the `punt-labs` database. This repo is indexed as the `cryptd` collection.
