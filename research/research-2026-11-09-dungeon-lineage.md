# Research: Zork, Wizardry, and non-commercial MUD lineage for the crypt PR/FAQ
**Date:** 2026-11-09
**Request:** Historical detail on Zork (Infocom/MIT), Wizardry (Sir-Tech 1981), and popular non-commercial MUDs (MUD1, AberMUD, TinyMUD/MOO, LPMud, DikuMUD and descendants), with dates, creators, mechanics, and documented figures.
**Claims investigated:** 18

Source quality note: most figures below come from Wikipedia articles (tertiary) and one primary source (Bartle's "Early MUD History"). Wikipedia text was read through a summarizing fetch tool, so exact wording is not quoted. Before printing any figure in the PR/FAQ, open the cited Wikipedia footnote and cite its underlying source (Computer Gaming World, InfoWorld, Bartle, Digital Antiquarian).

## Evidence Found

### Zork

**Claim**: Zork was written at MIT's Dynamic Modeling Group between 1977 and 1979 by Tim Anderson, Marc Blank, Bruce Daniels, and Dave Lebling, in MDL on a PDP-10.
**Verdict**: SUPPORTED
**Sources**:
- Wikipedia "Zork": development began May 1977; authors Anderson, Blank, Daniels, Lebling; MDL on PDP-10; last puzzle added February 1979; bug fixes to January 1981. The 1977 version had 32 treasures.
- Digital Antiquarian "The Birth of Infocom": three of the four (Anderson, Lebling, Blank) became Infocom founders. Infocom was founded 22 June 1979 with ten stockholders (per the Wikipedia Infocom article via search).
**Contradictory evidence**: None. Start date is May 1977 (Wikipedia); some sources say "1977-1979" for the whole effort.
**Recommendation**: Use as-is. Safe phrasing: "begun in 1977 by four members of MIT's Dynamic Modeling Group."

**Claim**: A Fortran port called Dungeon spread the game beyond MIT.
**Verdict**: SUPPORTED
**Sources**:
- Wikipedia "Zork": Fortran port by Bob Supnik released March 1978 as Dungeon; renamed after TSR raised a trademark concern over Dungeons & Dragons.
- Wikipedia "Multi-user dungeon" (quoting Bartle): Roy Trubshaw played a Fortran port called DUNGEN, which is why MUD has "Dungeon" in its name.
**Contradictory evidence**: None.
**Recommendation**: Use as-is. This is the direct line from Zork to MUD1.

**Claim**: The commercial trilogy was Zork I (1980), II (1981), III (1982), split because the game did not fit a home computer.
**Verdict**: SUPPORTED
**Sources**:
- Wikipedia "Zork": Zork I released for TRS-80 December 1980 via Personal Software (later VisiCorp); Infocom self-published from late 1981.
- Digital Antiquarian "ZIL and the Z-Machine": original game used 1 MB on the PDP-10; target was a 32K Apple II or TRS-80 with one floppy. Final story file 77K (18K dynamic). Rooms cut from 191 to 110, items from 211 to 117, parseable words from 911 to 617; about 100 locations kept for Zork I, the rest saved for the sequel.
**Contradictory evidence**: None.
**Recommendation**: Use as-is. The cut figures are a strong illustration of "finite hand-authored content" in the problem statement.

**Claim**: The Z-machine was a virtual machine designed so one game file ran on many incompatible microcomputers.
**Verdict**: SUPPORTED
**Sources**:
- Wikipedia "Z-machine": created by Joel Berez and Marc Blank in 1979; as of July 1984 Infocom published for 15 different computers plus generic CP/M and MS-DOS; Graham Nelson's later standard; Inform compiler released May 1993; Nelson called it "possibly the most portable virtual machine ever created."
- Digital Antiquarian "ZIL and the Z-Machine": interpreter called Zip, written in assembly per platform; static data paged in 1K blocks; text compressed to 5 bits per character plus abbreviations; influenced by UCSD P-machine; Zork compiled by ZIL (derived from MDL) with compiler Zilch.
- Digital Antiquarian "The Birth of Infocom" (comments): Bruce Daniels wrote the first Z-machine interpreter for the Apple II. Treat as a reader comment, not verified.
**Contradictory evidence**: Wikipedia credits Scott Cutler (via search summary) with the first interpreter for TRS-80. Both can be true (different platforms).
**Recommendation**: Use as-is; skip the Daniels interpreter claim unless verified.

**Claim**: The Infocom parser accepted full sentences, not two-word verb-noun commands.
**Verdict**: SUPPORTED
**Sources**:
- Wikipedia "Zork": Zork aimed for more complex input than the two-word commands of Colossal Cave Adventure.
- Zork manual (via IFWiki "Infocom-type parser" and infodoc.plover.net manuals): multiple objects separated by AND or commas (TAKE LAMP, JAR, FLUTE; PUT THE GOLD BAR AND THE PEARL IN THE TROPHY CASE); ALL refers to visible objects not inside containers.
- Vocabulary: 911 words in the MIT version, 617 in the commercial Zork I (Digital Antiquarian). One secondary source says 908 words with 71 actions; unverified and conflicts slightly with 911.
**Contradictory evidence**: Sentence support was bounded: verb, up to two noun phrases, prepositions. The parser was not open-ended natural language.
**Recommendation**: Revise to "understood sentences such as PUT THE GOLD BAR AND THE PEARL IN THE TROPHY CASE from a fixed vocabulary of a few hundred words." Cite the 617 figure, not 908.

**Claim**: The grue is the darkness mechanic.
**Verdict**: SUPPORTED
**Sources**:
- Wikipedia "Grue (monster)": added by Dave Lebling to replace pits that killed players in dark places; name from Jack Vance; the line "It is pitch black. You are likely to be eaten by a grue." Player needs a lamp or other light source.
**Contradictory evidence**: Page did not name Vance's Dying Earth specifically in the part read. Unverified detail.
**Recommendation**: Use the mechanic and the quoted line. Say "name borrowed from Jack Vance."

**Claim**: Zork sales numbers.
**Verdict**: PARTIALLY SUPPORTED
**Sources**:
- Wikipedia "Zork": Zork I 38,000 copies by end of 1982, about 150,000 in 1984, 380,000 by 1986. Zork II 170,000+, Zork III 130,000+ by 1986. Trilogy 680,000+ through 1986, over 760,000 by early 1989. Trilogy was more than one-third of Infocom's roughly 2 million game sales 1982-1986.
**Contradictory evidence**: Figures are Wikipedia's compilation; the underlying trade-press sources were not opened. Release-year and cumulative-year basis varies by line.
**Recommendation**: Add caveat: "about 680,000 copies of the trilogy through 1986." Verify against Wikipedia footnotes before final.

**Claim**: Zork's place as the canonical text adventure.
**Verdict**: SUPPORTED
**Sources**:
- Wikipedia "Zork": described as the most famous interactive fiction; Library of Congress game canon (2007) lists it among ten games; Microsoft released Zork I-III as open source in November 2025.
**Contradictory evidence**: Colossal Cave Adventure (1976) predates it. Zork is canonical for parser quality, not first.
**Recommendation**: Use "the best-known text adventure," not "first." The Library of Congress item should be confirmed from LOC.

### Wizardry

**Claim**: Wizardry was created by Andrew Greenberg and Robert Woodhead, developed from 1978, published by Sir-Tech for the Apple II in September 1981.
**Verdict**: SUPPORTED
**Sources**:
- Wikipedia "Wizardry: Proving Grounds of the Mad Overlord": development began 1978 at Cornell; playable by fall 1979; Applesoft BASIC rewritten in UCSD Pascal; about two and a half man-years; a year of playtesting; final September 1981; Revision 2 in 1982. Drew on PLATO games, notably Oubliette (1977).
- Search summary of the Wikipedia series article: Norman Sirotek formed Sir-Tech in 1980 to distribute it.
**Contradictory evidence**: Wikipedia names Greenberg as starting development; Woodhead joined later. Credit both as authors.
**Recommendation**: Use as-is.

**Claim**: Party of six, five races, four base classes, four elite classes, ten dungeon levels, 20x20 maze, no automap.
**Verdict**: SUPPORTED
**Sources**:
- Wikipedia (same article): up to six characters; five races (human, elf, dwarf, gnome, hobbit); three alignments; four base classes (fighter, priest, mage, thief); four elite classes (bishop, samurai, lord, ninja); ten levels under Trebor's castle; goal is to kill Werdna and take the amulet; seven spell tiers; players drew maps on graph paper included in the box; no saving inside the dungeon.
**Contradictory evidence**: None found. Werdna and Trebor are the authors' first names reversed (Andrew, Robert).
**Recommendation**: Use as-is.

**Claim**: Class change system.
**Verdict**: SUPPORTED
**Sources**:
- Wikipedia series article: elite classes require high attribute minimums and alignment; some cannot be rolled at creation, so players level up in one class and change.
- Steam and GOG community guides (tertiary, player-written): changing class resets to level 1 and zero XP, keeps attributes and known spells, loses class special abilities. Hit points recompute on each level and never drop. Only the Bishop is available at creation. A Thief with a Thieves' Dagger can become a Ninja without meeting the requirements.
**Contradictory evidence**: Player guides, not the manual. Details such as the "stats reset to race minimums" vary across versions and the 2023 remake (a Steam thread title suggests the remake resets stats to race minimums).
**Recommendation**: Cite the general mechanic only ("change class at the cost of dropping to level 1"). Verify details against the original Apple II manual before stating specifics.

**Claim**: Permadeath, with death handled in a way that punished the player.
**Verdict**: PARTIALLY SUPPORTED
**Sources**:
- Wizardry Wiki: character death is permanent and written to disk immediately; a wiped party is lost unless a new party enters the maze and recovers the bodies; no save mid-dungeon; returning to the castle restores resources but restarts at level 1; players bypassed permadeath by duplicating or ejecting save disks.
- Wikipedia: after a total party kill a new party must recover bodies and items.
**Contradictory evidence**: Death was not absolute. The temple could resurrect, with a chance of failure. The wiki fetched did not cover ash versus lost states or odds.
**Recommendation**: Say "harsh, disk-persisted death" rather than "permadeath," unless the ash/lost rules are verified from the manual.

**Claim**: Combat was turn-based.
**Verdict**: SUPPORTED
**Sources**:
- Wizardry Wiki: players assign actions, then resolution runs in random order influenced by dexterity; fixed and random encounters.
- Wikipedia: random enemy encounters in a pseudo-3D first-person maze.
**Contradictory evidence**: None.
**Recommendation**: Use as-is.

**Claim**: Wizardry sold well and mattered in Japan, and influenced Dragon Quest and Final Fantasy.
**Verdict**: SUPPORTED for Dragon Quest; PARTIALLY SUPPORTED for Final Fantasy
**Sources**:
- Wikipedia (Wizardry I): 24,000 copies by 30 June 1982; 200,000 in first three years, outselling Ultima; reportedly well over 500,000 by 1989. Temple of Apshai had sold 30,000 and Ultima 20,000 by mid-1982.
- Wikipedia (series): 1.5 million copies by 1987, 2 million by 1991, over 5 million by 1996. Published in Japan by ASCII; Japanese FM-7 release November 1985; Famicom release 22 December 1987; many ports (PC-8801, PC-9801, MSX2, X1, PC Engine, Super Famicom, PlayStation, Saturn, DS).
- Game Informer, Yuji Horii interview (19 November 2025, primary): Horii wanted to combine Ultima's 2D overworld with Wizardry's character growth on the Famicom. Horii in a 2000 Game Maestro interview: "I was super into Wizardry at the time."
- Wikipedia (series): Horii hid a Wizardry reference ("MONSTER SURPRISED YOU") in Portopia Serial Murder Case (1983).
**Contradictory evidence**: Final Fantasy: Wikipedia lists it among Wizardry-influenced games but no creator quote was found. Sakaguchi's link to Wizardry is unverified here. The Dragon Quest slime is said to be inspired by Wizardry by a secondary source only.
**Recommendation**: Cite Horii by name for Dragon Quest. Treat Final Fantasy as "influenced the genre it belongs to" unless a Sakaguchi interview is found.

### MUDs

**Claim**: MUD1 was created by Roy Trubshaw and Richard Bartle at Essex University, 1978-1980.
**Verdict**: SUPPORTED
**Sources**:
- Bartle, "Early MUD History" (1990, primary): first MUD written by Trubshaw in MACRO-10 in autumn 1978; BCPL rewrite from late 1979 to about Easter 1980 (the finished BCPL product was version 3). Bartle: "MUD was created and written by Roy Trubshaw and Richard Bartle at Essex University in the UK."
- Wikipedia "MUD1": Trubshaw graduated late 1980 and left Bartle in charge.
**Contradictory evidence**: Wikipedia's MUD1 page gives 1977 as the creation year; Bartle gives autumn 1978. Prefer Bartle.
**Recommendation**: Use 1978 with Bartle as the source.

**Claim**: MUD1 mechanics.
**Verdict**: SUPPORTED
**Sources**:
- Bartle: early versions were interconnected locations where you could move and chat; database (rooms, objects, commands) lived in a separate file, written in Trubshaw's MUDDL, and could be extended during play. When Trubshaw left there was no scoring, no mobiles, no containers. Bartle added the wizard mechanic (formerly "debug mode"), points, and the persona system. Machine memory was about 50K 36-bit words.
- Wikipedia "MUD1": first external players came via EPSS and ARPANET in spring 1980; 1983 remote access via British Telecom's Packet Switch Stream, 2am to 7am.
**Contradictory evidence**: None. Note "Zork article in December 1980 Byte" mentions MUD (Bartle).
**Recommendation**: Use as-is. Good fit for the claim that MUD content was hand-authored (MUDDL database) and extensible.

**Claim**: MUD1 was non-commercial at origin and later licensed commercially.
**Verdict**: SUPPORTED
**Sources**:
- Bartle: ran at Essex; code sent to Norway, Sweden, Australia, USA; "always full to capacity" in its overnight window; licensed to CompuServe as "British Legends."
- Wikipedia "MUD1": CompuServe licensed it in 1987 and pressured Essex to close its instance (account deleted October 1987); British Legends ran until late 1999; 2020 PDP-10 source released on GitHub under a custom not-for-profit license.
**Contradictory evidence**: The 2020 licence is custom not-for-profit, not an open-source licence in the OSI sense. A secondary source says Bartle's "Essex MUD" ran to late 1987.
**Recommendation**: Use as-is. Describe MUD1 as academic, later commercialized.

**Claim**: AberMUD was the first popular open-source MUD.
**Verdict**: SUPPORTED
**Sources**:
- Wikipedia "AberMUD": first version 1987 at University of Wales, Aberystwyth, written in B; ported to C in late 1988 (AberMUD2); released January 1989; copies sent to US players Vijay Subramaniam and Bill Wisner who spread it; called "the first popular open source MUD"; heavily influenced by MUD1; combat-oriented; AberMUD V later GPL.
- Wikipedia "Multi-user dungeon": AberMUD inspired TinyMUD, LPMud, and DikuMUD.
**Contradictory evidence**: Creator list conflicts. The fetched article names Alan Cox, Richard Acott, Jim Finnis, and Leon Thrane (Michael Lawrie later maintained it). Other memory-based lists differ. Bartle's page is a reply to Alan Cox about AberMUD.
**Recommendation**: Name Alan Cox as creator; hedge on the full list.

**Claim**: TinyMUD and MOO were social, user-built, non-combat.
**Verdict**: SUPPORTED
**Sources**:
- Wikipedia "TinyMUD": by James Aspnes, late 1989, a stripped-down version of Richard Skrenta's Monster; deliberately moved away from combat, with "D" claimed as "Domain" or "Dimension"; players build areas and puzzles; TinyMUD Classic ran August 1989 to April 1990; spawned TinyMUCK (Forth-based MUF), TinyMUSH; the "MU*" label.
- Wikipedia "LambdaMOO": founded 1990 by Pavel Curtis at Xerox PARC; MOO language; players program and build; population near 10,000 around 1994, over 300 connected at once; 1993 ballot system (66% approval); still running, volunteer-operated, free; home modelled on Curtis's California house. Julian Dibbell's "A Rape in Cyberspace" tested self-governance.
**Contradictory evidence**: None. Pavel Curtis' 1990 date comes only from Wikipedia.
**Recommendation**: Use as-is. Strong precedent for "players author the world."

**Claim**: LPMud (Lars Pensjö, 1989) introduced LPC and in-game wizard authoring.
**Verdict**: SUPPORTED
**Sources**:
- Wikipedia "LPMud": created 1989 by Lars Pensjö (first game Genesis, April 1989); C-like object-oriented LPC; split into driver (virtual machine that protects the server from builder errors) and mudlib; experienced players ("wizards") add rooms, weapons, monsters in-game; guilds instead of classes; derivatives LDMud, MudOS (debuted 18 February 1992), DGD (Felix Croes), SWLPC; notable MUDs Genesis, Threshold, Two Towers.
- Wikipedia "Multi-user dungeon": LPMud combined TinyMUD flexibility with AberMUD gameplay.
**Contradictory evidence**: Some sources give 1990 or 1991 for variants. Licensing terms for the original LPMud were not retrievable; Pensjö's code was distributed freely but terms were not verified.
**Recommendation**: Use with "1989." Do not assert a licence without checking.

**Claim**: DikuMUD (1990, Copenhagen) was free, hard-coded in world, hack and slash, four classes, and spawned most later codebases.
**Verdict**: SUPPORTED
**Sources**:
- Wikipedia "DikuMUD": five DIKU students (Sebastian Hammer, Michael Seifert, Hans Henrik Staerfeldt, Tom Madsen, Katja Nyboe); developed 1990-1991; opened to the public 3 February 1991; Gamma March 1991; four classes; group members share kill score; world hard-coded unlike TinyMUD and LPMud; spun off Circle, Silly, Merc; Merc to ROM and Envy; SMAUG from Diku and Merc. Raph Koster: Diku produced the greatest proliferation of gameworlds. Bartle called EverQuest "basically a DikuMUD with a graphical client bolted on"; Verant denied code derivation.
- Search summary of Wikipedia and CircleMUD sources: the 1990 Diku licence bans commercial use and charging for distribution, requires creator names in login and credits, and requires notifying the authors; the authors added an LGPL release in February 2020 (derivatives not auto-relicensed).
- Article external links: "700+ Dikumud MUD Listings" at The Mud Connector (undated).
**Contradictory evidence**: The Wikipedia text does not state licence terms; the licence claim comes from a search summary and needs confirmation from the actual licence text (circlemud.org/dlicense.html).
**Recommendation**: Use as-is for dates and creators. For "non-commercial," cite the licence text directly. Describe EverQuest line as Bartle's opinion.

**Claim**: CircleMUD, Merc, and ROM details.
**Verdict**: SUPPORTED (dates), PARTIALLY SUPPORTED (popularity)
**Sources**:
- CircleMUD (search summaries of fandom/Wikipedia/circlemud.org): Jeremy Elson at Johns Hopkins; derived from DikuMUD Gamma 0.0; version 1.0 ran January to September 1992 and was never released; first public release 16 July 1993; last Elson release 3.1 on 18 November 2002; 3.5 on 11 December 2006; renamed tbaMUD (version 3.51, 24 February 2007) by The Builder Academy, which adds world-building tools and scripting; licence moved to LGPL in 2020.
- Merc (mudbytes, igdb, Aarchon pages via search): by Michael Chastain (Furey), Michael Quan (Kahn), Mitchell Tse (Hatchet) at UC Berkeley; released 18 December 1992, about 30,000 lines of C; 2.2 in November 1993.
- ROM (fandom and mudconnect FAQ via search): Russ Taylor (Alander); started February 1993 from Merc 1.0; ROM II from July 1993; ROM 2.4b6 released May 1998; Rivers of MUD shut 31 March 2006; Aardwolf, "one of the most popular MUDs in the late 1990s," ran ROM 2.4.
**Contradictory evidence**: All from tertiary or wiki pages; no player counts per codebase were found. The "Rivers of MUD ... free for use" phrase is a fandom claim.
**Recommendation**: Use dates. For popularity use the 1995 Independent figure (below), not codebase counts.

**Claim**: Documented MUD player counts.
**Verdict**: PARTIALLY SUPPORTED
**Sources**:
- Wikipedia "Multi-user dungeon": in 1995 The Independent reported over 60,000 players regularly playing about 600 MUDs, up from 170 MUDs three years earlier. Mud Connector listed active MUDs; a 2009 article described the community as in decline.
- LambdaMOO: about 10,000 registered around 1994, 300+ concurrent.
- Bartle: MUD1 "always full" in its window; no numeric counts.
**Contradictory evidence**: No peak-concurrent figure for DikuMUD, ROM, or CircleMUD was found. The 1995 figure comes from a newspaper via Wikipedia.
**Recommendation**: Use the 1995 figure with attribution to The Independent. Flag that per-codebase counts are unknown.

## Bibliography Entries

```bibtex
@online{bartle1990earlymud,
  author       = {Bartle, Richard},
  title        = {Early {MUD} History},
  year         = {1990},
  url          = {https://mud.co.uk/richard/mudhist.htm},
  urldate      = {2026-11-09},
  note         = {Primary account by MUD1 co-creator: MACRO-10 start autumn 1978, BCPL rewrite by Easter 1980, MUDDL database, wizards and points, EPSS/ArpaNet access, CompuServe licence as British Legends.},
}

@online{wikipedia2026zork,
  author       = {{Wikipedia}},
  title        = {Zork},
  year         = {2026},
  url          = {https://en.wikipedia.org/wiki/Zork},
  urldate      = {2026-11-09},
  note         = {Tertiary. Authors, MDL/PDP-10 origin, Dungeon port, release dates, sales through 1989, Library of Congress canon.},
}

@online{jerz2000zork,
  author       = {Jerz, Dennis G.},
  title        = {Zork},
  url          = {https://jerz.setonhill.edu/if/canon/Zork.htm},
  urldate      = {2026-11-09},
  note         = {Listed in search results; not opened. Interactive fiction canon page; candidate source for verification.},
}

@online{maher2012birthinfocom,
  author       = {Maher, Jimmy},
  title        = {The Birth of {Infocom}},
  year         = {2012},
  url          = {https://www.filfre.net/2012/01/the-birth-of-infocom/},
  urldate      = {2026-11-09},
  note         = {The Digital Antiquarian. Founding of Infocom by Anderson, Lebling, Blank; rationale for a portable game design for incompatible microcomputers.},
}

@online{maher2012zilzmachine,
  author       = {Maher, Jimmy},
  title        = {{ZIL} and the {Z-Machine}},
  year         = {2012},
  url          = {https://www.filfre.net/2012/01/zil-and-the-z-machine/},
  urldate      = {2026-11-09},
  note         = {The Digital Antiquarian. Z-machine design, paging, 32K target, content cuts (191 to 110 rooms, 911 to 617 words).},
}

@online{wikipedia2026zmachine,
  author       = {{Wikipedia}},
  title        = {Z-machine},
  year         = {2026},
  url          = {https://en.wikipedia.org/wiki/Z-machine},
  urldate      = {2026-11-09},
  note         = {Tertiary. Berez and Blank 1979, 15 platforms by July 1984, Inform 1993, Nelson standard.},
}

@online{wikipedia2026grue,
  author       = {{Wikipedia}},
  title        = {Grue (monster)},
  year         = {2026},
  url          = {https://en.wikipedia.org/wiki/Grue_(monster)},
  urldate      = {2026-11-09},
  note         = {Tertiary. Lebling added the grue to replace pits; Jack Vance name source.},
}

@online{ifwiki2026infocomparser,
  author       = {{IFWiki}},
  title        = {Infocom-type parser},
  url          = {https://www.ifwiki.org/Infocom-type_parser},
  urldate      = {2026-11-09},
  note         = {Describes multiple-object handling and parser behavior; quotes Zork manual examples.},
}

@online{wikipedia2026wizardry1,
  author       = {{Wikipedia}},
  title        = {Wizardry: Proving Grounds of the Mad Overlord},
  year         = {2026},
  url          = {https://en.wikipedia.org/wiki/Wizardry:_Proving_Grounds_of_the_Mad_Overlord},
  urldate      = {2026-11-09},
  note         = {Tertiary. Authors, 1978 start, party of six, classes, ten levels, sales figures, Japan reception.},
}

@online{wikipedia2026wizardryseries,
  author       = {{Wikipedia}},
  title        = {Wizardry (video game series)},
  year         = {2026},
  url          = {https://en.wikipedia.org/wiki/Wizardry_(video_game_series)},
  urldate      = {2026-11-09},
  note         = {Tertiary. Series sales (5 million by 1996), ASCII in Japan, class change overview, Horii link.},
}

@online{wizardrywiki2026pgmo,
  author       = {{Wizardry Wiki}},
  title        = {Wizardry: Proving Grounds of the Mad Overlord},
  year         = {2026},
  url          = {https://wizardry.wiki.gg/wiki/Wizardry:_Proving_Grounds_of_the_Mad_Overlord},
  urldate      = {2026-11-09},
  note         = {Fan wiki. Permanent death written to disk, recovery of bodies, no mid-dungeon save, combat order.},
}

@online{gameinformer2025horii,
  author       = {{Game Informer}},
  title        = {Yuji Horii's Journey To Making {Dragon Quest}, As Told By The Man Himself},
  year         = {2025},
  url          = {https://gameinformer.com/exclusive-interview/2025/11/19/yuji-horiis-journey-to-making-dragon-quest-as-told-by-the-man},
  urldate      = {2026-11-09},
  note         = {Interview. Horii cites Ultima and Wizardry as inspirations for Dragon Quest.},
}

@online{wikipedia2026mud1,
  author       = {{Wikipedia}},
  title        = {{MUD1}},
  year         = {2026},
  url          = {https://en.wikipedia.org/wiki/MUD1},
  urldate      = {2026-11-09},
  note         = {Tertiary. Essex and ARPANET access, CompuServe licence, 2020 source release. Gives 1977 start, which conflicts with Bartle (1978).},
}

@online{wikipedia2026mud,
  author       = {{Wikipedia}},
  title        = {Multi-user dungeon},
  year         = {2026},
  url          = {https://en.wikipedia.org/wiki/Multi-user_dungeon},
  urldate      = {2026-11-09},
  note         = {Tertiary. Codebase family tree, 1995 Independent figure (60,000 players, 600 MUDs).},
}

@online{wikipedia2026abermud,
  author       = {{Wikipedia}},
  title        = {{AberMUD}},
  year         = {2026},
  url          = {https://en.wikipedia.org/wiki/AberMUD},
  urldate      = {2026-11-09},
  note         = {Tertiary. 1987 Aberystwyth origin, C port 1988, January 1989 release, first popular open-source MUD.},
}

@online{wikipedia2026tinymud,
  author       = {{Wikipedia}},
  title        = {{TinyMUD}},
  year         = {2026},
  url          = {https://en.wikipedia.org/wiki/TinyMUD},
  urldate      = {2026-11-09},
  note         = {Tertiary. Aspnes, 1989, social and non-combat, MU* family.},
}

@online{wikipedia2026lambdamoo,
  author       = {{Wikipedia}},
  title        = {{LambdaMOO}},
  year         = {2026},
  url          = {https://en.wikipedia.org/wiki/LambdaMOO},
  urldate      = {2026-11-09},
  note         = {Tertiary. Curtis, Xerox PARC, 1990; about 10,000 users near 1994.},
}

@online{wikipedia2026lpmud,
  author       = {{Wikipedia}},
  title        = {{LPMud}},
  year         = {2026},
  url          = {https://en.wikipedia.org/wiki/LPMud},
  urldate      = {2026-11-09},
  note         = {Tertiary. Pensjo 1989, LPC, driver and mudlib, wizard coding, MudOS and DGD.},
}

@online{wikipedia2026dikumud,
  author       = {{Wikipedia}},
  title        = {{DikuMUD}},
  year         = {2026},
  url          = {https://en.wikipedia.org/wiki/DikuMUD},
  urldate      = {2026-11-09},
  note         = {Tertiary. Five DIKU authors, opened 3 February 1991, four classes, derivative codebases, Bartle on EverQuest.},
}

@online{elson2006circlemud,
  author       = {Elson, Jeremy},
  title        = {{CircleMUD}},
  url          = {https://www.circlemud.org/},
  urldate      = {2026-11-09},
  note         = {Project home page; hosts the DikuMUD licence copy and administrator manual. Not opened directly; verify licence terms here.},
}
```

## Research Gaps

**Claim**: Zork vocabulary size and parser limits stated in precise terms.
**What's missing**: The "908 words, 71 actions" figure has no verified source. The Digital Antiquarian gives 911 (MIT) and 617 (Zork I).
**Suggested action**: Read the Infocom "Internal Secrets" PDF (eblong.com) or the Zork I manual and cite the primary document.

**Claim**: Wizardry ash versus lost status, temple resurrection odds, class change specifics.
**What's missing**: The Apple II manual or Sir-Tech documentation; the wiki fetched did not cover these.
**Suggested action**: Find the scanned Wizardry I manual (Internet Archive) and cite it.

**Claim**: Final Fantasy was influenced by Wizardry.
**What's missing**: A Sakaguchi or Square interview. Only the Wikipedia list was found.
**Suggested action**: Search for a Sakaguchi interview or drop the claim.

**Claim**: Peak player numbers for DikuMUD, CircleMUD, ROM, Merc, AberMUD, and LPMud.
**What's missing**: No per-codebase installation counts with dates. Only "700+ DikuMUD listings" (undated, from Wikipedia external link text) and the 1995 aggregate figure.
**Suggested action**: Check the Mud Connector archive, the rec.games.mud FAQ (mudconnect.com/mudfaq), and Raph Koster's writing on MUD families; otherwise state that counts are undocumented.

**Claim**: DikuMUD, LPMud, ROM, and CircleMUD were "free/non-commercial" by licence.
**What's missing**: Direct licence text. Diku non-commercial terms came from a search summary. LPMud and ROM licence terms were not retrieved.
**Suggested action**: Read circlemud.org/dlicense.html and the ROM licence file (license.doc in ROM 2.4 distribution) before using the claim.

**Claim**: Sales figures for Zork and Wizardry.
**What's missing**: Original trade-press sources behind the Wikipedia figures.
**Suggested action**: Open Wikipedia footnotes (InfoWorld, Computer Gaming World, Softalk) and cite those directly.

**Claim**: Zork parser as "verb-noun natural language" and "frobozz lexicon."
**What's missing**: Frobozz lexicon was not researched in detail beyond the Wizard of Frobozz name in Zork II.
**Suggested action**: Consult the Zork manuals and the Infocom lore page if the PR/FAQ wants this.

**Note on contradiction to the PR/FAQ thesis**: The evidence shows the originals were already extended by players and authors beyond a single hand-written author: LPMud wizards and TinyMUD/MOO users built the world in-game, and MUD1's database could be extended during play. "Hand-authored, finite content" fits Zork (content cut to fit 32K) and Wizardry, but MUDs partly answered it with crowd authorship. The PR/FAQ should position the LLM against that, not against a single-author baseline.
