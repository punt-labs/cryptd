# Research: TUI frameworks, terminal graphics protocols, immediate-mode UI, and LLM generatability for crypt
**Date:** 2026-11-09
**Request:** Technical build research for crypt's display/client layer: TUI frameworks, terminal emulators and image protocols, immediate-mode/2D UI frameworks, decision axes, plus a GENERATABILITY axis (how well an LLM can generate the display at runtime).
**Claims investigated:** 4 areas plus generatability

Verification: "fetched" = I read the page. "search" = from search-result summaries only; verify before quoting. Generatability rankings are my own analysis built on the cited facts; no source ranks frameworks by LLM-generatability.

## Evidence Found

### Area 1: TUI frameworks

**Bubble Tea / Lip Gloss / Bubbles v2 (Go, Charm).** Verdict: FITS as the current base; best Go option for a text-first client, weak as an image framework itself.
- v2 released 2026-02-23 (search); import paths moved to charm.land/*; latest seen v2.1.1 (search). Elm-style model/update/view; View() returns a tea.View struct (fetched, Discussion #1374).
- New "Cursed Renderer" based on ncurses-style diffing; synchronized updates (mode 2026) and Unicode mode 2027 on by default; automatic color downsampling; split mouse messages (click, release, wheel, motion); progressive keyboard enhancement (kitty keyboard protocol) where the terminal supports it (fetched).
- Images: the v2 discussion lists no image-specific features (fetched). A search summary claims "inline image rendering" support in v2; the primary discussion does not back that, so treat it as unverified. Images need a third-party library such as go-termimg (below).
- Layout: Lip Gloss does string-based layout (JoinHorizontal/Vertical, borders, sizing). No constraint solver or widget tree; large apps need hand-composed sub-models and message plumbing (search, tertiary).
- Weakness: terminal queries (XTVERSION, capability) are not supported by all terminals (fetched).

**tview / tcell (Go).** Verdict: FITS for form/table-heavy UIs; no image story.
- tcell is the low-level cell screen; tview is a retained, imperative widget tree (search). Full-window only, no inline mode (search, tertiary). Supports mouse. Search found no image support for either.
- Tcell cell-buffer model gives the renderer no hook for pixel graphics; an image library would have to write escape sequences around tcell's buffer (my inference, not sourced).

**termui, gocui (Go).** Not researched beyond the name. Gap.

**ratatui (Rust).** Immediate-mode: the app redraws the whole UI each frame into a buffer, which is diffed to the terminal. Image support through the third-party ratatui-image crate (fetched): Sixel, Kitty, iTerm2, with env-var guess then terminal query for protocol detection, font-size detection, and halfblock fallback. Its own matrix marks Alacritty, Konsole, Contour, ctx, Warp as not working; WezTerm works bug-free only with iTerm2; Sixel on the last line causes a scroll (#57); image resize blocks and should be offloaded. Not Go, so only a reference point; it shows what a mature detect-and-fall-back abstraction looks like.

**Textual (Python, Rich).** Retained widget tree with CSS-like styling, DOM and mouse support. No native image protocol support; textual-image adds Kitty (TGP), Sixel, halfcell, Unicode with auto selection (search). Sixel in Textual is described as not performant; textual-serve does not support it (search). Reference point only: Textual's CSS and widget DOM is the most declarative of the group.

**ncurses (C).** Classic cell-based API; no image support; not researched further.

### Area 2: Terminal graphics protocols and terminals

**Kitty graphics protocol (fetched from the spec).** APC escape sequences. Transmission modes: direct (base64, chunks up to 4096 bytes), file, temp file, shared memory. PNG sent as-is (f=100), or raw RGB/RGBA (search). Images can be placed multiple times, resized or moved by re-placing, and drawn below text with negative z-index. Unicode placeholders (U+10EEEE, v0.28.0+) let tmux and vim reposition images as ordinary text. Terminals listed in the spec: kitty, Ghostty, Konsole, WezTerm, iTerm2, Warp, patched st, wayst, xterm.js, plus some SSH clients.
- Caveat: iTerm2 and Warp are listed by the kitty spec as implementers, but a third-party compatibility matrix (search, tertiary) marks iTerm2 as no kitty protocol and ratatui-image marks Warp as not working. Conflicting; verify per terminal before relying on it.

**Sixel (fetched, arewesixelyet.com).** Supported: foot, iTerm2, konsole, mintty, mlterm, WezTerm, Windows Terminal, xterm, Contour, Rio, VS Code, xterm.js, Zellij, tmux, and others. Unsupported: Alacritty, Ghostty, kitty (deliberately), GNOME Terminal and other VTE terminals (stalled VTE issue #253), Terminal.app, Warp, PuTTY, Windows Console. Needs patch: st, Black Box, xfce-terminal. Windows Terminal Sixel arrived in Preview 1.22, mainline 1.23 (search). Sixel is palette-based (typically 256 colors) so an encoder must quantize and dither; it is the slowest protocol in go-termimg's numbers (about 90 ms versus about 800 microseconds for halfblocks, fetched).

**iTerm2 inline images (fetched, iterm2.com docs).** OSC 1337 File=...: base64 payload. Accepts any format macOS decodes (PNG, GIF, PDF, etc.). Width/height in cells, px, percent, or auto. 1 MiB sequence limit; older tmux caps at 256 bytes. Multipart mode from 3.5. Supported by iTerm2 and WezTerm; mintty also (search).

**Terminal-by-terminal stance (macOS and Linux relevant).**
- macOS Terminal.app: no inline images (search; arewesixelyet agrees on Sixel).
- iTerm2: iTerm2 protocol and Sixel.
- Ghostty: kitty protocol; no Sixel (arewesixelyet).
- WezTerm: iTerm2, Sixel, kitty (partial; ratatui-image says bug-free only with iTerm2).
- kitty: kitty protocol only.
- foot: Sixel.
- xterm: Sixel (needs enabling).
- Alacritty: none; maintainers have repeatedly declined Sixel (issue #910) and kitty graphics (search). A fork with patches exists.
- Windows Terminal: Sixel only (1.22+), no kitty graphics (search).
- GNOME Terminal and other VTE terminals: none.

**tmux (search).** Native Sixel from 3.4 if built with --enable-sixel (Homebrew builds enable it). No native kitty protocol; kitty and iTerm2 sequences only pass through with `allow-passthrough on` and are not clipped or redrawn, so images can stick after scroll. Kitty Unicode placeholders are the workaround. Passthrough bypasses tmux filtering, a security tradeoff.

**Go libraries.**
- blacktop/go-termimg (fetched): Kitty, Sixel, iTerm2, halfblocks; auto-detect with parallel detection; Bubble Tea ImageWidget; tmux passthrough; MIT; 68 stars, no tagged releases, 148 commits. Kitty Unicode-placeholder and relative placement marked "not recommended for production." Maturity: young, single maintainer.
- BourgeoisBear/rasterm and kenshaw/rasterm (search): encode to iTerm2, Kitty, Sixel. BourgeoisBear's detection uses $TERM_PROGRAM, which is not passed through tmux or ssh; kenshaw's fork asks tmux for the outer terminal and returns ErrTermGraphicsNotAvailable when passthrough is off. Smaller APIs.

**Practical read.** Fragmented. No single protocol covers the main macOS and Linux terminals: kitty-protocol terminals (kitty, Ghostty) lack Sixel; Sixel terminals mostly lack kitty. iTerm2 protocol plus Sixel covers iTerm2 and WezTerm. Terminal.app, Alacritty, GNOME Terminal, and any plain ssh-to-server session of unknown terminal get nothing. Safe baseline: text and Unicode block/halfblock rendering (works everywhere, lossy). Enhanced path: detect (DA1 query for Sixel attribute 4, env vars, kitty query) then pick kitty, iTerm2, or Sixel, falling back to halfblocks. Detection is itself unreliable under tmux and ssh.

### Area 3: Immediate-mode and 2D UI frameworks

- **Dear ImGui via giu / cimgui-go (Go).** Immediate-mode; giu builder-pattern API; requires cgo and a C/C++ toolchain (search). Tooling-oriented look; strong text and font handling in giu (search). Poor fit for a prose-heavy, atmospheric text game; fine for debug panels.
- **raylib-go.** cgo binding to raylib (search). Simple 2D/3D drawing; no built-in UI or rich text layout beyond basic fonts.
- **SDL2 / go-sdl2.** Not researched in this pass. Gap.
- **Ebitengine (Go).** Pure Go on desktop from v2.10, so no C compiler on Windows/macOS/Linux/FreeBSD; Android and iOS still need cgo; also WebAssembly (search). text/v2 package exists but text rendering is basic and IME support is experimental (search). Pixel/sprite oriented: you build your own widgets and text layout.
- **Gio (pure Go immediate-mode UI).** Surfaced incidentally as a cgo-free option (search); not evaluated.
- **What leaving the terminal costs.** Loses SSH play, zero-install, and the text identity; adds a native window, fonts, input, accessibility, and for ImGui/raylib a C toolchain. Gains: guaranteed image display on every platform, pixel-exact maps, no protocol fragmentation. A hybrid (terminal client plus an optional image/map window or browser tab fed by the same JSON-RPC events) is an option; no source evaluates it for text games.

### Area 4: Decision axes (no recommendation)

1. Terminal-only with protocol detection: keeps text identity, SSH-friendly, zero-install; images only on kitty/iTerm2/Sixel terminals; tmux and ssh degrade detection; needs halfblock or ASCII fallback for everything else.
2. Pure window (Ebiten, Gio, ImGui): uniform images, loses text identity and SSH; cgo cost for ImGui/raylib, none for Ebiten on desktop.
3. Hybrid: terminal for text, separate viewer (local window or browser) for generated images and maps; costs a second surface and sync.
Cross-cutting axes: image fidelity everywhere, cgo, SSH play, tmux compatibility, generatability (below), text-identity.

## Generatability (LLM generates the display at runtime)

Rating scale: HIGH = LLM emits plain data or text that the client validates and renders; MEDIUM = LLM emits data but the client must handle protocol or layout risk; LOW = LLM would have to write or drive code or pixels. Ratings are my analysis.

**Evidence base.**
- LLMs generate task-specific UIs that humans prefer (Chen et al., arXiv 2508.19227; Leviathan et al. 2604.09577, both in the existing file research-2026-11-09-generative-ui.md). Both use HTML or structured representations with reusable widgets, which supports the "emit data against a fixed widget vocabulary" pattern. Google's pages equal human experts in about 50% of cases.
- LLMs are weaker at constructing a spatial layout than at reasoning over one. Text2Space (arXiv 2604.14641, search): Qwen3-30B-A3B falls from 75.0% on text-only spatial reasoning to 66.6% when it must first build an ASCII layout; fine-tuning on layout construction adds over 7 points. PlanarBench (arXiv 2606.02010, search): models must produce crossing-free ASCII drawings from an edge list; checking is mechanical (parse, verify); best model scored 159.5 of 199. Visualization-of-Thought (arXiv 2404.03622, search): prompting GPT-4 to draw ASCII lifted tracking rate by 78.5 points. So ASCII maps are generatable and mechanically checkable, but not reliably correct in one shot.
- Dungeon-map generation by name: no benchmark found. Moonshine and Five-Dollar (in the existing file) are trained text-to-map models, not general LLM output.
- RPGBench (existing file): models keep mechanics inconsistent over long sessions without engine-held state. crypt's engine-owns-state split is the mitigation; the same logic applies to maps: the engine should own the map graph and the LLM should only style it.

**Ranking, best to worst for runtime LLM generation (with strengths and weaknesses).**

| Rank | Target | Generatability | Why |
|---|---|---|---|
| 1 | Plain text and Unicode/ASCII map rendered by the engine from a map graph, LLM supplies descriptions and a style hint | HIGH | LLM never emits layout; no protocol risk; works in every terminal including readline. Loses visual richness. |
| 2 | Bubble Tea with a fixed schema (JSON: panels, text spans, colors, a map object, an image reference) | HIGH | Client maps validated JSON to Lip Gloss styles; schema rejects bad output; LLM never writes Go. Lip Gloss is string-composed so any layout is bounded by what the client's renderer implements. |
| 3 | LLM-authored ASCII map emitted directly | MEDIUM | Plausible and checkable (parse for connectivity, size, legal glyphs) per the PlanarBench approach, but Text2Space shows construction is a weak point. Needs validation plus an engine fallback. |
| 4 | Image bytes handed to a terminal protocol (kitty PNG f=100, direct base64) | MEDIUM | Generation side is easy: an image model returns PNG, the client base64-encodes and chunks it, no re-encoding for kitty (search). Risk is entirely display-side: protocol detection, tmux, ssh, and the terminals with no support. Sixel needs client-side quantization and dithering, and iTerm2 caps at 1 MiB per sequence. The LLM does not touch escape codes if the client does the encoding; an LLM must never emit raw escape sequences (injection and corruption risk; my inference). |
| 5 | tview widget tree | MEDIUM-LOW | Imperative Go API; an LLM would have to emit code or the client would need its own schema-to-widget layer, same as Bubble Tea but with less benefit. Textual and its CSS/DOM are the most declarative in principle, but Python and no image support. |
| 6 | ratatui, ImGui, raylib, SDL2, Ebiten | LOW | Imperative or immediate-mode APIs built around compiled code. Runtime generation would need either a scripting layer, a hand-built schema interpreter, or the LLM writing code that is then compiled or interpreted (unsafe and slow). They display images well (raylib, Ebiten, SDL textures), but the LLM supplies only assets and data, not layout. Same schema-driven pattern as rank 2, at higher cost. |

**Best suited to "LLM generates the display at runtime":** a schema-driven client (JSON the LLM fills, the client validates and renders) on top of any framework, plus the engine rendering maps from state. Framework choice matters less than whether the client exposes a bounded widget vocabulary. The image path is MEDIUM: easy to produce bytes, hard to guarantee they display. Not evidenced: any study of generated images or maps staying consistent across a session; image-generation latency for a game turn (Unbounded reports about 1 second per scene with distillation versus about 5 seconds naive, existing file).

## Bibliography Entries

```bibtex
@online{kitty2026graphics,
  author  = {Goyal, Kovid},
  title   = {Terminal graphics protocol},
  year    = {2026},
  url     = {https://sw.kovidgoyal.net/kitty/graphics-protocol/},
  urldate = {2026-11-09},
  note    = {Primary spec: transmission modes, placements, Unicode placeholders, list of implementing terminals.},
}

@online{iterm2images,
  author  = {{iTerm2}},
  title   = {Inline Images Protocol},
  year    = {2026},
  url     = {https://iterm2.com/documentation-images.html},
  urldate = {2026-11-09},
  note    = {Primary spec: OSC 1337 File, multipart, size units, 1 MiB and tmux limits.},
}

@online{arewesixelyet,
  author  = {{Are We Sixel Yet}},
  title   = {Are We Sixel Yet?},
  year    = {2026},
  url     = {https://www.arewesixelyet.com/},
  urldate = {2026-11-09},
  note    = {Community-maintained Sixel support table by terminal; secondary source.},
}

@online{charm2026bubbletea2,
  author  = {{Charm}},
  title   = {Bubble Tea v2: What's New in v2 (Discussion 1374)},
  year    = {2026},
  url     = {https://github.com/charmbracelet/bubbletea/discussions/1374},
  urldate = {2026-11-09},
  note    = {Primary: Cursed Renderer, declarative View, mouse and keyboard changes. Lists no image features.},
}

@online{charm2026bubbletearelease,
  author  = {{Charm}},
  title   = {Bubble Tea v2.0.0 release},
  year    = {2026},
  url     = {https://github.com/charmbracelet/bubbletea/releases/tag/v2.0.0},
  urldate = {2026-11-09},
  note    = {Release page; search-result only, not fetched.},
}

@online{blacktop2025termimg,
  author  = {blacktop},
  title   = {go-termimg: Modern terminal image library for Go},
  year    = {2025},
  url     = {https://github.com/blacktop/go-termimg},
  urldate = {2026-11-09},
  note    = {Go library for Kitty, Sixel, iTerm2, halfblocks with Bubble Tea widget and tmux passthrough. 68 stars, no tagged releases (fetched).},
}

@online{bourgeoisbear2024rasterm,
  author  = {BourgeoisBear},
  title   = {rasterm: encode images to iTerm / Kitty / SIXEL inline graphics protocols},
  year    = {2024},
  url     = {https://github.com/BourgeoisBear/rasterm},
  urldate = {2026-11-09},
  note    = {Go encoder with a terminal feature matrix; detection relies on TERM_PROGRAM. Search-result only.},
}

@online{kenshaw2025rasterm,
  author  = {Kenshaw},
  title   = {rasterm (fork with tmux-aware detection)},
  year    = {2025},
  url     = {https://github.com/kenshaw/rasterm},
  urldate = {2026-11-09},
  note    = {Go encoder; asks tmux for the outer terminal; returns ErrTermGraphicsNotAvailable when passthrough is off. Search-result only.},
}

@online{benjajaja2025ratatuiimage,
  author  = {benjajaja},
  title   = {ratatui-image},
  year    = {2025},
  url     = {https://github.com/benjajaja/ratatui-image},
  urldate = {2026-11-09},
  note    = {Rust image widget for ratatui: protocol detection, halfblock fallback, per-terminal compatibility matrix (fetched).},
}

@online{lnqs2025textualimage,
  author  = {lnqs},
  title   = {textual-image: Render images in the terminal with Textual and rich},
  year    = {2025},
  url     = {https://github.com/lnqs/textual-image},
  urldate = {2026-11-09},
  note    = {Python image widget for Textual/Rich; TGP, Sixel, halfcell, Unicode. Search-result only.},
}

@online{ms2025winterm122,
  author  = {{Microsoft}},
  title   = {Windows Terminal Preview 1.22 Release},
  year    = {2025},
  url     = {https://devblogs.microsoft.com/commandline/windows-terminal-preview-1-22-release/},
  urldate = {2026-11-09},
  note    = {Announces Sixel support; no kitty graphics. Search-result only.},
}

@online{alacritty910sixel,
  author  = {{Alacritty}},
  title   = {Sixel support (issue 910)},
  url     = {https://github.com/alacritty/alacritty/issues/910},
  urldate = {2026-11-09},
  note    = {Maintainer stance against graphics protocols; cited via search summary, not fetched.},
}

@online{vtdn2026tmuxpassthrough,
  author  = {{VT Developer Network}},
  title   = {tmux passthrough},
  year    = {2026},
  url     = {https://vtdn.dev/docs/dcs/tmux-passthrough/},
  urldate = {2026-11-09},
  note    = {DCS passthrough mechanics and allow-passthrough; search-result only.},
}

@online{hajimehoshi2026ebiten,
  author  = {Hoshi, Hajime},
  title   = {Ebitengine: a dead simple 2D game engine for Go},
  year    = {2026},
  url     = {https://github.com/hajimehoshi/ebiten},
  urldate = {2026-11-09},
  note    = {Pure Go on desktop from v2.10; cgo on Android and iOS. Search-result only.},
}

@online{allendang2025cimguigo,
  author  = {Dang, Allen},
  title   = {cimgui-go: Go wrapper for Dear ImGui via cimgui},
  year    = {2025},
  url     = {https://github.com/AllenDang/cimgui-go},
  urldate = {2026-11-09},
  note    = {Binding underneath giu; requires cgo. Search-result only.},
}

@online{allendang2025giu,
  author  = {Dang, Allen},
  title   = {giu: Cross-platform GUI for Go based on Dear ImGui},
  year    = {2025},
  url     = {https://github.com/AllenDang/giu},
  urldate = {2026-11-09},
  note    = {Immediate-mode Go GUI; requires CGO_ENABLED=1. Search-result only.},
}

@misc{text2space2026,
  title         = {Learning to Draw {ASCII} Improves Spatial Reasoning in Language Models},
  year          = {2026},
  eprint        = {2604.14641},
  archiveprefix = {arXiv},
  url           = {https://arxiv.org/abs/2604.14641},
  note          = {Text2Space: ASCII layout construction is a bottleneck (75.0 to 66.6 percent on one model); also at COLM 2026. Authors not verified; search-result only.},
}

@misc{planarbench2026,
  title         = {{PlanarBench}: Evaluating {LLM} Spatial Reasoning via Planar Graph Drawing},
  year          = {2026},
  eprint        = {2606.02010},
  archiveprefix = {arXiv},
  url           = {https://arxiv.org/abs/2606.02010},
  note          = {Mechanically checked ASCII drawing generation; best model 159.5 of 199. Authors not verified; search-result only.},
}

@misc{wu2024vot,
  author        = {Wu, Wenshan and others},
  title         = {Mind's Eye of {LLMs}: Visualization-of-Thought Elicits Spatial Reasoning in Large Language Models},
  year          = {2024},
  eprint        = {2404.03622},
  archiveprefix = {arXiv},
  url           = {https://arxiv.org/abs/2404.03622},
  note          = {ASCII-art visualization prompting raised GPT-4 tracking rate by 78.5 points. Author list not verified; search-result only.},
}
```

Generative-UI and game-content sources (Leviathan 2026, Chen 2025, RPGBench, Unbounded, Moonshine) are in /Users/jfreeman/Coding/punt-labs/cryptd/research/research-2026-11-09-generative-ui.md; reuse those keys.

## Research Gaps

**Claim**: Bubble Tea v2 has built-in inline image support.
**What's missing**: One search summary says yes; the v2 discussion (fetched) lists none. Read the Bubble Tea v2 source and charmbracelet/x (ansi, kitty, sixel packages) directly.
**Suggested action**: Check the repo before assuming an image path through Charm libraries alone.

**Claim**: termui, gocui, SDL2/go-sdl2, raylib-go details, Gio.
**What's missing**: Not researched in this pass.
**Suggested action**: Fetch each repo README if any becomes a candidate.

**Claim**: Per-terminal protocol support (iTerm2 and Warp with kitty protocol; WezTerm quirks).
**What's missing**: Sources conflict (kitty spec lists iTerm2 and Warp; third-party matrices and ratatui-image say otherwise). Support also changes release to release.
**Suggested action**: Test on the target terminals (Ghostty, iTerm2, WezTerm, Terminal.app, kitty) and keep a support matrix in the repo.

**Claim**: Image protocol behavior inside tmux and over ssh.
**What's missing**: Only secondary sources; no first-party tmux docs read.
**Suggested action**: Fetch the tmux changelog and test passthrough with a crypt prototype.

**Claim**: LLMs generate coherent ASCII dungeon maps reliably.
**What's missing**: No dungeon-map benchmark found. Text2Space and PlanarBench suggest construction is error-prone but checkable. Author lists and details are from search summaries.
**Suggested action**: Build a validator (connectivity, bounds, glyph set) and measure crypt's own pass rate, or have the engine render maps from a graph.

**Claim**: Runtime image generation latency and cost fit a game turn; images stay consistent across a session.
**What's missing**: Unbounded gives one data point; nothing on consistency across a session.
**Suggested action**: Prototype and measure; treat as open.

**Claim**: Hybrid terminal-plus-window architecture works for text games.
**What's missing**: No source evaluating it.
**Suggested action**: Accept as an assumption and prototype if images in unsupported terminals matter.

Housekeeping: `research/` is not in /Users/jfreeman/Coding/punt-labs/cryptd/.gitignore (the file is managed by a rollout script, and I did not rewrite it). Add `research/` under the repo-specific section if the cache should stay uncommitted.
