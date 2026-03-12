# SLM Improvement Strategy

Systematic plan for improving SLM interpreter accuracy from 63.5% to 90%+.

## Baseline

- **Model**: gemma3:1b (Q4_K_M, 815MB, runs on M2 Metal)
- **Eval harness**: 63 labeled test cases across 14 action categories
- **Current accuracy**: 63.5% (40/63 pass, 22 fail, 1 parse error)
- **Target accuracy**: 90%+ (pass threshold: 80%)
- **Eval command**: `make eval-slm`

## Failure Analysis

| Category | Count | Examples | Root Cause |
|----------|-------|---------|------------|
| Single-char aliases | 8 | `n`, `s`, `l`, `i`, `a`, `?` | System prompt has no alias table |
| Item ID normalization | 3 | `rusty key` → should be `rusty_key` | SLM returns natural spacing despite snake_case rule |
| Verb confusion | 6 | `drop` → `take`, `wield` → `move`, `guard` → `attack` | No grounding in valid game state |
| Meta commands | 3 | `quit`, `exit`, `?` | SLM doesn't recognize game meta-verbs |
| Ambiguous intent | 2 | `examine room` → needs item, `look at potion` → should be examine | Overlapping verb semantics |

## Improvement Levers (in order)

### Lever 1: Context Injection

**What**: Pass game state into the user message so the SLM knows what's valid.

**Current user message**: just the raw input string (e.g., `"take sword"`)

**Proposed user message**:
```
Room: entrance
Items here: short_sword, rusty_key
Exits: south, west
Enemies: (none)
Equipment: (none)
Spells: fireball, heal

Player input: take sword
```

**Why this helps**: The SLM sees `short_sword` in the item list and can resolve
"sword" → `short_sword`. It sees valid exits and won't hallucinate directions.
The action space collapses from open-ended to a small enumeration.

**Expected impact**: Fixes item resolution failures, reduces verb confusion (SLM
can see what's actually possible), eliminates hallucinated targets.

**Cost**: ~100-200 extra tokens per request. Negligible on 1B model.

### Lever 2: System Prompt Engineering

**What**: Improve the static system prompt with alias tables, few-shot examples,
and explicit edge case handling.

**Additions**:
1. **Alias table**: `n=north, s=south, e=east, w=west, l=look, i=inventory, a=attack, ?=help, q=quit`
2. **Few-shot examples** (5-10 covering failure categories):
   ```
   Input: "n" → {"type":"move","direction":"north"}
   Input: "grab the key" → {"type":"take","item_id":"rusty_key"}
   Input: "?" → {"type":"help"}
   ```
3. **Item ID format rule reinforcement**: "Always use snake_case: spaces become underscores, strip articles (the, a, an)."
4. **Meta command emphasis**: "quit, exit, q all map to {"type":"quit"}. help, ? map to {"type":"help"}."

**Expected impact**: Fixes single-char alias failures (~8 cases), meta command
failures (~3 cases), and reinforces ID normalization (~3 cases). Estimated +14
cases → ~86% accuracy.

**Cost**: ~200 extra system prompt tokens. One-time, amortized across all requests.

### Lever 3: Fine-Tuning

**What**: Supervised fine-tuning on our domain, then reinforcement learning using
the engine as a reward signal.

**Phase A — Supervised Fine-Tuning (SFT)**:
- Training data: the 63 eval cases + synthetic expansions (paraphrases, typos,
  partial names, mixed case). Target: 500-1000 labeled pairs.
- Fine-tune gemma3:1b using ollama's `Modelfile` + `FROM` + `ADAPTER` workflow,
  or use `unsloth` / `axolotl` for LoRA fine-tuning.
- Eval against held-out subset to measure generalization.

**Phase B — Reinforcement Learning from Engine Feedback (RLEF)**:
- The game engine is a deterministic reward function: submit the parsed action,
  observe success/failure. No human labeling needed.
- Reward signal: +1 if engine accepts the action and it matches intent, -1 if
  engine rejects it or action is wrong.
- Can use DPO (Direct Preference Optimization) or PPO with the engine in the loop.
- Generate training pairs by: (1) sample N completions per input at temperature>0,
  (2) run each through the engine, (3) preferred = accepted action, rejected = failed action.

**Expected impact**: Domain-specialized 1B model should reach 95%+ on our narrow
action space. The task is well-defined (15 verbs, bounded targets) — ideal for
fine-tuning.

**Cost**: Requires training infrastructure (GPU time for LoRA, ~1 hour on
consumer GPU). Ongoing cost: retrain when action space changes.

### Lever 4: Bigger Model (last resort)

**What**: Move to gemma3:4b, llama3.2:3b, or phi3:mini.

**When**: Only if levers 1-3 plateau below 90% accuracy.

**Trade-offs**: 2-4x inference latency, 2-4x memory. On M2 with 16GB unified
memory, a 4B Q4 model (~2.5GB) fits comfortably but response time increases
from ~600ms to ~1.5-2s.

**Eval command**: `make eval-slm SLM_MODEL=gemma3:4b`

## Architecture Note: Fallback Chain

Current chain: **SLM → Rules** (SLM tries first, falls back to rules on failure).

Consider flipping to **Rules → SLM**: route exact-match inputs (aliases, known
verbs) through the rules interpreter first (instant, deterministic), only invoke
the SLM for inputs the rules parser can't handle. Benefits:
- Eliminates latency for simple commands
- SLM only called for genuinely ambiguous natural language
- Single-char alias failures become irrelevant (rules catches them)

This is orthogonal to the four levers above and can be done independently.

## Measurement

Every change is measured against the eval harness:
```bash
make eval-slm                          # default model (gemma3:1b)
make eval-slm SLM_MODEL=gemma3:4b     # test bigger model
```

The eval harness exits non-zero below 80% accuracy, making it a CI gate once
we cross the threshold.

## Narrator SLM

The narrator SLM is separate from the interpreter SLM. It uses three specialized
prompts (room, moment, examine) at temperature 0.7. Improvement strategy is
similar but lower priority — template narration is a solid fallback, and
narrator quality is subjective (no accuracy metric). Future work: human
preference eval for narrator output quality.
