#!/usr/bin/env bash
#
# Requires bash 4+ (mapfile). On macOS, /bin/bash is 3.2; use Homebrew bash.
#
# Tests for validate-ai-report-default.sh.
#
# Two layers:
#   1. Repository-state invariant: this repository's reusable workflow must
#      contain exactly one parsable gaze-version default in canonical tag
#      form, so the release gate can never silently stop parsing it.
#   2. Synthetic cases: each case writes a controlled workflow file and
#      asserts the validator's exit status.

set -euo pipefail
export LC_ALL=C

repository_root="$(git rev-parse --show-toplevel)"
validator="$repository_root/.github/scripts/validate-ai-report-default.sh"
temporary_directory="$(mktemp -d)"
trap 'rm -rf "$temporary_directory"' EXIT

failures=0

# --- Repository-state invariant --------------------------------------------

# A release tag that cannot match yields exit 1 only when exactly one default
# was parsed; exit 2 means the workflow's default is no longer parsable.
repo_status=0
"$validator" -- v0.0.0-invariant-probe \
  "$repository_root/.github/workflows/reusable_gaze_ai_report.yml" >/dev/null 2>&1 || repo_status=$?
if [[ "$repo_status" -ne 1 ]]; then
  printf 'FAIL: repository workflow gaze-version default is not parsable (status %d)\n' "$repo_status" >&2
  failures=$((failures + 1))
fi

# --- Synthetic cases ---------------------------------------------------------

write_workflow() {
  local path="$temporary_directory/$1.yml"
  printf '%s\n' "$2" >"$path"
  printf '%s\n' "$path"
}

run_case() {
  local name="$1" tag="$2" file="$3" expected_status="$4"
  local status=0
  "$validator" -- "$tag" "$file" >/dev/null 2>&1 || status=$?
  if [[ "$status" -ne "$expected_status" ]]; then
    printf 'FAIL: %s: expected status %d, got %d\n' "$name" "$expected_status" "$status" >&2
    failures=$((failures + 1))
  else
    printf 'ok: %s\n' "$name"
  fi
}

inputs_header='on:
  workflow_call:
    inputs:'

matching="$(write_workflow matching "$inputs_header
      gaze-version:
        description: \"gaze release tag\"
        required: false
        # keep equal to the release tag
        default: \"v2.4.0\"
        type: string
      go-version:
        default: \"1.25\"")"
run_case "double-quoted default matches" v2.4.0 "$matching" 0
run_case "default differs from tag" v2.5.0 "$matching" 1
run_case "pre-release tag requires exact default" v2.4.0-rc.1 "$matching" 1

single_quoted="$(write_workflow single "$inputs_header
      gaze-version:
        default: 'v2.4.0'  # trailing comment")"
run_case "single-quoted default with comment matches" v2.4.0 "$single_quoted" 0

unquoted_pre="$(write_workflow unquoted "$inputs_header
      gaze-version:
        default: v2.5.0-rc.1")"
run_case "unquoted pre-release default matches" v2.5.0-rc.1 "$unquoted_pre" 0

sibling_default="$(write_workflow sibling "$inputs_header
      gaze-version:
        required: false
      go-version:
        default: \"v2.4.0\"")"
run_case "sibling input default is not attributed" v2.4.0 "$sibling_default" 2

duplicate="$(write_workflow duplicate "$inputs_header
      gaze-version:
        default: \"v2.4.0\"
jobs:
  other:
    with:
      gaze-version:
        default: \"v2.4.0\"")"
run_case "duplicate gaze-version defaults" v2.4.0 "$duplicate" 2

run_case "missing workflow file" v2.4.0 "$temporary_directory/absent.yml" 2
run_case "malformed release tag" 2.4.0 "$matching" 2

usage_status=0
"$validator" v2.4.0 >/dev/null 2>&1 || usage_status=$?
if [[ "$usage_status" -ne 2 ]]; then
  printf 'FAIL: missing -- separator: expected status 2, got %d\n' "$usage_status" >&2
  failures=$((failures + 1))
else
  printf 'ok: missing -- separator\n'
fi

if [[ "$failures" -ne 0 ]]; then
  printf '%d case(s) failed\n' "$failures" >&2
  exit 1
fi
printf 'all validate-ai-report-default cases passed\n'
