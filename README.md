# cryptd

Game engine and server for [crypt](https://github.com/punt-labs/crypt) — a text
adventure game playable via Claude Code, CLI, or (future) web client.

## Two Binaries

| Binary | Role | Description |
|--------|------|-------------|
| `cryptd` | **Server** | Game engine, interpreter, and narrator as a network service |
| `crypt` | **Client** | Player-facing CLI — connects to the server, displays the game |

### cryptd (server)

```bash
cryptd serve                           # daemonize, default Unix socket
cryptd serve -f --listen :9000         # foreground, TCP
cryptd serve -t --scenario minimal     # testing mode (stdin/stdout, no network)
cryptd validate scenario.yaml          # validate scenario YAML
```

The server owns the game engine, command interpreter (SLM with Rules fallback),
and narrator (SLM with Template fallback). Normal mode accepts free text from
the player and returns narrated display text. Passthrough mode (`--passthrough`)
exposes the raw MCP tool surface for Claude Code.

### crypt (client)

```bash
crypt                                  # connect to local server (auto-starts if needed)
crypt --addr host:9000                 # connect to remote server
crypt --scenario unix-catacombs        # auto-start with scenario
```

The client connects to `cryptd serve`, sends natural language text, and renders
the game with readline input and ASCII status display. If the server is not
running on the local socket, the client forks it automatically.

The [crypt plugin](https://github.com/punt-labs/crypt) for Claude Code connects
to `cryptd serve --passthrough` for raw MCP tool access.

## Status

M8 (server thin slice) is complete. See [docs/build-plan.md](docs/build-plan.md)
for the roadmap.

## Documentation

- [docs/architecture.pdf](docs/architecture.pdf) — Technical architecture specification
- [docs/build-plan.md](docs/build-plan.md) — Development build plan
- [docs/testing.md](docs/testing.md) — Test and verification architecture
- [docs/plan.md](docs/plan.md) — Architecture evolution plan
- [DESIGN.md](DESIGN.md) — Architectural decision records (DES-001–026)
- [docs/distribution.md](docs/distribution.md) — Binary distribution specification
