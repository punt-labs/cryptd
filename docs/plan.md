# Dungeon — Architecture Evolution Plan

## Problem Statement

The current Dungeon is a pure-prompt text adventure: Claude is the parser, rules engine,
narrator, and state machine all at once. This makes it elegant but fragile — rule enforcement
is probabilistic, performance degrades on long adventures, and there is no proper display
surface. The game cannot easily scale to multi-player (Biff integration) in its current form.

## Proposed Architecture

### Core Principle: Play Mode as First-Class Concept

The engine exposes three swappable interfaces. A **play mode** is a named combination of
implementations. All modes use the same engine, the same save files, the same scenarios.
Switching modes mid-adventure is valid — save in `dm`, resume in `solo`.

```text
┌───────────────────────────────────────────────────────────────────────────┐
│                           PLAY MODES                                      │
│                                                                           │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────┐  │
│  │  dm             │  │  solo           │  │  headless               │  │
│  │  ─────────────  │  │  ─────────────  │  │  ──────────────────────  │  │
│  │  Interpreter:   │  │  Interpreter:   │  │  Interpreter:           │  │
│  │    LLM (Claude) │  │    SLM (ollama) │  │    SLM or rules-based   │  │
│  │  Narrator:      │  │  Narrator:      │  │  Narrator:              │  │
│  │    LLM (Claude) │  │    SLM (ollama) │  │    template             │  │
│  │  Renderer:      │  │  Renderer:      │  │  Renderer:              │  │
│  │    Lux renderer  │  │    CLI renderer │  │    CLI / stdout         │  │
│  │  Engine:        │  │  Engine:        │  │  Engine:                │  │
│  │    daemon       │  │    daemon       │  │    daemon               │  │
│  └─────────────────┘  └─────────────────┘  └─────────────────────────┘  │
└───────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                  GAME ENGINE DAEMON  (`cryptd serve`)                   │
│  ─────────────────────────────────────────────────────────────────────  │
│  GameState · Character · DungeonMap · Combat · Inventory · Save/Load    │
│  Game-as-goroutine: each game exclusively owns its engine and state     │
│  Session identity via initialize handshake (session_id param)           │
│  Concurrent sessions with session resume on reconnect                   │
│  Deterministic. No LLM calls. No orchestration.                         │
└─────────────────────────────────────────────────────────────────────────┘

### The Three Interfaces

**`CommandInterpreter`** — maps player free-text input to engine actions

| Implementation | Used in | How |
|---|---|---|
| `LLMInterpreter` | `dm` | Calls back to Claude via registered MCP callback; Claude parses intent and returns a structured action |
| `SLMInterpreter` | `solo`, `headless` | Calls local ollama model (e.g. phi-3-mini) with a compact system prompt; classifies intent to action |
| `RulesInterpreter` | `headless` fallback | Regex/keyword matching over known verb-noun pairs; no model required |

**`Narrator`** — generates human-readable text from game events

| Implementation | Used in | How |
|---|---|---|
| `LLMNarrator` | `dm` | Claude generates rich, contextual narration with full adventure context |
| `SLMNarrator` | `solo` | Local SLM generates short atmospheric narration from room seed text + event |
| `TemplateNarrator` | `headless` | Fills canned templates: `"You move north. You are in the {room.name}."` |

**`Renderer`** — presents game state to the player

| Implementation | Used in | How |
|---|---|---|
| `LuxRenderer` | `dm`, `solo` | Calls Lux via MCP tools; full HUD with map, HP bars, combat UI |
| `CLIRenderer` | `headless`, SSH | ANSI terminal output; ASCII map, text HP bar, stdin input loop |

### Engine Deployment

The engine always runs as `cryptd serve`. There is no embedded engine mode. All play
modes — dm, solo, headless — connect to the daemon. See DES-025.

```text
crypt (thin client) ─────► cryptd serve    (TCP or Unix socket)
crypt plugin ──────────►       │
future web client ─────►       │
cryptd serve -t ───────► stdin/stdout (testing mode, no network)
```

- `cryptd serve` — game server daemon (JSON-RPC 2.0 over NDJSON). Supports Unix sockets
  (`--socket`) for local dev and TCP (`--listen`) for remote play. Client-agnostic.
- `cryptd serve -t --scenario <id>` — testing mode on stdin/stdout (no network, implies `-f`). Requires `--scenario`. Optional: `--script` for file input, `--json` for JSON transcript (requires `--script`).
- `cryptd serve --passthrough` — passthrough mode for Claude Code MCP clients (no
  interpreter/narrator; Claude acts as its own DM).
- `crypt` — thin CLI client. Connects to `cryptd serve`. For the default local Unix socket, auto-starts `cryptd serve` if available on PATH (no auto-start for remote/TCP via `--addr`).
- `crypt --session <id>` — resume a previous session by ID.

Play mode (interpreter + narrator selection) is server configuration via `--api-key`,
`--passthrough`, and inference tier auto-detection. The client is agnostic — it sends
text and displays responses regardless of which tier the server uses.

**Session routing (M10):** The daemon handles concurrent sessions natively via
game-as-goroutine architecture. Session identity is established in the `initialize`
handshake (`session_id` param, `has_game` optional response field; absent means false). Session resume works by
reconnecting with the same `session_id`. mcp-proxy remains a future option for Claude
Code MCP stdio multiplexing but is no longer a blocker for multi-session play.

**Daemon scope**: Game logic and push routing only. No LLM calls, no orchestration logic.
The beads project deleted 70K lines after their daemon grew too large — this daemon stays small.

### What Stays the Same Across All Modes

- Engine API (all MCP tools work identically)
- Save file format (cross-mode compatible)
- Scenario YAML format
- Combat resolution, inventory rules, map mechanics
- Lux HUD layout (DM and Solo both use it)

### Responsibility Boundaries

| What | `dm` | `solo` | `headless` |
|---|---|---|---|
| Room narration | LLM | SLM | Template |
| Free-text command parsing | LLM | SLM | Rules/regex |
| Scenario / map generation | LLM | Pre-made only | Pre-made only |
| Encounter design, lore | LLM | Pre-made only | Pre-made only |
| State transitions (move, equip…) | Engine | Engine | Engine |
| Combat resolution | Engine | Engine | Engine |
| Inventory rules | Engine | Engine | Engine |
| Fog-of-war, save/load | Engine | Engine | Engine |
| Rendering | Lux | Lux or CLI | CLI |
| Nav/combat button input | Lux→Engine | Lux→Engine | stdin→Engine |

### Input Flow (DM mode)

```text
Lux button (move N) ──────────────────────→ engine.move("N")
                                                    │
                                            state updated
                                                    │
                                            LuxRenderer.update(map, stats)
                                                    │
                            LLMNarrator called ─────┘
                            (generates room description,
                             checks for events/lore)

Free text "I search the walls for a hidden door"
         │
         ▼
  LLMInterpreter → engine.search("walls", hint="secret door")
         │
         ▼
  returns: found=true, item="lever"  OR  nothing
         │
         ▼
  LLMNarrator generates result text
         │
         ▼
  LuxRenderer.update(log entry)
```

---

## Game Engine Design (Wizardry I + Zork I inspired)

### Character

Single character per game session. Party = Biff extension (see below).

- **Classes**: Fighter, Mage, Thief, Priest (Wizardry-style)
- **Stats**: STR, INT, DEX, CON, WIS, CHA (D&D-adjacent)
- **Leveling**: XP thresholds, stat gains on level-up
- **HP / MP**: class-dependent, restored by rest/items
- **Gold**: dropped by enemies, spent in shops (Phase 4)
- **Equipment slots**: weapon, armor, ring, amulet

### Map

- Grid-based dungeon (Wizardry's 20×20 floor style)
- Pre-generated by the DM as a room graph before play starts
- Fog of war: unvisited rooms are hidden on the map canvas
- Room types: corridor, chamber, stairway (up/down), treasure vault, boss chamber
- Connections: `open`, `locked` (requires key), `hidden`, `one-way`, `trapped`

### Combat (turn-based, Wizardry-style)

- Initiative roll determines turn order
- Player actions: Attack, Cast Spell, Use Item, Flee, Defend
- Enemy AI: deterministic patterns defined in scenario YAML
- Damage: weapon dice + STR modifier vs armor class
- Conditions: poisoned, asleep, paralyzed, confused (duration-based)
- Death: configurable per scenario — permadeath or respawn at last rest point
- XP awarded on defeat; loot rolled from loot table

### Inventory

- Weight limit (STR-dependent)
- Item categories: weapon, armor, potion, scroll, key, misc
- Items can be: equipped, consumed, dropped, examined
- Lore: DM/SLM generates flavor text when examined
- Cursed items: equip-on-identify, removable only by spell

### Scenario Format (YAML)

```yaml
scenario:
  title: "The Depths of Grimhold"
  description: "A rotting keep swallowed by the earth."
  starting_room: entrance
  death: respawn          # permadeath | respawn
  character_classes: [fighter, mage, thief, priest]

rooms:
  entrance:
    name: "Dungeon Entrance"
    description_seed: "Crumbling stone arch, torchlight flickers."
    connections:
      north: {room: corridor_a, type: open}
      down:  {room: level_2,   type: stairway}
    items: [{id: torch, quantity: 2}]
    encounter: null

  goblin_lair:
    name: "Goblin Lair"
    description_seed: "Filth, bones, low ceiling. Eyes in the dark."
    connections:
      south: {room: corridor_a,    type: open}
      east:  {room: treasure_vault, type: locked, key: rusty_key}
    items: []
    encounter:
      enemies:
        - {type: goblin, count: 3, hp: 12, atk: 4, def: 2, xp: 15}
      loot_table:
        - {item: rusty_key,    chance: 0.5}
        - {item: gold,         amount: 2d6}

items:
  rusty_sword:    {name: "Rusty Sword",    type: weapon, damage: 1d6,   weight: 3, value: 5}
  health_potion:  {name: "Health Potion",  type: potion, effect: {heal: 2d4+2}, weight: 0.5}
  rusty_key:      {name: "Rusty Key",      type: key,    weight: 0.1,   value: 0}
```

### Save File Format

```json
{
  "schema_version": "1.0",
  "play_mode": "dm",
  "scenario": "grimhold",
  "timestamp": "2026-03-10T18:00:00Z",
  "character": {
    "name": "Aldric",
    "class": "fighter",
    "level": 3,
    "hp": 45, "max_hp": 60,
    "mp": 0,  "max_mp": 0,
    "stats": {"str": 16, "dex": 12, "con": 14, "int": 9, "wis": 10, "cha": 11},
    "xp": 1240,
    "gold": 48,
    "inventory": [],
    "equipped": {"weapon": "rusty_sword"},
    "conditions": []
  },
  "party": [],
  "dungeon": {
    "current_room": "goblin_lair",
    "visited_rooms": ["entrance", "corridor_a", "goblin_lair"],
    "room_state": {
      "goblin_lair":    {"cleared": true, "items_taken": ["rusty_key"]},
      "treasure_vault": {"door_locked": false}
    }
  },
  "adventure_log": []
}
```

`party` is always present (empty for single-player). `play_mode` is advisory — the engine
accepts a `--mode` override on resume. Save files live in `.dungeon/saves/<slot>.json`
(gitignored).

---

## Lux Display Layout

```text
┌─────────────────────────────────────────────────────────────────┐
│ Menu: [Game ▾] [Character ▾] [Map ▾]                            │
├──────────────────────────────────┬──────────────────────────────┤
│                                  │ ♥ HP  ████████░░ 45/60       │
│   DUNGEON MAP                    │ ✦ MP  ░░░░░░░░░░  0/0        │
│   (draw canvas, grid rooms,      │ ⚔ Fighter Lv3    XP: 1240   │
│    corridors, fog of war,        ├──────────────────────────────┤
│    player ●, locked doors ⊠,     │ EXITS                        │
│    visited/unvisited fill)       │         [↑ North]            │
│                                  │  [← West]        [East →]    │
│                                  │         [↓ South]            │
│                                  │         [▼ Stairs down]      │
├──────────────────────────────────┼──────────────────────────────┤
│ NARRATION (markdown, scrolling)  │ COMBAT (when active)         │
│                                  │ Goblin  ████░░ 8/12 HP       │
│ "The goblin lair reeks of smoke  │ [⚔ Attack]  [🎒 Use Item]    │
│  and old blood. Three goblins    │ [🛡 Defend]  [🏃 Flee]        │
│  snap at your heels."            │                              │
│                                  │ Round 2 — Your turn          │
└──────────────────────────────────┴──────────────────────────────┘
```

Update strategy:

- `show()` on scene transitions (new room, enter/exit combat)
- `update()` for incremental changes (HP tick, log append, fog reveal)
- Navigation buttons → engine directly (no LLM round trip)
- Combat action buttons → engine for resolution → Narrator for flavor text

---

## MCP Tool Interface (Game Engine)

The interface is language-agnostic. Go and Python implementations are drop-in equivalent.

### Scenario and Session

| Tool | Args | Returns |
|---|---|---|
| `new_game` | `scenario_id, char_name, char_class, mode?` | full game state |
| `load_game` | `slot?, mode?` | full game state |
| `save_game` | `slot?` | save path |
| `list_saves` | — | `[{slot, scenario, char, mode, timestamp}]` |
| `list_scenarios` | — | `[{id, title, description}]` |

### Navigation

| Tool | Args | Returns |
|---|---|---|
| `move` | `direction, character_id?` | new\_room, encounter?, items?, event? |
| `look` | — | current room: exits, items, enemies |
| `get_map` | — | room graph with visited/fog state |

### Character and Inventory

| Tool | Args | Returns |
|---|---|---|
| `get_character` | — | stats, HP, MP, level, XP, gold, conditions |
| `get_inventory` | — | item list with weights, equipped slots |
| `pick_up` | `item_id` | success, weight\_remaining |
| `drop` | `item_id` | success |
| `equip` | `item_id` | success, stat\_delta |
| `use_item` | `item_id, target?` | effect result |
| `examine` | `target` | raw detail (Narrator expands with lore) |

### Combat

| Tool | Args | Returns |
|---|---|---|
| `get_combat_state` | — | turn order, HP, conditions |
| `attack` | `target_id, weapon?, character_id?` | roll, damage, target HP, outcome |
| `cast_spell` | `spell_id, target?, character_id?` | effect, MP cost, outcome |
| `defend` | `character_id?` | defense\_bonus until next turn |
| `use_item_combat` | `item_id, target?, character_id?` | effect result |
| `flee` | `character_id?` | success (DEX check), penalty on fail |
| `end_combat` | — | xp\_gained, loot\_found |

### Search and Interaction

| Tool | Args | Returns |
|---|---|---|
| `search` | `target?, hint?` | found items/secrets or nothing |
| `interact` | `object_id, action?` | outcome (lever pulled, door opened…) |
| `rest` | — | HP/MP restored, random encounter check |

---

## Multiplayer Extension Points (designed in, not yet implemented)

The engine is party-ready from day one:

- `GameState` carries `party: []Character` (length 1 for single-player)
- `move`, `attack`, `flee`, `defend` all accept an optional `character_id`
- Combat turn order already handles multiple actors
- Save file carries a `party` array (empty until multiplayer is added)

With `cryptd serve` as a general-purpose game server (DES-025), multiplayer may
not require Biff as the transport. Players connect to the same server instance
directly. However, Biff's communication primitives (mailboxes, `/wall`, `/talk`,
presence) could still add value as a coordination layer between players — social
features on top of the game, not game state transport. This will be evaluated
when multiplayer is implemented (M13).

---

## Implementation Phases

### Phase 1 — Engine Core + All Three Modes (MVP)

Build the engine and all three interface implementations together so `dungeon solo`
is playable before the LLM layer exists. This validates the engine design first.

- Go engine: `GameState`, `Character`, `DungeonMap`, `Inventory` (no LLM dependencies)
- `CommandInterpreter` interface + `SLMInterpreter` (ollama) + `RulesInterpreter` (regex)
- `Narrator` interface + `SLMNarrator` + `TemplateNarrator`
- `Renderer` interface + `CLIRenderer` (ANSI terminal)
- `LuxRenderer` (Lux MCP tool calls)
- Scenario YAML schema + `classic-fantasy-dungeon.yaml` migration
- Save/load: `.dungeon/saves/<slot>.json`
- `cryptd serve` daemon (NDJSON over Unix socket/TCP, session identity routing)
- `crypt` auto-starts server if not running
- `cryptd serve -t` testing mode (stdin/stdout, no network)

### Phase 2 — DM Mode + Lux HUD

- `LLMInterpreter` + `LLMNarrator`: MCP callback into Claude
- SKILL.md rewritten for DM role (narration, scenario generation, lore on examine)
- Full Lux HUD: map canvas, HP/MP bars, nav buttons, narration log
- `cryptd serve --passthrough` for Claude Code MCP clients (invoked by `/crypt` skill)
- `/crypt:create` — DM generates a new scenario YAML interactively

### Phase 3 — Combat System

- Turn-based combat: initiative, weapon dice, armor class, conditions
- Enemy AI patterns from scenario YAML
- Lux combat panel: HP bars, action buttons, turn indicator
- XP, leveling, loot rolls
- Spells (Mage/Priest)

### Phase 4 — Rich World

- Shops, traps (DEX save), cursed items, locked doors
- Multiple bundled scenarios
- DM generates richer encounters with backstory and puzzles

### Phase 5 — Biff Multi-player

- Party management (one character per Biff participant)
- Turn tokens via Biff `/write`
- Shared save state
- DM narrates for the full party

---

## Open Questions

- **mcp-proxy**: Session routing is now implemented natively (M10). mcp-proxy remains
  a future option for Claude Code MCP stdio multiplexing (one shim per session, shared
  daemon) but is no longer a blocker. The engine daemon API is the same either way.
- **Permadeath default**: configurable in scenario YAML — default `respawn`
