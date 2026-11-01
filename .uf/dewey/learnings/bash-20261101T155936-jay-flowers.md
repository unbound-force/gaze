---
tag: bash
author: jay-flowers
category: gotcha
created_at: 2026-11-01T15:59:36Z
identity: bash-20261101T155936-jay-flowers
tier: draft
---

Bash process substitution loses the inner command's exit status: `mapfile -t lines < <(grep pattern file)` cannot distinguish "no match" (grep exit 1, the intended legacy-skip signal) from "could not read file" (grep exit 2, a repo-state failure) — under `set -euo pipefail` the substitution's status is silently discarded and the fail-open path goes unnoticed. When a grep result drives a skip-vs-fail decision, redirect to a temp file and capture the status explicitly (`matches="$(mktemp)"; trap 'rm -f' EXIT; grep ... >"$matches" || grep_status=$?`; only status > 1 is an error), then mapfile from the file. Verified in gaze's validate-workflow-floor.sh (design D2 post-review revision), which closed the gate's only reachable fail-open path this way after a council pass flagged it; the sibling validate-module-major.sh uses plain process substitution because awk over a missing go.mod fails loudly anyway.
