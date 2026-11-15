# cryptd

> Text adventure game server, terminal client, and scenario tools.

[![License](https://img.shields.io/github/license/punt-labs/cryptd)](LICENSE)
[![CI](https://img.shields.io/github/actions/workflow/status/punt-labs/cryptd/test.yml?label=CI)](https://github.com/punt-labs/cryptd/actions/workflows/test.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/punt-labs/cryptd.svg)](https://pkg.go.dev/github.com/punt-labs/cryptd)
[![Working Backwards](https://img.shields.io/badge/Working_Backwards-hypothesis-lightgrey)](./prfaq.pdf)

cryptd is a single-player text adventure: a dungeon crawl you play by typing
commands such as `go north` or `attack`. It builds three programs: `cryptd`, the
game server, which applies every game rule; `crypt`, the terminal client you play
with; and `crypt-admin`, which generates and checks dungeon files (called
*scenarios*). You can also play from Claude Code through the `crypt mcp` bridge.

**Platforms:** Linux (CI runs on Ubuntu)

## Quick Start

There are no binary releases. Install from source; you need Git, `make`, and the
Go toolchain version named in [go.mod](go.mod).

```bash
git clone https://github.com/punt-labs/cryptd.git
```

```bash
cd cryptd
```

`make install` builds the three programs and copies them and the man pages to
`/usr/local`. Use `sudo` if you cannot write there, or set another prefix with
`make install PREFIX="$HOME/.local"`.

```bash
make install
```

Start a game from the repository directory. `crypt` starts the `cryptd` server
for you if it is not already running.

```bash
crypt
```

<details>
<summary>Verify the install</summary>

From the repository directory, start a game in the plain text interface with a
fighter named Ada in the bundled UNIX Catacombs scenario:

```bash
crypt --plain --scenario unix-catacombs --name Ada
```

You should see the first room, `[login_terminal]`, and a `>` prompt. Type
`quit` to leave.

</details>

<details>
<summary>Uninstall</summary>

Run from the repository directory, with the same `PREFIX` you installed with:

```bash
make uninstall
```

</details>

## Features

- **Rules in code** --- movement, combat, inventory, spells, leveling, and saves are decided by the Go engine, never by a language model
- **Plays with no AI service** --- built-in keyword parsing and fixed text templates work with nothing else installed
- **Optional language model** --- a Claude API key or a local model (ollama or llama.cpp) parses free-form commands and writes longer descriptions
- **Terminal interface** --- a full-screen interface with health and experience bars, a compass of exits, inventory, and single-key combat actions; `--plain` gives a line-by-line prompt
- **Resumable sessions** --- disconnect and reconnect to the same game with `--session`
- **Claude Code play** --- `crypt mcp` exposes game actions as tools Claude Code can call
- **Scenario generation** --- `crypt-admin` builds a dungeon from any directory tree and checks scenario files for errors

## What It Looks Like

A scripted run of the bundled scenario. Room IDs appear in brackets; the text
comes from the built-in templates.

```text
[login_terminal]
You look around Login Prompt. A green phosphor terminal flickers in the darkness. The cursor blinks steadily. UNIX System V — Catacomb Edition. You are a process, newly spawned, standing before the system's login gate. A crumpled man page lies on the floor. The filesystem stretches south. Exits: south. You see: Man Page, Short Sword.

> take man page
[login_terminal]
You pick up the Man Page.

> south
[home_dir]
You enter /home/guest. A sparse home directory. A .bashrc file glows faintly on the floor. Three paths branch out: a well-worn path west to /tmp, a narrow corridor east to /var/log, and a passage south with /etc/shadow stenciled above it. Exits: east, north, south, west. You see: Alias Shield, Health Potion.

> west
[tmp_wasteland]
You enter /tmp Wasteland. A chaotic landscape of abandoned temp files. Half-finished downloads, orphaned sockets, and stale lock files litter the ground. A Segfault Daemon shambles between the debris, corrupting everything it touches. A pipe juts from the ground nearby. Exits: east, south. You see: Pipe Connector. Combat begins! You face: Segfault Daemon

> attack
[tmp_wasteland]
You strike the Segfault Daemon for 6 damage. The Segfault Daemon attacks you for 2 damage.
```

Dice rolls differ from run to run.

## Commands

### crypt (play)

| Command | What it does |
|---------|-------------|
| `crypt` | Connect to the local server, starting it if needed, and open the game menu (new game, resume, quit) |
| `crypt --scenario <id>` | Skip the menu and start a new game in that scenario |
| `crypt --scenario <id> --name <name> --class <class>` | Start with a character name and class (`fighter`, `mage`, `priest`, `thief`) |
| `crypt --session <id>` | Resume a game by session ID; `crypt --plain` prints the ID when it connects |
| `crypt --plain` | Use the line-by-line prompt instead of the full-screen interface |
| `crypt --addr <host:port>` | Connect to a server listening on TCP |
| `crypt --socket <path>` | Connect to a server on a different Unix socket |
| `crypt mcp` | Run a bridge on standard input/output that lets Claude Code play through tool calls |

### In-Game Commands

The built-in parser accepts these words. With a language model configured, other
phrasings also work.

| Command | What it does |
|---------|-------------|
| `go north`, `north`, `n` | Move (`north`, `south`, `east`, `west`, `up`, `down`) |
| `look` | Describe the current room |
| `examine <thing>`, `x <thing>` | Describe an item |
| `take <item>`, `drop <item>` | Pick up or drop an item |
| `equip <item>`, `unequip <item>` | Wear or remove a weapon, armor, ring, or amulet |
| `use <item>` | Use a consumable, such as a health potion |
| `attack`, `defend`, `flee` | Act in combat |
| `cast <spell> [target]` | Cast a spell (mage and priest) |
| `inventory`, `i` | List what you carry |
| `save [slot]`, `load [slot]` | Save or load a game (default slot `quicksave`) |
| `help` | List commands |
| `quit` | Leave the game |

### cryptd (server)

| Command | What it does |
|---------|-------------|
| `cryptd serve` | Start the server in the background on `~/.crypt/daemon.sock` |
| `cryptd serve -f` | Run the server in the foreground |
| `cryptd serve --listen :9000` | Listen on TCP instead of the Unix socket |
| `cryptd serve --scenario <id>` | Set the default scenario for new games |
| `cryptd serve -t --scenario <id>` | Play one game directly on this terminal, with no network |
| `cryptd serve -t --scenario <id> --script <file>` | Run commands from a file, one per line |
| `cryptd serve --model <name>` | Choose the Claude model used when an API key is set |

### crypt-admin (scenarios)

| Command | What it does |
|---------|-------------|
| `crypt-admin generate --topology tree --source <dir> --title <title> --output <dir>/` | Build a scenario whose rooms follow a directory tree |
| `crypt-admin generate ... --db <file>` | Also keep an editable SQLite copy of the generated scenario |
| `crypt-admin export --db <file> --output <dir>/` | Write a SQLite copy back out as a scenario directory |
| `crypt-admin validate <path>` | Check a scenario file or directory and print `OK` |

Each room has at most six exits (north, south, east, west, up, down), so `generate` fails on a directory tree where one directory has too many neighbors.

For example, this checks the bundled directory-format scenario:

```bash
crypt-admin validate scenarios/unix-megadungeon/
```

The man pages (`man cryptd`, `man crypt`, `man crypt-admin`) list every flag.

## Setup

| Setting | Default | Purpose |
|---------|---------|---------|
| `CRYPT_SCENARIO_DIR` | `scenarios` in the current directory | Where the server looks for scenarios |
| `CRYPTD_API_KEY` | unset | Anthropic API key; enables Claude for parsing and descriptions |
| `--socket` | `~/.crypt/daemon.sock` | Unix socket for server and client |

**Scenarios.** The bundled scenarios are in the repository's `scenarios/`
directory: `unix-catacombs` (9 rooms) and `unix-megadungeon` (53 rooms). The
server resolves scenario IDs against `scenarios` in its working directory, so to
play from anywhere else, point `CRYPT_SCENARIO_DIR` at the repository's copy:

```bash
export CRYPT_SCENARIO_DIR="/path/to/cryptd/scenarios"
```

**Language model (optional).** At startup the server picks the first that works:
Claude, if `CRYPTD_API_KEY` is set and the key is accepted; then
[ollama](https://ollama.com) at `localhost:11434` (preferring `gemma3:1b`,
`llama3.2:3b`, `smollm2:135m`); then llama.cpp at `localhost:8080`; then the
built-in parser and templates. Even with a model, commands the built-in parser
recognizes skip the model, and combat results and errors use the built-in templates.

**Files.** A background server logs to `~/.crypt/cryptd.log` and records its
process ID in `~/.crypt/cryptd.pid`. Saved games go to `.dungeon/saves/` in the
server's working directory.

**Claude Code.** Register the bridge as an MCP server (a program that gives
Claude Code tools to call) in your project:

```bash
claude mcp add -s project crypt -- crypt mcp
```

Claude Code then calls game actions (`move`, `attack`, `save_game`, and others)
and writes the descriptions itself; the server only applies the rules. The
repository also contains a development plugin manifest
(`.claude-plugin/plugin.json`, named `crypt-dev`) and a `/crypt` skill in
`skills/crypt/`. No plugin is published.

## How It Works

`cryptd` holds every game and applies all rules. Clients connect to it over a Unix
socket or TCP and exchange JSON-RPC messages, one JSON object per line. One
server runs many games at once, and each connection's game is kept separate.

A session runs in one of two modes:

| Mode | Used by | Who reads your text and writes descriptions |
|------|---------|--------------------------------------------|
| Normal | `crypt` | The server: Claude, a local model, or the built-in parser and templates |
| Passthrough | `crypt mcp` (Claude Code) | Claude Code; the server returns structured results |

## Game Rules

| System | Implementation |
|--------|---------------|
| **Character Creation** | Point-buy: 8 points across 6 stats (base 10 each); four classes |
| **Movement** | Six directions (N/S/E/W/Up/Down), locked doors, hidden exits |
| **Combat** | Turn-based: initiative (DEX + 1d20), attack/defend/flee/cast, enemy behavior (aggressive/cautious/scripted), XP equal to the enemy's maximum HP |
| **Armor** | Equipped armor reduces incoming damage by its `defense` value (minimum 1 damage); stacks with defending |
| **Consumables** | `use` consumes an item for its effect (for example, health potions heal 2d6 HP) |
| **Inventory** | Weight limit 50.0; slots for weapon, armor, ring, and amulet |
| **Spells** | MP cost, limited to mage and priest, damage and healing, dice-based power |
| **Leveling** | XP tables per class (fighter cheapest, mage most expensive), maximum level 10, CON adds HP per level, stat gains on level-up |
| **Save/Load** | Named slots (default `quicksave`) stored as JSON |

## Documentation

[Architecture (as built)](docs/architecture-as-built.md) |
[Gameplay mechanics](docs/gameplay.pdf) |
[Design decisions](DESIGN.md) |
[PR/FAQ (planned direction)](prfaq.pdf) |
[Roadmap](docs/build-plan.md) |
[Contributing](CONTRIBUTING.md) |
[Changelog](CHANGELOG.md)

## License

MIT
