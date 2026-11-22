# Advisory agent checks (DES-085)

Requires the next ethos release after v5.1.1. This migration was exercised
with ethos main `f4f884fb07e6a0c7381bc9c3cd05a84191a597e1`, built with
`make build` in `~/repos/ethos` (version `5.1.1-13-gf4f884f`). Put that binary
first on `PATH`, including when committing and starting Claude Code.

Project policy is `.punt-labs/ethos/checks/crypt-go.yaml`, selected by
`checks: crypt-go` in `.punt-labs/ethos.yaml`. Edit checks run in parallel
within ethos's 60-second stage cap:

- Go: gofmt is blocking; package vet and race-enabled tests are advisory.
  Their timeouts are 5, 20 and 30 seconds respectively. The defaults were
  raised to allow cold compilation; timeouts report advisory feedback for
  vet/tests rather than blocking partially completed changes or TDD.
- Shell: shellcheck is advisory, with the default 10-second timeout.
- Commit: blocking `make check`, 10-minute timeout (below the 15-minute cap).
- Handoff: blocking `make check`, 20-minute timeout (below the 30-minute cap).

Keep the full commit gate: warm `make check` took about six seconds in the
old-hook benchmark, inexpensive enough to enforce CLAUDE.md's existing
before-every-commit rule. Do not use mutating `gofmt {files}` in a hook;
`gofmt -l` observes without modifying the working tree or index.

## Measurements

Ubuntu Linux, 8 logical cores; Python `time.perf_counter()` wall time.
Three serial runs per cell, same files and PostToolUse payloads; only file
mtimes were changed, not contents. Go build caches were not cleared.

| File | Old median | New median | Runs each |
| --- | ---: | ---: | ---: |
| `internal/engine/engine.go` | 5.679 s | 1.417 s | 3 |
| `README.md` | 0.016 s | 0.010 s | 3 |

Old Go runs: 52.083, 5.675, 5.679 seconds (first includes cold builds).
New Go runs: 1.422, 1.417, 1.414 seconds. New Markdown runs:
0.011, 0.010, 0.010 seconds. All reported exit 0. Markdown was already
skipped by the old hook: this is not a full-gate speedup for Markdown.
An all-package race-test sweep took 4.33 seconds wall time; the slowest
reported package was `internal/inference` at 3.036 seconds. This supports
keeping package vet/tests advisory at edit, rather than formatting alone.
These measurements do not guarantee cold builds finish before the timeout.

The explicit `make check` run passed in 5.91 seconds; the configured
SubagentStop handoff hook passed in 5.76 seconds (one run each).

An initial candidate used a 20-minute commit timeout. The real post-edit
entry point rejected it with an advisory naming the 15-minute commit cap;
those runs are excluded from the new-hook measurements above. After fixing
the timeout, Go success produced no policy diagnostics, and the intentional
test failure below produced `hookSpecificOutput.additionalContext`, naming
`unit-tests`, `exit status 1` and `TestAdvisoryProbe`, with hook exit 0 and
no blocking decision. The real hook strictly decodes the entire selected
check set; there is no `ethos validate` command in this build.

## Reproduce

From a clean cryptd checkout, with the built ethos on PATH:

```bash
make -C "$HOME/repos/ethos" build
export PATH="$HOME/repos/ethos:$PATH"
export CLAUDE_PROJECT_DIR="$PWD"
mkdir -p .tmp
python3 - <<'PY'
import json
import os
import pathlib
import statistics
import subprocess
import time

root = pathlib.Path.cwd()
base = "276f76c242105a6f837728a1772ef0405199cd93"
old = subprocess.check_output(
    ["git", "show", f"{base}:.claude/agents/bwk.md"], text=True)
command = json.loads(next(line.split("command: ", 1)[1]
    for line in old.splitlines() if "command: " in line))
for file in ["internal/engine/engine.go", "README.md"]:
    os.utime(file, None)
    payload = json.dumps({"hook_event_name": "PostToolUse",
        "tool_name": "Edit", "tool_input": {"file_path": str(root / file)}})
    for label, argv in [
        ("before", ["sh", "-c", command]),
        ("after", ["ethos", "hook", "post-edit", "--agent", "bwk"]),
    ]:
        times = []
        for i in range(3):
            start = time.perf_counter()
            p = subprocess.run(argv, input=payload, text=True,
                capture_output=True)
            times.append(time.perf_counter() - start)
            print(label, file, i + 1, times[-1], "exit", p.returncode)
            print(p.stdout, p.stderr)
        print("median", statistics.median(times))
PY
/usr/bin/time -f 'make check wall: %e s' make check
```

Prove advisory failure (do not commit the probe; do not overwrite an
existing file of that name):

```bash
test ! -e internal/engine/advisory_probe_test.go || exit 1
cat > internal/engine/advisory_probe_test.go <<'GO'
package engine

import "testing"

func TestAdvisoryProbe(t *testing.T) {
	t.Fatal("intentional DES-085 advisory proof")
}
GO
printf '%s\n' "{\"hook_event_name\":\"PostToolUse\",\"tool_name\":\"Edit\",\"tool_input\":{\"file_path\":\"$PWD/internal/engine/advisory_probe_test.go\"}}" |
  ethos hook post-edit --agent bwk
printf 'advisory hook exit: %s\n' "$?"
rm internal/engine/advisory_probe_test.go
```

Regenerate agents through the same entry point as SessionStart:

```bash
printf '%s\n' '{"session_id":"cryptd-advisory-migration","hook_event_name":"SessionStart"}' |
  ethos hook session-start
printf '%s\n' '{"session_id":"cryptd-advisory-migration","agent_id":"advisory-handoff-probe","agent_type":"bwk"}' |
  /usr/bin/time -f 'handoff hook wall: %e s' ethos hook subagent-stop
```

The new generator also namespaces `baseline-ops` as `ethos:baseline-ops`
and `ethos-dev:baseline-ops` in all 19 generated agents. This is the only
agent change beyond the post-edit shim and its 70-second timeout. The three
unmarked reviewer files are preserved byte-for-byte. SessionStart also
removed a legacy sessions ignore entry; that unrelated tracked change was
reverted. SessionStart refreshes local Git hook shims, which are not tracked.

## Feedback trade-offs

Go edits no longer run repository-wide vet/tests, staticcheck or
markdownlint. Shell edits run shellcheck instead of the full gate. Edits
to YAML, module/workspace files and Makefiles no longer run the full gate.
Markdown remains unchecked at edit. All receive the full quick gate at
commit and handoff. Integration, E2E, acceptance and coverage were never
part of `make check` here; they remain in `make check-full`, not per edit.
The handoff hook applies to SubagentStop; mission-result integration is
not implemented by this ethos version.
