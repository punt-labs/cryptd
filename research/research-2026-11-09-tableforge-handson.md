# Research: TableForge hands-on session (primary, operator playtest)
**Date:** 2026-11-09
**Request:** First-hand evaluation of TableForge (tableforge.gg), the closest commercial neighbor to crypt, by the operator playing a live campaign in Safari on macOS.
**Source quality:** Primary. One session, one player, one browser. Seven screenshots on the operator's Desktop (`Screenshot 2026-11-09 at 18.35.47.png` through `18.36.36.png`). Observations are the operator's and are quoted or paraphrased; nothing here is from the vendor.

## Evidence Found

**Claim**: TableForge's interaction model is slow and sluggish in a browser.
**Verdict**: SUPPORTED (operator's direct report).
**Observation**: "This app is slow, and feels very sluggish in a browser." Later in the same session the browser tab froze outright and the operator had to reload the page to continue. Cause not isolated (front-end rendering, audio, or the long-lived chat DOM are candidates); one occurrence, one browser. The input is a free-text box plus mouse-driven action chips (Attack, Cast Spell, Use Feature, Use Item, Dodge, Dash, Help, Investigate, Roll Dice, Primer, Short Rest, Long Rest) revealed by a Quick Actions button. Every turn is a Send click and a wait.
**Operator's split of the slowness (two layers)**: (1) Expected inference latency: the narrator appears to be a large model, and the session appears to generate TTS audio as well, so each turn pays model time plus synthesis time. That part is understood and is what crypt's predictive cache targets. (2) The browser itself: in the operator's words, the browser "acts like it is about to have a heart attack," independent of waiting on the server. That is a client-side load problem (rendering, audio, DOM growth) and it is the one that made the session feel unplayable; inference latency alone would have been a wait, not a freeze.
**Relevance to crypt**: Tempo is a felt differentiator, and the two layers argue for the terminal client on separate grounds. The predictive cache hides layer 1. Layer 2 does not exist in a terminal: no page, no DOM, no audio pipeline fighting the renderer for the main thread; vox plays audio out of process. Wizardry's keyboard navigation and keyboard-driven combat were, in the operator's words, part of what made it better than these games. crypt's terminal client can hold that tempo; a browser chat surface cannot.

**Claim**: Play requires many clicks per turn.
**Verdict**: SUPPORTED (operator: "I am also having to click a lot").
**Observation**: A turn is a mouse path: open Quick Actions, pick a chip (or type), press Send, then click the play button under each message to hear it, and click tabs (Journal, Party, Companions) to see state the Wizardry screen showed at all times. Nothing observed was keyboard-reachable.
**Relevance**: Wizardry's combat was one keypress per decision with the party status always on screen. Click count per turn is a measurable tempo metric crypt can report against: target one keypress per action, zero clicks to see party state.

**Claim**: Voice is present but not integrated into the flow.
**Verdict**: SUPPORTED.
**Observation**: Characters speak only when the player presses a play button under each message; "Group voice narration autoplay" exists as a setting and defaults OFF. Portraits animate when speaking, in "a janky sort of way." Background music plays.
**Relevance**: Per-companion voice is table stakes in this category (Questie has it too). What is absent is voice that arrives with the turn without a click. Autoplay-off as the default suggests the vendor's own latency or quality concerns.

**Claim**: Spoken narration was written for the eye, not the ear.
**Verdict**: SUPPORTED (operator: "the TTS sentences were very long").
**Observation**: The narrator's text is literary prose (see the Thrain Mapcarver passage in the screenshots: two paragraphs, sentences of 20 to 30 words, semicolons, nested clauses). A spoken turn runs about four such sentences. The operator's verdict: "That's a mistake." The same text is handed to TTS unchanged, so the player waits through synthesis of four long sentences of prose that was never shaped for speech, every turn.
**Relevance**: Two consequences for crypt. (1) Latency: TTS time scales with character count, so long narration doubles the wait on top of model time. (2) Prompting: the narrator needs a spoken register when voice is on, distinct from the reading register: one or two short sentences per turn, no semicolons, one image per sentence, the way a Zork room description is two lines and then the prompt returns. That is a Chapter 5 prompt-engineering exercise with two measurable targets (sentences per spoken turn, target at most two; words per sentence, target under 15), and a reason vox's synthesis should start on the first sentence rather than wait for the full paragraph. Length belongs to the player: a "tell me more" keypress asks for the long version; the default is short.

**Claim**: TableForge ships a rules engine with a visible character sheet and grid map.
**Verdict**: SUPPORTED (visible in screenshots).
**Observation**: Party tab shows a 5e sheet (HP 12/12, AC 13, speed 30ft, initiative +3, six ability scores with modifiers, XP 6 of 300, gold 10). Map panel shows a 20x20 grid with tokens and furniture, labeled "The Wrong Heroes' Send-Off." Quest log with completion state. Streak counter ("9 turns more starts a streak") in the header.
**Relevance**: Confirms the research finding that TableForge's engine owns the numbers. The streak counter is an engagement mechanic, not a game mechanic.

**Claim**: Companions were absent in this session.
**Verdict**: SUPPORTED for the free tier as played.
**Observation**: Companions tab reads "No companions travel with the party yet" with a 0/2 badge; Party tab shows "Room for 1 more" and an invite link. The two speaking characters (Pim Aldergast, Thrain Mapcarver) are DM-voiced NPCs, one of whom narrates in the second person.
**Relevance**: The research file's "1 to 4 companions by plan" was not exercised here. Companion behavior, memory, and death were not observed and remain unverified.

**Claim**: The experience would not sustain hours of play.
**Verdict**: Operator judgment, one session.
**Observation**: "This could not be that much fun for hours from my POV."
**Relevance**: The genre's long-session appeal (Wizardry, MUDs) came from tempo and agency, not presentation. TableForge is visually polished ("visually pretty") and still fails the tempo test for this player.

## What this adds to the competitive FAQ

- The differentiator the web research could not see: **tempo**. Keyboard-first play with instant engine response, the Wizardry model, against a click-send-wait chat surface. This is a felt UX difference and it is verified first-hand.
- Presentation polish (portraits, grid map, music, voice on demand) does not by itself produce a game someone plays for hours. That supports crypt's build order: party and tempo first, presentation gated behind "it never makes you wait."
- The predictive-caching thesis gets a concrete target: TableForge's wait is the thing to beat, measured.

## Research Gaps

- One session, one tier, no companions seated. Companion memory and death on a paid plan were not observed.
- No timing was captured. A follow-up should record seconds from Send to first narration text and to voice start, over ten turns.
- Combat was not reached; the keyboard-vs-click comparison in combat is inferred from the action-chip UI, not observed in a fight.
- "Sluggish" is not separated into network latency, model latency, and front-end rendering; a browser performance trace would split them.
- The freeze was observed once. Whether it reproduces, and whether it is tied to session length, audio, or the map panel, is unknown.
