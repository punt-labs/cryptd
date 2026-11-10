# Research: Open-source neighbors of crypt (deterministic engine + LLM)
**Date:** 2026-11-09
**Request:** Map open-source and commercial projects similar to crypt for the competitive-landscape and differentiation FAQs; find out whether "deterministic rules engine + LLM narration, LLM never decides rules" already exists.
**Claims investigated:** 5 categories plus the novelty claim

Method: web search summaries plus WebFetch of GitHub READMEs on 2026-11-09. Stars and commit counts are as shown on the page; last-commit dates were not exposed by the fetcher, so maturity is stated as commit count. READMEs are self-descriptions, not independent evaluations. Extends research-2026-11-09-llm-mud-landscape.md (Thread 4), which found about 8 hybrid repos; this pass fetched the READMEs.

## Evidence Found

### Headline finding
**Claim**: "Deterministic rules engine owns rules, LLM narrates, LLM never decides a rule" is a differentiator.
**Verdict**: CONTRADICTED as a novelty claim. At least 8 open-source repos state the same thesis, several in almost the same words. All fetched ones are young (0 to 46 stars, 64 to 3,032 commits). None has a large community.
**Nearest neighbors to crypt (read their READMEs before the PR/FAQ ships):**
1. kliuchnikovv/dnd: Go, MIT, console game, deterministic Go core that "never calls a model"; models only narrate. Has a fallback chain across providers and runs fully offline with the narrator disabled. Reproducible from (case, seed, input). 0 stars, 332 commits. Closest to crypt in language, terminal form, and tiered/offline story. Differs: detective-case game, not a dungeon crawl; Master/Actor/Guard model roles; no MCP bridge or multiplayer found.
2. Mnehmos/rpg-mcp: TypeScript, ISC (README) or MIT (GitHub sidebar), conflicting; 46 stars, 128 commits. "LLMs propose, never execute." Engine rolls dice, checks AC, updates HP, validates spells. MCP server, so it runs in Claude Code. Closest to crypt's MCP angle and the "LLM never decides" wording. Differs: MCP-only; no TUI, no local-model or rules-only fallback tier found.
3. thegreystone/rpg-mcp: Java (Quarkus), BSD-3-Clause, 5 stars, 90 commits. "The AI improvises the fiction; the engine owns the truth." Server-side dice, SQLite state, audited GM override. MCP server. SRD 5.2.1 rules. Differs from crypt: a GM override exists (the LLM can bend rules under audit); MCP only.
4. shadowgate15/the-dungeon-master: TypeScript (SvelteKit, SQLite), license not stated (SRD content CC-BY-4.0), 0 stars, 64 commits. Rules, dice, HP, inventory in code; LLM "never invents numbers or state" and acts through tools. Local-first, runs with a local model. Closest in written philosophy. No rules-only fallback found.
5. slowbutfast/open-dungeon: Node plus Python, MIT, 0 stars, 178 commits. Zork-style text adventure, LLM output is a "proposal" the engine validates; score, moves, location are engine-computed. Has both a web app and an MCP server. `MOCK_LLM=1` gives a scripted-narrator mode with no model. Closest to crypt's "playable with no LLM" claim.
6. Open Dungeon Master / Open Game Master (AlecVlad/open-game-master, a fork of Lebbitheplow/open-game-master; opengamemaster.com): TypeScript/Next.js, MIT, SQLite, 3,032 commits (largest repo seen). Server-side engines own dice, HP, slots, conditions; model changes state only through validated tools. Local model, API key, or CLI agent. No full no-LLM fallback described (a storyteller is required).
7. datashaman/ai-ttrpg (from prior research, not fetched): "deterministic rules, event-sourced state"; "the LLM proposes; the engine disposes"; mechanics usable without an LLM.
8. Also reported in search summaries only (not fetched): timoncool/dungeon-ultimate (offline D&D 5e engine fork of open-dungeon), dguywhoknows/dungeon-narrator (roguelike, seeded maze, d20, AI adjudicates free-form actions as structured JSON the engine clamps), EdwardAThomson/DungeonGPT-JS (code d20, AI narrates), danielbsimpson/personal-dungeon-master ([ROLL] tags resolved by RNG), Bobby-Gray/open-tabletop-gm (LLM-agnostic GM framework, mechanics in Python), west132/AI-GM-Engine (rules spec plus stateless Python helper).
**What crypt still appears to differ on (needs verification against these repos and crypt's DESIGN.md):**
- Go engine with three modes of narration (large API model, small local model, rules plus templates) in one product, with a terminal TUI and Claude Code bridge both first-class. kliuchnikovv/dnd matches the Go, terminal, and fallback parts; open-dungeon matches the mock-narrator and MCP parts. No single repo found matches all.
- Party of humans plus LLM-played characters in one session. Not found in any fetched hybrid repo. TaleWeave AI (prior research, MIT, 14 stars) has AI characters and multiplayer, but its LLM drives character actions and the engine does not own rules to the same degree.
- Both caveats rest on README skims, not code reads.
**Recommendation**: Remove any "first" or "only" language. Write the thesis as an established pattern and name two or three neighbors in the competitive FAQ. Place the differentiation in the combination (offline tiers, terminal plus MCP, human/LLM party) and verify each element first.

### Category 1: Open-source LLM dungeon masters and AI text adventures (pure LLM)
**Verdict**: SUPPORTED. The big, popular open-source projects here are pure-LLM front ends with no rules engine.
- KoboldCpp (LostRuins/koboldcpp): C++/Python, AGPL-3.0, about 12.2k stars. Local GGUF inference server with the KoboldAI Lite web UI, which includes an adventure mode. State lives only in the prompt (context, memory, author's note, world info). Pure-LLM. Specific adventure-mode details were not found in this pass.
- SillyTavern: JavaScript, AGPL-3.0, about 35k stars, 300+ contributors, last push reported Oct 2, 2026. Chat and roleplay front end over many LLM back ends; character cards, lorebooks, visual novel mode. Pure-LLM; any "RPG" is prompt convention or user scripting. TavernAI is its ancestor (Feb 2023 fork).
- Oobabooga text-generation-webui: not researched in this pass.
- Open AI Dungeon clones: no popular one found; the live ones in search results are the hybrid repos above.
**Recommendation**: Frame these as a different category (collaborative storytelling front ends). They have the community; the hybrid repos do not.

### Category 2: Open-source tabletop and D&D tools with LLMs
**Verdict**: SUPPORTED. Two groups.
- Standalone 5e apps with an engine (listed above: Open Game Master, rpg-mcp x2, the-dungeon-master). These are the nearest neighbors.
- Foundry VTT modules (search summaries; star counts not collected). Foundry has its own dnd5e rules automation, so the modules inherit a real ruleset, but the LLM often drives actions:
  - cjkennedy1972/foundryvtt-ai-gm: autonomous 5e GM, LLM generates campaigns and NPC-turn prompts, works with midi-qol. Hybrid in the sense that Foundry's engine resolves rolls; LLM decision authority over rulings not verified.
  - Hogun974/lakour-mj-ia: AI narrator (Claude or ChatGPT) with dnd5e. Narrator-focused.
  - Foundry MCP (alexivenkov/foundry-api-bridge, foundry-mcp.com): module plus MCP server so an MCP client can roll dice, run combat, move tokens. Engine is Foundry's; the LLM calls tools.
  - rayners/fvtt-foundry-familiar (MIT), WalrusM0rj/llm-chat (MIT), daxiongmao87/Legend Lore: GM assistants and content generators; no rules role.
- Research-side contrast: CriticaLLM (arXiv 2610.21525) uses a second LLM as rules judge. Rule enforcement by LLM is the opposite design from crypt.
**Recommendation**: Say "Foundry modules put an LLM on top of an existing VTT ruleset; crypt ships its own engine." Check licenses before naming any.

### Category 3: Commercial products (brief)
**Verdict**: PARTIALLY SUPPORTED; sourced from prior research only, not re-verified.
- AI Dungeon (Latitude): pure-LLM with Story Cards and memory aids. Company-reported 8M+ registered players. No rules engine.
- NovelAI: pure-LLM story and adventure generation; not researched here.
- Hidden Door: not researched here; do not describe mechanics without checking.
- RoleForge, Friends & Fables (fables.gg), DungeonsDeep.ai: marketing claims of rules engine plus LLM narrator; unverified.
- Character.ai-style roleplay: chat only, no mechanics (general knowledge, not sourced here).
- Infobip D&D engine (ShiftMag, Sep 2025): company reports dropping pure-LLM after accuracy fell below 65 percent; moved dice and turn order to code.

### Category 4: LLM-in-terminal and MCP/Claude Code games
**Verdict**: SUPPORTED. This is less rare than assumed; at least 6 MCP game servers exist.
- punt-labs/dungeon: the org's own earlier project (Claude Code plugin; entire engine is a SKILL.md prompt plus a Node MCP server for state I/O). Pure-LLM referee by design. The repo URL returned 404 during this pass (renamed or made private); recheck before citing. Search listing described it as "clickable prototype stage."
- shawnrushefsky/dmcp: TypeScript, MIT, 14 stars, 92 commits. MCP server for AI DMs. Dice and some ability checks are code, but rules are "agent-designed" and stored as state; combat tools mostly track state. Mostly LLM-owned rules. This sits on the pure-LLM side of the boundary.
- Compose-and-Dragons/dungeon-mcp-server: MCP dungeon crawl with ASCII map, NPCs, monsters. Rules enforcement not checked.
- bellsanct/mcp-dungeon-game: Japanese idle/incremental dungeon crawler MCP server. Not LLM-narrated by design as far as seen.
- Nexlen/mud-mcp (prior research): MCP MUD server with LLM NPC dialogue.
- travisjones-dev/adventure-game: Python, Claude API tool-use for inventory. LLM as DM with a persistent inventory store; not an engine.
- Mnehmos/rpg-mcp, thegreystone/rpg-mcp, open-dungeon: hybrid MCP servers (above).
- Terminal TUI plus LLM: kliuchnikovv/dnd (console). No Bubble Tea or similar LLM TUI game found specifically.
**Recommendation**: Do not claim "Claude Code game" as new. The distinguishing claim has to be the hybrid engine plus tiered fallback, not the MCP channel.

### Category 5: Frameworks for building LLM games
**Verdict**: SUPPORTED, with few that match build-vs-adopt.
- Evennia (Python, BSD-licensed per general knowledge, not re-verified here): mature MUD framework; rules written by the developer in Python; has an LLM contrib (see prior research). Possible foundation, but Python and MUD-oriented.
- TextWorld (Microsoft, Python): generates and simulates text games for RL agents; used as the engine core in Peng et al. 2024. Research tool, not a product framework.
- Concordia (Google DeepMind, cited via the OpenGame paper): LLM-driven narrative simulation that substitutes mechanics with LLM adjudication; opposite side of the boundary. Not fetched.
- open-tabletop-gm and west132/AI-GM-Engine: GM frameworks with rules in code; small.
- STORY2GAME (arXiv 2505.03547): LLM generates content for a minimal game engine; research prototype.
- Nothing found that is a Go library for this. Build-vs-adopt: adoptable pieces exist in TypeScript and Python; none found in Go beyond kliuchnikovv/dnd.
**Not found**: a widely adopted, maintained framework that provides a deterministic rules core plus LLM adapter interfaces as a reusable library.

## Bibliography Entries

```bibtex
@misc{kliuchnikovv2026dnd,
  author       = {{kliuchnikovv}},
  title        = {dnd: a console detective RPG with a deterministic Go rules engine and an LLM narrator},
  year         = {2026},
  url          = {https://github.com/kliuchnikovv/dnd},
  urldate      = {2026-11-09},
  note         = {MIT, Go, 0 stars, 332 commits. Core never calls a model; fallback chain; offline with narrator disabled. Year is an estimate.},
}

@misc{mnehmos2026rpgmcp,
  author       = {{Mnehmos}},
  title        = {rpg-mcp: rules-enforced RPG backend for LLM game masters},
  year         = {2026},
  url          = {https://github.com/Mnehmos/rpg-mcp},
  urldate      = {2026-11-09},
  note         = {TypeScript, 46 stars, 128 commits. License conflicts between sidebar (MIT) and README (ISC). "LLMs propose, never execute." Year is an estimate.},
}

@misc{thegreystone2026rpgmcp,
  author       = {{thegreystone}},
  title        = {rpg-mcp: persistent rules-aware RPG engine as an MCP server},
  year         = {2026},
  url          = {https://github.com/thegreystone/rpg-mcp},
  urldate      = {2026-11-09},
  note         = {Java (Quarkus), BSD-3-Clause, 5 stars, 90 commits. Engine owns dice and state; audited GM override. Year is an estimate.},
}

@misc{shadowgate152026dungeonmaster,
  author       = {{shadowgate15}},
  title        = {the-dungeon-master: an AI dungeon master with deterministic rules and state},
  year         = {2026},
  url          = {https://github.com/shadowgate15/the-dungeon-master},
  urldate      = {2026-11-09},
  note         = {TypeScript, SQLite, 0 stars, 64 commits. Application license not stated; SRD 5.1 content CC-BY-4.0. Year is an estimate.},
}

@misc{slowbutfast2026opendungeon,
  author       = {{slowbutfast}},
  title        = {open-dungeon: open source AI Dungeon with an engine-validated LLM narrator},
  year         = {2026},
  url          = {https://github.com/slowbutfast/open-dungeon},
  urldate      = {2026-11-09},
  note         = {MIT, Node and Python, 0 stars, 178 commits. Web app plus MCP server; MOCK_LLM mode needs no model. Year is an estimate.},
}

@misc{alecvlad2026opengamemaster,
  author       = {{AlecVlad}},
  title        = {open-game-master (fork of Lebbitheplow/open-game-master; formerly Open Dungeon Master)},
  year         = {2026},
  url          = {https://github.com/AlecVlad/open-game-master},
  urldate      = {2026-11-09},
  note         = {MIT, TypeScript, 3,032 commits. Server-side engines own 5e rules; local-first. Star count on this fork not meaningful. Year is an estimate.},
}

@online{opengamemaster2026site,
  author       = {{Open Game Master}},
  title        = {Open Game Master: AI Game Master for D\&D 5e, on your machine},
  year         = {2026},
  url          = {https://opengamemaster.com/},
  urldate      = {2026-11-09},
  note         = {Project site. Tagline: the narrator never owns the numbers.},
}

@misc{rushefsky2026dmcp,
  author       = {Rushefsky, Shawn},
  title        = {dmcp: MCP server for AI dungeon masters in text-based RPGs},
  year         = {2026},
  url          = {https://github.com/shawnrushefsky/dmcp},
  urldate      = {2026-11-09},
  note         = {MIT, 14 stars, 92 commits. Dice and some checks in code; rule systems are agent-designed. Mostly LLM-owned rules. Year is an estimate.},
}

@misc{lostruins2026koboldcpp,
  author       = {{LostRuins}},
  title        = {KoboldCpp},
  year         = {2026},
  url          = {https://github.com/LostRuins/koboldcpp},
  urldate      = {2026-11-09},
  note         = {AGPL-3.0, about 12.2k stars (search summary). Local inference with KoboldAI Lite UI incl. adventure mode. Pure-LLM.},
}

@misc{sillytavern2026,
  author       = {{SillyTavern contributors}},
  title        = {SillyTavern: LLM Frontend for Power Users},
  year         = {2026},
  url          = {https://github.com/SillyTavern/SillyTavern},
  urldate      = {2026-11-09},
  note         = {AGPL-3.0, about 35k stars (search summary), last push Oct 2026. Chat and roleplay front end. Pure-LLM.},
}

@misc{cjkennedy2026foundryaigm,
  author       = {{cjkennedy1972}},
  title        = {foundryvtt-ai-gm: AI Game Master for D\&D 5e in Foundry VTT},
  year         = {2026},
  url          = {https://github.com/cjkennedy1972/foundryvtt-ai-gm},
  urldate      = {2026-11-09},
  note         = {From search summary only; license and stars not collected.},
}

@misc{alexivenkov2026foundrymcp,
  author       = {{alexivenkov}},
  title        = {Foundry MCP / foundry-api-bridge},
  year         = {2026},
  url          = {https://foundry-mcp.com/},
  urldate      = {2026-11-09},
  note         = {Open-source Foundry module plus MCP server; lets MCP clients roll dice and run combat through Foundry's own rules. From search summary only.},
}

@misc{puntlabs2026dungeon,
  author       = {{Punt Labs}},
  title        = {dungeon: a text adventure engine for Claude Code},
  year         = {2026},
  url          = {https://github.com/punt-labs/dungeon},
  urldate      = {2026-11-09},
  note         = {Prompt-only engine (SKILL.md) plus Node MCP state server. URL returned 404 on 2026-11-09; verify status before citing.},
}

@misc{microsoft2018textworld,
  author       = {C{\^o}t{\'e}, Marc-Alexandre and others},
  title        = {{TextWorld}: A Learning Environment for Text-based Games},
  year         = {2018},
  eprint       = {1806.11532},
  archiveprefix = {arXiv},
  url          = {https://arxiv.org/abs/1806.11532},
  note         = {Engine that generates and simulates text games; used as core in Peng et al. 2024. Author list abbreviated.},
}
```

## Research Gaps

**Claim**: Maturity of each hybrid repo (last commit date, activity, user base).
**What's missing**: The fetcher did not return last-commit dates. Star counts are 0 to 46; commit counts 64 to 3,032.
**Suggested action**: Check commit history with `gh api` or the GitHub MCP for the six main neighbors before quoting "young" or "active."

**Claim**: Whether any single project combines deterministic engine, tiered model fallback (API, local, rules-only), terminal TUI, MCP bridge, and human plus LLM party.
**What's missing**: README skims only. No code read. kliuchnikovv/dnd and open-dungeon each cover part of this.
**Suggested action**: Read the engine code and multiplayer support of kliuchnikovv/dnd, Mnehmos/rpg-mcp, and open-dungeon; search GitHub topics `llm-dungeon-master`, `ai-rpg`, `mcp-game` for multiplayer claims.

**Claim**: KoboldAI adventure mode, Oobabooga, NovelAI, Hidden Door mechanics.
**What's missing**: No primary documentation fetched for these.
**Suggested action**: Read the KoboldAI Lite and NovelAI docs; keep commercial entries to one line until then.

**Claim**: Foundry VTT AI modules actually let the LLM decide, or the Foundry engine decide, rulings.
**What's missing**: Only search summaries.
**Suggested action**: Read foundryvtt-ai-gm README and the Foundry MCP tool list.

**Claim**: Licenses of Mnehmos/rpg-mcp (MIT vs ISC conflict) and the-dungeon-master (not stated).
**What's missing**: LICENSE file contents.
**Suggested action**: Read the LICENSE files before listing either as open source in the PR/FAQ.

**Claim**: punt-labs/dungeon status.
**What's missing**: URL returned 404.
**Suggested action**: Ask the operator whether it was renamed or archived; it is the org's own prior art and should be described accurately.
