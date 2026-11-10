# Research: LLM-generated on-demand UI/UX and game presentation
**Date:** 2026-11-09
**Request:** Academic and authoritative sources on runtime LLM-generated UI/UX, applied to crypt (LLM-generated maps, images, scene descriptions)
**Claims investigated:** 5 threads

Verification level: "fetched" means I read the arXiv abstract page. "search" means details come from search-result summaries only; check before quoting numbers.

## Evidence Found

### Thread 1: Generative UI / on-demand UI. WELL SUPPORTED (new field, 2025-2026, mostly preprints)

- Leviathan, Valevski, et al. (Google), "Generative UI: LLMs are Effective UI Generators", arXiv 2604.09577, Feb 2026 (fetched). Gemini 3 Pro emits interactive web pages per prompt. Humans strongly prefer them to markdown output; results are "at least comparable" to human-expert pages in 50% of cases. Releases PAGEN, an expert-crafted dataset. Search summaries add: 73.5% win rate vs top search result, 50% vs 35.3% for developer-built sites, generation latency excluded from the comparison. Preprint plus Google Research blog (industry). STRONGEST industry-backed source.
- Chen, Zhang, Zhang, Shao, Yang (Stanford SALT), "Generative Interfaces for Language Models", arXiv 2508.19227, Aug 2025; ACL 2026 Findings (search). LLM generates a task-specific UI instead of chat text. Humans prefer it in over 70% of cases (search); iterative refinement beats one-shot by 14% win rate (search). Peer-reviewed. STRONGEST academic source.
- Cao, Jiang, Xia (UCSD), "Generative and Malleable User Interfaces with Generative and Evolving Task-Driven Data Model" (Jelly), CHI 2025, DOI 10.1145/3706598.3713285 (search). Interface regenerated from an LLM-built task data model; data persists across UI changes. Peer-reviewed. Most-cited HCI anchor.
- Min, Jiang, Huang, Xia, "Gradual Generation of User Interfaces as a Design Method for Malleable Software", UIST 2025, arXiv 2601.17975 (search). Five pages; layered generation so users can discover customizations. Peer-reviewed, short.
- Bernstein, "The Interface of Theseus: The Rise of Just-In-Time Interfaces", arXiv 2609.06770, Sep 2026, UIST Visions 2026 (fetched). Position paper: interfaces become "rapid, discardable, on-demand," generated in the moment. Best framing citation for "on-demand presentation is the general pattern." A vision paper, not an evaluation.
- Cifliku, "Hidden Technical Debt in Generative (GenUI) and Malleable User Interfaces", arXiv 2604.16354, CHI 2026 workshop (fetched). Names barriers and calls for new evaluation methods. Workshop paper.
- "GenerativeGUI: Dynamic GUI Generation Leveraging LLMs for Enhanced User Interaction on Chat Interfaces", CHI 2025 Extended Abstracts, DOI 10.1145/3706599.3719743 (search). LLM emits HTML each conversation turn. Late-breaking-work level.
- ACM CHI 2026 workshop "What does Generative UI mean for HCI Practice?" (DOI 10.1145/3772363.3778757, search) and a CHI 2026 "GenUI meets HCI" runtime-UI position paper (search) say generated UI mutates too much and needs bounded, stabilized adaptation.
Contradicting or limiting evidence: Google's own results are below human experts in half the cases and exclude latency. Cifliku and the CHI 2026 workshop paper flag change aversion, security and lack of data formats.

### Thread 2: LLMs emitting UI code or components. SUPPORTED, thinner on usability and latency
- Covered by the Google and Stanford papers above (HTML/JS and structured UI representations with reusable widgets such as map, chart, clock).
- Bunian/Wu-style work: "UI Layout Generation with LLMs Guided by UI Grammar", arXiv 2310.15455 (search only; authors unverified). Earlier, narrower layout-generation work.
- Vercel v0 and similar products: not researched; no peer-reviewed evaluation located. Treat as industry only.
- Latency is the weak point: Google's comparison excludes generation time. No source found with a systematic latency-versus-usability study.

### Thread 3: Adaptive UI by LLM (HCI venues). PARTIALLY SUPPORTED
- Jelly (CHI 2025) and Gradual Generation (UIST 2025) above are the peer-reviewed core. Foundational non-LLM adaptive UI (SUPPLE, UIST 2008; Gajos and Weld) appears as a citation in these works (search); I did not verify its bibliographic details.
- arXiv 2412.16837 (reinforcement-learning adaptive UI) and arXiv 2602.03154 ("Intelligent Front-End Personalization") surfaced in search; not read, quality unknown. Not recommended without a check.
- "Efficient Personalization of Generative User Interfaces" arXiv 2604.09876 and "Maru" arXiv 2608.25565 surfaced; not read.

### Thread 4: LLM-generated game content as presentation. WELL SUPPORTED for narration and levels; THIN for images and maps at play time
- Gallotta, Todd, Zammit, Earle, Liapis, Togelius, Yannakakis, "Large Language Models and Games: A Survey and Roadmap", IEEE Transactions on Games 2024, arXiv 2402.18659, DOI 10.1109/TG.2024.3461510 (search). Surveys LLM roles in games, including game master and PCG. Peer-reviewed. STRONGEST survey.
- Sudhakaran et al., "MarioGPT: Open-Ended Text2Level Generation through Large Language Models", NeurIPS 2023, arXiv 2302.05981 (search). Text prompt to tile level. Peer-reviewed. Level generation, not runtime narration.
- Li, Li, Ruiz, Bansal, et al. (Google/UNC), "Unbounded: A Generative Infinite Game of Character Life Simulation", ICLR 2025, arXiv 2410.18975 (search). LLM game engine plus diffusion images; distilled Gemma-2B and latent consistency models bring each new scene to about 1 second against 5 seconds naive. Closest analogue to crypt's on-demand images, and the only source found that reports image latency. Peer-reviewed. Author order not verified.
- Earle/Merino-style text-to-map work: "Moonshine: Distilling Game Content Generators into Steerable Generative Models", arXiv 2408.09594, and "The Five-Dollar Model", arXiv 2308.04052 (search). Text-to-dungeon-map from Brogue data with LLM-labelled maps. Preprints; authors not verified. Directly relevant to ad-hoc maps.
- "Narrative-to-Scene Generation: An LLM-Driven Pipeline for 2D Game Environments", arXiv 2509.04481 (search). Preprint, unread.
- Zhu, Martin, Head, Callison-Burch, "CALYPSO: LLMs as Dungeon Master's Assistants", AIIDE 2023, DOI 10.1609/aiide.v19i1.27534, arXiv 2308.07540 (search). Live-use study: LLM text was high-fidelity enough to read to players; DMs wanted to keep creative agency. Peer-reviewed.
- Callison-Burch et al., "Dungeons and Dragons as a Dialog Challenge for Artificial Intelligence", EMNLP 2022, arXiv 2210.07109 (search). Models could assist but not fully run D&D. Peer-reviewed.
- Song, Zhu, Callison-Burch, "You Have Thirteen Hours in Which to Solve the Labyrinth: Enhancing AI Game Masters with Function Calling", Wordplay Workshop at ACL 2024, arXiv 2409.06949 (fetched). Function calls for dice and state improve narrative quality and state consistency under human evaluation. Directly supports crypt's L4/L1 split (engine owns state, LLM narrates).
- "Static Vs. Agentic Game Master AI for Facilitating Solo Role-Playing Experiences", arXiv 2502.19519 (search only; title only).
- AI Dungeon: no peer-reviewed player study found. Found: Mitchell, "Repetition and Defamiliarization in AI Dungeon and Project December" (Electronic Book Review, humanities essay); "Genre, Bias, and Narrative Logic in AI Dungeon" (ResearchGate, venue unverified); GPT-for-Games scoping reviews arXiv 2404.17794 and 2411.00308 (describe AI Dungeon). Do not present AI Dungeon as having a strong empirical literature.
- Survey: "Procedural Content Generation in Games: A Survey with Insights on Emerging LLM Integration", arXiv 2410.15644 (search).

### Thread 5: Evaluating generated UI and game content. SUPPORTED, new and fragmented
- RPGBench, Yu, Shen, Meng, Lee, Yin, Cui, Xu, Zhu, Shi, Li, Smola, arXiv 2502.00595, submitted to ICML 2025 (fetched; acceptance not confirmed). Mixes rule-checked mechanics with LLM-as-judge ratings; finds models write engaging stories but keep game mechanics inconsistent over long sessions. STRONGEST for evaluation-driven development of a game engine. Preprint.
- Góngora, Chiruzzo, Méndez, Gervás, "Skill Check: Some Considerations on the Evaluation of Gamemastering Models for Role-playing Games", GALA 2023, arXiv 2309.13702 (fetched). Three test categories for GM systems, applied to ChatGPT, Bard, OpenAssistant.
- Song et al. 2024 (above): unit tests plus human evaluation of state consistency.
- EvoGenUI-Bench, arXiv 2608.29387 (search): 150 five-turn tasks; checks that UI, derived state, external state and assistant claims stay synchronized across turns. Relevant to session consistency. Preprint.
- FlowEval, arXiv 2605.04165; Design Theater, arXiv 2607.22928; Persona Panel, arXiv 2607.28439; PerceptUI, arXiv 2606.05697; "MLLM as a UI Judge", arXiv 2510.08783 (all search only). LLM-judge and simulated-user evaluation of generated UI.
- Chen et al. 2025 (above): LLM judge (listwise ranker) agrees with human pairwise labels 69.0% of the time (search). Supports using LLM judges with a stated agreement ceiling.
- GenQuest, arXiv 2510.04498 (search): participants reported logical inconsistencies across branches in an LLM text adventure.

## Bibliography Entries

Entries marked (v) were read on arXiv; others rest on search results, so verify authors and pages before final use.

```bibtex
@misc{leviathan2026generativeui,
  author       = {Leviathan, Yaniv and Valevski, Dani and Kalman, Matan and Lumen, Danny and Segalis, Eyal and Molad, Eyal and Pasternak, Shlomi and Natchu, Vishnu and Nygaard, Valerie and Venkatachary, Srinivasan and Manyika, James and Matias, Yossi},
  title        = {Generative {UI}: {LLMs} are Effective {UI} Generators},
  year         = {2026},
  eprint       = {2604.09577},
  archiveprefix = {arXiv},
  url          = {https://arxiv.org/abs/2604.09577},
  note         = {Google Research. Gemini 3 Pro generates interactive pages per prompt; preferred over markdown, comparable to human experts in 50 percent of cases; PAGEN dataset. Preprint (v).},
}

@misc{chen2025generativeinterfaces,
  author       = {Chen, Jiaqi and Zhang, Yanzhe and Zhang, Yutong and Shao, Yijia and Yang, Diyi},
  title        = {Generative Interfaces for Language Models},
  year         = {2025},
  eprint       = {2508.19227},
  archiveprefix = {arXiv},
  url          = {https://arxiv.org/abs/2508.19227},
  note         = {Stanford. LLM-generated task-specific UIs preferred by humans in over 70 percent of cases; accepted to ACL 2026 Findings (per search).},
}

@inproceedings{cao2025jelly,
  author       = {Cao, Yining and Jiang, Peiling and Xia, Haijun},
  title        = {Generative and Malleable User Interfaces with Generative and Evolving Task-Driven Data Model},
  booktitle    = {Proceedings of the 2025 CHI Conference on Human Factors in Computing Systems},
  year         = {2025},
  doi          = {10.1145/3706598.3713285},
  note         = {Jelly system. Peer-reviewed.},
}

@inproceedings{min2025gradual,
  author       = {Min, Bryan and Jiang, Peiling and Huang, Zhicheng and Xia, Haijun},
  title        = {Gradual Generation of User Interfaces as a Design Method for Malleable Software},
  booktitle    = {Proceedings of the 38th Annual ACM Symposium on User Interface Software and Technology (UIST '25)},
  year         = {2025},
  eprint       = {2601.17975},
  archiveprefix = {arXiv},
  url          = {https://arxiv.org/abs/2601.17975},
  note         = {Layered UI generation for discoverable customization. Five pages.},
}

@misc{bernstein2026theseus,
  author       = {Bernstein, Michael S.},
  title        = {The Interface of Theseus: The Rise of Just-In-Time Interfaces},
  year         = {2026},
  eprint       = {2609.06770},
  archiveprefix = {arXiv},
  url          = {https://arxiv.org/abs/2609.06770},
  note         = {UIST Visions 2026 position paper: on-demand, discardable generated interfaces (v).},
}

@misc{cifliku2026technicaldebt,
  author       = {Cifliku, Besjon},
  title        = {Hidden Technical Debt in Generative ({GenUI}) and Malleable User Interfaces},
  year         = {2026},
  eprint       = {2604.16354},
  archiveprefix = {arXiv},
  url          = {https://arxiv.org/abs/2604.16354},
  note         = {CHI 2026 workshop paper; barriers to GenUI adoption and call for evaluation methods (v).},
}

@article{gallotta2024llmgames,
  author       = {Gallotta, Roberto and Todd, Graham and Zammit, Marvin and Earle, Sam and Liapis, Antonios and Togelius, Julian and Yannakakis, Georgios N.},
  title        = {Large Language Models and Games: A Survey and Roadmap},
  journal      = {IEEE Transactions on Games},
  year         = {2024},
  doi          = {10.1109/TG.2024.3461510},
  eprint       = {2402.18659},
  archiveprefix = {arXiv},
  note         = {Survey of LLM roles in games including game master and PCG.},
}

@inproceedings{sudhakaran2023mariogpt,
  author       = {Sudhakaran, Shyam and Gonz{\'a}lez-Duque, Miguel and Glanois, Claire and Freiberger, Matthias and Najarro, Elias and Risi, Sebastian},
  title        = {{MarioGPT}: Open-Ended Text2Level Generation through Large Language Models},
  booktitle    = {Advances in Neural Information Processing Systems (NeurIPS)},
  year         = {2023},
  eprint       = {2302.05981},
  archiveprefix = {arXiv},
  note         = {Text-prompted game level generation.},
}

@inproceedings{li2025unbounded,
  author       = {Li, Jialu and Li, Yuanzhen and Ruiz, Nataniel and Bansal, Mohit and others},
  title        = {Unbounded: A Generative Infinite Game of Character Life Simulation},
  booktitle    = {International Conference on Learning Representations (ICLR)},
  year         = {2025},
  eprint       = {2410.18975},
  archiveprefix = {arXiv},
  note         = {LLM game engine plus diffusion images; about 1 second per scene versus about 5 seconds naive. Full author list not verified.},
}

@misc{moonshine2024,
  title        = {Moonshine: Distilling Game Content Generators into Steerable Generative Models},
  year         = {2024},
  eprint       = {2408.09594},
  archiveprefix = {arXiv},
  url          = {https://arxiv.org/abs/2408.09594},
  note         = {Text-to-dungeon-map models trained on Brogue maps labelled by an LLM. Authors not verified; fill in before use.},
}

@inproceedings{zhu2023calypso,
  author       = {Zhu, Andrew and Martin, Lara and Head, Andrew and Callison-Burch, Chris},
  title        = {{CALYPSO}: {LLMs} as Dungeon Master's Assistants},
  booktitle    = {Proceedings of the AAAI Conference on Artificial Intelligence and Interactive Digital Entertainment (AIIDE)},
  year         = {2023},
  pages        = {380--390},
  doi          = {10.1609/aiide.v19i1.27534},
  eprint       = {2308.07540},
  archiveprefix = {arXiv},
  note         = {Live-use study; LLM text usable for direct narration; DMs want to keep agency.},
}

@inproceedings{callisonburch2022dnd,
  author       = {Callison-Burch, Chris and others},
  title        = {Dungeons and Dragons as a Dialog Challenge for Artificial Intelligence},
  booktitle    = {Proceedings of EMNLP},
  year         = {2022},
  eprint       = {2210.07109},
  archiveprefix = {arXiv},
  note         = {Models can assist but not autonomously run D\&D. Author list truncated; verify.},
}

@inproceedings{song2024labyrinth,
  author       = {Song, Jaewoo and Zhu, Andrew and Callison-Burch, Chris},
  title        = {You Have Thirteen Hours in Which to Solve the Labyrinth: Enhancing {AI} Game Masters with Function Calling},
  booktitle    = {Wordplay Workshop at ACL 2024},
  year         = {2024},
  eprint       = {2409.06949},
  archiveprefix = {arXiv},
  url          = {https://arxiv.org/abs/2409.06949},
  note         = {Function calling improves narrative quality and state consistency; human evaluation plus unit tests (v).},
}

@misc{yu2025rpgbench,
  author       = {Yu, Pengfei and Shen, Dongming and Meng, Silin and Lee, Jaewon and Yin, Weisu and Cui, Andrea Yaoyun and Xu, Zhenlin and Zhu, Yi and Shi, Xingjian and Li, Mu and Smola, Alex},
  title        = {{RPGBench}: Evaluating Large Language Models as Role-Playing Game Engines},
  year         = {2025},
  eprint       = {2502.00595},
  archiveprefix = {arXiv},
  url          = {https://arxiv.org/abs/2502.00595},
  note         = {Rule-based plus LLM-judge evaluation; models write engaging stories but keep mechanics inconsistent (v). Submitted to ICML 2025; acceptance unconfirmed.},
}

@inproceedings{gongora2023skillcheck,
  author       = {G{\'o}ngora, Santiago and Chiruzzo, Luis and M{\'e}ndez, Gonzalo and Gerv{\'a}s, Pablo},
  title        = {Skill Check: Some Considerations on the Evaluation of Gamemastering Models for Role-playing Games},
  booktitle    = {Games and Learning Alliance (GALA)},
  year         = {2023},
  eprint       = {2309.13702},
  archiveprefix = {arXiv},
  url          = {https://arxiv.org/abs/2309.13702},
  note         = {Three test categories for GM dialogue systems (v).},
}

@misc{evogenuibench2026,
  title        = {{EvoGenUI-Bench}: Evaluating {LLMs} as Multi-Turn Generative {UI} Assistants},
  year         = {2026},
  eprint       = {2608.29387},
  archiveprefix = {arXiv},
  url          = {https://arxiv.org/abs/2608.29387},
  note         = {Multi-turn consistency of UI, state and claims. Authors not verified.},
}

@misc{mllmuijudge2025,
  title        = {{MLLM} as a {UI} Judge: Benchmarking Multimodal {LLMs} for Predicting Human Perception of User Interfaces},
  year         = {2025},
  eprint       = {2510.08783},
  archiveprefix = {arXiv},
  url          = {https://arxiv.org/abs/2510.08783},
  note         = {LLM-as-judge versus human UI ratings. Full title and authors not verified.},
}
```

## Research Gaps

**Claim**: LLM-generated maps and images produced at play time improve a text adventure for players.
**What's missing**: A player study of runtime image or map generation in a text game. Unbounded reports latency and image consistency but its user evidence was not checked. Moonshine and Five-Dollar are model papers without player outcomes.
**Suggested action**: Read the Unbounded user-study section; otherwise accept as assumption and make it crypt's own evaluation target.

**Claim**: AI Dungeon has an empirical research record.
**What's missing**: No peer-reviewed player study found. Only a humanities essay, an unverified ResearchGate paper and scoping reviews.
**Suggested action**: Cite only the scoping reviews and CALYPSO; search ACM DL directly for CHI PLAY.

**Claim**: Runtime-generated UI has acceptable latency.
**What's missing**: Google's evaluation excludes generation time. No systematic latency-versus-satisfaction study found.
**Suggested action**: Treat as open; measure in crypt.

**Claim**: Generated content stays consistent across a long session.
**What's missing**: RPGBench and Song et al. show consistency fails without engine-held state. No study covers consistency of generated images or maps across a session. EvoGenUI-Bench covers UI only.
**Suggested action**: Use the engine-owns-state pattern as the cited mitigation; build own consistency checks.

**Claim**: Vercel v0 and other products demonstrate the pattern.
**What's missing**: Not researched; no peer-reviewed evaluation found.
**Suggested action**: Cite the vendor docs if needed, as industry only.

Caveats: the arXiv identifiers 26xx are 2026 papers I could not cross-check beyond arXiv pages or search summaries. Several preprints are under 9 months old and unreviewed.
