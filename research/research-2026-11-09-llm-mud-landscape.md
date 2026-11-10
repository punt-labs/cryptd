# Research: LLMs in MUDs and text games, and the deterministic-engine-plus-LLM hybrid
**Date:** 2026-11-09
**Request:** Document who else is combining LLMs with MUDs and text games (5 threads: MUDs with LLM features, AI Dungeon lineage, LLM NPCs, the hybrid pattern, adoption numbers) for the crypt PR/FAQ competitive-landscape and differentiation FAQs.
**Claims investigated:** 5 threads

Method note: web search summaries plus WebFetch of the primary pages (GitHub READMEs, arXiv abstracts, vendor/blog pages). Star counts and dates are as seen on 2026-11-09. Repo READMEs are project self-descriptions, not evaluations. Direct r/MUD and Mud Coders threads were not retrievable.

## Evidence Found

### Thread 1: Existing MUDs and MUD codebases with LLM features
**Verdict**: SUPPORTED (several named projects), but all are small hobbyist or demo projects. No established, long-running MUD with a large player base was found that ships LLM features.

Sources:
- LlamaTale (neph1, GitHub, project; 202 stars, 1,684 commits): fork of the archived Tale IF/MUD library. LLM describes locations and NPCs, generates NPC dialogue with memory and sentiment, idle actions, automated quests, an "Anything" story mode that generates a world from a few questions, and "Game Master" commands. README says "Combat results are rolled by the MUD, but the LLM tries to describe them in a more fluid way." This is a genuine hybrid for combat. Strongest MUD-specific match. Single-player-oriented lineage; multiplayer status not confirmed.
- TaleWeave AI (ssube, GitHub, MIT, 14 stars, 150 commits): open-source engine for text adventures and MUDs with AI NPCs, Discord and web front ends, multiplayer (humans and NPCs). Engine tracks entity state (mood, hunger, weather) and summarizes it into prompts; LLM drives character planning and actions. Closest to crypt's "LLM-played characters in a multiplayer game" idea.
- SpacetimeMUD (Clockwork Labs, GitHub, 14 stars, 34 commits): MUD on SpacetimeDB (Rust server). A Python AI agent connects as a player and uses the OpenAI API to build rooms and run conversational NPCs from player prompts. LLM acts as a world builder through the same interface as a player.
- SINGLE-PLAYER-AI-MUD (undergroundrap, GitHub, AGPL-3.0, 2 stars, 169 commits): FastAPI plus Next.js plus local LM Studio. README states all authoritative logic (combat, loot, XP) is deterministic Python; LLM does names, descriptions, NPC dialogue, quest flavor, and zone generation in a deterministic-skeleton-then-LLM two-phase split. Runs without the LLM via fallbacks. Self-described hybrid. Single player.
- Evennia LLM contrib (Griatch, Evennia project docs, 2023): official contrib adding an async LLMClient and a `talk` command for one AI NPC. Docs flag that external API calls are "currently untested." Shows the main Python MUD framework treats LLM NPCs as an optional add-on.
- Coder's Haven hackathon post, "GPT4-x-Alpaca Part 2: Creating a MUD with Evennia" (community post): LLM generates room descriptions on first entry; author reports 10-15 seconds latency per new room.
- Nexlen/mud-mcp (GitHub): MCP stateful MUD server with AI-generated NPC dialogue (seen only in search summary; not fetched).
- AGIMUD, Berga, arXiv 2609.26927 (Sep 2026, preprint, single author): socio-affective multi-agent architecture; MUD where "both agents and humans can interact simultaneously in real time." Closest academic item to LLM-played characters in a MUD; LLM role not verified beyond the abstract.
- NeoMud (terrymaster, Show HN): Kotlin MUD built with Claude Code. LLMs are used for development tooling (designer, playtest, bugfix agents that file about 40 GitHub issues), not in gameplay. Relevant as an agent-playtests-a-MUD example, not an LLM-in-game example.
- HN comment on Evennia thread (community post): skeptical take that players care more about UX than "a million different room descriptions"; you can write rooms yourself.

**Contradictory evidence**: None found to the claim that this is happening; the evidence is that it is happening at hobby scale. Not found: any named long-running commercial or major hobby MUD (e.g. one listed on MudVerse or Mud Connector) announcing LLM features. This is absence in search, not proof of absence.
**Recommendation**: Do not claim "first LLM-enhanced MUD." Say LLM-enhanced MUD frameworks exist (LlamaTale, TaleWeave AI, SpacetimeMUD, an Evennia contrib) and are small, mostly single-maintainer projects.

### Thread 2: LLM-as-dungeon-master and AI text adventures (AI Dungeon lineage), failure modes
**Verdict**: SUPPORTED for the lineage and failure modes. Failure-mode evidence from peer-reviewed sources is stronger than from AI Dungeon community material.

Sources:
- AI Dungeon (Latitude; started on GPT-2, later GPT-3 and others; Wikipedia, company site; news/industry): pure-LLM story generator. Offers Story Cards, Plot Essentials and a Memory Bank as mitigations.
- AI Dungeon help page "Why does the AI forget or mix things up?" (Latitude, primary): documents that older story text is trimmed when context limits are exceeded.
- RPGBench, Yu et al. (Amazon, arXiv 2502.00595, Feb 2025; submitted to ICML 2025): benchmark of LLMs as RPG engines. Finds top models create "engaging stories but often struggle to implement consistent, verifiable game mechanics," especially in long or complex sessions. Best citable evidence for the failure mode crypt avoids.
- Infobip ShiftMag, "Dungeons & Dragons (D&D) Meets AI: A Smarter Game Engine" (Sep 29, 2025, industry blog): reports their pure-LLM approach dropped below 65% accuracy as context grew; switched to deterministic dice and turn order in code. Vendor self-report with no method detail.
- Bicking, "Intra: design notes on an LLM-driven text adventure" (blog, Jul 1, 2025): practitioner list of failure modes: ungrounded narrative, fixation, always responding, lost memory, object hallucination and inflation, prompt hacking, and latency and concurrency problems in multiplayer. He keeps coded ground-truth state but lets the LLM resolve outcomes, and calls it not yet cleanly separated.
- Dungeons and Dragons as a Dialog Challenge for AI, Callison-Burch et al. (arXiv 2210.07109): fine-tuned model reaches 58% joint accuracy on game state tracking.
- SCORE, arXiv 2503.23512: LLM narrative coherence failures over long stories and a RAG-based mitigation.
- Blog posts from competing products (arcanumrpgs, lore-keeper, dungeonsdeep.ai): claims that AI Dungeon does not track HP, dice or character sheets. These are commercial competitors; treat as tertiary and biased.

**Contradictory evidence**: AI Dungeon's growth (see Thread 5) shows many players accept or like the pure-LLM experience; incoherence is not a market-killing flaw for that audience. Bicking's blog also argues pure collaborative storytelling is a different product category from a game with real state. KoboldAI and community AI-adventure tooling were not researched in detail (see gaps).
**Recommendation**: Cite RPGBench as the peer-reviewed support for "LLMs alone are inconsistent at mechanics." Cite AI Dungeon's own help page for context trimming. Describe AI Dungeon as a different category (collaborative storytelling), not as a failed competitor.

### Thread 3: LLM-driven NPCs and agents in games generally
**Verdict**: SUPPORTED. Peer-reviewed foundation exists; commercial adoption is real but limited.

Sources:
- Generative Agents: Interactive Simulacra of Human Behavior, Park, O'Brien, Cai, Morris, Liang, Bernstein (UIST '23, doi 10.1145/3586183.3606763; arXiv 2304.03442; peer-reviewed): 25 LLM agents in a Sims-like town (Smallville) with memory, reflection and planning; the Valentine's Day party emerges from one seed. Foundational citation. It is a simulation sandbox, not a rules-driven game.
- Large Language Models and Games: A Survey and Roadmap, Gallotta et al. (IEEE Transactions on Games; arXiv 2402.18659; peer-reviewed survey): categorizes LLM roles in games including player, NPC, game master, game mechanic, and designer. Good source for the role taxonomy and for saying the space is well mapped.
- Player-Driven Emergence in LLM-Driven Game Narrative, Peng et al. (Microsoft; IEEE Conference on Games 2024; arXiv 2404.17027; peer-reviewed): text-adventure mystery with a TextWorld core plus GPT-4 NPCs and feedback; 28 players; emergent narrative nodes found. A hybrid with a deterministic engine underneath.
- NVIDIA ACE adoption (NVIDIA pages and trade press, industry): ACE reported in inZOI (shipped March 2025), PUBG, NARAKA per GDC 2026 coverage. Ubisoft NEO NPC (Inworld, GDC 2024) had not shipped in a game as of March 2026 per one secondary report. Inworld and Convai are vendor platforms (secondary, vendor claims).
- TextQuests (arXiv 2507.23701) and BALROG (ICLR 2025, arXiv 2411.13543): benchmarks of LLM agents playing Infocom games and TextWorld. They cover LLMs as players of single-player IF. No work found on LLM agents playing networked MUDs over telnet.

**Contradictory evidence**: A critical review ("LLM-based Human Simulations Have Not Yet Been Reliable," arXiv 2501.08579) questions reliability of LLM-simulated humans. Not read in depth.
**Recommendation**: Cite Park et al. for LLM-played characters and Gallotta et al. for the landscape. Do not imply LLM NPCs in multiplayer text games are unexplored; Generative Agents and TaleWeave AI both occupy that ground.

### Thread 4: The hybrid pattern (deterministic engine owns rules, LLM owns narration and creativity)
**Verdict**: SUPPORTED that the pattern exists and is articulated, in the research literature and (much more) in community and open-source projects. Not novel. It is also not yet a named, standard term in peer-reviewed work; no single canonical paper was found. Pure-LLM generators remain the majority.

Peer-reviewed and preprint:
- Creating Hybrid Interactive Narrative Systems with LLM, Clerc, Lourdeaux, Sallak, Ravaine (IEEE ICTAI 2026, arXiv 2611.02807, submitted 2 Nov 2026): "hybrid Experience Managers" combining run-time LLM generation with symbolic narrative control over a shared symbolic world model with explicit preconditions and effects. Three managers compared; "no manager dominates on every property." Closest peer-level articulation of the pattern. Frames it as narrative control, not as a combat and rules engine.
- Peng et al. 2404.17027 (above): TextWorld engine as fixed logic plus GPT-4 for dynamic output.
- FIREBALL, Zhu et al. (ACL 2023, arXiv 2305.01528): about 25,000 D&D sessions with structured game state from the Avrae bot; shows LLMs given game state generate more grounded narration and can emit executable engine commands. Evidence for state-to-narration grounding.
- CriticaLLM, Berlaimont et al. (arXiv 2610.21525, Oct 2026 preprint): LLM Game Master plus a Judge LLM that validates player input to follow D&D rules. Rule enforcement is by a second LLM, not by code. A contrast to crypt's approach.
- RPGBench (above): includes verifiable mechanics enabling deterministic LLM-free evaluation, and finds LLMs struggle to implement them.

Open-source and community (all self-descriptions; none independently evaluated; star counts not collected):
- datashaman/ai-ttrpg: "deterministic rules, event-sourced state, and LLM-powered interpretation and narration"; "the LLM proposes; the engine disposes"; mechanics usable without an LLM.
- lee-b/traditional-game-master (fork of Open Game Master): "the narrator never owns the numbers"; state changes only through server-validated tools.
- AlexD2112/butterfly: app owns dice, XP and resources; LLM narrates and plays NPCs.
- HosseinGorji05/eecs3311-ai-game-master: "an LLM narrates and a deterministic rules engine owns the game state" (looks like a course project).
- RedStoneManL/llm-rpg-engine, LoreKit (DEV Community), west132/AI-GM-Engine: same split in different forms.
- ACHIVX Forum post "The agent game-master split: a deterministic rules engine the model must not override vs a narrative layer it may improvise": the clearest prose statement of the principle that was found.
- Infobip D&D engine (ShiftMag, Sep 2025): "hybrid by intent," with a thought-action-observation loop and every decision checked in code.
- LlamaTale and SINGLE-PLAYER-AI-MUD (Thread 1): hybrid for combat and economy respectively.
- dkurokawa/llm-npc-engine: LLM only produces speech, progression is deterministic code.
- Commercial: RoleForge, Friends & Fables (fables.gg) and DungeonsDeep.ai claim a rules engine plus LLM narrator (marketing; unverified; the last is also a source of competitor-bias blog posts).

**Contradictory evidence (against "this is rare")**: the volume of independent open-source projects stating the same thesis in nearly the same words suggests the pattern is common among developers, especially for D&D-style TTRPG engines. Evidence in favor of "still unsettled": the ICTAI 2026 paper treats the design space as open ("no manager dominates"), CriticaLLM enforces rules with an LLM, and Bicking's design keeps the LLM in charge of resolution.
**What was not found**: a MUD-specific, multiplayer, persistent-world system that pairs a deterministic engine with an LLM Dungeon Master and LLM-played characters in one product. The pieces exist separately (LlamaTale for MUD plus hybrid combat, TaleWeave AI for multiplayer plus AI characters, ai-ttrpg for engine-owns-rules). Combination is the only differentiation these results support, and it rests on a limited search.
**Recommendation**: Replace any claim of novelty for "engine owns rules, LLM owns narration" with "an established pattern; crypt's contribution is X." Candidates, each needing verification against crypt's design docs: engine-as-DM with LLM modes optional (normal vs passthrough), MCP and Claude Code as the client, and multiplayer plus LLM-played characters in one engine. Do not assert any of these as unique without a wider search.

### Thread 5: Scale and adoption signals
**Verdict**: PARTIALLY SUPPORTED. AI Dungeon numbers are company-reported. No counts for LLM-enabled MUDs exist.

Sources:
- Latitude site (company, primary but self-reported): 8M+ registered players, 68M+ adventures, 6M+ scenarios, 1T+ tokens per month.
- Latitude blog (June 2025, company): daily active users up over 70% in six months; over 11 million minutes of usage per day.
- Lean AI Leaderboard post on X (May 2025, third-party summary): $7M+ ARR, under 20 people, profitable.
- Latitude (2020, via Yahoo/funding news): 1.5 million monthly active users then.
- WeavAI blog (Apr 2026, competitor/tertiary): "over 2 million monthly active users." Treat as unverified.
- Open-source MUD projects: LlamaTale 202 stars, TaleWeave AI 14, SpacetimeMUD 14, SINGLE-PLAYER-AI-MUD 2. These are the only community-activity signals found.

**Contradictory evidence**: An analytics aggregator listing 83.3K monthly visits to the site is likely a web-traffic estimate, not the player base.
**Not findable**: number of MUDs that have added LLM features; r/MUD and Mud Coders activity; unit adoption of the Evennia contrib; paid AI Dungeon subscriber count.
**Recommendation**: Cite AI Dungeon as "company-reported 8M+ registered players" only. Say the hobbyist LLM-MUD segment is small and unmeasured.

## Bibliography Entries

```bibtex
@inproceedings{park2023generativeagents,
  author       = {Park, Joon Sung and O'Brien, Joseph C. and Cai, Carrie J. and Morris, Meredith Ringel and Liang, Percy and Bernstein, Michael S.},
  title        = {Generative Agents: Interactive Simulacra of Human Behavior},
  booktitle    = {Proceedings of the 36th Annual ACM Symposium on User Interface Software and Technology (UIST '23)},
  year         = {2023},
  doi          = {10.1145/3586183.3606763},
  eprint       = {2304.03442},
  archiveprefix = {arXiv},
  url          = {https://arxiv.org/abs/2304.03442},
  note         = {Peer-reviewed. 25 LLM agents with memory, reflection and planning in a Sims-like town.},
}

@article{gallotta2024llmgames,
  author       = {Gallotta, Roberto and Todd, Graham and Zammit, Marvin and Earle, Sam and Liapis, Antonios and Togelius, Julian and Yannakakis, Georgios N.},
  title        = {Large Language Models and Games: A Survey and Roadmap},
  journal      = {IEEE Transactions on Games},
  year         = {2024},
  eprint       = {2402.18659},
  archiveprefix = {arXiv},
  url          = {https://arxiv.org/abs/2402.18659},
  note         = {Peer-reviewed survey. Taxonomy of LLM roles in games including NPC, game master, and game mechanic.},
}

@inproceedings{peng2024playerdriven,
  author       = {Peng, Xiangyu and Quaye, Jessica and Rao, Sudha and Xu, Weijia and Botchway, Portia and Brockett, Chris and Jojic, Nebojsa and DesGarennes, Gabriel and Lobb, Ken and Xu, Michael and Leandro, Jorge and Jin, Claire and Dolan, Bill},
  title        = {Player-Driven Emergence in LLM-Driven Game Narrative},
  booktitle    = {IEEE Conference on Games},
  year         = {2024},
  eprint       = {2404.17027},
  archiveprefix = {arXiv},
  url          = {https://arxiv.org/abs/2404.17027},
  note         = {Peer-reviewed. TextWorld engine core with GPT-4 NPCs; 28 players.},
}

@misc{yu2025rpgbench,
  author       = {Yu, Pengfei and Shen, Dongming and Meng, Silin and Lee, Jaewon and Yin, Weisu and Cui, Andrea Yaoyun and Xu, Zhenlin and Zhu, Yi and Shi, Xingjian and Li, Mu and Smola, Alex},
  title        = {{RPGBench}: Evaluating Large Language Models as Role-Playing Game Engines},
  year         = {2025},
  eprint       = {2502.00595},
  archiveprefix = {arXiv},
  url          = {https://arxiv.org/abs/2502.00595},
  note         = {Preprint (submitted to ICML 2025). Top LLMs write engaging stories but struggle with consistent, verifiable mechanics.},
}

@inproceedings{clerc2026hybridnarrative,
  author       = {Clerc, Jules and Lourdeaux, Domitile and Sallak, Mohamed and Ravaine, Marc},
  title        = {Creating Hybrid Interactive Narrative Systems with {LLM}: From Formal Description to Evaluation},
  booktitle    = {38th IEEE International Conference on Tools with Artificial Intelligence (ICTAI 2026)},
  year         = {2026},
  eprint       = {2611.02807},
  archiveprefix = {arXiv},
  url          = {https://arxiv.org/abs/2611.02807},
  note         = {Accepted venue per arXiv listing. Hybrid experience managers: LLM generation with symbolic world model.},
}

@inproceedings{zhu2023fireball,
  author       = {Zhu, Andrew and Aggarwal, Karmanya and Feng, Alexander and Martin, Lara J. and Callison-Burch, Chris},
  title        = {{FIREBALL}: A Dataset of Dungeons and Dragons Actual-Play with Structured Game State Information},
  booktitle    = {Proceedings of ACL 2023},
  year         = {2023},
  pages        = {4171--4193},
  eprint       = {2305.01528},
  archiveprefix = {arXiv},
  url          = {https://arxiv.org/abs/2305.01528},
  note         = {Peer-reviewed. About 25,000 D&D sessions with Avrae game state; state improves narration.},
}

@misc{berlaimont2026criticallm,
  author       = {Berlaimont, Ga{\"e}tan and Vanwichelen, Justin and Ronval, Beno{\^i}t and Verhaeghe, H{\'e}l{\`e}ne},
  title        = {{CriticaLLM}: Using {LLMs} as Autonomous Game Master for Tabletop Role-Playing Games},
  year         = {2026},
  eprint       = {2610.21525},
  archiveprefix = {arXiv},
  url          = {https://arxiv.org/abs/2610.21525},
  note         = {Preprint. LLM Game Master plus Judge LLM that validates input; rules enforced by an LLM, not code.},
}

@misc{berga2026agimud,
  author       = {Berga, David},
  title        = {Building Socio-Affective Artificial Intelligence for Interactive Multi-Agent Simulations},
  year         = {2026},
  eprint       = {2609.26927},
  archiveprefix = {arXiv},
  url          = {https://arxiv.org/abs/2609.26927},
  note         = {Preprint. AGIMUD: MUD where agents and humans interact in real time. LLM role not verified.},
}

@misc{zhu2022dnddialog,
  author       = {Callison-Burch, Chris and others},
  title        = {Dungeons and Dragons as a Dialog Challenge for Artificial Intelligence},
  year         = {2022},
  eprint       = {2210.07109},
  archiveprefix = {arXiv},
  url          = {https://arxiv.org/abs/2210.07109},
  note         = {Preprint. 58 percent joint accuracy on game state tracking. Author list incomplete; check before final citation.},
}

@misc{neph12024llamatale,
  author       = {{neph1}},
  title        = {{LlamaTale}: Giving the power of {LLMs} to a {MUD} lib},
  year         = {2024},
  url          = {https://github.com/neph1/LlamaTale},
  urldate      = {2026-11-09},
  note         = {Open-source repo, 202 stars. Combat rolled by the MUD, described by the LLM. Year is an estimate; creation date not verified.},
}

@misc{ssube2024taleweave,
  author       = {{ssube}},
  title        = {{TaleWeave AI}},
  year         = {2024},
  url          = {https://github.com/ssube/taleweave-ai},
  urldate      = {2026-11-09},
  note         = {Open-source repo, MIT, 14 stars. Multiplayer text adventure and MUD engine with AI NPCs. Year is an estimate.},
}

@misc{clockworklabs2024spacetimemud,
  author       = {{Clockwork Labs}},
  title        = {{SpacetimeMUD}},
  year         = {2024},
  url          = {https://github.com/clockworklabs/spacetime-mud},
  urldate      = {2026-11-09},
  note         = {Open-source repo, 14 stars. AI agent builds rooms and NPCs via OpenAI API. Year is an estimate.},
}

@misc{undergroundrap2026aimud,
  author       = {{undergroundrap}},
  title        = {{SINGLE-PLAYER-AI-MUD}},
  year         = {2026},
  url          = {https://github.com/undergroundrap/SINGLE-PLAYER-AI-MUD},
  urldate      = {2026-11-09},
  note         = {Open-source repo, AGPL-3.0, 2 stars. Deterministic Python rules, local LLM for flavor. Year is an estimate.},
}

@misc{griatch2023evenniallm,
  author       = {Griatch},
  title        = {Evennia {LLM} contrib},
  year         = {2023},
  url          = {https://www.evennia.com/docs/latest/Contribs/Contrib-Llm.html},
  urldate      = {2026-11-09},
  note         = {Official Evennia docs. Async LLM client and one talkative NPC; external API usage flagged untested.},
}

@misc{datashaman2026aittrpg,
  author       = {{datashaman}},
  title        = {ai-ttrpg: An {AI}-assisted tabletop {RPG} engine with deterministic rules, event-sourced state, and {LLM}-powered interpretation and narration},
  year         = {2026},
  url          = {https://github.com/datashaman/ai-ttrpg},
  urldate      = {2026-11-09},
  note         = {Open-source repo. "The LLM proposes; the engine disposes." Year is an estimate; not fetched directly.},
}

@misc{lee2026opengamemaster,
  author       = {{lee-b}},
  title        = {traditional-game-master (fork of Open Game Master)},
  year         = {2026},
  url          = {https://github.com/lee-b/traditional-game-master},
  urldate      = {2026-11-09},
  note         = {Open-source repo. Backend computes dice, HP and conditions; model changes state only via validated tools. Not fetched directly.},
}

@online{infobip2025dnd,
  author       = {Pelivanovic, Ivan},
  title        = {Dungeons \& Dragons ({D\&D}) Meets {AI}: A Smarter Game Engine},
  year         = {2025},
  url          = {https://shiftmag.dev/dungeons-dragons-dnd-ai-game-engine-6240/},
  urldate      = {2026-11-09},
  note         = {Industry blog, 29 Sep 2025. Infobip hybrid engine; pure-LLM accuracy fell below 65 percent with context length. Vendor self-report.},
}

@online{bicking2025intra,
  author       = {Bicking, Ian},
  title        = {Intra: design notes on an {LLM}-driven text adventure},
  year         = {2025},
  url          = {https://ianbicking.org/blog/2025/07/intra-llm-text-adventure},
  urldate      = {2026-11-09},
  note         = {Practitioner blog, 1 Jul 2025. Failure modes of LLM-run text games and partial state separation.},
}

@online{latitude2025growth,
  author       = {{Latitude}},
  title        = {AI Dungeon and Latitude company site and blog},
  year         = {2025},
  url          = {https://latitude.io/},
  urldate      = {2026-11-09},
  note         = {Company-reported: 8M+ registered players, 68M+ adventures; DAU up over 70 percent in six months (June 2025). Not audited.},
}

@online{latitude2025help,
  author       = {{Latitude}},
  title        = {Why does the {AI} forget or mix things up?},
  year         = {2025},
  url          = {https://help.aidungeon.com/faq/why-does-the-ai-forget-or-mix-things-up},
  urldate      = {2026-11-09},
  note         = {AI Dungeon help page. Context limits trim older story text.},
}

@inproceedings{shridhar2025textquests,
  author       = {{TextQuests authors}},
  title        = {{TextQuests}: How Good are {LLMs} at Text-Based Video Games?},
  year         = {2025},
  eprint       = {2507.23701},
  archiveprefix = {arXiv},
  url          = {https://arxiv.org/abs/2507.23701},
  note         = {Preprint. 25 Infocom games. Author list not collected; fill in before use.},
}

@inproceedings{paglieri2025balrog,
  author       = {{BALROG authors}},
  title        = {{BALROG}: Benchmarking Agentic {LLM} and {VLM} Reasoning On Games},
  booktitle    = {ICLR 2025},
  year         = {2025},
  eprint       = {2411.13543},
  archiveprefix = {arXiv},
  url          = {https://arxiv.org/abs/2411.13543},
  note         = {Peer-reviewed. Includes TextWorld. Author list not collected; fill in before use.},
}
```

## Research Gaps

**Claim**: Established or long-running MUDs (MudVerse and Mud Connector listings, Mud Coders Guild) have adopted LLM features.
**What's missing**: No direct r/MUD, Mud Coders Slack, or MudVerse evidence retrieved; search engines returned GitHub and HN only.
**Suggested action**: Search r/MUD and the Mud Coders Guild by hand; ask maintainers of two or three active MUDs.

**Claim**: AI Dungeon is a pure LLM generator with no rules engine, and how its successors compare.
**What's missing**: No primary technical source from Latitude on architecture; KoboldAI, NovelAI, Character.AI-style tools and "Friends & Fables" were not examined.
**Suggested action**: Read Latitude engineering posts; check KoboldAI and NovelAI docs before naming them in the FAQ.

**Claim**: crypt's engine-owns-rules, LLM-owns-narration split is unusual.
**What's missing**: The pattern is common in open source (about 8 repos found). Star counts, dates, quality and whether any combines multiplayer, persistent world, LLM-played characters and MCP were not verified. No peer-reviewed paper names the pattern in a MUD context.
**Suggested action**: Do a second pass on GitHub topics (`ai-rpg`, `mud`, `llm-mud`) and read the ai-ttrpg and TaleWeave code before writing a differentiation claim.

**Claim**: LLM agents can play MUDs as characters.
**What's missing**: Only AGIMUD (LLM role unverified) and TaleWeave AI found. Benchmarks (TextQuests, BALROG) cover single-player IF, not networked MUDs.
**Suggested action**: Read the AGIMUD paper body; search for "LLM agent MUD telnet".

**Claim**: Scale of LLM-in-MUD adoption.
**What's missing**: No counts exist in what was found. AI Dungeon figures are self-reported; the 2M MAU figure comes from a competitor blog.
**Suggested action**: Accept as unmeasurable; state the data as company-reported.

**Data-quality caveats**: GitHub creation years in the bibliography are estimates. Several arXiv IDs (2609.*, 2610.*, 2611.*) are recent preprints read from their abstract pages only. The TextQuests, BALROG and D&D dialog entries have incomplete author lists. The 2611.02807 ICTAI listing is from the arXiv page, not the conference program.
