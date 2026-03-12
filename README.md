# cryptd

Game engine and server for [crypt](https://github.com/punt-labs/crypt) — a text
adventure game playable via Claude Code, CLI, or (future) web client.

## Two Binaries

| Binary | Role | Description |
|--------|------|-------------|
| `cryptd` | **Server** | Game engine as a network service |
| `crypt` | **Client** | Player-facing CLI with embedded and network modes |

### cryptd (server)

```bash
cryptd serve --listen :9000          # TCP (remote play, multiplayer)
cryptd serve --socket ~/.crypt/d.sock  # Unix socket (local dev)
```

The server exposes 15 MCP tools as JSON-RPC 2.0 over NDJSON. It holds game
state, resolves all game logic deterministically, and accepts connections from
any client. It does not know or care what brain (LLM/SLM/templates) or UI
(CLI/Lux/Claude Code) the client uses.

### crypt (client)

```bash
crypt connect --server host:9000     # play via remote server
crypt solo --scenario minimal        # embedded engine, local SLM
crypt headless --scenario minimal    # embedded engine, templates
crypt autoplay --scenario minimal --script file  # scripted playback
```

Embedded modes (`solo`, `headless`, `autoplay`) run the engine in-process — no
server required. `connect` mode speaks the same protocol as the server.

The [crypt plugin](https://github.com/punt-labs/crypt) for Claude Code connects
to `cryptd serve` and provides the MCP tool surface + SKILL.md for DM mode.

## Architecture

Three orthogonal axes:

| Axis | Options |
|------|---------|
| **Engine access** | Embedded (in-process) or Server (network) |
| **Brain** | LLM (Claude), SLM (ollama), Templates |
| **UI** | CLI, Lux (ImGui), Claude Code conversation |

See [DES-025](DESIGN.md) for the full design rationale.

## Status

M8 (daemon thin slice) in progress. See [docs/build-plan.md](docs/build-plan.md)
for the implementation roadmap.

## Documentation

- [docs/architecture.pdf](docs/architecture.pdf) — Technical architecture specification
- [docs/build-plan.md](docs/build-plan.md) — Development build plan (14 milestones)
- [docs/testing.md](docs/testing.md) — Test and verification architecture
- [docs/plan.md](docs/plan.md) — Architecture evolution plan
- [DESIGN.md](DESIGN.md) — Architectural decision records (DES-001–025)
- [docs/distribution.md](docs/distribution.md) — Binary distribution specification
