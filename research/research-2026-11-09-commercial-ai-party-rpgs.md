# Research: Commercial products combining rules engine, AI dungeon master, multiplayer parties, and AI companions
**Date:** 2026-11-09
**Request:** Examine TableForge, Friends & Fables, Hidden Door, Auferet, Questie, and AI Dungeon multiplayer for the competitive-landscape FAQ of prfaq.tex. Record rules adjudication, party size, companions, memory, permadeath, open source/offline/MCP, pricing, launch date, review data. Then state which of five crypt properties each product lacks.
**Claims investigated:** 6 products, 5 property checks

Method note. Each fact is tagged **fetched** (I read the vendor page through WebFetch; the fetch tool returns a model-written summary of the page, so quotes are as relayed) or **search** (web search summary only). Every vendor page is marketing copy. "Not mentioned" always means "not mentioned on the pages I fetched". It is a gap in my search, not proof the product lacks the feature. Vendor FAQs, pricing pages, and the TableForge blog comparison were fetched; other pages on each site (changelogs, docs, Discord, blogs) were not.

## Evidence Found

### 1. TableForge (tableforge.gg)
**Claim**: A commercial product pairs a code rules engine, an AI DM, multiplayer, and AI companions.
**Verdict**: SUPPORTED. TableForge is the closest commercial match to the engine-plus-narrator pattern.
**Sources**:
- Rules (fetched, blog + home): "The game engine runs the game, and the AI DM narrates on top of it." Comparison table: "Separate rules engine (not LLM)", combat "Programmatic". Blog: TableForge "gives final authority over dice, damage, conditions, and resources to code" and uses AI "for interpretation and narration." FAQ (fetched): rules engine governs "Dice rolls, HP tracking, spell slot management, condition application, and combat resolution." Rules basis is SRD 2024 (CC BY 4.0).
- Party size (fetched): up to 6 humans, set by host plan (Free 2, Hearthfire 2, Fellowship 4, Legendary 6). Real-time and async. Guests play free; only the host subscribes.
- Companions (fetched): "travelling NPC companions that fight beside you": 1 (Free), 2 (Hearthfire), 4 (Fellowship, Legendary). Blog: "AI companions can fill out a small party in a fight." Whether companions obey the same rules as humans: FAQ is silent (fetched; gap).
- Memory (fetched): "Persistent cross-session memory", "Complete campaign memory is persistent and comprehensive"; "NPCs, locations, player decisions, quest state, and narrative threads all carry forward." Blog lists stored state: objective, milestones, NPC relationships, world state, scene summaries. No statement that companion memory is checked against engine state; the DM memory is described as stored game state, which is a partial overlap with property (a).
- Character persistence (fetched): items, gold, level carry between campaigns; HP and spell slots refresh on arrival.
- Permadeath: not mentioned (fetched FAQ, home, blog). Search found nothing.
- Open source / offline / MCP: none mentioned (fetched FAQ, home). FAQ says "Any modern browser... No downloads or plugins required." Search summary: hosted commercial product.
- Pricing (fetched): Free $0 (60 turns, 1 campaign); Hearthfire $14.99/mo; Fellowship $29.99/mo; Legendary $39.99/mo. Voice narration minutes on paid plans; custom portraits, battle maps, choice of DM.
- Launch date: not stated (fetched). Blog post published Aug 14, 2026, competitor facts "as of October 17, 2026".
- Reviews: none on vendor pages (fetched). Search found only TableForge's own comparison posts. No third-party review counts found.
**Contradictory evidence**: The blog compares itself favorably to rivals; it is self-authored. The statement "Friends & Fables... frames Franz as referee" is TableForge's characterization of a competitor.
**Recommendation**: Cite TableForge as the main commercial engine-plus-narrator product. Do not claim crypt is the only game with a code-authoritative rules layer. Distinguish on the other properties.

### 2. Friends & Fables (fables.gg)
**Claim**: A commercial AI-DM product with multiplayer parties.
**Verdict**: SUPPORTED for AI DM and multiplayer. PARTIALLY SUPPORTED for a code rules engine; the vendor wording says the AI adjudicates.
**Sources**:
- Rules (fetched, home): the AI GM Franz "narrates, adjudicates rules, and reacts to your world in real time." Combat is "Tactical turn-based combat based on D&D 5e rules. Battlemaps, spells, abilities, monsters, and more." SRD 5.1 under CC BY 4.0. Dice, HP, inventory mechanics are not described beyond a feature heading "Manage Inventory". TableForge's blog (fetched; competitor claim) says Franz "narrates and adjudicates rules." Third-party (search, Arcanum RPGs June 2026): rates it 3.9/5, tied top on "Mechanical Depth" and "Determinism & Fairness" axes, and says one action is sent to up to a dozen specialized models in parallel. That suggests model-side adjudication with many models, not a separate code engine; unverified because the Arcanum page was not fetched.
- Party size (fetched): "a party of up to 6 players". Pricing page (fetched): 3 / 4 / 5 / 6 player campaigns by plan. Friends play without their own subscription.
- Companions (fetched home and pricing): not mentioned. The only AI entity named is Franz. Gap in my search (other pages not read).
- Memory (fetched): Franz has "advanced memory and customization"; tracked state "Quests, inventory, health, spells, maps, lore" (TableForge blog's summary).
- Permadeath, open source, offline, MCP: none mentioned (fetched).
- Pricing (fetched /pricing): Free $0 (25 turns/day); Starter $19.95/mo; Pro $29.95/mo; Legend $39.95/mo; yearly "2 Months Free". Premium model costs credits per turn.
- Launch (search): Product Hunt datePublished 2024-10-24, one directory says Jul 12 2024; the vendor site gives no date. Beta status per DreamGen review (search, May 2026): "Still in beta."
- Reviews (search): Product Hunt 4.7 from 15 reviews; vendor claim "100,000+ players and world builders" (fetched). No Steam page found (browser product). Reddit threads not retrieved.
- TableForge blog (fetched, competitor claim): a September 2026 patch note reportedly pauses new development until 2027. Not verified at source.
**Recommendation**: Say Friends & Fables has an AI-adjudicated 5e game with up to six human players and no AI companions found on its site.

### 3. Hidden Door (hiddendoor.co)
**Claim**: A commercial AI story game with a rules layer.
**Verdict**: PARTIALLY SUPPORTED. It has a world-state engine, not dice, HP, or inventory.
**Sources**:
- Rules (fetched, home): a "Story engine" that "turns every intent into story beats and world-state changes, checked against canon before anyone sees them." Shown example: intent, planned beat, "Canon check", updated world state. No dice, HP, or inventory on the page. Search (Bicking review, Aug 2025): "light mechanics system" and the critique that it reads as collaborative storytelling more than a game.
- Party size (fetched): no number given; a mockup shows "24 players online"; social games are described as multiplayer with friends. Multiple humans: implied, not specified.
- Companions (fetched): not explicit. "Character chat" with world characters.
- Memory (fetched): "He remembers what you told him last night"; engine claims to "keep stories coherent across multi-hour sessions." Mechanism not described.
- Permadeath (fetched): the world-creator Death setting offers "None / Normal / Permadeath". This is the only product in the set with a stated permadeath setting. It is a creator toggle on characters in a story; the page does not say companions are covered or that reload is blocked.
- MCP (fetched): a "Creator MCP" lets you "build and extend it from Claude and other AI assistants." This is for authoring worlds, not for a player's model running the game.
- Open source, offline: not mentioned (fetched).
- Pricing (fetched /pricing): Free tier "Jump into any world, without limits"; Creator tier for building and publishing worlds; dollar amounts did not render. Search (Aug 2025 coverage): credit system modeled on Audible.
- Launch (search): public web launch August 2025; company founded 2020; $9 million raised (search, Forbes/GamesBeat coverage).
- Reviews (search): no Steam or Reddit counts found. Independent review by Ian Bicking (Aug 2025) is critical ("the game is ungrounded"). Arcanum RPGs 2026 review is favorable.
**Recommendation**: Cite Hidden Door as the product with a permadeath option and an MCP creator interface. Do not claim crypt is the first with MCP or permadeath in the category.

### 4. Auferet (auferet.com/multiplayer-ai-rpg)
**Verdict**: PARTIALLY SUPPORTED. The AI decides outcomes; a dice roller and stat tracker exist.
**Sources**:
- Rules (fetched): "Stat tracking and a dice roller are built in, including 5e and Pathfinder 2e modes." "The AI runs the world, the NPCs, and the outcomes." The page warns "AI narration can still apply a rule incorrectly." That is the vendor stating the model can break rules.
- Party size (fetched): no maximum stated; demo used two accounts; FAQ confirms two-player. Four group modes: host only, turn based, free for all, shared world. One shared record of each player's location and last action is read by every player's GM before replying.
- Companions (fetched): "the Game Master can portray companions and other characters"; no count, no separate mechanics.
- Memory (fetched): "The full memory system is on every plan, including free." Warns shared details may be missed or contradicted.
- Permadeath, open source, MCP: not mentioned (fetched). Offline: not offered; browser plus Android app.
- Pricing (fetched): free with 10 daily claimable credits; paid from $10/month (500 monthly credits plus 20 daily).
- Launch (search, IndieDB listing): released Dec 23, 2025. Page copyright 2025-2026.
- Reviews: none on page (fetched); no independent reviews found (search).

### 5. Questie (questie.ai/multiplayer-ai-party-mode)
**Verdict**: PARTIALLY SUPPORTED. Party Mode is one human plus AI companions; there is no rules engine.
**Sources**:
- Rules (fetched): no rules engine. Only mechanic: "the built-in dice roll feature, so you can roll a d20 and watch your companions react to the outcome." Companions react conversationally.
- Party size (fetched): one human plus up to four AI companions per voice room; multiple humans cannot join one room (audio can be routed to Discord or OBS).
- Companions (fetched): up to four, each with own synthesized voice, personality, backstory; they converse with each other with turn-taking. This is the only product in the set with per-companion voice (property e, voice part).
- Memory (fetched): "Companions remember your squad's shared history, party jokes, and campaign lore across sessions." Mechanism not stated; no engine state exists to verify against.
- Permadeath, open source, offline, MCP: not mentioned (fetched). Browser, desktop, and mobile apps.
- Pricing (fetched): Party Mode requires the Legend plan (7,500 monthly credits). Search: Legend is $149.99/month (search only; the fetched page showed no dollar figure).
- Launch: not stated (fetched).
- Reviews: vendor claims 50,000+ gamers (fetched); search snippet quotes 15,000+ in live sessions (search; conflicting figures). Independent reviews (RoboRhythms, Stork.ai) positive for solo gamers; a 404 Media report found the AI gave confident wrong advice about an FFXIV raid (search).
**Recommendation**: Questie is an AI-companion voice product for gamers. It competes with crypt's companions only if the FAQ frames companions as chat partners.

### 6. AI Dungeon multiplayer (Latitude)
**Verdict**: SUPPORTED. No rules engine; multiplayer with no hard cap.
**Sources**:
- Rules (search; Arcanum RPGs article fetched): "No character sheets, dice or rules." Community scripts such as Hashtag DnD must be added to a scenario. TableForge blog (fetched; competitor): "Generative story, no rules engine."
- Party size (fetched, Latitude help page): "Currently, there is no set limit for how many people can join a Multiplayer game." The "(1/4)" is "simply a suggestion". Open eight-digit join code. Turns are not strictly sequenced.
- Companions (fetched help page): no mention.
- Memory (fetched): none in multiplayer docs. Arcanum (fetched): every player shares one context window; roughly 2k-4k tokens on free models, 8k Journey, 16k Legend, 32k Mythic/Ultimate; Story Cards check against a minimum of four recent actions.
- Permadeath, open source, offline, MCP: not mentioned (fetched).
- Pricing (search, App Store and Arcanum): Journey $14.99, Legend $29.99, Mythic $49.99, Ultimate $99.99 per month; host's tier applies to all players. The Arcanum article (fetched) lists the same tiers.
- Launch (search, Wikipedia): Windows/browser Dec 5, 2019; Android/iOS Dec 17, 2019. Multiplayer launch date not found.
- Reviews (search): iOS 4.4 from about 26,600 ratings (App Store listing); Google Play over 105,000 reviews, 4.0-4.2 average. Counts are from search summaries, not read at the store page.

### Gap analysis: crypt properties each product does NOT appear to have
Legend: **NF** = not found on the fetched vendor pages (gap in my search); **Contra** = a vendor statement points the other way; **Has** = vendor states it.

| Property | TableForge | Friends & Fables | Hidden Door | Auferet | Questie | AI Dungeon |
|---|---|---|---|---|---|---|
| (a) Companion memory across sessions verified against engine-recorded state | NF for companions. DM memory is stored game state (partial overlap) | NF (no companions found) | NF; memory claimed, mechanism unstated | Contra: "AI narration can still apply a rule incorrectly"; shared record is model-read; no verification claimed | NF; memory claimed, no engine state exists | Contra: no engine state; shared context window forgets |
| (b) Permanent companion death, no reload | NF | NF | Partial: "Permadeath" creator setting for characters; companions and reload not addressed | NF | NF | NF |
| (c) Open source plus fully offline rules-only tier | NF; hosted, browser-only | NF; browser-only | NF | Contra for offline: browser plus Android, online | NF; online | NF (closed; no offline) |
| (d) Claude Code / MCP bridge where the player's own model runs the game | NF | NF | Partial: Creator MCP for authoring worlds from Claude; the game is run by Hidden Door's story engine | NF | NF | NF |
| (e) Generated presentation (panels, layered music, per-companion voice) | Partial: generated art, battle maps, DM voice (minutes metered); no layered music or per-companion voice found | Partial: image generation, battlemaps, text-to-speech; no per-companion voice | NF | NF | Has per-companion voice; no panels or layered music found | NF |

Reading of the table. No fetched page supports (a), (b) for companions, (c), or (d) as crypt defines them. Every "NF" cell rests on reading one to three pages per vendor. The strongest absence evidence is (c): all six are hosted browser or app services and none mentions source code or local models. The weakest absence evidence is (e) for TableForge and Friends & Fables, whose pages mention art, maps, and voice that I did not read in detail. Hidden Door's permadeath setting and Creator MCP both partly overlap with (b) and (d) and should be named in the FAQ.

**Recommendation for the FAQ**: Claim only what the evidence supports: "Among six commercial products reviewed (page reads, November 2026), none stated open-source code with an offline rules-only mode, and none stated a bridge that lets the player's own model run the game." Do not write "no competitor has permadeath" or "no competitor has MCP." Do not claim crypt is first to separate rules from narration; TableForge states that design.

## Bibliography Entries

```bibtex
@online{tableforge2026multiplayercompare,
  author  = {{TableForge}},
  title   = {Best Multiplayer AI Dungeon Master for a Group Without a DM},
  year    = {2026},
  url     = {https://tableforge.gg/blog/multiplayer-ai-dungeon-master-comparison},
  urldate = {2026-11-09},
  note    = {Vendor-authored comparison of TableForge, Friends \& Fables, The Endlessness, LoreKeeper, AI Dungeon, Roll20, Foundry. Published Aug 14, 2026, updated Oct 20, 2026. Tertiary and self-interested for competitor claims.},
}

@online{tableforge2026home,
  author  = {{TableForge}},
  title   = {TableForge: AI Dungeon Master for D\&D 5e},
  year    = {2026},
  url     = {https://tableforge.gg/},
  urldate = {2026-11-09},
  note    = {Primary vendor page. Separate rules engine, companions per plan, pricing \$0 to \$39.99 per month, party size up to 6.},
}

@online{tableforge2026faq,
  author  = {{TableForge}},
  title   = {TableForge FAQ},
  year    = {2026},
  url     = {https://tableforge.gg/faq},
  urldate = {2026-11-09},
  note    = {Primary vendor page. Rules engine governs dice, HP, spell slots, conditions, combat. No mention of permadeath, offline, open source, or MCP.},
}

@online{fables2026home,
  author  = {{Friends \& Fables}},
  title   = {Friends \& Fables: AI TTRPG and World Building Game},
  year    = {2026},
  url     = {https://fables.gg},
  urldate = {2026-11-09},
  note    = {Primary vendor page. AI GM Franz narrates and adjudicates rules; up to 6 players; SRD 5.1.},
}

@online{fables2026pricing,
  author  = {{Friends \& Fables}},
  title   = {Friends \& Fables Pricing},
  year    = {2026},
  url     = {https://fables.gg/pricing},
  urldate = {2026-11-09},
  note    = {Primary vendor page. Free, \$19.95, \$29.95, \$39.95 per month; 3 to 6 player campaigns.},
}

@online{hiddendoor2026home,
  author  = {{Hidden Door}},
  title   = {Hidden Door},
  year    = {2026},
  url     = {https://hiddendoor.co},
  urldate = {2026-11-09},
  note    = {Primary vendor page. Story engine with canon check, Death setting None/Normal/Permadeath, Creator MCP.},
}

@online{hiddendoor2026pricing,
  author  = {{Hidden Door}},
  title   = {Hidden Door Pricing},
  year    = {2026},
  url     = {https://hiddendoor.co/pricing},
  urldate = {2026-11-09},
  note    = {Primary vendor page. Free and Creator tiers; dollar amounts did not render.},
}

@online{bicking2025hiddendoor,
  author  = {Bicking, Ian},
  title   = {Hidden Door At Launch: Design Review of an LLM-Driven Story Game},
  year    = {2025},
  url     = {https://ianbicking.org/blog/2025/08/hidden-door-design-review-llm-driven-game},
  urldate = {2026-11-09},
  note    = {Independent practitioner review (seen in search summary only). Critical: ungrounded narrative.},
}

@online{auferet2026multiplayer,
  author  = {{Auferet}},
  title   = {Multiplayer AI RPG},
  year    = {2026},
  url     = {https://auferet.com/multiplayer-ai-rpg},
  urldate = {2026-11-09},
  note    = {Primary vendor page. Dice roller and stat tracking; AI decides outcomes; four group modes; free plus paid from \$10 per month.},
}

@online{questie2026partymode,
  author  = {{Questie.ai}},
  title   = {Multiplayer AI Chatbot and Party Mode},
  year    = {2026},
  url     = {https://www.questie.ai/multiplayer-ai-party-mode},
  urldate = {2026-11-09},
  note    = {Primary vendor page. One human plus up to four voiced AI companions; Legend plan only; d20 roll with companion reactions.},
}

@online{latitude2026multiplayerfaq,
  author  = {{Latitude}},
  title   = {Do you support Multiplayer?},
  year    = {2026},
  url     = {https://help.aidungeon.com/faq/do-you-support-multiplayer},
  urldate = {2026-11-09},
  note    = {Primary vendor help page. No set player limit; host's settings apply to all players.},
}

@online{arcanum2026aidungeonmultiplayer,
  author  = {{Arcanum RPGs}},
  title   = {AI Dungeon Multiplayer (2026): How It Works, What It Costs, and Why the Table Forgets Faster},
  year    = {2026},
  url     = {https://arcanumrpgs.com/blog/ai-dungeon-multiplayer/},
  urldate = {2026-11-09},
  note    = {Third-party competitor-adjacent blog, verified Sep 25, 2026. No dice or rules; shared context window; tier pricing.},
}

@online{apple2026aidungeon,
  author  = {{Latitude Inc.}},
  title   = {AI Dungeon: RPG and Story Maker (App Store listing)},
  year    = {2026},
  url     = {https://apps.apple.com/us/app/ai-dungeon-rpg-story-maker/id1491268416},
  urldate = {2026-11-09},
  note    = {Seen in search summary only: 4.4 rating, about 26,600 ratings; in-app purchase price list.},
}
```

## Research Gaps

**Claim**: Companions in TableForge act under the same rules as human characters.
**What's missing**: TableForge pages say a rules engine handles combat but never address companion mechanics. A playthrough page (tableforge.gg/playthrough) and the companion section of the home page were not read in detail.
**Suggested action**: Fetch tableforge.gg/playthrough; otherwise state only that companions "fight beside you."

**Claim**: Friends & Fables has AI companions, and its adjudication is model-side.
**What's missing**: Only the home and pricing pages were read. The Arcanum review (3.9/5, "up to a dozen specialized models") was seen in a search summary. Patch notes, docs, and Discord were not read.
**Suggested action**: Fetch fables.gg patch notes and the Arcanum review before asserting either point.

**Claim**: Hidden Door permadeath covers companions or blocks reload.
**What's missing**: The page shows only a "None / Normal / Permadeath" selector. Whether companions can die and whether saves can be reloaded is unknown.
**Suggested action**: Read Hidden Door world-creation docs or the early-access blog.

**Claim**: Questie Legend costs $149.99 per month, and 15,000 versus 50,000 users.
**What's missing**: The fetched page had no price and a 50,000-user claim; a search summary had $149.99 and 15,000. Conflicting, unresolved.
**Suggested action**: Fetch questie.ai/pricing before quoting a price or user count.

**Claim**: Player review counts for these products (Steam, app stores, Reddit).
**What's missing**: Only AI Dungeon (App Store about 26,600 ratings; Google Play 105,000+ reviews) and Friends & Fables (Product Hunt 15 reviews) yielded counts, both from search summaries. No Steam pages exist for the browser products found. Reddit threads were not retrievable.
**Suggested action**: Fetch the App Store and Play listings directly; search r/DnD, r/Solo_Roleplaying, r/AIDungeon by hand.

**Claim**: Launch dates for TableForge, Hidden Door's exact public date, Questie, and AI Dungeon multiplayer.
**What's missing**: No vendor page gives a launch date for TableForge or Questie. AI Dungeon's multiplayer launch date was not found.
**Suggested action**: Use the Wayback Machine first-capture date, or write "launched in 2025-2026" for dated claims.

**Claim**: None of the six products offers open source, offline play, or a player-model bridge.
**What's missing**: Five to seven pages per vendor were read at most. Absence is unproven.
**Suggested action**: Phrase the FAQ as "none of the six stated" and cite the pages read, with the date.
