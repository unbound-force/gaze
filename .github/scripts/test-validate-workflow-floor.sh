#!/usr/bin/env bash
#
# Tests for validate-workflow-floor.sh.
#
# Two layers:
#   1. Repository-state invariants — this repository itself must stay
#      governed by the floor validation. If the minWorkflowTag constant
#      moves or the reusable workflow disappears, the validator's
#      legacy-skip path would silently skip releases; fail here instead.
#   2. Synthetic-repo cases — each case builds a throwaway git repo with a
#      controlled scaffold.go, workflow file, and tag layout, then asserts
#      the validator's exit status.

set -euo pipefail
export LC_ALL=C

repository_root="$(git rev-parse --show-toplevel)"
validator="$repository_root/.github/scripts/validate-workflow-floor.sh"
temporary_directory="$(mktemp -d)"
trap 'rm -rf "$temporary_directory"' EXIT

workflow_file=".github/workflows/reusable_gaze_ai_report.yml"

# --- Repository-state invariants -------------------------------------------

floor_lines=()
mapfile -t floor_lines < <(grep -E '^[[:space:]]*const minWorkflowTag = "[^"]*"' \
  "$repository_root/internal/scaffold/scaffold.go")
if [[ ${#floor_lines[@]} -ne 1 ]]; then
  printf 'expected exactly one minWorkflowTag declaration in internal/scaffold/scaffold.go, found %d\n' \
    "${#floor_lines[@]}" >&2
  exit 1
fi
repo_floor="$(printf '%s\n' "${floor_lines[0]}" | sed -E 's/^[[:space:]]*const minWorkflowTag = "([^"]*)".*$/\1/')"
if [[ ! "$repo_floor" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?$ ]]; then
  printf 'malformed minWorkflowTag value in internal/scaffold/scaffold.go: %s\n' "$repo_floor" >&2
  exit 1
fi
if [[ ! -f "$repository_root/$workflow_file" ]]; then
  printf 'repository is missing %s\n' "$workflow_file" >&2
  exit 1
fi

# --- Synthetic-repo helpers --------------------------------------------------

repo=""

init_repo() {
  repo="$temporary_directory/$1"
  mkdir -p "$repo"
  git -C "$repo" init -q
  git -C "$repo" config user.email test@example.com
  git -C "$repo" config user.name "Gaze Test"
}

write_scaffold() {
  mkdir -p "$repo/internal/scaffold"
  if [[ -n "$1" ]]; then
    printf 'package scaffold\n\n%s\n' "$1" >"$repo/internal/scaffold/scaffold.go"
  else
    printf 'package scaffold\n' >"$repo/internal/scaffold/scaffold.go"
  fi
}

write_workflow() {
  mkdir -p "$repo/.github/workflows"
  printf 'name: Reusable Gaze AI Report\n' >"$repo/$workflow_file"
}

commit_repo() {
  git -C "$repo" add -A
  git -C "$repo" commit -qm "test commit"
}

run_case() {
  local name="$1"
  local tag="$2"
  local case_repo="$3"
  local expected_status="$4"

  set +e
  "$validator" -- "$tag" "$case_repo" >/dev/null 2>&1
  actual_status=$?
  set -e

  if [[ "$actual_status" -ne "$expected_status" ]]; then
    printf '%s: expected status %s, got %s\n' \
      "$name" "$expected_status" "$actual_status" >&2
    exit 1
  fi
}

# --- Synthetic-repo cases ----------------------------------------------------

# Standard floor-bearing repo: floor v2.3.0, workflow present, floor tag
# exists at a commit that contains the workflow.
init_repo standard
write_scaffold 'const minWorkflowTag = "v2.3.0"'
write_workflow
commit_repo
git -C "$repo" tag v2.3.0
standard_repo="$repo"

run_case below-floor v2.2.9 "$standard_repo" 1
run_case at-floor v2.3.0 "$standard_repo" 0
run_case above-floor-valid v2.4.0 "$standard_repo" 0
run_case floor-prerelease v2.3.0-rc1 "$standard_repo" 1
run_case later-prerelease v2.4.0-rc1 "$standard_repo" 0
run_case malformed-tag release-2 "$standard_repo" 2

# Above the floor, but the floor tag does not exist in the repository.
init_repo untagged-floor
write_scaffold 'const minWorkflowTag = "v2.3.0"'
write_workflow
commit_repo
run_case above-floor-missing-floor-tag v2.4.0 "$repo" 1

# Above the floor, but the floor tag points at a commit that predates the
# workflow.
init_repo floor-predates-workflow
write_scaffold 'const minWorkflowTag = "v2.3.0"'
commit_repo
git -C "$repo" tag v2.3.0
write_workflow
commit_repo
run_case above-floor-tag-predates-workflow v2.4.0 "$repo" 1

# Legacy tree predating floor governance: no constant, validator skips.
init_repo legacy
write_scaffold ''
write_workflow
commit_repo
run_case legacy-no-constant v2.4.0 "$repo" 0

# Malformed floor value in the constant.
init_repo malformed-floor
write_scaffold 'const minWorkflowTag = "banana"'
write_workflow
commit_repo
run_case malformed-floor v2.4.0 "$repo" 1

# Duplicate floor declarations.
init_repo duplicate-floor
write_scaffold 'const minWorkflowTag = "v2.3.0"
const minWorkflowTag = "v2.4.0"'
write_workflow
commit_repo
run_case duplicate-floor v2.4.0 "$repo" 1

# Floor-bearing tree missing the workflow the floor points at — even a tag
# exactly at the floor must not pass.
init_repo floor-without-workflow
write_scaffold 'const minWorkflowTag = "v2.3.0"'
commit_repo
run_case floor-without-workflow v2.3.0 "$repo" 1

printf 'workflow-floor and repository-state validation passed: 12 cases\n'
