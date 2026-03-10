# cryptd

The crypt game engine daemon.

`cryptd` is the Go backend for [crypt](https://github.com/punt-labs/crypt) — a
text adventure game for Claude Code. It implements the game engine, all play
modes, and the MCP server interface.

## Play Modes

| Mode | Command | Description |
|---|---|---|
| `dm` | `cryptd dm` | LLM as Dungeon Master; Lux HUD display |
| `solo` | `cryptd solo` | SLM narration; no Claude required |
| `headless` | `cryptd headless` | Fully automated; no display, no model |
| daemon | `cryptd serve` | Shared engine for multi-session DM mode |

## Status

Design stage. See [docs/build-plan.md](docs/build-plan.md) for the
implementation roadmap.

## Documentation

- [docs/architecture.pdf](docs/architecture.pdf) — Technical architecture specification
- [docs/build-plan.md](docs/build-plan.md) — Development build plan (14 milestones)
- [docs/testing.md](docs/testing.md) — Test and verification architecture
- [docs/plan.md](docs/plan.md) — Architecture evolution plan
- [DESIGN.md](DESIGN.md) — Architectural decision records (DES-001–022)
