#!/usr/bin/env bash
set -euo pipefail

ethos=$1
scratch=$2

# First-run Ethos storage repair can update .gitignore in the disposable checkout.
if ! git diff --quiet -- .gitignore; then
    git archive HEAD .gitignore | tar -xf -
fi
[[ -z "$(git status --porcelain)" ]]

printf '%s\n' '{"session_id":"replay","agent_id":"worker","agent_type":"implementer"}' |
    HOME="$scratch" ETHOS_SESSION=replay ETHOS_AGENT_ID=worker CLAUDE_AGENT_TYPE=implementer \
    "$ethos" hook subagent-stop > "$scratch/replay.json"
if [[ -s "$scratch/replay.json" ]]; then
    cat "$scratch/replay.json" >&2
    exit 1
fi
log="$scratch/.punt-labs/ethos/logs/ethos.log"
grep -F 'stage=handoff check=handoff-gate outcome=launched ' "$log"
grep -F 'stage=handoff check=handoff-gate outcome=completed ' "$log"
