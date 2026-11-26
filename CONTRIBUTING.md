# Contributing to cryptd

## Prerequisites

- The Go toolchain version named in [go.mod](go.mod)
- `make`
- Node.js with `npx`, for Markdown lint
- [staticcheck](https://staticcheck.dev), for `make lint`

## Build

```bash
make build
```

This writes `cryptd`, `crypt`, and `crypt-admin` to the repository root.
`make help` lists every target.

## Quality Gates

Run the quick gate before every commit: `go vet`, unit tests with the race
detector, staticcheck, and Markdown lint.

```bash
make check
```

Run the full gate before opening a pull request. It adds integration tests,
engine coverage (fails below 90%), a build, end-to-end tests, and acceptance
tests (scripted playthroughs).

```bash
make check-full
```

Both gates must finish with zero errors and zero failing tests.

### Ethos handoff snapshot

With an Ethos version that supports the DES-086 snapshot profile and
`handoff fetch-runtime`, fetch the published runtime before the first handoff
and whenever the pin changes. From the cryptd checkout on Linux amd64 or
macOS arm64, run:

```bash
ethos handoff fetch-runtime
```

Ethos verifies the archive against the project's snapshot profile before
placing it in `~/.punt-labs/ethos/runtimes/`. The handoff itself never
downloads the archive and runs the same blocking `make check` as a live
checkout.

## Playing From a Checkout

The `play` targets build the binaries and set `CRYPT_SCENARIO_DIR` for you.

```bash
make play-unix
```

`make play` uses the small `minimal` test scenario in `testdata/scenarios/`,
and `make play-mega` uses the 53-room `unix-megadungeon` scenario.

## Scripted Demos and Balance Checks

`make demo` runs the scripted playthroughs in `testdata/demos/` against
`cryptd serve -t --script`.

```bash
make demo
```

`make eval-balance-quick` runs 100 automated fighter sessions against the
`minimal` scenario and reports outcomes. `make eval-balance` runs 1000 sessions
across all classes.

```bash
make eval-balance-quick
```

## Local Language Model

To test the local-model path, install [ollama](https://ollama.com) and pull a
small model. The server prefers `gemma3:1b`, then `llama3.2:3b`, then
`smollm2:135m`.

```bash
ollama pull gemma3:1b
```

## Pull Requests

- Add an entry under `## [Unreleased]` in [CHANGELOG.md](CHANGELOG.md) for any change in behavior.
- Update [README.md](README.md) when user-facing commands, flags, or defaults change.
- Record design decisions with rejected alternatives in [DESIGN.md](DESIGN.md).

## Further Reading

- [Testing](docs/testing.md) --- test layers, fixtures, and fakes
- [Architecture (as built)](docs/architecture-as-built.md) --- packages, protocol, and concurrency
- [Distribution plan](docs/distribution.md) --- planned release channels (not yet available)
- [Build plan](docs/build-plan.md) --- milestones and planned work
