# Research: "openclaw" as an agent-authored social site, and agent-populated worlds
**Date:** 2026-11-09
**Request:** Identify "openclaw" (agent-authored social media); survey agent-only social networks; anchor the pattern in research (Generative Agents).
**Claims investigated:** 4

## Evidence Found

**Claim**: "openclaw" is a social-media site where AI agents author and populate the content.
**Verdict**: PARTIALLY SUPPORTED. The user is almost certainly remembering Moltbook, the agent-only social network built for OpenClaw agents. OpenClaw itself is the agent framework, not the site.
**Sources**:
- OpenClaw (agent framework). Open-source personal AI agent by Peter Steinberger, formerly Clawdbot, then Moltbot, renamed after Anthropic raised a name conflict with Claude. Per TechXplore and Milvus (secondary/tertiary). 114,000+ GitHub stars per one secondary source. OpenAI hired Steinberger in early 2026 (reported in the Meta/Moltbook coverage).
- Moltbook (moltbook.com). Reddit-style network where only AI agents post; humans observe. Launched 2026-01-28 by Matt Schlicht (Octane AI CEO), with COO Ben Parr. Topic communities are "Submolts." Agents join by reading a markdown skill file and fetch a heartbeat file every four hours. Schlicht said he wrote no code himself (vibe-coded with his assistant "Clawd Clawderberg"). Sources: Wikipedia, Axios, CNBC, Built In, IEEE Spectrum (secondary).
- Scale: 32,000 agents in the first 72 hours (chiefaiofficer, tertiary). As of 2026-06-06 the site claimed 206,839 verified agents of 2,895,874 registered (Wikipedia).
- Acquisition: Meta acquired Moltbook, announced 2026-03-10 (Axios first), team into Meta Superintelligence Labs, terms undisclosed.
- Third-party "OpenClaw Social" / "OpenClaw Forum" sites (openclawsocial.org, openclawsocial.forum, openclawforum.social) exist but appear to be unofficial information portals. I did not verify any of them as an agent-populated network.
- Academic study: Manik and Wang, arXiv 2602.02625 (2026-02-02), 39,026 posts and 5,712 comments from 14,490 agents. 18.4% of posts contain action-inducing language; risky instructions drew more cautionary replies than neutral posts. Also arXiv 2602.13458 "MoltNet" (surfaced in search, not read).
**Contradictory evidence**: Autonomy and scale are disputed. Wiz found 1.5M registered agents traced to about 17,000 human owners (88:1), no rate limiting or identity checks, and a humans-can-post-as-agents hole (Supabase key exposed, no row-level security; 1.5M tokens, about 35,000 emails, 4,060 DMs exposed; patched within hours). CNBC found some interactions driven by explicit human direction. MIT Technology Review called it "AI theater"; Simon Willison called the output "complete slop" (both via Wikipedia, not read at source).
**Recommendation**: Cite Moltbook, not "openclaw," as the example. Say "built for OpenClaw agents." Add the caveat that agent autonomy is disputed and human operators prompt many agents.

**Claim**: "OpenClaw" is also the name of a Captain Claw reimplementation.
**Verdict**: SUPPORTED, and unrelated to social media.
**Sources**:
- pjasicek/OpenClaw (GitHub): C++/SDL2 GPL-3.0 reimplementation of the 1997 platformer Captain Claw, by Petr Jasicek, needs original CLAW.REZ assets, many forks. Primary (repo).
**Contradictory evidence**: None. Name collision only.
**Recommendation**: Do not conflate with the agent framework.

**Claim**: Other agent-populated or agent-only social networks exist.
**Verdict**: SUPPORTED.
**Sources**:
- Chirper.ai: launched April 2023; X-like network where humans create an AI persona and the agent then posts autonomously (Boing Boing 2023-04-27). Zhu et al., arXiv 2504.10286 (also ACM): 65,000+ agents, 7.7M AI-generated posts, agents self-disclose 1.83x more than humans, notable abusive content, 76.42% of agents in one strongly connected component. Primary (paper).
- SocialAI: by Michael Sayman (Friendly Apps), launched 2024-09-16, iPhone only. One human per account; all followers are LLM bots (choose supporters, trolls, etc.). A private diary, not a shared network. TechCrunch 2024-09-17. Later Sayman joined Meta (per search summary, not verified at source).
- Butterflies AI: Vu Tran (ex-Snap), public launch June 2024 (TechCrunch 2024-06-18), $4.8M seed led by Coatue (Nov 2023). Mixed human and AI-persona network. Not agent-only.
**Contradictory evidence**: Butterflies and SocialAI include or center a human; only Chirper.ai and Moltbook are agent-only.
**Recommendation**: Use Chirper.ai (2023, studied) and Moltbook (2026, viral) as the agent-only precedents.

**Claim**: Stanford's Generative Agents (Smallville) anchors "world populated by autonomous agents."
**Verdict**: SUPPORTED.
**Sources**:
- Park, O'Brien, Cai, Morris, Liang, Bernstein. "Generative Agents: Interactive Simulacra of Human Behavior." UIST '23, 2023-10-29 to 11-01. arXiv 2304.03442 (2023-04-07), DOI 10.1145/3586183.3606763. 25 agents in a Sims-like sandbox, memory/reflection/planning, unscripted Valentine's Day party. Code: joonspk-research/generative_agents.
- Altera Project Sid, arXiv 2411.00114 (2024-10-31): hundreds of agents (about 500; MIT Tech Review says up to 1,000) in Minecraft developing roles, voting on tax rules, spreading a parody religion. Code: altera-al/project-sid.
- OpenLife (arXiv 2606.31046) and Superminds Test (arXiv 2604.22452) surfaced in search; titles only, not read.
**Contradictory evidence**: Smallville is a designed sandbox with 25 agents, not an open authored world; Project Sid results are the company's own report.
**Recommendation**: Use Park et al. as the primary anchor, Project Sid as scale evidence.

## Bibliography Entries

```bibtex
@online{wikipedia2026moltbook,
  author  = {{Wikipedia}},
  title   = {Moltbook},
  year    = {2026},
  url     = {https://en.wikipedia.org/wiki/Moltbook},
  urldate = {2026-11-09},
  note    = {Launch 2026-01-28, Schlicht, OpenClaw link, scale, authenticity disputes, Meta acquisition. Tertiary.},
}

@online{axios2026metamoltbook,
  author  = {Fried, Ina},
  title   = {Exclusive: Meta acquires Moltbook, the social network for AI agents},
  year    = {2026},
  url     = {https://www.axios.com/2026/03/10/meta-facebook-moltbook-agent-social-network},
  urldate = {2026-11-09},
  note    = {First report of the 2026-03-10 acquisition. Secondary.},
}

@online{cnbc2026metamoltbook,
  author  = {{CNBC}},
  title   = {Meta gets into social networks for AI agents with acquisition of viral Moltbook platform},
  year    = {2026},
  url     = {https://www.cnbc.com/2026/03/10/meta-social-networks-ai-agents-moltbook-acquisition.html},
  urldate = {2026-11-09},
  note    = {Acquisition coverage; reports human direction behind some agent interactions. Secondary.},
}

@online{techxplore2026openclaw,
  author  = {{TechXplore}},
  title   = {OpenClaw's AI agent does everything, even social media},
  year    = {2026},
  url     = {https://techxplore.com/news/2026-02-openclaw-ai-agent-social-media.html},
  urldate = {2026-11-09},
  note    = {Background on OpenClaw (formerly Clawdbot, Moltbot) and Steinberger. Secondary.},
}

@online{ieee2026moltbook,
  author  = {{IEEE Spectrum}},
  title   = {Moltbook, agentic AI, and OpenClaw},
  year    = {2026},
  url     = {https://spectrum.ieee.org/moltbook-agentic-ai-agents-openclaw},
  urldate = {2026-11-09},
  note    = {Explainer on Moltbook and OpenClaw. Search result only, not read. Secondary.},
}

@online{wiz2026moltbook,
  author  = {{Wiz Research}},
  title   = {Moltbook database exposure (as reported by third parties)},
  year    = {2026},
  url     = {https://www.aicerts.ai/news/wiz-reveals-moltbook-database-security-breach-1-5m-keys-exposed/},
  urldate = {2026-11-09},
  note    = {Secondhand report: 1.5M tokens exposed, about 17,000 human owners (88:1). Locate the original Wiz post before citing. Tertiary.},
}

@misc{manik2026moltbook,
  author       = {Manik, Md Motaleb Hossen and Wang, Ge},
  title        = {{OpenClaw} Agents on {Moltbook}: Risky Instruction Sharing and Norm Enforcement in an Agent-Only Social Network},
  year         = {2026},
  eprint       = {2602.02625},
  archiveprefix = {arXiv},
  url          = {https://arxiv.org/abs/2602.02625},
  note         = {39,026 posts, 5,712 comments, 14,490 agents. Primary.},
}

@misc{zhu2025chirper,
  author       = {Zhu et al.},
  title        = {Characterizing an {LLM}-driven Social Network: The Case of {Chirper.ai}},
  year         = {2025},
  eprint       = {2504.10286},
  archiveprefix = {arXiv},
  url          = {https://arxiv.org/abs/2504.10286},
  note         = {65,000+ agents, 7.7M posts. Author list truncated in search results; verify.},
}

@online{boingboing2023chirper,
  author  = {{Boing Boing}},
  title   = {Chirper is a social network for artificial intelligences},
  year    = {2023},
  url     = {https://boingboing.net/2023/04/27/chirper-is-a-social-network-for-artificial-intelligences.html},
  urldate = {2026-11-09},
  note    = {April 2023 launch coverage. Tertiary.},
}

@online{techcrunch2024socialai,
  author  = {{TechCrunch}},
  title   = {{SocialAI} offers a Twitter-like diary where AI bots respond to your posts},
  year    = {2024},
  url     = {https://techcrunch.com/2024/09/17/socialai-offers-a-twitter-like-diary-where-ai-bots-respond-to-your-posts/},
  urldate = {2026-11-09},
  note    = {Sayman, launched 2024-09-16, all followers are bots. Secondary.},
}

@online{techcrunch2024butterflies,
  author  = {{TechCrunch}},
  title   = {Former Snap engineer launches Butterflies, a social network where AIs and humans coexist},
  year    = {2024},
  url     = {https://techcrunch.com/2024/06/18/former-snap-engineer-launches-butterflies-a-social-network-where-ais-and-humans-coexist},
  urldate = {2026-11-09},
  note    = {Vu Tran, June 2024 launch. Secondary.},
}

@inproceedings{park2023generative,
  author    = {Park, Joon Sung and O'Brien, Joseph C. and Cai, Carrie J. and Morris, Meredith Ringel and Liang, Percy and Bernstein, Michael S.},
  title     = {Generative Agents: Interactive Simulacra of Human Behavior},
  booktitle = {Proceedings of the 36th Annual ACM Symposium on User Interface Software and Technology (UIST '23)},
  year      = {2023},
  doi       = {10.1145/3586183.3606763},
  url       = {https://arxiv.org/abs/2304.03442},
  note      = {Smallville, 25 agents. Primary.},
}

@misc{altera2024projectsid,
  author       = {{Altera.AL}},
  title        = {Project Sid: Many-agent simulations toward {AI} civilization},
  year         = {2024},
  eprint       = {2411.00114},
  archiveprefix = {arXiv},
  url          = {https://arxiv.org/abs/2411.00114},
  note         = {Hundreds of agents in Minecraft. Primary, self-reported.},
}

@online{jasicek2016openclawgame,
  author  = {Jasicek, Petr},
  title   = {{OpenClaw}: Reimplementation of Captain Claw (1997) platformer},
  url     = {https://github.com/pjasicek/OpenClaw},
  urldate = {2026-11-09},
  note    = {Unrelated game project sharing the name. Year of first commit not verified.},
}
```

## Research Gaps

**Claim**: No site literally named "openclaw" is an agent-populated network.
**What's missing**: I did not open openclawsocial.org, openclawsocial.forum, or openclawforum.social. Search snippets call them unofficial portals.
**Suggested action**: Ask the user whether "openclaw" meant Moltbook. If not, open those three sites.

**Claim**: Moltbook human-vs-agent ratio and "AI theater" critiques.
**What's missing**: Original Wiz post, MIT Technology Review article, and the CNBC piece were not read directly; figures come from Wikipedia and aggregator summaries.
**Suggested action**: Fetch the Wiz blog and MIT Tech Review before quoting numbers in the PR/FAQ.

**Claim**: Persistent multi-agent authored worlds beyond Smallville and Project Sid.
**What's missing**: a16z AI Town, OpenLife (2606.31046), Superminds Test (2604.22452), and MoltNet (2602.13458) were not read. Smallville and Sid are not worlds authored by agents as scenario designers.
**Suggested action**: Targeted search if the PR/FAQ needs agent-authored (not agent-populated) world precedent.

**Housekeeping**: `research/` is not in /Users/jfreeman/Coding/punt-labs/cryptd/.gitignore. I did not edit it because the file is managed by a rollout script and existing research files are already untracked.
