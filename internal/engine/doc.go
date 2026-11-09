// Package engine implements the deterministic game rules machine: the
// Dungeon Master. It resolves all state transitions — movement and look,
// inventory (pick up, drop, equip, unequip, examine, use), turn-based combat
// (start, attack, defend, flee, enemy turns), spells, leveling, and
// save/load. The engine holds no state of its own; every method operates on a
// caller-owned *model.GameState. It never calls LLM or SLM services —
// "deterministic" means the rules run in code, not in a model (see
// docs/architecture-as-built.md and DES-030).
package engine
