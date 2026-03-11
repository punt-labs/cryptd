# Copilot Instructions — cryptd

## GitHub PR Context: No Sub-PRs

If you are running as a GitHub-triggered agent on a pull request or issue
(not as the local Copilot CLI), do NOT create branches or sub-PRs. Post
review comments only. The `copilot-pull-request-reviewer` app handles
code review on this repo; the coding agent should not push code.

## No "Pre-existing" Excuse

There is no such thing as a "pre-existing" issue. If you see a problem — in code
you wrote, code a reviewer flagged, or code you happen to be reading — you fix it.
Do not classify issues as "pre-existing" to justify ignoring them. Do not suggest
that something is "outside the scope of this change." If it is broken and you can
see it, it is your problem now.

## Project State

**M0–M2 are complete.** The engine, headless mode, and game loop are implemented
and tested. See `bd ready` for the next unblocked work.

The binary is `cryptd`. Working CLI subcommands: `cryptd headless`,
`cryptd validate`. Future: `cryptd dm`, `cryptd solo`, `cryptd serve`.

## Code Review Workflow

Every milestone ends with a PR merged to `main`. The review cycle:

1. **Create the PR** — always request `copilot-pull-request-reviewer` at creation:
   ```bash
   gh pr create --title "..." --body "..." --reviewer copilot-pull-request-reviewer
   ```
2. **Wait for reviews** — both `copilot-pull-request-reviewer` (GitHub Copilot) and
   Cursor Bugbot will post inline review comments.
3. **Fix all real issues** — commit fixes to the feature branch. No "pre-existing"
   exemptions.
4. **Request re-review** — add the reviewer again:
   ```bash
   gh pr edit <number> --add-reviewer copilot-pull-request-reviewer
   ```
5. **Repeat** until the last round surfaces no substantial new issues.
6. **Merge** — squash or merge commit, then close the milestone epic bead.

### Copilot Reviewer vs. Copilot Coding Agent

These are two completely different GitHub features:

| Feature                         | What it does                                   | How it's triggered                                      |
|---------------------------------|------------------------------------------------|---------------------------------------------------------|
| `copilot-pull-request-reviewer` | Posts inline review comments on PRs            | Added as a PR reviewer                                  |
| Copilot coding agent            | Creates branches and sub-PRs with code changes | `@copilot` mentions, issue assignment, or repo settings |

**We use only the reviewer.** The coding agent is not part of this workflow.
If it creates sub-PRs, close them without merging and delete the branches.

## Architecture and Conventions

See `CLAUDE.md` for the full specification. Key points for reviewers:

- **L4/L1 split**: LLM is the Dungeon Master; Go engine is the deterministic
  rules machine. No game rule in LLM output.
- **Three interfaces**: `CommandInterpreter`, `Narrator`, `Renderer` — all in
  `internal/model/interfaces.go`.
- **Dependency direction**: strictly downward. Engine knows nothing about
  interpreters. Interpreters know nothing about narrators.
- **Headless mode** is the CI workhorse — zero external dependencies.

## Quality Gates

Every PR must pass before merge:

```bash
go vet ./...
go test -race -count=1 ./...
go test -race -tags integration -count=1 ./...
go test -cover -coverprofile=coverage.out ./internal/engine/...
go tool cover -func=coverage.out           # engine must be >= 90%
staticcheck ./...
npx markdownlint-cli2 "**/*.md" "#node_modules"
```

## Documentation Map

| File                             | Contents                                                              |
|----------------------------------|-----------------------------------------------------------------------|
| `CLAUDE.md`                      | Full project specification: architecture, standards, workflow, testing |
| `DESIGN.md`                      | Authoritative decision log (DES-001–022)                              |
| `docs/build-plan.md`             | 14-milestone roadmap with guiding principles and red lines            |
| `docs/plan.md`                   | Architecture evolution plan: interfaces, engine design, MCP tools     |
| `docs/testing.md`                | Full test architecture: pyramid, fixture layout, fakes, CI config     |
| `docs/distribution.md`           | Binary distribution: GitHub Releases, GoReleaser, Homebrew, trust tiers |
| `docs/architecture.tex` / `.pdf` | Technical architecture specification (LaTeX)                          |
| `docs/review.md`                 | Compliance review of predecessor project; gap list still relevant     |
